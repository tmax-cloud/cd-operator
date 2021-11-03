package manifestmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/cluster"
	"github.com/tmax-cloud/cd-operator/pkg/httpclient"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type plainYamlManager struct {
	Client client.Client
	context.Context
	httpclient.HTTPClient
}

type DownloadURL struct {
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Path        string `json:"path"`
}

func NewPlainYamlManager(ctx context.Context, cli client.Client, httpCli httpclient.HTTPClient) ManifestManager {
	return &plainYamlManager{
		Client:     cli,
		Context:    ctx,
		HTTPClient: httpCli,
	}
}

func (m *plainYamlManager) Sync(app *cdv1.Application, forced bool) error {
	urls, err := m.getManifestURLList(app)
	if err != nil {
		log.Error(err, "GetManifestURLList failed..")
		return err
	}
	oldDeployResources, err := m.getDeployResourceList(app)
	if err != nil {
		log.Error(err, "GetDeployResourceList failed")
		return err
	}

	updatedDeployResources := make(map[string]*cdv1.DeployResource)

	for _, url := range urls {
		manifestRawobj, err := m.objectFromManifest(url, app)
		if err != nil {
			log.Error(err, "Get object from manifest failed..")
			return err
		}
		updatedDeployResource, err := m.updateDeployResource(manifestRawobj, app)
		if err != nil {
			log.Error(err, "NewDeployResource failed..")
			return err
		}
		updatedDeployResources[updatedDeployResource.Name] = updatedDeployResource

		manifestModifiedObj, err := m.compareDeployWithManifest(manifestRawobj)
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

	for _, oldDeployResource := range oldDeployResources.Items {
		if updatedDeployResources[oldDeployResource.Name] == nil {
			if err := m.deleteDeployResource(&oldDeployResource); err != nil {
				log.Error(err, "DeleteDeployResource failed..")
				return err
			}
		}
	}

	if app.Spec.SyncPolicy.AutoSync {
		app.Status.Sync.Status = cdv1.SyncStatusCodeSynced
	}

	return nil
}

func (m *plainYamlManager) Clear(app *cdv1.Application) error {
	deployedResourceList, err := m.getDeployResourceList(app)
	if err != nil {
		return err
	}
	for _, deployedResource := range deployedResourceList.Items {
		if err := m.deleteDeployResource(&deployedResource); err != nil {
			return err
		}
	}
	return nil
}

// GetManifestURL gets a url of manifest file
func (m *plainYamlManager) getManifestURLList(app *cdv1.Application) ([]string, error) {
	apiBaseURL := app.Spec.Source.GetAPIUrl()
	repo := app.Spec.Source.GetRepository()
	revision := app.Spec.Source.TargetRevision // branch, tag, sha..
	path := app.Spec.Source.Path
	gitToken, err := app.GetToken(m.Client)
	if err != nil {
		return nil, err
	}

	var manifestURLs []string

	manifestURLs, err = m.recursivePathCheck(apiBaseURL, repo, path, revision, gitToken, manifestURLs)
	if err != nil {
		return nil, err
	}

	return manifestURLs, nil
}

func (m *plainYamlManager) recursivePathCheck(apiBaseURL, repo, path, revision, gitToken string, manifestURLs []string) ([]string, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", apiBaseURL, repo, path, revision)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	// Get download_url of manifest file
	if gitToken != "" {
		req.Header.Add("Authorization", gitToken)
	}

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		log.Error(err, "http Get failed..")
		return nil, err
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf(resp.Status)
		log.Error(err, "http response error..")
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "Read response body failed..")
		return nil, err
	}

	var downloadURLs []DownloadURL
	var downloadURL DownloadURL

	if err := json.Unmarshal(body, &downloadURLs); err != nil {
		if err := json.Unmarshal(body, &downloadURL); err != nil {
			log.Error(err, "Unmarshal failed..")
			return nil, err
		}
		downloadURLs = append(downloadURLs, downloadURL)
	}

	for i := range downloadURLs {
		if downloadURLs[i].Type == "file" {
			manifestURLs = append(manifestURLs, downloadURLs[i].DownloadURL)
		} else if downloadURLs[i].Type == "dir" {
			manifestURLs, err = m.recursivePathCheck(apiBaseURL, repo, downloadURLs[i].Path, revision, gitToken, manifestURLs)
			if err != nil {
				return nil, err
			}
		}
	}

	return manifestURLs, nil
}

