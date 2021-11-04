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
	"github.com/tmax-cloud/cd-operator/pkg/httpclient"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
	repoURL        string
	path           string
	targetRevision string

	expectedStatusCode int
	expectedErrOccur   bool
	expectedResult     []string
}

func TestGetManifestURL(t *testing.T) {
	// Set loggers
	if os.Getenv("CI") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}

	testBody := map[string]string{
		"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook?ref=main":       `[{"name":"guestbook-test-svc.yaml","path":"guestbook/guestbook-test-svc.yaml","sha":"e8a4a27fbae4042ba3428098c0b899f3665c39e4","size":141,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-test-svc.yaml?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-test-svc.yaml","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/e8a4a27fbae4042ba3428098c0b899f3665c39e4","download_url":"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-test-svc.yaml","type":"file","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-test-svc.yaml?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/e8a4a27fbae4042ba3428098c0b899f3665c39e4","html":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-test-svc.yaml"}},{"name":"guestbook-ui-deployment.yaml","path":"guestbook/guestbook-ui-deployment.yaml","sha":"8a0975e363539eacfba296559ad6385cbedd1245","size":389,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-ui-deployment.yaml?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-ui-deployment.yaml","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/8a0975e363539eacfba296559ad6385cbedd1245","download_url":"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-deployment.yaml","type":"file","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-ui-deployment.yaml?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/8a0975e363539eacfba296559ad6385cbedd1245","html":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-ui-deployment.yaml"}},{"name":"guestbook-ui-svc.yaml","path":"guestbook/guestbook-ui-svc.yaml","sha":"fa173a2b2e84c2a3566a1572bbc65a72155b58d1","size":145,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-ui-svc.yaml?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-ui-svc.yaml","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/fa173a2b2e84c2a3566a1572bbc65a72155b58d1","download_url":"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-svc.yaml","type":"file","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/guestbook-ui-svc.yaml?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/fa173a2b2e84c2a3566a1572bbc65a72155b58d1","html":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/guestbook-ui-svc.yaml"}},{"name":"test","path":"guestbook/test","sha":"7eb2aed0d0aadb4fd268b7e7921e9eb9c61d2a1e","size":0,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/test?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/tree/main/guestbook/test","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/trees/7eb2aed0d0aadb4fd268b7e7921e9eb9c61d2a1e","download_url":null,"type":"dir","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/test?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/trees/7eb2aed0d0aadb4fd268b7e7921e9eb9c61d2a1e","html":"https://github.com/tmax-cloud/cd-example-apps/tree/main/guestbook/test"}}]`,
		"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/test?ref=main":  `[{"name":"guestbook-testui-deployment.yaml","path":"guestbook/test/guestbook-testui-deployment.yaml","sha":"28322ec77cc65392aee4a6ea312a7a8e67e04a71","size":399,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/test/guestbook-testui-deployment.yaml?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/test/guestbook-testui-deployment.yaml","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/28322ec77cc65392aee4a6ea312a7a8e67e04a71","download_url":"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/test/guestbook-testui-deployment.yaml","type":"file","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/guestbook/test/guestbook-testui-deployment.yaml?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/28322ec77cc65392aee4a6ea312a7a8e67e04a71","html":"https://github.com/tmax-cloud/cd-example-apps/blob/main/guestbook/test/guestbook-testui-deployment.yaml"}}]`,
		"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/deployment.yaml?ref=main": `{"name":"deployment.yaml","path":"deployment.yaml","sha":"2d0f44780d8fe8108524a77f96d10da2231e1e90","size":345,"url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/deployment.yaml?ref=main","html_url":"https://github.com/tmax-cloud/cd-example-apps/blob/main/deployment.yaml","git_url":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/2d0f44780d8fe8108524a77f96d10da2231e1e90","download_url":"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/deployment.yaml","type":"file","content":"YXBpVmVyc2lvbjogYXBwcy92MQpraW5kOiBEZXBsb3ltZW50Cm1ldGFkYXRh\nOgogIG5hbWU6IHRlc3QtZGVwbG95LWZyb20tZ2l0CnNwZWM6CiAgdGVtcGxh\ndGU6CiAgICBtZXRhZGF0YToKICAgICAgbmFtZTogbmdpbngKICAgICAgbGFi\nZWxzOgogICAgICAgIGFwcHM6IHRlc3QtYXBwCiAgICBzcGVjOgogICAgICBj\nb250YWluZXJzOgogICAgICAgIC0gbmFtZTogbmdpbngtY29udGFpbmVyCiAg\nICAgICAgICBpbWFnZTogbmdpbngKICAgICAgICAgIHBvcnRzOgogICAgICAg\nICAgICAtIGNvbnRhaW5lclBvcnQ6IDgwCiAgc2VsZWN0b3I6CiAgICBtYXRj\naExhYmVsczoKICAgICAgYXBwczogdGVzdC1hcHAK\n","encoding":"base64","_links":{"self":"https://api.github.com/repos/tmax-cloud/cd-example-apps/contents/deployment.yaml?ref=main","git":"https://api.github.com/repos/tmax-cloud/cd-example-apps/git/blobs/2d0f44780d8fe8108524a77f96d10da2231e1e90","html":"https://github.com/tmax-cloud/cd-example-apps/blob/main/deployment.yaml"}}`,
	}

	mockHTTPClient := &httpclient.MockHTTPClient{}
	mockClient := fake.NewClientBuilder().Build()
	m := plainYamlManager{Client: mockClient, Context: context.Background(), HTTPClient: mockHTTPClient}

	tc := map[string]getManifestURLTestCase{
		"githubValidURLDir": {
			repoURL:            "https://github.com/tmax-cloud/cd-example-apps",
			path:               "guestbook",
			targetRevision:     "main",
			expectedStatusCode: 200,
			expectedErrOccur:   false,
			expectedResult:     []string{"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-test-svc.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-deployment.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/guestbook-ui-svc.yaml", "https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/guestbook/test/guestbook-testui-deployment.yaml"},
		},
		"githubValidURLFile": {
			repoURL:            "https://github.com/tmax-cloud/cd-example-apps",
			path:               "deployment.yaml",
			targetRevision:     "main",
			expectedStatusCode: 200,
			expectedErrOccur:   false,
			expectedResult:     []string{"https://raw.githubusercontent.com/tmax-cloud/cd-example-apps/main/deployment.yaml"},
		},
		"githubInvalidURL": {
			repoURL:        "https://github.com/tmax-cloud/cd-example-apps-fake",
			path:           "guestbook",
			targetRevision: "main",

			expectedStatusCode: 404,
			expectedErrOccur:   true,
		},
		// TODO: tc for gitlab & other apiURL
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
			mockHTTPClient.DoFunc = func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader(testBody[r.URL.String()])),
					StatusCode: c.expectedStatusCode,
				}, nil
			}
			result, err := m.getManifestURLList(app)
			if c.expectedErrOccur {
				require.Error(t, err)
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
	m := plainYamlManager{Client: mockClient, Context: context.Background(), HTTPClient: mockHTTPClient}

	server := newTestServer()

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
		"otherCluster": {
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
			destinationName:      "test",
			destinationNameSpace: "test",
			expectedErrOccur:     false,
			expectedRawObj:       &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 80}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},
		},
		"noExistCluster": {
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
			destinationName:      "test2",
			destinationNameSpace: "test",
			expectedErrOccur:     true,
			expectedErrMsg:       "unable to find cluster secret test2-kubeconfig: secrets \"test2-kubeconfig\" not found",
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

			sec := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-kubeconfig",
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

			m.Client = fake.NewClientBuilder().WithScheme(s).WithObjects(app, sec).Build()
			manifestRawObj, err := m.objectFromManifest(c.url, app)
			if c.expectedErrOccur {
				require.Equal(t, c.expectedErrMsg, err.Error())
			} else {
				expectedRaw, _ := c.expectedRawObj.MarshalJSON()
				manifestRaw, _ := manifestRawObj.MarshalJSON()
				require.Equal(t, expectedRaw, manifestRaw)
				require.NoError(t, err)
			}
		})
	}
}

