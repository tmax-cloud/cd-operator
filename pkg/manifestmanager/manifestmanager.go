package manifestmanager

import (
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
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
