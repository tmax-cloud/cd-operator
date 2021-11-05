package manifestmanager

import (
	"context"
	"strings"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("manifest-manager")

type ManifestManager interface {
	Sync(app *cdv1.Application, forced bool) error
	Clear(app *cdv1.Application) error
}

func bytesToUnstructuredObject(obj *runtime.RawExtension) (*unstructured.Unstructured, error) {
	var in runtime.Object
	var scope conversion.Scope // While not actually used within the function, need to pass in
	if err := runtime.Convert_runtime_RawExtension_To_runtime_Object(obj, &in, scope); err != nil {
		return nil, err
	}

	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(in)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: unstrObj}, nil
}

func splitMultipleObjectsYAML(rawYAML []byte) []string {
	splitObjects := strings.Split(string(rawYAML), "---")
	return splitObjects
}

func getDeployResourceList(cli client.Client, app *cdv1.Application) (*cdv1.DeployResourceList, error) {
	deployResourceList := &cdv1.DeployResourceList{}

	if err := cli.List(context.Background(), deployResourceList, client.MatchingLabels{"cd.tmax.io/application": app.Name + "-" + app.Namespace}); err != nil {
		return nil, err
	}
	return deployResourceList, nil
}

func updateDeployResource(cli client.Client, unstObj *unstructured.Unstructured, app *cdv1.Application) (*cdv1.DeployResource, error) {
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

	if err := cli.Get(context.Background(), types.NamespacedName{
		Name:      strings.ToLower(app.Name + "-" + unstObj.GetKind() + "-" + unstObj.GetName() + "-" + unstObj.GetNamespace()),
		Namespace: app.Namespace}, deployResource); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
		if err := cli.Create(context.Background(), deployResource); err != nil {
			return nil, err
		}
		return deployResource, nil
	}

	return deployResource, nil
}

func deleteDeployResource(cli client.Client, deployResource *cdv1.DeployResource) error {
	deployedObj := &unstructured.Unstructured{}

	if err := cli.Delete(context.Background(), deployResource); err != nil {
		log.Error(err, "Delete DeployResource error..")
		return err
	}

	deployedObj.SetAPIVersion(deployResource.Spec.APIVersion)
	deployedObj.SetKind(deployResource.Spec.Kind)
	deployedObj.SetName(deployResource.Spec.Name)
	deployedObj.SetNamespace(deployResource.Spec.Namespace)

	if err := cli.Get(context.Background(), types.NamespacedName{Namespace: deployedObj.GetNamespace(), Name: deployedObj.GetName()}, deployedObj); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Get deprecated resource error..")
			return err
		}
		return nil
	}

	if err := cli.Delete(context.Background(), deployedObj); err != nil {
		log.Error(err, "Delete deprecated resource error..")
		return err
	}

	return nil
}