type compareDeployWithTestCase struct {
	manifestObj *unstructured.Unstructured
	deployedObj *unstructured.Unstructured

	expectedObj      *unstructured.Unstructured
	expectedErrOccur bool
	expectedErrMsg   string
}

func TestCompareDeployWithManifest(t *testing.T) {
	tc := map[string]compareDeployWithTestCase{
		"notFound": {
			manifestObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": "80", "targetPort": "80"}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},
			deployedObj: nil,

			expectedObj:      &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": "80", "targetPort": "80"}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},
			expectedErrOccur: true,
			expectedErrMsg:   `services "guestbook-ui" not found`,
		},
		"inSync": {
			manifestObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": "80", "targetPort": "80"}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},
			deployedObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 80}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},

			expectedObj:      nil,
			expectedErrOccur: false,
		},
		"outSync": {
			manifestObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": "80", "targetPort": "80"}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},
			deployedObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "guestbook-ui", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 8080}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},

			expectedObj:      &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"creationTimestamp": interface{}(nil), "name": "guestbook-ui", "namespace": "test", "resourceVersion": "999"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": "80", "targetPort": "80"}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}, "status": map[string]interface{}{"loadBalancer": map[string]interface{}{}}}},
			expectedErrOccur: false,
		},
	}

	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	mockHTTPClient := &httpclient.MockHTTPClient{}
	mockClient := fake.NewClientBuilder().Build()
	m := plainYamlManager{Client: mockClient, Context: context.Background(), HTTPClient: mockHTTPClient}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			if c.deployedObj != nil {
				m.Client = fake.NewClientBuilder().WithScheme(s).WithObjects(c.deployedObj).Build()
			} else {
				m.Client = fake.NewClientBuilder().WithScheme(s).Build()
			}
			manifestObj, err := m.compareDeployWithManifest(c.manifestObj)

			if c.expectedErrOccur {
				require.Equal(t, c.expectedErrMsg, err.Error())
			} else {
				require.NoError(t, err)
			}
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
	m := plainYamlManager{Client: mockClient, Context: context.Background(), HTTPClient: mockHTTPClient}

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

