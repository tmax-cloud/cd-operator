package v1

import (
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/internal/apiserver"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"github.com/tmax-cloud/cd-operator/internal/wrapper"
	"github.com/tmax-cloud/cd-operator/pkg/apiserver/apis/v1/applications"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// APIVersion of the apis
	APIVersion = "v1"
)

type handler struct {
	approvalsHandler apiserver.APIHandler
	appHandler       apiserver.APIHandler
}

// NewHandler instantiates a new v1 api handler
func NewHandler(parent wrapper.RouterWrapper, cli client.Client, logger logr.Logger) (apiserver.APIHandler, error) {
	handler := &handler{}

	// /v1
	versionWrapper := wrapper.New(fmt.Sprintf("/%s/%s", apiserver.APIGroup, APIVersion), nil, handler.versionHandler)
	if err := parent.Add(versionWrapper); err != nil {
		return nil, err
	}

	// /v1/namespaces/<namespace>
	namespaceWrapper := wrapper.New("/namespaces/{namespace}", nil, nil)
	if err := versionWrapper.Add(namespaceWrapper); err != nil {
		return nil, err
	}

	// /v1/namespaces/<namespace>/Applications
	appHandler, err := applications.NewHandler(namespaceWrapper, cli, logger)
	if err != nil {
		return nil, err
	}
	handler.appHandler = appHandler

	return handler, nil
}

func (h *handler) versionHandler(w http.ResponseWriter, _ *http.Request) {
	apiResourceList := &metav1.APIResourceList{}
	apiResourceList.Kind = "APIResourceList"
	apiResourceList.GroupVersion = fmt.Sprintf("%s/%s", apiserver.APIGroup, APIVersion)
	apiResourceList.APIVersion = APIVersion

	apiResourceList.APIResources = []metav1.APIResource{
		{
			Name:       fmt.Sprintf("%s/%s", cdv1.ApplicationKind, cdv1.ApplicationAPIRunPre),
			Namespaced: true,
		},
		{
			Name:       fmt.Sprintf("%s/%s", cdv1.ApplicationKind, cdv1.ApplicationAPIRunPost),
			Namespaced: true,
		},
		{
			Name:       fmt.Sprintf("%s/%s", cdv1.ApplicationKind, cdv1.ApplicationAPIWebhookURL),
			Namespaced: true,
		},
	}

	_ = utils.RespondJSON(w, apiResourceList)
}
