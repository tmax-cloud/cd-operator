package applications

import (
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/internal/apiserver"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"github.com/tmax-cloud/cd-operator/internal/wrapper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// APIVersion of the api
	APIVersion = "v1"

	appParamKey = "appName"
)

type handler struct {
	k8sClient client.Client
	log       logr.Logger

	authorizer apiserver.Authorizer
}

// NewHandler instantiates a new integration configs api handler
func NewHandler(parent wrapper.RouterWrapper, cli client.Client, logger logr.Logger) (apiserver.APIHandler, error) {
	handler := &handler{k8sClient: cli, log: logger}

	// Authorizer
	authClient, err := utils.AuthClient()
	if err != nil {
		return nil, err
	}
	handler.authorizer = apiserver.NewAuthorizer(authClient, apiserver.APIGroup, APIVersion, "create")

	// /applications/<Application>
	appWrapper := wrapper.New(fmt.Sprintf("/%s/{%s}", cdv1.ApplicationKind, appParamKey), nil, nil)
	if err := parent.Add(appWrapper); err != nil {
		return nil, err
	}
	appWrapper.Router().Use(handler.authorizer.Authorize)

	// /applications/<Application>/webhookurl
	webhookURLWrapper := wrapper.New("/"+cdv1.ApplicationAPIWebhookURL, []string{http.MethodGet}, handler.webhookURLHandler)
	if err := appWrapper.Add(webhookURLWrapper); err != nil {
		return nil, err
	}

	return handler, nil
}
