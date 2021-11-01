package manifestmanager

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	v1 "k8s.io/api/core/v1"
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
	utilruntime.Must(v1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	m := ManifestManager{Context: context.Background()}
	m.Client = fake.NewClientBuilder().WithLists(testDeployList).WithScheme(s).Build()

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			deployList, err := m.GetDeployResourceList(c.app)
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
	utilruntime.Must(v1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	m := ManifestManager{Context: context.Background()}
	m.Client = fake.NewClientBuilder().WithLists(testDeployList).WithObjects(app, testDeployResource).WithScheme(s).Build()

	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			deployResource, err := m.UpdateDeployResource(c.unstObj, app)
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
	utilruntime.Must(v1.AddToScheme(s))
	utilruntime.Must(cdv1.AddToScheme(s))

	m := ManifestManager{Context: context.Background()}

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
			testDeployedObject := &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "exist-drs",
					Namespace: "test",
				},
			}
			m.Client = fake.NewClientBuilder().WithLists(testDeployList).WithObjects(testDeployResource1, testDeployResource2, testDeployedObject).WithScheme(s).Build()

			err := m.DeleteDeployResource(c.deployResource)

			if !c.expectedErrOccur {
				require.NoError(t, err)
				err := m.Client.Get(m.Context, types.NamespacedName{Namespace: c.deployResource.Namespace, Name: c.deployResource.Name}, c.deployResource)
				require.True(t, errors.IsNotFound(err))
				err = m.Client.Get(m.Context, types.NamespacedName{Namespace: c.deployResource.Spec.Namespace, Name: c.deployResource.Spec.Name}, &v1.Service{})
				require.True(t, errors.IsNotFound(err))
			} else {
				require.Equal(t, c.expectedErrMsg, err.Error())
			}
		})
	}
}
