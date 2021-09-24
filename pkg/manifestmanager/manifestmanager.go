package manifestmanager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("manifest-manager")

type ManifestManager struct {
}

type DownloadURL struct {
	Download_URL string `json:"download_url"`
}

// GetManifestURL gets a url of manifest file
func (m *ManifestManager) GetManifestURL(app *cdv1.Application) (string, error) {
	apiUrl := app.Spec.Git.GetAPIUrl()
	repo := app.Spec.Git.Repository // {owner}/{repo}
	revision := app.Spec.Revision   // branch, tag, sha..
	path := app.Spec.Path

	apiURL := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", apiUrl, repo, path, revision)
	log.Info(apiURL)

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

	// TODO - kubectl apply -f <<download_url>>
	return nil
}
