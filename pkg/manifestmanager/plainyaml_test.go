package manifestmanager

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/httpclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
