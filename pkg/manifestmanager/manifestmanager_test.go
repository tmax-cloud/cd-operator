package manifestmanager

import (
	"testing"

	"github.com/bmizerany/assert"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
)

func TestGetManifestURL(t *testing.T) {
	var m ManifestManager
	// api.github.com/repos/argoproj/argocd-example-apps/contents/guestbook/guestbook-ui-svc.yaml?ref=master

	app := &cdv1.Application{}
	app.Spec = cdv1.ApplicationSpec{
		Git: cdv1.GitConfig{
			Type:       cdv1.GitTypeGitHub,
			Repository: "argoproj/argocd-example-apps",
		},
		Revision:     "master",
		Path:         "guestbook/guestbook-ui-svc.yaml",
		ManifestType: cdv1.ApplicationSourceTypePlainYAML,
	}

	result, err := m.GetManifestURL(app)
	t.Log(result)
	assert.Equal(t, err, nil)
	assert.Equal(t, result, "https://raw.githubusercontent.com/argoproj/argocd-example-apps/master/guestbook/guestbook-ui-svc.yaml")
}

func TestApplyManifest(t *testing.T) {
	var m ManifestManager
	url := "https://raw.githubusercontent.com/argoproj/argocd-example-apps/master/guestbook/guestbook-ui-svc.yaml"

	err := m.ApplyManifest(url)
	assert.Equal(t, err, nil)
}
