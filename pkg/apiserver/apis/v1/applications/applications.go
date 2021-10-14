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

package applications

import (
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/internal/apiserver"
	authorization "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/tmax-cloud/cd-operator/internal/wrapper"
)

const (
	// APIVersion for the api
	APIVersion = "v1"

	applicationNameParamKey = "applicationName"
)

type handler struct {
	k8sClient client.Client
	log       logr.Logger

	authorizer apiserver.Authorizer
}

// NewHandler instantiates a new approvals api handler
func NewHandler(parent wrapper.RouterWrapper, cli client.Client, authCli *authorization.AuthorizationV1Client, logger logr.Logger) (apiserver.APIHandler, error) {
	handler := &handler{k8sClient: cli, log: logger}

	// Authorizer
	handler.authorizer = apiserver.NewAuthorizer(authCli, apiserver.APIGroup, APIVersion, "update")

	// /applications/<application>
	applicationWrapper := wrapper.New(fmt.Sprintf("/%s/{%s}", cdv1.APIKindApplication, applicationNameParamKey), nil, nil)
	if err := parent.Add(applicationWrapper); err != nil {
		return nil, err
	}
	applicationWrapper.Router().Use(handler.authorizer.Authorize)

	// /applications/<application>/sync
	syncWrapper := wrapper.New("/"+cdv1.ApplicationAPISync, []string{http.MethodPut}, handler.syncHandler)
	if err := applicationWrapper.Add(syncWrapper); err != nil {
		return nil, err
	}

	return handler, nil
}
