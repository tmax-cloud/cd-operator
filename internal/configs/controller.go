package configs

import (
	corev1 "k8s.io/api/core/v1"
)

// Configs to be configured by command line arguments

// Names of config maps
const (
	ConfigMapNameCDConfig = "cd-config"
)

// GcChan is a channel to call gc logic
var GcChan = make(chan struct{}, 1)

// ApplyControllerConfigChange is a configmap handler for cd-config configmap
func ApplyControllerConfigChange(cm *corev1.ConfigMap) error {
	getVars(cm.Data, map[string]operatorConfig{
		"externalHostName": {Type: cfgTypeString, StringVal: &ExternalHostName},                // External Hostname
		"ingressClass":     {Type: cfgTypeString, StringVal: &IngressClass, StringDefault: ""}, // Ingress class
		"ingressHost":      {Type: cfgTypeString, StringVal: &IngressHost, StringDefault: ""},  // Ingress host
	})

	// Init
	if !Initiated {
		Initiated = true
		if len(InitCh) < cap(InitCh) {
			InitCh <- struct{}{}
		}
	}

	// Reconfigure GC
	if len(GcChan) < cap(GcChan) {
		GcChan <- struct{}{}
	}

	return nil
}

// Configs for manager
var (
	// ExternalHostName to be used for webhook server (default is ingress host name)
	ExternalHostName string
	// IngressClass is a class for ingress instance
	IngressClass string
	// IngressHost is a host for ingress instance
	IngressHost string
)
