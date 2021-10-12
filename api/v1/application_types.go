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

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"fmt"
	"net/url"
)

// ApplicationKind is kind string
const (
	ApplicationKind = "Applications"
)

// Condition keys for IApplication
const (
	ApplicationConditionReady             = status.ConditionType("ready")
	ApplicationConditionWebhookRegistered = status.ConditionType("webhook-registered")
)

// ApplicationConditionReasonNoGitToken is a Reason key
const (
	ApplicationConditionReasonNoGitToken = "noGitToken"
)

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// Source is a reference to the location of the application's manifests or chart
	Source ApplicationSource `json:"source"`
	// Destination is a reference to the target Kubernetes server and namespace
	Destination ApplicationDestination `json:"destination"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	// Conditions of IntegrationConfig
	Conditions status.Conditions `json:"conditions"`
	Secrets    string            `json:"secrets,omitempty"` // TODO 왜 필요해?
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

type ApplicationSource struct {
	// RepoURL is the URL to the repository (Git) that contains the application manifests
	RepoURL string `json:"repoURL"`
	// Path is a directory path within the Git repository, and is only valid for applications sourced from Git.
	Path string `json:"path,omitempty"`
	// TargetRevision defines the revision of the source to sync the application to.
	// In case of Git, this can be commit, tag, or branch. If omitted, will equal to HEAD.
	// In case of Helm, this is a semver tag for the Chart's version.
	TargetRevision string `json:"targetRevision,omitempty"`
	// Token is a token for accessing the remote git server. It can be empty, if you don't want to register a webhook
	// to the git server
	Token *GitToken `json:"token,omitempty"`
}

func (source *ApplicationSource) GetRepository() string {
	u, err := url.Parse(source.RepoURL)
	if err != nil {
		panic(err)
	}

	return u.Path[1:]
}

func (source *ApplicationSource) GetAPIUrl() string {
	u, err := url.Parse(source.RepoURL)
	if err != nil {
		panic(err)
	}

	fmt.Println(u.Host)

	if u.Host == "github.com" {
		return GithubDefaultAPIUrl
	} else if u.Host == "gitlab.com" {
		return GitlabDefaultAPIUrl
	} else {
		// TODO - github, gitlab가 아닌 경우
		return ""
	}
}

func (source *ApplicationSource) GetGitType() GitType {
	u, err := url.Parse(source.RepoURL)
	if err != nil {
		panic(err)
	}

	fmt.Println(u.Host)

	if u.Host == "github.com" {
		return GitTypeGitHub
	} else if u.Host == "gitlab.com" {
		return GitTypeGitLab
	} else {
		// TODO - github, gitlab default가 아닌 경우
		return GitTypeFake
	}
}

type ApplicationDestination struct {
	// Namespace specifies the target namespace for the application's resources.
	// The namespace will only be set for namespace-scoped resources that have not set a value for .metadata.namespace
	Namespace string `json:"namespace,omitempty"`
	// Name specifies the target cluster's name
	Name string `json:"name"`
}

// TODO
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

const (
	ConfigMapNameCDConfig      = "cd-config"
	ConfigMapNamespaceCDSystem = "cd-system"
)

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}

// GetToken fetches git access token from IntegrationConfig
func (app *Application) GetToken(c client.Client) (string, error) {
	tokenStruct := app.Spec.Source.Token

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
// TODO: modify to use config controller & expose controller
func (app *Application) GetWebhookServerAddress(c client.Client) string {
	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: ConfigMapNameCDConfig, Namespace: ConfigMapNamespaceCDSystem}, cm); err != nil {
		return ""
	}
	currentExternalHostName := cm.Data["externalHostName"]
	return fmt.Sprintf("http://%s/webhook/%s/%s", currentExternalHostName, app.Namespace, app.Name)
}
