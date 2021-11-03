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
	"context"
	"fmt"
	"net/http"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/sync"
	"k8s.io/apimachinery/pkg/types"

	"github.com/tmax-cloud/cd-operator/internal/apiserver"
	"github.com/tmax-cloud/cd-operator/internal/utils"

	"github.com/gorilla/mux"
)

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="authorization.k8s.io",resources=subjectaccessreviews,verbs=get;list;watch;create;update;patch

func (h *handler) syncHandler(w http.ResponseWriter, req *http.Request) {
	h.updateDeploy(w, req)
}

func (h *handler) updateDeploy(w http.ResponseWriter, req *http.Request) {
	reqID := utils.RandomString(10)
	log := h.log.WithValues("request", reqID)

	vars := mux.Vars(req)

	ns, nsExist := vars[apiserver.NamespaceParamKey]
	applicationName, nameExist := vars[applicationNameParamKey]
	if !nsExist || !nameExist {
		_ = utils.RespondError(w, http.StatusBadRequest, "url is malformed")
		return
	}
	app := &cdv1.Application{}
	if err := h.k8sClient.Get(context.Background(), types.NamespacedName{Name: applicationName, Namespace: ns}, app); err != nil {
		log.Info(err.Error())
		_ = utils.RespondError(w, http.StatusBadRequest, fmt.Sprintf("req: %s, no Application %s/%s is found", reqID, ns, applicationName))
		return
	}
	// sync resources with manitests
	// TODO: application의 sync status, sync 옵션 등 추가하여 분기 필요.
	if err := sync.CheckSync(h.k8sClient, app, true); err != nil {
		log.Info(err.Error())
		return
	}
}
