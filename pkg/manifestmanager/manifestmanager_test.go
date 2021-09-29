package manifestmanager

import (
	"fmt"
	"testing"

	"github.com/bmizerany/assert"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
)

func TestGetManifestURL(t *testing.T) {
	var m ManifestManager
	// https://github.com/tmax-cloud/cd-operator.git
	// api.github.com/repos/argoproj/argocd-example-apps/contents/guestbook/guestbook-ui-svc.yaml?ref=master

	app := &cdv1.Application{
		Spec: cdv1.ApplicationSpec{
			Source: cdv1.ApplicationSource{
				RepoURL:        "https://github.com/tmax-cloud/cd-example-apps",
				Path:           "guestbook/guestbook-ui-svc.yaml", // 아직 single yaml만 가능
				TargetRevision: "main",
			},
		},
	}

	result, err := m.GetManifestURL(app)
	fmt.Println(result)
	t.Log(result)
	assert.Equal(t, err, nil)
	assert.Equal(t, result, "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-svc.yaml")
}

// func TestApplyManifest(t *testing.T) {
// 	// Set loggers
// 	if os.Getenv("CD") != "true" {
// 		logrus.SetLevel(logrus.InfoLevel)
// 		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
// 	}

// 	var m ManifestManager
// 	url := "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-svc.yaml"

// 	app := &cdv1.Application{
// 		Spec: cdv1.ApplicationSpec{
// 			Source: cdv1.ApplicationSource{
// 				RepoURL:        "https://github.com/tmax-cloud/cd-example-apps",
// 				Path:           "guestbook/guestbook-ui-svc.yaml", // 아직 single yaml만 가능
// 				TargetRevision: "main",
// 			},
// 		},
// 	}

// 	err := m.ApplyManifest(url, app)
// 	assert.Equal(t, err, nil)
// }
