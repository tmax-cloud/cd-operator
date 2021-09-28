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
	"github.com/operator-framework/operator-lib/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
}

func (source *ApplicationSource) GetRepository() string {
	//ex) https://github.com/tmax-cloud/cd-operator.git
	u, err := url.Parse(source.RepoURL)
	if err != nil {
		panic(err)
	}

	fmt.Println(u.Path)
	// '/tmax-cloud/cd-operator' 앞에 루트와 함께 파싱됨

	return u.Path[1:]
}

func (source *ApplicationSource) GetAPIUrl() string {
	//ex) https://github.com/tmax-cloud/cd-operator.git
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
		// TODO - github, gitlab default가 아닌 경우
		return ""
	}
}

type ApplicationDestination struct {
	// Server specifies the URL of the target cluster and must be set to the Kubernetes control plane API
	Server string `json:"server,omitempty"`
	// Namespace specifies the target namespace for the application's resources.
	// The namespace will only be set for namespace-scoped resources that have not set a value for .metadata.namespace
	Namespace string `json:"namespace,omitempty"`
	// Name is an alternate way of specifying the target cluster by its symbolic name
	Name string `json:"name,omitempty"`
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

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
