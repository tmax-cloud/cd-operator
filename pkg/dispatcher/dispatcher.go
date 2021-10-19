package dispatcher

import (
	"fmt"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/git"
	"github.com/tmax-cloud/cd-operator/pkg/sync"
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
		app.Status.Sync.Status = cdv1.SyncStatusCodeOutOfSync
		if err := sync.CheckSync(d.Client, app, true); err != nil {
			return err
		}
	}
	return nil
}
