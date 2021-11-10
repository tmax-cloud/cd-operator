package manifestmanager

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	gitfake "github.com/tmax-cloud/cd-operator/pkg/git/fake"
	"github.com/tmax-cloud/cd-operator/pkg/httpclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// type syncTestCase struct {
// 	app *cdv1.Application
// 	forced bool

// 	expectedErrOccur bool
// 	expectedErrMsg string
// }

func TestSync(t *testing.T) {
	// TODO
}

type getManifestURLTestCase struct {
	path           string
	targetRevision string

	expectedErrOccur bool
	expectedErrMsg   string
	expectedResult   []string
}

func TestGetManifestURL(t *testing.T) {
	// Set loggers
	if os.Getenv("CI") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}

	tc := map[string]getManifestURLTestCase{
		"pathErr": {
			path:           "invalid",
			targetRevision: "main",

			expectedErrOccur: true,
			expectedErrMsg:   "404 not found",
		},
		"validFilePath": {
			path:           "deployment.yaml",
			targetRevision: "main",

			expectedErrOccur: false,
			expectedResult:   []string{"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/deployment.yaml"},
		},
		"validDirPath": {
			path:           "guestbook",
			targetRevision: "main",

			expectedErrOccur: false,
			expectedResult:   []string{"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-test-svc.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-deployment.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-svc.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/test/guestbook-testui-deployment.yaml"},
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			app := &cdv1.Application{
				Spec: cdv1.ApplicationSpec{
					Source: cdv1.ApplicationSource{
						RepoURL:        "https://test.com/tmax-cloud/cd-example-apps",
						Path:           c.path,
						TargetRevision: c.targetRevision,
					},
				},
			}
			mockClient := fake.NewClientBuilder().Build()
			m := plainYamlManager{DefaultCli: mockClient, TargetCli: mockClient, Context: context.Background(), GitCli: &gitfake.Client{Repository: app.Spec.Source.GetRepository(), K8sClient: mockClient}}

			result, err := m.getManifestURLList(app)
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

type objectFromManifestTestCase struct {
	url                  string
	body                 string
	destinationName      string
	destinationNameSpace string

	expectedErrOccur bool
	expectedErrMsg   string
	expectedRawObj   *unstructured.Unstructured
}

// TODO: multiple objects tc 추가
func TestObjectFromManifest(t *testing.T) {
	// Set loggers
	if os.Getenv("CI") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}

	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	mockHTTPClient := &httpclient.MockHTTPClient{}
	mockClient := fake.NewClientBuilder().Build()
	m := plainYamlManager{DefaultCli: mockClient, TargetCli: mockClient, Context: context.Background(), HTTPClient: mockHTTPClient}

	tc := map[string]objectFromManifestTestCase{
		"default": {
			url: "validURL",
			body: `apiVersion: v1
kind: Service
metadata:
  name: guestbook-ui
spec:
  ports:
  - port: 80
    targetPort: 80
  selector:
    app: guestbook-ui`,
			destinationName:      "",
			destinationNameSpace: "",
			expectedErrOccur:     false,
			expectedRawObj:       &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 80}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},
		},
		"otherNameSpace": {
			url: "validURL",
			body: `apiVersion: v1
kind: Service
metadata:
  name: guestbook-ui
spec:
  ports:
  - port: 80
    targetPort: 80
  selector:
    app: guestbook-ui`,
			destinationName:      "",
			destinationNameSpace: "test",
			expectedErrOccur:     false,
			expectedRawObj:       &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 80}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},
		},
	}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			app := &cdv1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: cdv1.ApplicationSpec{
					Destination: cdv1.ApplicationDestination{
						Name:      c.destinationName,
						Namespace: c.destinationNameSpace,
					},
				},
			}

			mockHTTPClient.GetFunc = func(url string) (*http.Response, error) {
				return &http.Response{
					Body: io.NopCloser(strings.NewReader(c.body)),
				}, nil
			}

			m.DefaultCli = fake.NewClientBuilder().WithScheme(s).WithObjects(app).Build()
			manifestRawObjs, err := m.objectFromManifest(c.url, app)
			if c.expectedErrOccur {
				require.Equal(t, c.expectedErrMsg, err.Error())
			} else {
				expectedRaw, _ := c.expectedRawObj.MarshalJSON()
				manifestRaw, _ := manifestRawObjs[0].MarshalJSON()
				require.Equal(t, expectedRaw, manifestRaw)
				require.NoError(t, err)
			}
		})
	}
}

type compareDeployWithTestCase struct {
	manifestObj *unstructured.Unstructured
	deployedObj *unstructured.Unstructured

	expectedObj        *unstructured.Unstructured
	expectedSyncStatus cdv1.SyncStatusCode
	expectedErrOccur   bool
	expectedErrMsg     string
}

