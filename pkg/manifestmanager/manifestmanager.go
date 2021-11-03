package manifestmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	jsonpatch "github.com/evanphx/json-patch"
	gohelm "github.com/mittwald/go-helm-client"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"github.com/tmax-cloud/cd-operator/pkg/cluster"
	"github.com/tmax-cloud/cd-operator/pkg/httpclient"
	"github.com/tmax-cloud/cd-operator/util/gitclient"
	"github.com/tmax-cloud/cd-operator/util/helmclient"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

var log = logf.Log.WithName("manifest-manager")

type ManifestManager struct {
	Client client.Client
	context.Context
	httpclient.HTTPClient
	helmClient *helmclient.Client
}

type DownloadURL struct {
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
	Path        string `json:"path"`
}

// GetManifestURL gets a url of manifest file
func (m *ManifestManager) GetManifestURLList(app *cdv1.Application) ([]string, error) {
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

func (m *ManifestManager) recursivePathCheck(apiBaseURL, repo, path, revision, gitToken string, manifestURLs []string) ([]string, error) {
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

func (m *ManifestManager) ObjectFromManifest(url string, app *cdv1.Application) (*unstructured.Unstructured, error) {
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

func (m *ManifestManager) CompareDeployWithManifest(manifestObj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
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

func (m *ManifestManager) ApplyManifest(exist bool, manifestObj *unstructured.Unstructured) error {
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

func bytesToUnstructuredObject(obj *runtime.RawExtension) (*unstructured.Unstructured, error) {
	var in runtime.Object
	var scope conversion.Scope // While not actually used within the function, need to pass in
	if err := runtime.Convert_runtime_RawExtension_To_runtime_Object(obj, &in, scope); err != nil {
		return nil, err
	}

	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(in)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: unstrObj}, nil
}

func (m *ManifestManager) GitRepoClone(app *cdv1.Application) error {
	repo := app.Spec.Source.RepoURL
	revision := app.Spec.Source.TargetRevision

	localPath := "/tmp/repo-" + utils.RandomString(5)

	app.Spec.Source.Helm.ClonedRepoPath = localPath

	err := gitclient.Clone(repo, localPath, revision)
	if err != nil {
		panic(err)
	}

	return nil
}

func (m *ManifestManager) InstallHelmChart(app *cdv1.Application) error {
	if m.helmClient == nil {
		m.initHelmClient()
	}

	// 로컬에 저장된 경로를 이용하여 chart install
	randomString := utils.RandomString(5)
	releaseName := "release-" + randomString
	app.Spec.Source.Helm.ReleaseName = releaseName

	chartPath := app.Spec.Source.Path
	chartLocalPath := app.Spec.Source.Helm.ClonedRepoPath + "/" + chartPath

	var namespace string
	if app.Spec.Destination.Namespace == "" {
		namespace = "default"
	} else {
		namespace = app.Spec.Destination.Namespace
	}

	err := m.helmClient.InstallChart(releaseName, chartLocalPath, namespace)
	if err != nil {
		panic(err)
	}

	// 로컬에 clone된 Repo 디렉토리 삭제
	os.RemoveAll(app.Spec.Source.Helm.ClonedRepoPath)
	return nil
}

func (m *ManifestManager) UninstallRelease(app *cdv1.Application) error {
	if m.helmClient == nil {
		m.initHelmClient()
	}

	err := m.helmClient.UninstallReleaseByName(app.Spec.Source.Helm.ReleaseName)
	if err != nil {
		panic(err)
	}

	return nil
}

func (m *ManifestManager) initHelmClient() {
	opt := &gohelm.Options{
		RepositoryCache:  "/tmp/.helmcache",
		RepositoryConfig: "/tmp/.helmrepo",
		Debug:            true,
		Linting:          true,
	}

	goHelmClient, err := gohelm.New(opt)
	if err != nil {
		panic(err)
	}

	m.helmClient = &helmclient.Client{Client: goHelmClient}
}
