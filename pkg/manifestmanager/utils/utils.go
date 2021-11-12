package utils

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
)

func BytesToUnstructuredObject(bytes []byte) (*unstructured.Unstructured, error) {
	rawExt := &runtime.RawExtension{Raw: bytes}
	var in runtime.Object
	var scope conversion.Scope // While not actually used within the function, need to pass in

	// Currently, Convert_runtime_RawExtension_To_runtime_Object always return nil
	_ = runtime.Convert_runtime_RawExtension_To_runtime_Object(rawExt, &in, scope)

	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(in)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{Object: unstrObj}, nil
}

func SplitMultipleObjectsYAML(rawYAML []byte) []string {
	splitObjects := strings.Split(string(rawYAML), "\n---\n")
	return splitObjects
}
