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

// TODO: resource 일단 다 배포하고 실패한 resource들 error를 array로
func CheckSync(cli client.Client, app *cdv1.Application, forced bool) error {
	log.Info("Checking Sync status...")
	mgr := manifestmanager.ManifestManager{Client: cli, Context: context.Background()}
	urls, err := mgr.GetManifestURLList(app)
	if err != nil {
		log.Error(err, "GetManifestURLList failed..")
		return err
	}
	oldDeployResources, err := mgr.GetDeployResourceList(app)
	if err != nil {
		log.Error(err, "GetDeployResourceList failed")
		return err
	}

	updatedDeployResources := make(map[string]*cdv1.DeployResource)

	for _, url := range urls {
		manifestRawobj, err := mgr.ObjectFromManifest(url, app)
		if err != nil {
			log.Error(err, "Get object from manifest failed..")
			return err
		}
		updatedDeployResource, err := mgr.UpdateDeployResource(manifestRawobj, app)
		if err != nil {
			log.Error(err, "NewDeployResource failed..")
			return err
		}
		updatedDeployResources[updatedDeployResource.Name] = updatedDeployResource

		manifestModifiedObj, err := mgr.CompareDeployWithManifest(manifestRawobj)
		if manifestModifiedObj == nil && err != nil {
			log.Error(err, "Compare deployed resource with manifest failed..")
			return err
		}
		if manifestModifiedObj != nil && (app.Spec.SyncPolicy.AutoSync || forced) {
			exist := (err == nil)
			if err := mgr.ApplyManifest(exist, app, manifestModifiedObj); err != nil {
				log.Error(err, "Apply manifest failed..")
				return err
			}
		}
	}

	for _, oldDeployResource := range oldDeployResources.Items {
		if updatedDeployResources[oldDeployResource.Name] == nil {
			if err := mgr.DeleteDeployResource(&oldDeployResource); err != nil {
				log.Error(err, "DeleteDeployResource failed..")
				return err
			}
		}
	}

	if app.Spec.SyncPolicy.AutoSync {
		app.Status.Sync.Status = cdv1.SyncStatusCodeSynced
	}

	return nil
}
