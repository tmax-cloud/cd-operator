package manifestmanager

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type getManifestURLTestCase struct {
	repoURL        string
	path           string
	targetRevision string

	expectedErrOccur bool
	expectedErrMsg   string
	expectedResult   []string
}

func TestGetManifestURL(t *testing.T) {
	// Set loggers
	if os.Getenv("CD") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}
	var m ManifestManager
	// https://github.com/tmax-cloud/cd-operator.git
	// api.github.com/repos/argoproj/argocd-example-apps/contents/guestbook/guestbook-ui-svc.yaml?ref=master

	tc := map[string]getManifestURLTestCase{
		"githubValidURL": {
			repoURL:          "https://github.com/tmax-cloud/cd-example-apps",
			path:             "guestbook",
			targetRevision:   "main",
			expectedErrOccur: false,
			expectedResult:   []string{"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-deployment.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-svc.yaml"},
		},
		"githubInvalidURL": {
			repoURL:          "https://github.com/tmax-cloud/cd-example-apps-fake",
			path:             "guestbook",
			targetRevision:   "main",
			expectedErrOccur: true,
			expectedErrMsg:   "404 Not Found",
		},
		// TODO: tc for gitlab & other apiURL
		// "gitlabValidURL": {

		// },
		// "gitlabInvalidURL": {

		// },
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			app := &cdv1.Application{
				Spec: cdv1.ApplicationSpec{
					Source: cdv1.ApplicationSource{
						RepoURL:        c.repoURL,
						Path:           c.path, // 아직 single yaml만 가능
						TargetRevision: c.targetRevision,
					},
				},
			}
			result, err := m.GetManifestURLList(app)
			if c.expectedErrOccur {
				require.Error(t, err)
				require.Equal(t, c.expectedErrMsg, err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, c.expectedResult, result)
			}
		})
	}
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