type getDeployResourceListTestCase struct {
	app *cdv1.Application

	expectedListLength int
	expectedErrOccur   bool
	expectedErrMsg     string
}

var testDeployList = &cdv1.DeployResourceList{
	Items: []cdv1.DeployResource{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "application1-deployment-resource1-test",
				Namespace: "default",
				Labels:    map[string]string{"cd.tmax.io/application": "application1-default"},
			},
			Application: "application1",
			Spec: cdv1.DeployResourceSpec{
				APIVersion: "apps/v1",
				Name:       "resource1",
				Kind:       "Deployment",
				Namespace:  "test",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "application1-service-resource2-test",
				Namespace: "default",
				Labels:    map[string]string{"cd.tmax.io/application": "application1-default"},
			},
			Application: "application1",
			Spec: cdv1.DeployResourceSpec{
				APIVersion: "v1",
				Name:       "resource2",
				Kind:       "Service",
				Namespace:  "test",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "application2-service-resource3-test",
				Namespace: "default",
				Labels:    map[string]string{"cd.tmax.io/application": "application2-default"},
			},
			Application: "application1",
			Spec: cdv1.DeployResourceSpec{
				APIVersion: "v1",
				Name:       "resource3",
				Kind:       "Service",
				Namespace:  "test",
			},
		},
	},
}

