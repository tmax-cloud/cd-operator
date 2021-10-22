package manifestmanager

import (
	"strings"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (m *ManifestManager) GetDeployResourceList(app *cdv1.Application) (*cdv1.DeployResourceList, error) {
	deployResourceList := &cdv1.DeployResourceList{}

	if err := m.Client.List(m.Context, deployResourceList, client.MatchingLabels{"cd.tmax.io/application": app.Name + "-" + app.Namespace}); err != nil {
		return nil, err
	}
	return deployResourceList, nil
}

func (m *ManifestManager) UpdateDeployResource(unstObj *unstructured.Unstructured, app *cdv1.Application) (*cdv1.DeployResource, error) {
	deployResource := &cdv1.DeployResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(app.Name + "-" + unstObj.GetKind() + "-" + unstObj.GetName() + "-" + unstObj.GetNamespace()),
			Namespace: app.Namespace,
			Labels:    map[string]string{"cd.tmax.io/application": app.Name + "-" + app.Namespace},
		},
		Application: app.Name,
		Spec: cdv1.DeployResourceSpec{
			APIVersion: unstObj.GetAPIVersion(),
			Name:       unstObj.GetName(),
			Kind:       unstObj.GetKind(),
			Namespace:  unstObj.GetNamespace(),
		},
	}

	if err := m.Client.Get(m.Context, types.NamespacedName{
		Name:      strings.ToLower(app.Name + "-" + unstObj.GetKind() + "-" + unstObj.GetName() + "-" + unstObj.GetNamespace()),
		Namespace: app.Namespace}, deployResource); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		if err := m.Client.Create(m.Context, deployResource); err != nil {
			return nil, err
		}
		return deployResource, nil
	}

	return deployResource, nil
}

func (m *ManifestManager) DeleteDeployResource(deployResource *cdv1.DeployResource) error {
	deployedObj := &unstructured.Unstructured{}

	if err := m.Client.Delete(m.Context, deployResource); err != nil {
		log.Error(err, "Delete DeployResource error..")
		return err
	}

	deployedObj.SetAPIVersion(deployResource.Spec.APIVersion)
	deployedObj.SetKind(deployResource.Spec.Kind)
	deployedObj.SetName(deployResource.Spec.Name)
	deployedObj.SetNamespace(deployResource.Spec.Namespace)

	if err := m.Client.Get(m.Context, types.NamespacedName{Namespace: deployedObj.GetNamespace(), Name: deployedObj.GetName()}, deployedObj); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Get deprecated resource error..")
			return err
		}
		return nil
	}

	if err := m.Client.Delete(m.Context, deployedObj); err != nil {
		log.Error(err, "Delete deprecated resource error..")
		return err
	}

	return nil
}
