package dispatcher

import (
	"fmt"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/git"
	"github.com/tmax-cloud/cd-operator/pkg/manifestmanager"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Dispatcher dispatches IntegrationJob when webhook is called
// A kind of 'plugin' for webhook handler
type Dispatcher struct {
	Client client.Client
}

// Handle handles push events
func (d Dispatcher) Handle(webhook *git.Webhook, app *cdv1.Application) error {
	push := webhook.Push
	if push == nil {
		return fmt.Errorf("push struct is nil")
	}

	// Push일 경우
	if webhook.EventType == git.EventTypePush && push != nil {
		var mgr manifestmanager.ManifestManager
		urls, err := mgr.GetManifestURLList(app)
		if err != nil {
			return err
		}
		for _, url := range urls {
			if err = mgr.ApplyManifest(url, app); err != nil {
				return err
			}
		}
	}
	return nil
}