func TestGetDeployResourceList(t *testing.T) {
	if os.Getenv("CI") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}

	tc := map[string]getDeployResourceListTestCase{
		"app1": {
			app: &cdv1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "application1",
					Namespace: "default",
				},
			},
			expectedListLength: 2,
			expectedErrOccur:   false,
		},
		"app2": {
			app: &cdv1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "application2",
					Namespace: "default",
				},
			},
			expectedListLength: 1,
			expectedErrOccur:   false,
		},
	}

	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	mockHTTPClient := &httpclient.MockHTTPClient{}
	mockClient := fake.NewClientBuilder().WithLists(testDeployList).WithScheme(s).Build()
	m := plainYamlManager{Client: mockClient, Context: context.Background(), HTTPClient: mockHTTPClient}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			deployList, err := m.getDeployResourceList(c.app)
			if c.expectedErrOccur {
				require.Equal(t, c.expectedErrMsg, err.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, c.expectedListLength, len(deployList.Items))
			}
		})
	}
}

type updateDeployResourceTestCase struct {
	unstObj *unstructured.Unstructured

	expectedDeployResource *cdv1.DeployResource
	expectedErrOccur       bool
	expectedErrMsg         string
}

func TestUpdateDeployResource(t *testing.T) {
	tc := map[string]updateDeployResourceTestCase{
		"exist": {
			unstObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "exist-obj", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 80}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},

			expectedDeployResource: &cdv1.DeployResource{
				TypeMeta: metav1.TypeMeta{
					Kind:       "DeployResource",
					APIVersion: "cd.tmax.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "application-service-exist-obj-test",
					Namespace:       "default",
					ResourceVersion: "999",
					Labels:          map[string]string{"cd.tmax.io/application": "application-default"},
				},
				Application: "application",
				Spec: cdv1.DeployResourceSpec{
					APIVersion: "v1",
					Name:       "exist-obj",
					Kind:       "Service",
					Namespace:  "test",
				},
			},
			expectedErrOccur: false,
		},
		"create": {
			unstObj: &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "v1", "kind": "Service", "metadata": map[string]interface{}{"name": "new-obj", "namespace": "test"}, "spec": map[string]interface{}{"ports": []interface{}{map[string]interface{}{"port": 80, "targetPort": 80}}, "selector": map[string]interface{}{"app": "guestbook-ui"}}}},

			expectedDeployResource: &cdv1.DeployResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "application-service-new-obj-test",
					Namespace:       "default",
					Labels:          map[string]string{"cd.tmax.io/application": "application-default"},
					ResourceVersion: "1",
				},
				Application: "application",
				Spec: cdv1.DeployResourceSpec{
					APIVersion: "v1",
					Name:       "new-obj",
					Kind:       "Service",
					Namespace:  "test",
				},
			},
			expectedErrOccur: false,
		},
	}

	app := &cdv1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "application",
			Namespace: "default",
		},
	}

	testDeployResource := &cdv1.DeployResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "application-service-exist-obj-test",
			Namespace: "default",
			Labels:    map[string]string{"cd.tmax.io/application": "application-default"},
		},
		Application: "application",
		Spec: cdv1.DeployResourceSpec{
			APIVersion: "v1",
			Name:       "exist-obj",
			Kind:       "Service",
			Namespace:  "test",
		},
	}

	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	mockHTTPClient := &httpclient.MockHTTPClient{}
	mockClient := fake.NewClientBuilder().WithLists(testDeployList).WithObjects(app, testDeployResource).WithScheme(s).Build()
	m := plainYamlManager{Client: mockClient, Context: context.Background(), HTTPClient: mockHTTPClient}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			deployResource, err := m.updateDeployResource(c.unstObj, app)
			if !c.expectedErrOccur {
				require.NoError(t, err)
				require.Equal(t, c.expectedDeployResource, deployResource)
				err := m.Client.Get(m.Context, types.NamespacedName{Namespace: c.expectedDeployResource.Namespace, Name: c.expectedDeployResource.Name}, c.expectedDeployResource)
				require.NoError(t, err)
			} else {
				require.Equal(t, c.expectedErrMsg, err.Error())
			}
		})
	}
}

