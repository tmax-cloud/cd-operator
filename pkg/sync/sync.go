package sync

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/pkg/manifestmanager"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("sync")

const (
	defaultSyncCheckPerod = 60
)

func SetDefaultSyncStatus(app *cdv1.Application) {
	app.Status.Sync.Status = cdv1.SyncStatusCodeUnknown
	app.Status.Sync.TimeCheck = 0
}

func SetDefaultSyncCheckPerod(app *cdv1.Application) {
	app.Spec.SyncPolicy.SyncCheckPeriod = defaultSyncCheckPerod
}

func PeriodicSyncCheck(cli client.Client, app *cdv1.Application, checking chan bool) {
	randNum := rand.Int()
	log.Info(fmt.Sprintf("Periodic sync check %d start..", randNum))

	for <-checking {
		app.Status.Sync.TimeCheck++
		if app.Status.Sync.TimeCheck >= app.Spec.SyncPolicy.SyncCheckPeriod {
			log.Info(fmt.Sprintf("Periodic sync check %d", randNum))
			app.Status.Sync.TimeCheck = 0
			if err := CheckSync(cli, app, false); err != nil {
				log.Error(err, "")
			}
		}
		checking <- true
		time.Sleep(time.Second * 1)
	}
	log.Info(fmt.Sprintf("Periodic sync check %d finished", randNum))
}

// TODO: resource 일단 다 배포하고 실패한 resource들 error를 array로
func CheckSync(cli client.Client, app *cdv1.Application, forced bool) error {
	log.Info("Checking Sync status...")
	mgr := manifestmanager.ManifestManager{Client: cli, Context: context.Background()}
	urls, err := mgr.GetManifestURLList(app)
	if err != nil {
		log.Error(err, "")
		return err
	}
	for _, url := range urls {
		obj, err := mgr.ObjectFromManifest(url, app)
		if err != nil {
			log.Error(err, "")
			return err
		}
		manifestObj, err := mgr.CompareWithManifest(obj)
		if manifestObj == nil && err != nil {
			log.Error(err, "")
			return err
		}
		if manifestObj != nil && (app.Spec.SyncPolicy.AutoSync || forced) {
			exist := (err == nil)
			if err := mgr.ApplyManifest(exist, app, manifestObj); err != nil {
				log.Error(err, "")
				return err
			}
		}
	}
	if app.Spec.SyncPolicy.AutoSync {
		app.Status.Sync.Status = cdv1.SyncStatusCodeSynced
	}

	return nil
}