func TestCompareDeployWithManifest(t *testing.T) {
	tc := map[string]compareDeployWithTestCase{
		"notFound": {
			manifestObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": "80", "targetPort": "80"}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},
			deployedObj: nil,

			expectedObj:        &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": "80", "targetPort": "80"}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},
			expectedSyncStatus: cdv1.SyncStatusCodeOutOfSync,
			expectedErrOccur:   true,
			expectedErrMsg:     `services "guestbook-ui" not found`,
		},
		"inSync": {
			manifestObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": "80", "targetPort": "80"}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},
			deployedObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 80}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},

			expectedObj:        nil,
			expectedSyncStatus: cdv1.SyncStatusCodeUnknown,
			expectedErrOccur:   false,
		},
		"outSync": {
			manifestObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": "80", "targetPort": "80"}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},
			deployedObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 8080}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},

			expectedObj:        &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"creationTimestamp": interface{}(nil), "name": "guestbook-ui", "namespace": "test", "resourceVersion": "999"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": "80", "targetPort": "80"}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}, "status": map[string]interface{}{"loadBalancer": map[string]interface{}{}}}},
			expectedSyncStatus: cdv1.SyncStatusCodeOutOfSync,
			expectedErrOccur:   false,
		},
	}

	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	mockHTTPClient := &httpclient.MockHTTPClient{}
	mockClient := fake.NewClientBuilder().Build()
	m := plainYamlManager{DefaultCli: mockClient, TargetCli: mockClient, Context: context.Background(), HTTPClient: mockHTTPClient}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			app := &cdv1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Status: cdv1.ApplicationStatus{
					Sync: cdv1.SyncStatus{
						Status: cdv1.SyncStatusCodeUnknown,
					},
				},
			}
			if c.deployedObj != nil {
				m.TargetCli = fake.NewClientBuilder().WithScheme(s).WithObjects(c.deployedObj).Build()
			} else {
				m.TargetCli = fake.NewClientBuilder().WithScheme(s).Build()
			}
			manifestObj, err := m.compareDeployWithManifest(app, c.manifestObj)

			if c.expectedErrOccur {
				require.Equal(t, c.expectedErrMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, c.expectedSyncStatus, app.Status.Sync.Status)
			require.Equal(t, c.expectedObj, manifestObj)
		})
	}
}

type applyManifestTestCase struct {
	exist       bool
	manifestObj *unstructured.Unstructured

	expectedErrOccur bool
	expectedErrMsg   string
}

func TestApplyManifest(t *testing.T) {
	tc := map[string]applyManifestTestCase{
		"createSuccess": {
			exist:       false,
			manifestObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "newObj", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 80}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},

			expectedErrOccur: false,
		},
		"createFail": {
			exist:       false,
			manifestObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "metadata": map[string]interface{}{"name": "newObj", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": "80", "targetPort": "80"}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},

			expectedErrOccur: true,
			expectedErrMsg:   "Object 'Kind' is missing in 'unstructured object has no kind'",
		},
		"updateSuccess": {
			exist:       true,
			manifestObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"creationTimestamp": interface{}(nil), "name": "existObj", "namespace": "test", "resourceVersion": "999"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 80}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}, "status": map[string]interface{}{"loadBalancer": map[string]interface{}{}}}},

			expectedErrOccur: false,
		},
		"updateFail": {
			exist:       true,
			manifestObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"creationTimestamp": interface{}(nil), "name": "existObj", "namespace": "test", "resourceVersion": "999"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}, "status": map[string]interface{}{"loadBalancer": map[string]interface{}{}}}},

			expectedErrOccur: true,
			expectedErrMsg:   "Operation cannot be fulfilled on services \"existObj\": object was modified",
		},
	}

	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	existObj := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"creationTimestamp": interface{}(nil), "name": "existObj", "namespace": "test", "resourceVersion": "999"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 8080}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}, "status": map[string]interface{}{"loadBalancer": map[string]interface{}{}}}}

	mockHTTPClient := &httpclient.MockHTTPClient{}
	mockClient := fake.NewClientBuilder().WithScheme(s).WithObjects(existObj).Build()
	m := plainYamlManager{DefaultCli: mockClient, TargetCli: mockClient, Context: context.Background(), HTTPClient: mockHTTPClient}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			err := m.applyManifest(c.exist, c.manifestObj)
			if c.expectedErrOccur {
				require.Equal(t, c.expectedErrMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

type setTargetClientTestCase struct {
	destinationName string

	expectedErrOccur bool
	expectedErrMsg   string
}

func TestSetTargetClient(t *testing.T) {
	tc := map[string]setTargetClientTestCase{
		"defaultCluster": {
			destinationName:  "",
			expectedErrOccur: false,
		},
		"otherCluster": {
			destinationName:  "exist",
			expectedErrOccur: false,
		},
		"noExistCluster": {
			destinationName:  "noexist",
			expectedErrOccur: true,
			expectedErrMsg:   "unable to find cluster secret noexist-kubeconfig: secrets \"noexist-kubeconfig\" not found",
		},
	}

	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	mockHTTPClient := &httpclient.MockHTTPClient{}
	mockClient := fake.NewClientBuilder().Build()
	m := plainYamlManager{DefaultCli: mockClient, TargetCli: mockClient, Context: context.Background(), HTTPClient: mockHTTPClient}

	server := newTestServer()

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			app := &cdv1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test",
				},
				Spec: cdv1.ApplicationSpec{
					Destination: cdv1.ApplicationDestination{
						Name: c.destinationName,
					},
				},
			}

			sec := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exist-kubeconfig",
					Namespace: app.Namespace,
				},
				StringData: map[string]string{
					"value": `apiVersion: v1
clusters:
- cluster:
    server: ` + server.URL + `
  name: test
contexts:
- context:
    cluster: test
    user: test-admin
  name: test-admin@test
current-context: test-admin@test
kind: Config
preferences: {}
users:
- name: test-admin
`,
				},
			}

			sec.Data = make(map[string][]byte)
			sec.Data["value"] = []byte(sec.StringData["value"])

			m.DefaultCli = fake.NewClientBuilder().WithScheme(s).WithObjects(app, sec).Build()
			err := m.setTargetClient(app)
			if c.expectedErrOccur {
				require.Equal(t, c.expectedErrMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func newTestServer() *httptest.Server {
	router := mux.NewRouter()

	return httptest.NewServer(router)
}
