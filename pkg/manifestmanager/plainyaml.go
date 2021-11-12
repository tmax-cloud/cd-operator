package manifestmanager

import (
	"context"
	"encoding/json"
	"fmt"

	jsonpatch "github.com/evanphx/json-patch"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/cluster"
	"github.com/tmax-cloud/cd-operator/pkg/git"
	"github.com/tmax-cloud/cd-operator/pkg/httpclient"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type plainYamlManager struct {
	DefaultCli client.Client
	// Client for multi cluster
	TargetCli client.Client
	context.Context
	httpclient.HTTPClient
	GitCli git.Client
}

func NewPlainYamlManager(ctx context.Context, cli client.Client, httpCli httpclient.HTTPClient, gitCli git.Client) ManifestManager {
	return &plainYamlManager{
		DefaultCli: cli,
		TargetCli:  cli,
		Context:    ctx,
		HTTPClient: httpCli,
		GitCli:     gitCli,
	}
}

func (m *plainYamlManager) Sync(app *cdv1.Application, forced bool) error {
	if err := m.setTargetClient(app); err != nil {
		log.Error(err, "setTargetClient failed..")
		return err
	}

	var manifestInfos []string
	manifestInfos, err := m.GitCli.GetManifestInfos(app.Spec.Source.Path, app.Spec.Source.TargetRevision, manifestInfos)
	if err != nil {
		log.Error(err, "GetManifestURLList failed..")
		return err
	}
	oldDeployResources, err := getDeployResourceList(m.DefaultCli, app)
	if err != nil {
		log.Error(err, "GetDeployResourceList failed")
		return err
	}

	updatedDeployResources := make(map[string]*cdv1.DeployResource)

	for _, info := range manifestInfos {
		manifestRawobjs, err := m.GitCli.ObjectFromManifest(info, app.Spec.Destination.Namespace)
		if err != nil {
			log.Error(err, "Get object from manifest failed..")
			return err
		}

		for _, manifestRawobj := range manifestRawobjs {
			updatedDeployResource, err := updateDeployResource(m.DefaultCli, manifestRawobj, app)
			if err != nil {
				log.Error(err, "NewDeployResource failed..")
				return err
			}
			updatedDeployResources[updatedDeployResource.Name] = updatedDeployResource

			manifestModifiedObj, err := m.compareDeployWithManifest(app, manifestRawobj)
			if manifestModifiedObj == nil && err != nil {
				log.Error(err, "Compare deployed resource with manifest failed..")
				return err
			}
			if manifestModifiedObj != nil && (app.Spec.SyncPolicy.AutoSync || forced) {
				exist := (err == nil)
				if err := m.applyManifest(exist, manifestModifiedObj); err != nil {
					log.Error(err, "Apply manifest failed..")
					return err
				}
			}
		}
	}

	for _, oldDeployResource := range oldDeployResources.Items {
		if updatedDeployResources[oldDeployResource.Name] == nil {
			if err := m.clearApplicationResources(&oldDeployResource); err != nil {
				log.Error(err, "clearApplicationResources failed..")
				return err
			}
		}
	}

	if app.Status.Sync.Status == cdv1.SyncStatusCodeUnknown {
		app.Status.Sync.Status = cdv1.SyncStatusCodeSynced
	}

	return nil
}

func (m *plainYamlManager) Clear(app *cdv1.Application) error {
	if err := m.setTargetClient(app); err != nil {
		return err
	}

	deployedResourceList, err := getDeployResourceList(m.DefaultCli, app)
	if err != nil {
		return err
	}

	for _, deployedResource := range deployedResourceList.Items {
		if err := m.clearApplicationResources(&deployedResource); err != nil {
			return err
		}
	}

	return nil
}

func (m *plainYamlManager) compareDeployWithManifest(app *cdv1.Application, manifestObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	deployedObj := manifestObj.DeepCopy()
	if err := m.TargetCli.Get(m.Context, types.NamespacedName{
		Namespace: deployedObj.GetNamespace(),
		Name:      deployedObj.GetName()}, deployedObj); err != nil {
		if errors.IsNotFound(err) {
			app.Status.Sync.Status = cdv1.SyncStatusCodeOutOfSync
			return manifestObj, err
		}
		return nil, err
	}

	bytedDeployedObj, _ := deployedObj.MarshalJSON()
	bytedManifestObj, _ := manifestObj.MarshalJSON()

	patchedByte, _ := jsonpatch.MergePatch(bytedDeployedObj, bytedManifestObj)

	patchedObj := make(map[string]interface{})
	if err := json.Unmarshal(patchedByte, &patchedObj); err != nil {
		return nil, err
	}

	manifestObj.SetUnstructuredContent(patchedObj)
	if err := m.TargetCli.Update(m.Context, manifestObj, client.DryRunAll); err != nil {
		return nil, err
	}

	if fmt.Sprintf("%v", deployedObj) != fmt.Sprintf("%v", manifestObj) {
		app.Status.Sync.Status = cdv1.SyncStatusCodeOutOfSync
		log.Info("Deployed resource is not in-synced with manifests. Sync..")
		return manifestObj, nil
	}

	return nil, nil
}

func (m *plainYamlManager) applyManifest(exist bool, manifestObj *unstructured.Unstructured) error {
	if !exist {
		log.Info("Create..")
		if err := m.TargetCli.Create(m.Context, manifestObj); err != nil {
			log.Error(err, "Creating Object failed..")
			return err
		}
	} else {
		if err := m.TargetCli.Update(m.Context, manifestObj); err != nil {
			return err
		}
	}
	return nil
}

func (m *plainYamlManager) setTargetClient(app *cdv1.Application) error {
	if app.Spec.Destination.Name != "" {
		cfg, err := cluster.GetApplicationClusterConfig(m.Context, m.DefaultCli, app)
		if err != nil {
			log.Error(err, "GetConfig failed..")
			return err
		}

		s := runtime.NewScheme()
		utilruntime.Must(cdv1.AddToScheme(s))
		c, err := client.New(cfg, client.Options{Scheme: s})
		if err != nil {
			log.Error(err, "Create client failed..")
			return err
		}
		m.TargetCli = c
	} else {
		m.TargetCli = m.DefaultCli
	}
	return nil
}

func (m *plainYamlManager) clearApplicationResources(deployResource *cdv1.DeployResource) error {
	deployedObj := &unstructured.Unstructured{}

	if err := m.DefaultCli.Delete(m.Context, deployResource); err != nil {
		log.Error(err, "Delete DeployResource error..")
		return err
	}

	deployedObj.SetAPIVersion(deployResource.Spec.APIVersion)
	deployedObj.SetKind(deployResource.Spec.Kind)
	deployedObj.SetName(deployResource.Spec.Name)
	deployedObj.SetNamespace(deployResource.Spec.Namespace)

	if err := m.TargetCli.Get(m.Context, types.NamespacedName{Namespace: deployedObj.GetNamespace(), Name: deployedObj.GetName()}, deployedObj); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Get deprecated resource error..")
			return err
		}
		return nil
	}

	if err := m.TargetCli.Delete(m.Context, deployedObj); err != nil {
		log.Error(err, "Delete deprecated resource error..")
		return err
	}

	return nil
}
