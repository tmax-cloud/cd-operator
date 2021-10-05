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

/* TODO : TestServer 적용 완료. Scheme 에러 수정한 커밋 반영 후 주석 해제 할 것
func TestApplyManifest(t *testing.T) {
	// Set loggers
	if os.Getenv("CD") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}

	var m ManifestManager
	server := newTestServer()
	app := &cdv1.Application{
		Spec: cdv1.ApplicationSpec{
			Source: cdv1.ApplicationSource{
				RepoURL:        "https://github.com/tmax-cloud/cd-example-apps",
				Path:           "guestbook/guestbook-ui-svc.yaml",
				TargetRevision: "main",
			},
		},
	}

	err := m.ApplyManifest(server.URL, app)
	assert.Equal(t, err, nil)
}

func newTestServer() *httptest.Server {
	router := mux.NewRouter()

	router.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			_ = req.Body.Close()
		}()
		// yaml은 tab 말고 space로만 구문 가능
		data := `apiVersion: v1
kind: Service
metadata:
  name: guestbook-ui
spec:
  ports:
  - port: 80
    targetPort: 80
  selector:
    app: guestbook-ui

`
		io.WriteString(w, data)
	})

	return httptest.NewServer(router)
}
*/
