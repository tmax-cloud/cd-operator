package manifestmanager

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bmizerany/assert"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
		"githubValidURLDir": {
			repoURL:          "https://github.com/tmax-cloud/cd-example-apps",
			path:             "guestbook",
			targetRevision:   "main",
			expectedErrOccur: false,
			expectedResult:   []string{"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-test-svc.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-deployment.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-svc.yaml"},
		},
		"githubValidURLFile": {
			repoURL:          "https://github.com/tmax-cloud/cd-example-apps",
			path:             "deployment.yaml",
			targetRevision:   "main",
			expectedErrOccur: false,
			expectedResult:   []string{"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/deployment.yaml"},
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

func TestApplyManifest(t *testing.T) {
	// Set loggers
	if os.Getenv("CD") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}

	s := runtime.NewScheme()
	utilruntime.Must(cdv1.AddToScheme(s))
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

	fakeCli := fake.NewFakeClientWithScheme(s, app)
	m := ManifestManager{Client: fakeCli}
	err := m.ApplyManifest(server.URL, app)

	//TODO : 아웃풋인 DeployResource을 활용해서 Test 짜기
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
  name: guestbook-ui-test
spec:
  ports:
  - port: 80
    targetPort: 80
  selector:
    app: guestbook-ui

`
		_, err := io.WriteString(w, data)
		if err != nil {
			return
		}
	})

	return httptest.NewServer(router)
}
