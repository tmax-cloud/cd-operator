package manifestmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	jsonpatch "github.com/evanphx/json-patch"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/cluster"
	"github.com/tmax-cloud/cd-operator/pkg/httpclient"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type plainYamlManager struct {
	DefaultCli client.Client
	// Client for multi cluster
	TargetCli client.Client
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
		DefaultCli: cli,
		TargetCli:  cli,
		Context:    ctx,
		HTTPClient: httpCli,
	}
}

func (m *plainYamlManager) Sync(app *cdv1.Application, forced bool) error {
	if err := m.setTargetClient(app); err != nil {
		log.Error(err, "setTargetClient failed..")
		return err
	}

	urls, err := m.getManifestURLList(app)
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

	for _, url := range urls {
		manifestRawobjs, err := m.objectFromManifest(url, app)
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
	}

	for _, oldDeployResource := range oldDeployResources.Items {
		if updatedDeployResources[oldDeployResource.Name] == nil {
			if err := m.clearApplicationResources(&oldDeployResource); err != nil {
				log.Error(err, "clearApplicationResources failed..")
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

// GetManifestURL gets a url of manifest file
func (m *plainYamlManager) getManifestURLList(app *cdv1.Application) ([]string, error) {
	apiBaseURL := app.Spec.Source.GetAPIUrl()
	repo := app.Spec.Source.GetRepository()
	revision := app.Spec.Source.TargetRevision // branch, tag, sha..
	path := app.Spec.Source.Path
	gitToken, err := app.GetToken(m.DefaultCli)
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

func (m *plainYamlManager) objectFromManifest(url string, app *cdv1.Application) ([]*unstructured.Unstructured, error) {
	var manifestRawObjs []*unstructured.Unstructured

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

	stringYAMLManifests := splitMultipleObjectsYAML(body)

	for _, stringYAMLManifest := range stringYAMLManifests {
		byteYAMLManifest := []byte(stringYAMLManifest)

		bytes, err := yaml.YAMLToJSON(byteYAMLManifest)
		if err != nil {
			log.Error(err, "YAMLToJSON failed..")
			return nil, err
		}

		if string(bytes) == "null" {
			continue
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
		manifestRawObjs = append(manifestRawObjs, manifestRawObj)
	}
	return manifestRawObjs, nil
}

func (m *plainYamlManager) compareDeployWithManifest(manifestObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	deployedObj := manifestObj.DeepCopy()
	if err := m.TargetCli.Get(m.Context, types.NamespacedName{
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
	if err := m.TargetCli.Update(m.Context, manifestObj, client.DryRunAll); err != nil {
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

	if err := m.DefaultCli.Delete(context.Background(), deployResource); err != nil {
		log.Error(err, "Delete DeployResource error..")
		return err
	}

	deployedObj.SetAPIVersion(deployResource.Spec.APIVersion)
	deployedObj.SetKind(deployResource.Spec.Kind)
	deployedObj.SetName(deployResource.Spec.Name)
	deployedObj.SetNamespace(deployResource.Spec.Namespace)

	if err := m.TargetCli.Get(context.Background(), types.NamespacedName{Namespace: deployedObj.GetNamespace(), Name: deployedObj.GetName()}, deployedObj); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Get deprecated resource error..")
			return err
		}
		return nil
	}

	if err := m.TargetCli.Delete(context.Background(), deployedObj); err != nil {
		log.Error(err, "Delete deprecated resource error..")
		return err
	}

	return nil
}
