package manifestmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

var log = logf.Log.WithName("manifest-manager")

type ManifestManager struct {
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
	log.Info(apiURL)

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

	var downloadURL []DownloadURL

	if err := json.Unmarshal(body, &downloadURL); err != nil {
		log.Error(err, "Unmarshal failed..")
		return nil, err
	}

	for i := range downloadURL {
		if downloadURL[i].Type == "file" {
			manifestURLs = append(manifestURLs, downloadURL[i].DownloadURL)
		} else if downloadURL[i].Type == "dir" {
			manifestURLs, err = recursivePathCheck(apiBaseURL, repo, downloadURL[i].Path, revision, manifestURLs)
			if err != nil {
				return nil, err
			}
		}
	}

	return manifestURLs, nil
}

func (m *ManifestManager) ApplyManifest(url string, app *cdv1.Application) error {
	resp, err := http.Get(url)
	if err != nil {
		log.Error(err, "http Get failed..")
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "Read response body failed..")
		return err
	}

	json, err := yaml.YAMLToJSON(body)
	if err != nil {
		log.Error(err, "YAMLToJSON failed..")
		return err
	}

	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "GetConfig failed..")
		return err
	}

	c, err := client.New(cfg, client.Options{})
	if err != nil {
		log.Error(err, "Create client failed..")
		return err
	}

	rawExt := &runtime.RawExtension{Raw: json}
	unstObj, err := BytesToUnstructuredObject(rawExt)
	if err != nil {
		log.Error(err, "BytesToUnstructuredObject failed..")
		return err
	}

	// TODO - fix it. use Application.Spec.Destination.Namespace
	if len(unstObj.GetNamespace()) == 0 {
		unstObj.SetNamespace("default")
	}

	if err := c.Create(context.Background(), unstObj); err != nil {
		log.Error(err, "Creating Object failed..")
		// TODO
		// it can be 'services "guestbook-ui" already exists' err.
		// return err
	}

	if err := createResource(unstObj, app, c); err != nil {
		panic(err)
	}

	return nil
}

func BytesToUnstructuredObject(obj *runtime.RawExtension) (*unstructured.Unstructured, error) {
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

func createResource(unstObj *unstructured.Unstructured, app *cdv1.Application, c client.Client) error {
	obj := &cdv1.DeployResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name + "-" + unstObj.GetKind() + "-" + unstObj.GetName(),
			Namespace: app.Name,
		},
		Application: app.Namespace,
		Spec: cdv1.DeployResourceSpec{
			Name:      unstObj.GetName(),
			Kind:      unstObj.GetKind(),
			Namespace: unstObj.GetNamespace(),
		},
	}

	if err := c.Create(context.Background(), obj); err != nil {
		log.Info("err")
		panic(err)
	}
	log.Info("err")

	return nil
}
