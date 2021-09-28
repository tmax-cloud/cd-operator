package configs

import (
	"testing"

	"github.com/bmizerany/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestApplyControllerConfigChange(t *testing.T) {
	tc := map[string]controllerTestCase{
		"default": {ConfigMap: &corev1.ConfigMap{
			Data: map[string]string{},
		}, AssertFunc: func(t *testing.T, err error) {
			assert.Equal(t, true, err == nil)

			assert.Equal(t, "", ExternalHostName)
		}},
		"noError": {ConfigMap: &corev1.ConfigMap{
			Data: map[string]string{
				"externalHostName": "external.host.name",
			},
		}, AssertFunc: func(t *testing.T, err error) {
			assert.Equal(t, true, err == nil)
			assert.Equal(t, "external.host.name", ExternalHostName)
		}},
	}

	for name, c := range tc {
		ExternalHostName = ""
		t.Run(name, func(t *testing.T) {
			err := ApplyControllerConfigChange(c.ConfigMap)
			c.AssertFunc(t, err)
		})
	}
}