func (m *plainYamlManager) objectFromManifest(url string, app *cdv1.Application) (*unstructured.Unstructured, error) {
	resp, err := m.HTTPClient.Get(url)
	if err != nil {
		log.Error(err, "http Get failed..")
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "Read response body failed..")
		return nil, err
	}

	bytes, err := yaml.YAMLToJSON(body)
	if err != nil {
		log.Error(err, "YAMLToJSON failed..")
		return nil, err
	}

	if app.Spec.Destination.Name != "" {
		cfg, err := cluster.GetApplicationClusterConfig(m.Context, m.Client, app)
		if err != nil {
			log.Error(err, "GetConfig failed..")
			return nil, err
		}

		s := runtime.NewScheme()
		utilruntime.Must(cdv1.AddToScheme(s))
		c, err := client.New(cfg, client.Options{Scheme: s})
		if err != nil {
			log.Error(err, "Create client failed..")
			return nil, err
		}
		m.Client = c
	}

	rawExt := &runtime.RawExtension{Raw: bytes}
	manifestRawObj, err := bytesToUnstructuredObject(rawExt)
	if err != nil {
		log.Error(err, "BytesToUnstructuredObject failed..")
		return nil, err
	}

	if len(manifestRawObj.GetNamespace()) == 0 {
		manifestRawObj.SetNamespace(app.Spec.Destination.Namespace)
	}
	return manifestRawObj, nil
}

func (m *plainYamlManager) compareDeployWithManifest(manifestObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	deployedObj := manifestObj.DeepCopy()
	if err := m.Client.Get(m.Context, types.NamespacedName{
		Namespace: deployedObj.GetNamespace(),
		Name:      deployedObj.GetName()}, deployedObj); err != nil {
		if errors.IsNotFound(err) {
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
	if err := m.Client.Update(m.Context, manifestObj, client.DryRunAll); err != nil {
		return nil, err
	}

	if fmt.Sprintf("%v", deployedObj) != fmt.Sprintf("%v", manifestObj) {
		log.Info("Deployed resource is not in-synced with manifests. Sync..")
		return manifestObj, nil
	}

	return nil, nil
}

func (m *plainYamlManager) applyManifest(exist bool, manifestObj *unstructured.Unstructured) error {
	if !exist {
		log.Info("Create..")
		if err := m.Client.Create(m.Context, manifestObj); err != nil {
			log.Error(err, "Creating Object failed..")
			return err
		}
	} else {
		if err := m.Client.Update(m.Context, manifestObj); err != nil {
			return err
		}
	}
	return nil
}

func (m *plainYamlManager) getDeployResourceList(app *cdv1.Application) (*cdv1.DeployResourceList, error) {
	deployResourceList := &cdv1.DeployResourceList{}

	if err := m.Client.List(m.Context, deployResourceList, client.MatchingLabels{"cd.tmax.io/application": app.Name + "-" + app.Namespace}); err != nil {
		return nil, err
	}
	return deployResourceList, nil
}

func (m *plainYamlManager) updateDeployResource(unstObj *unstructured.Unstructured, app *cdv1.Application) (*cdv1.DeployResource, error) {
	deployResource := &cdv1.DeployResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(app.Name + "-" + unstObj.GetKind() + "-" + unstObj.GetName() + "-" + unstObj.GetNamespace()),
			Namespace: app.Namespace,
			Labels:    map[string]string{"cd.tmax.io/application": app.Name + "-" + app.Namespace},
		},
		Application: app.Name,
		Spec: cdv1.DeployResourceSpec{
			APIVersion: unstObj.GetAPIVersion(),
			Name:       unstObj.GetName(),
			Kind:       unstObj.GetKind(),
			Namespace:  unstObj.GetNamespace(),
		},
	}

	if err := m.Client.Get(m.Context, types.NamespacedName{
		Name:      strings.ToLower(app.Name + "-" + unstObj.GetKind() + "-" + unstObj.GetName() + "-" + unstObj.GetNamespace()),
		Namespace: app.Namespace}, deployResource); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		if err := m.Client.Create(m.Context, deployResource); err != nil {
			return nil, err
		}
		return deployResource, nil
	}

	return deployResource, nil
}

func (m *plainYamlManager) deleteDeployResource(deployResource *cdv1.DeployResource) error {
	deployedObj := &unstructured.Unstructured{}

	if err := m.Client.Delete(m.Context, deployResource); err != nil {
		log.Error(err, "Delete DeployResource error..")
		return err
	}

	deployedObj.SetAPIVersion(deployResource.Spec.APIVersion)
	deployedObj.SetKind(deployResource.Spec.Kind)
	deployedObj.SetName(deployResource.Spec.Name)
	deployedObj.SetNamespace(deployResource.Spec.Namespace)

	if err := m.Client.Get(m.Context, types.NamespacedName{Namespace: deployedObj.GetNamespace(), Name: deployedObj.GetName()}, deployedObj); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Get deprecated resource error..")
			return err
		}
		return nil
	}

	if err := m.Client.Delete(m.Context, deployedObj); err != nil {
		log.Error(err, "Delete deprecated resource error..")
		return err
	}

	return nil
}
