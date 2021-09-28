package manifestmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
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
	Download_URL string `json:"download_url"`
}

// GetManifestURL gets a url of manifest file
func (m *ManifestManager) GetManifestURL(app *cdv1.Application) (string, error) {
	apiUrl := app.Spec.Source.GetAPIUrl()
	repo := app.Spec.Source.GetRepository()
	revision := app.Spec.Source.TargetRevision // branch, tag, sha..
	path := app.Spec.Source.Path

	apiURL := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", apiUrl, repo, path, revision)

	// Get download_url of manifest file
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Error(err, "http Get failed..")
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err, "Read response body failed..")
		return "", err
	}

	var downloadUrl DownloadURL
	if err := json.Unmarshal(body, &downloadUrl); err != nil {
		log.Error(err, "Unmarshal failed..")
		return "", err
	}

	return downloadUrl.Download_URL, nil
}

func (m *ManifestManager) ApplyManifest(url string) error {
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
