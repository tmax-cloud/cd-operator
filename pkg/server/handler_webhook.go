package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
)

var webhookPath = fmt.Sprintf("/webhook/{%s}/{%s}", paramKeyNamespace, paramKeyAppName)

type webhookHandler struct {
	k8sClient client.Client
}

func (h *webhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqID := utils.RandomString(10)
	log := logger.WithValues("request", reqID)

	vars := mux.Vars(r)

	ns, nsExist := vars[paramKeyNamespace]
	appName, appNameExist := vars[paramKeyAppName]

	if !nsExist || !appNameExist {
		_ = utils.RespondError(w, http.StatusBadRequest, fmt.Sprintf("req: %s, path is not in form of '%s'", reqID, webhookPath))
		log.Info("Bad request for path", "path", r.RequestURI)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		_ = utils.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("req: %s, cannot read webhook body", reqID))
		log.Info("cannot read webhook body", "error", err.Error())
		return
	}

	app := &cdv1.Application{}
	if err := h.k8sClient.Get(context.Background(), types.NamespacedName{Name: appName, Namespace: ns}, app); err != nil {
		_ = utils.RespondError(w, http.StatusBadRequest, fmt.Sprintf("req: %s, cannot get Application %s/%s", reqID, ns, appName))
		log.Info("Bad request for path", "path", r.RequestURI, "error", err.Error())
		return
	}

	gitCli, err := utils.GetGitCli(app, h.k8sClient)
	if err != nil {
		log.Info("Cannot initialize git cli", "error", err.Error())
		_ = utils.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("req: %s, err: %s", reqID, err.Error()))
		return
	}

	// Convert webhook
	wh, err := gitCli.ParseWebhook(r.Header, body)
	if err != nil {
		_ = utils.RespondError(w, http.StatusInternalServerError, fmt.Sprintf("req: %s, cannot parse webhook body", reqID))
		log.Info("Cannot parse webhook", "error", err.Error())
		return
	}

	if wh == nil {
		return
	}

	// Call plugin functions
	if err := HandleEvent(wh, app); err != nil {
		log.Error(err, "")
	}
}
