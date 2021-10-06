package manifestmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	jsonpatch "github.com/evanphx/json-patch"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

var log = logf.Log.WithName("manifest-manager")

type ManifestManager struct {
	Client client.Client
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

	bytes, err := yaml.YAMLToJSON(body)
	if err != nil {
		log.Error(err, "YAMLToJSON failed..")
		return err
	}

	rawExt := &runtime.RawExtension{Raw: bytes}
	unstObj, err := bytesToUnstructuredObject(rawExt)
	if err != nil {
		log.Error(err, "BytesToUnstructuredObject failed..")
		return err
	}

	// TODO - fix it. use Application.Spec.Destination.Namespace
	if len(unstObj.GetNamespace()) == 0 {
		unstObj.SetNamespace("default")
	}

	if err := m.Client.Get(context.Background(), types.NamespacedName{
		Namespace: unstObj.GetNamespace(),
		Name:      unstObj.GetName()}, unstObj); err != nil {
		if !errors.IsNotFound(err) { // 에러가 404일 때만 Create 시도
			return err
		}

		log.Info("Create..")
		if err := m.Client.Create(context.Background(), unstObj); err != nil {
			log.Error(err, "Creating Object failed..")
			return err
		}
	} else {
		log.Info("This object alrealy exists..Update it")
		unstr := unstObj.DeepCopy()

		// get already existing k8s object as unstructured type
		if err = m.Client.Get(context.Background(), types.NamespacedName{
			Namespace: unstObj.GetNamespace(),
			Name:      unstObj.GetName(),
		}, unstr); err != nil {
			return err
		}

		bytedUnstr, _ := unstr.MarshalJSON()
		bytedUnstObj, _ := unstObj.MarshalJSON()
		patchedByte, _ := jsonpatch.MergePatch(bytedUnstr, bytedUnstObj)

		finalPatch := make(map[string]interface{})
		if err := json.Unmarshal(patchedByte, &finalPatch); err != nil {
			return err
		}

		unstr.SetUnstructuredContent(finalPatch)
		if err = m.Client.Update(context.Background(), unstObj); err != nil {
			return err
		}
	}

	if err := createResource(unstObj, app, m.Client); err != nil {
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

func createResource(unstObj *unstructured.Unstructured, app *cdv1.Application, c client.Client) error {
	// TODO
	// 1. DeploySource의 Namespace가 unstObj의 Namespace을 따를지, app의 Namespace을 따를지 결정해야 함
	// 2. Namespace의 대한 정보가 두 번 중복되는 부분도 확인 필요
	// 3. Application 필드로 app.Namespace가 들어가는게 맞는지?
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
		// TODO : Fix "no kind is registered for the type v1.DeployResource in scheme "pkg/runtime/scheme.go:101" [recovered]"
		panic(err)
	}
	return nil
}
