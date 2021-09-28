/*
Copyright 2021.

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

package v1

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-lib/status"
	"github.com/tmax-cloud/cd-operator/internal/configs"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplicationKind is kind string
const (
	ApplicationKind = "applications"
)

// Condition keys for IApplication
const (
	ApplicationConditionReady             = status.ConditionType("ready")
	ApplicationConditionWebhookRegistered = status.ConditionType("webhook-registered")
)

const (
	ApplicationConditionReasonNoGitToken = "noGitToken"
)

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// Git config for target repository
	Git          GitConfig             `json:"git"`
	Revision     string                `json:"revision"`
	Path         string                `json:"path"`
	ManifestType ApplicationSourceType `json:"manifestType"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	// Conditions of Application
	Conditions status.Conditions `json:"conditions"`
	Secrets    string            `json:"secrets,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

// ApplicationSourceType specifies the type of the application's source
type ApplicationSourceType string

const (
	ApplicationSourceTypePlainYAML ApplicationSourceType = "PlainYAML"
	ApplicationSourceTypeHelm      ApplicationSourceType = "Helm"
	ApplicationSourceTypeKustomize ApplicationSourceType = "Kustomize"
	ApplicationSourceTypeKsonnet   ApplicationSourceType = "Ksonnet"
	ApplicationSourceTypeDirectory ApplicationSourceType = "Directory"
	ApplicationSourceTypePlugin    ApplicationSourceType = "Plugin"
)

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}

// GetToken fetches git access token from Application
func (app *Application) GetToken(c client.Client) (string, error) {
	tokenStruct := app.Spec.Git.Token

	// Empty token
	if tokenStruct == nil {
		return "", nil
	}

	// Get from value
	if tokenStruct.ValueFrom == nil {
		if tokenStruct.Value != "" {
			return tokenStruct.Value, nil
		}
		return "", fmt.Errorf("token is empty")
	}

	// Get from secret
	secretName := tokenStruct.ValueFrom.SecretKeyRef.Name
	secretKey := tokenStruct.ValueFrom.SecretKeyRef.Key
	secret := &corev1.Secret{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: app.Namespace}, secret); err != nil {
		return "", err
	}
	token, ok := secret.Data[secretKey]
	if !ok {
		return "", fmt.Errorf("token secret/key %s/%s not valid", secretName, secretKey)
	}
	return string(token), nil
}

// GetWebhookServerAddress returns Server address which webhook events will be received
func (app *Application) GetWebhookServerAddress() string {
	return fmt.Sprintf("http://%s/webhook/%s/%s", configs.ExternalHostName, app.Namespace, app.Name)
}

const (
	ApplicationAPIRunPre     = "runpre"
	ApplicationAPIRunPost    = "runpost"
	ApplicationAPIWebhookURL = "webhookurl"
)

// ApplicationAPIReqRunPreBody is a body struct for Application's api request
type ApplicationAPIReqRunPreBody struct {
	BaseBranch string `json:"base_branch"`
	HeadBranch string `json:"head_branch"`
}

// ApplicationAPIReqRunPostBody is a body struct for Application's api request
type ApplicationAPIReqRunPostBody struct {
	Branch string `json:"branch"`
}

// ApplicationAPIReqWebhookURL is a body struct for Application's api request
type ApplicationAPIReqWebhookURL struct {
	URL    string `json:"url"`
	Secret string `json:"secret"`
}
