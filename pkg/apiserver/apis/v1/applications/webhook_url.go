package applications

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/internal/apiserver"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"k8s.io/apimachinery/pkg/types"
)

func (h *handler) webhookURLHandler(w http.ResponseWriter, req *http.Request) {
	reqID := utils.RandomString(10)
	log := h.log.WithValues("request", reqID)

	// Get ns/resource name
	vars := mux.Vars(req)

	ns, nsExist := vars[apiserver.NamespaceParamKey]
	resName, nameExist := vars[appParamKey]
	if !nsExist || !nameExist {
		log.Info("url is malformed")
		_ = utils.RespondError(w, http.StatusBadRequest, "url is malformed")
		return
	}

	// Get Application
	app := &cdv1.Application{}
	if err := h.k8sClient.Get(context.Background(), types.NamespacedName{Name: resName, Namespace: ns}, app); err != nil {
		log.Info(err.Error())
		_ = utils.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("req: %s, cannot get Application %s/%s", reqID, ns, resName))
		return
	}

	_ = utils.RespondJSON(w, cdv1.ApplicationAPIReqWebhookURL{
		URL:    app.GetWebhookServerAddress(),
		Secret: app.Status.Secrets,
	})
}
