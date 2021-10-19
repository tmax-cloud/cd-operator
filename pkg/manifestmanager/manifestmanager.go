package manifestmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/cluster"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	var manifestURLs []string

	manifestURLs, err := recursivePathCheck(apiBaseURL, repo, path, revision, manifestURLs)
	if err != nil {
		return nil, err
	}

	return manifestURLs, nil
}

func recursivePathCheck(apiBaseURL, repo, path, revision string, manifestURLs []string) ([]string, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", apiBaseURL, repo, path, revision)

	// Get download_url of manifest file
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Error(err, "http Get failed..")
		return nil, err
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf(resp.Status)
		log.Error(err, "http response error..")
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
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
			manifestURLs, err = recursivePathCheck(apiBaseURL, repo, downloadURLs[i].Path, revision, manifestURLs)
			if err != nil {
				return nil, err
			}
		}
	}

	return manifestURLs, nil
}

func (m *ManifestManager) ObjectFromManifest(url string, app *cdv1.Application) (*unstructured.Unstructured, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Error(err, "http Get failed..")
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
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

	bytedManifestObj, _ = manifestObj.MarshalJSON()

	if string(bytedDeployedObj) != string(bytedManifestObj) {
		log.Info("Deployed resource does not synced with manifests..")
		return manifestObj, nil
	}

	return nil, nil
}

func (m *ManifestManager) ApplyManifest(exist bool, app *cdv1.Application, manifestObj *unstructured.Unstructured) error {
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

	if err := m.createDeployResource(manifestObj, app); err != nil {
		panic(err)
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

func (m *ManifestManager) createDeployResource(unstObj *unstructured.Unstructured, app *cdv1.Application) error {
	obj := &cdv1.DeployResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(app.Name + "-" + unstObj.GetKind() + "-" + unstObj.GetName()),
			Namespace: app.Name,
		},
		Application: app.Name,
		Spec: cdv1.DeployResourceSpec{
			Name:      unstObj.GetName(),
			Kind:      unstObj.GetKind(),
			Namespace: unstObj.GetNamespace(),
		},
	}

	if err := m.Client.Get(m.Context, types.NamespacedName{Name: app.Name}, &v1.Namespace{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if err := m.Client.Create(m.Context, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: app.Name}}); err != nil {
			panic(err)
		}
	}

	if err := m.Client.Get(m.Context, types.NamespacedName{
		Name:      strings.ToLower(app.Name + "-" + unstObj.GetKind() + "-" + unstObj.GetName()),
		Namespace: app.Name}, obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if err := m.Client.Create(m.Context, obj); err != nil {
			panic(err)
		}
	}

	return nil
}
