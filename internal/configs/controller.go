/*
 Copyright 2021 The CI/CD Operator Authors

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package configs

import (
	corev1 "k8s.io/api/core/v1"
)

// Configs to be configured by command line arguments

// Names of config maps
const (
	ConfigMapNameCDConfig = "cd-config"
)

var controllerConfigUpdateChan []chan struct{}

// RegisterControllerConfigUpdateChan registers a channel which accepts controller config's update event
func RegisterControllerConfigUpdateChan(ch chan struct{}) {
	controllerConfigUpdateChan = append(controllerConfigUpdateChan, ch)
}

// ApplyControllerConfigChange is a configmap handler for cicd-config configmap
func ApplyControllerConfigChange(cm *corev1.ConfigMap) error {
	getVars(cm.Data, map[string]operatorConfig{
		"externalHostName": {Type: cfgTypeString, StringVal: &ExternalHostName},                     // External Hostname
		"exposeMode":       {Type: cfgTypeString, StringVal: &ExposeMode, StringDefault: "Ingress"}, // Expose mode
		"ingressClass":     {Type: cfgTypeString, StringVal: &IngressClass, StringDefault: ""},      // Ingress class
		"ingressHost":      {Type: cfgTypeString, StringVal: &IngressHost, StringDefault: ""},       // Ingress host
	})

	// Init
	if !ControllerInitiated {
		ControllerInitiated = true
		if len(ControllerInitCh) < cap(ControllerInitCh) {
			ControllerInitCh <- struct{}{}
		}
	}

	// Notify channels (non-blocking way)
	for _, ch := range controllerConfigUpdateChan {
		if len(ch) < cap(ch) {
			ch <- struct{}{}
		}
	}

	return nil
}

// Configs for manager
var (
	// ExternalHostName to be used for webhook server (default is ingress host name)
	ExternalHostName string

	// CurrentExternalHostName is NOT a configurable variable! it just stores current hostname which will be used for
	// exposing webhook/result server
	CurrentExternalHostName string

	// ExposeMode is a mode to be used for exposing the webhook server (Ingress/LoadBalancer/ClusterIP)
	ExposeMode string

	// IngressClass is a class for ingress instance
	IngressClass string

	// IngressHost is a host for ingress instance
	IngressHost string
)