type deleteDeployResourceTestCase struct {
	deployResource *cdv1.DeployResource

	expectedErrOccur bool
	expectedErrMsg   string
}

func TestDeleteDeployResource(t *testing.T) {
	tc := map[string]deleteDeployResourceTestCase{
		"drsExist": {
			deployResource: &cdv1.DeployResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "application-service-exist-drs-test",
					Namespace:       "default",
					Labels:          map[string]string{"cd.tmax.io/application": "application-default"},
					ResourceVersion: "1",
				},
				Application: "application",
				Spec: cdv1.DeployResourceSpec{
					APIVersion: "v1",
					Name:       "exist-drs",
					Kind:       "Service",
					Namespace:  "test",
				},
			},
			expectedErrOccur: false,
		},
		"noDrsExist": {
			deployResource: &cdv1.DeployResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "application-service-no-drs-test",
					Namespace:       "default",
					Labels:          map[string]string{"cd.tmax.io/application": "application-default"},
					ResourceVersion: "1",
				},
				Application: "application",
				Spec: cdv1.DeployResourceSpec{
					APIVersion: "v1",
					Name:       "no-drs",
					Kind:       "Service",
					Namespace:  "test",
				},
			},
			expectedErrOccur: true,
			expectedErrMsg:   `deployresources.cd.tmax.io "application-service-no-drs-test" not found`,
		},
		"noObjExist": {
			deployResource: &cdv1.DeployResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "application-service-no-obj-test",
					Namespace:       "default",
					Labels:          map[string]string{"cd.tmax.io/application": "application-default"},
					ResourceVersion: "1",
				},
				Application: "application",
				Spec: cdv1.DeployResourceSpec{
					APIVersion: "v1",
					Name:       "no-obj",
					Kind:       "Service",
					Namespace:  "test",
				},
			},
			expectedErrOccur: false,
		},
	}

	s := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	mockHTTPClient := &httpclient.MockHTTPClient{}
	mockClient := fake.NewClientBuilder().WithScheme(s).Build()
	m := &plainYamlManager{Client: mockClient, Context: context.Background(), HTTPClient: mockHTTPClient}

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			testDeployResource1 := &cdv1.DeployResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "application-service-exist-drs-test",
					Namespace: "default",
					Labels:    map[string]string{"cd.tmax.io/application": "application-default"},
				},
				Application: "application",
				Spec: cdv1.DeployResourceSpec{
					APIVersion: "v1",
					Name:       "exist-drs",
					Kind:       "Service",
					Namespace:  "test",
				},
			}
			testDeployResource2 := &cdv1.DeployResource{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "application-service-no-obj-test",
					Namespace: "default",
					Labels:    map[string]string{"cd.tmax.io/application": "application-default"},
				},
				Application: "application",
				Spec: cdv1.DeployResourceSpec{
					APIVersion: "v1",
					Name:       "no-obj",
					Kind:       "Service",
					Namespace:  "test",
				},
			}
			testDeployedObject := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exist-drs",
					Namespace: "test",
				},
			}
			m.Client = fake.NewClientBuilder().WithLists(testDeployList).WithObjects(testDeployResource1, testDeployResource2, testDeployedObject).WithScheme(s).Build()

			err := m.deleteDeployResource(c.deployResource)

			if !c.expectedErrOccur {
				require.NoError(t, err)
				err := m.Client.Get(m.Context, types.NamespacedName{Namespace: c.deployResource.Namespace, Name: c.deployResource.Name}, c.deployResource)
				require.True(t, errors.IsNotFound(err))
				err = m.Client.Get(m.Context, types.NamespacedName{Namespace: c.deployResource.Spec.Namespace, Name: c.deployResource.Spec.Name}, &corev1.Service{})
				require.True(t, errors.IsNotFound(err))
			} else {
				require.Equal(t, c.expectedErrMsg, err.Error())
			}
		})
	}
}

func newTestServer() *httptest.Server {
	router := mux.NewRouter()

	return httptest.NewServer(router)
}