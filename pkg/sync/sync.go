package sync

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/manifestmanager"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	log              = logf.Log.WithName("sync")
	PlainYamlManager manifestmanager.ManifestManager
	HelmManager      manifestmanager.ManifestManager
)

const (
	defaultSyncCheckPerod = 60
)

func SetDefaultSyncStatus(app *cdv1.Application) {
	app.Status.Sync.Status = cdv1.SyncStatusCodeUnknown
	app.Status.Sync.TimeCheck = 0
}

func SetDefaultSyncCheckPeriod(app *cdv1.Application) {
	app.Spec.SyncPolicy.SyncCheckPeriod = defaultSyncCheckPerod
}

func PeriodicSyncCheck(cli client.Client, app *cdv1.Application, done chan bool, ticker *time.Ticker) {
	randNum := rand.Int()
	log.Info(fmt.Sprintf("Periodic sync check %d start..", randNum))

	for {
		select {
		case <-done:
			log.Info(fmt.Sprintf("Periodic sync check %d finished", randNum))
			return
		case <-ticker.C:
			log.Info(fmt.Sprintf("Periodic sync check %d", randNum))
			if err := CheckSync(cli, app, false); err != nil {
				log.Error(err, "")
			}
		}
	}
}

func CheckSync(cli client.Client, app *cdv1.Application, forced bool) error {
	log.Info("Checking Sync status...")

	if PlainYamlManager == nil {
		PlainYamlManager = manifestmanager.NewPlainYamlManager(context.Background(), cli, http.DefaultClient)
	}
	if HelmManager == nil {
		HelmManager = manifestmanager.NewHelmManager(context.Background(), cli)
	}

	var mgr manifestmanager.ManifestManager

	switch app.Spec.Source.Type {
	case cdv1.ApplicationSourceTypePlainYAML:
		mgr = PlainYamlManager
	case cdv1.ApplicationSourceTypeHelm:
		mgr = HelmManager
	default:
		err := fmt.Errorf("get sync manager failed")
		return err
	}

	if err := mgr.Sync(app, forced); err != nil {
		return err
	}

	return nil
}
