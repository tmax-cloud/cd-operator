package cluster

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type getApplicationClusterConfigTestCase struct {
	secretName string
	value      string

	expectedHost     string
	expectedErrOccur bool
	expectedErrMsg   string
}

func TestGetApplicationClusterConfig(t *testing.T) {
	//Set loggers
	if os.Getenv("CI") != "true" {
		logrus.SetLevel(logrus.InfoLevel)
		ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	}

	tc := map[string]getApplicationClusterConfigTestCase{
		"existSecret": {
			secretName: "exist",
			value: `apiVersion: v1
clusters:
- cluster:
    certificate-authority: "./test_cert/ca.crt"
    server: https://test.server.io:6443
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes
kind: Config
preferences: {}
users:
- name: kubernetes-admin
  user:
    client-certificate: "./test_cert/tls.crt"
    client-key: "./test_cert/tls.key"
`,
			expectedHost:     "https://test.server.io:6443",
			expectedErrOccur: false,
		},
		"noExistSecret": {
			secretName: "no-exist",
			value: `apiVersion: v1
clusters:
- cluster:
    certificate-authority: "./test_cert/ca.crt"
    server: https://test.server.io:6443
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: kubernetes-admin
  name: kubernetes-admin@kubernetes
current-context: kubernetes-admin@kubernetes
kind: Config
preferences: {}
users:
- name: kubernetes-admin
user:
client-certificate: "./test_cert/tls.crt"
    client-key: "./test_cert/tls.key"
	`,
			expectedHost:     "https://test.server.io:6443",
			expectedErrOccur: true,
			expectedErrMsg:   "unable to find cluster secret no-exist-kubeconfig: secrets \"no-exist-kubeconfig\" not found",
		},
	}

	s := runtime.NewScheme()
	utilruntime.Must(cdv1.AddToScheme(s))
	utilruntime.Must(corev1.AddToScheme(s))

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			err := createFakeCert()
			require.NoError(t, err)

			defer func() {
				err = removeFakeCert()
				require.NoError(t, err)
			}()

			app := &cdv1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Spec: cdv1.ApplicationSpec{
					Destination: cdv1.ApplicationDestination{
						Name:      c.secretName,
						Namespace: "default",
					},
				},
			}

			sec := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exist-kubeconfig",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"value": []byte(c.value),
				},
			}

			fakeCli := fake.NewClientBuilder().WithScheme(s).WithObjects(app, sec).Build()
			config, err := GetApplicationClusterConfig(context.Background(), fakeCli, app)

			if !c.expectedErrOccur {
				require.Equal(t, c.expectedHost, config.Host)
				require.NoError(t, err)
			} else {
				require.Equal(t, c.expectedErrMsg, err.Error())
			}
		})
	}
}

func createFakeCert() error {
	if err := os.Mkdir("./test_cert/", os.ModePerm); err != nil {
		return err
	}
	if err := ioutil.WriteFile("./test_cert/ca.crt", []byte("test"), 0644); err != nil {
		return err
	}
	if err := ioutil.WriteFile("./test_cert/tls.key", []byte("test"), 0644); err != nil {
		return err
	}
	if err := ioutil.WriteFile("./test_cert/tls.crt", []byte("test"), 0644); err != nil {
		return err
	}
	return nil
}

func removeFakeCert() error {
	if err := os.RemoveAll("./test_cert"); err != nil {
		return err
	}
	return nil
}
