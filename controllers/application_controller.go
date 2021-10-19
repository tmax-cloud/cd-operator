/*
Copyright 2021.

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

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"github.com/tmax-cloud/cd-operator/pkg/sync"
	corev1 "k8s.io/api/core/v1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	finalizer = "cd.tmax.io/finalizer"
)

var (
	syncStopFlags map[string]chan bool
	syncTickers   map[string]*time.Ticker
	syncPeriods   map[string]int64
)

//+kubebuilder:rbac:groups=cd.tmax.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cd.tmax.io,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cdapi.tmax.io,resources=applications/sync,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete

// Reconcile reconciles Application
func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("Application", req.NamespacedName)

	instance := &cdv1.Application{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "")
		return ctrl.Result{}, err
	}

	if instance.Status.Sync.Status == "" {
		sync.SetDefaultSyncStatus(instance)
	}

	if instance.Spec.SyncPolicy.SyncCheckPeriod == 0 {
		sync.SetDefaultSyncCheckPeriod(instance)
	}

	original := instance.DeepCopy()

	if err := r.manageSyncRoutine(instance); err != nil {
		log.Error(err, "manageSyncRoutine failed..")
		return ctrl.Result{}, err
	}

	// New Condition default
	cond := instance.Status.Conditions.GetCondition(cdv1.ApplicationConditionReady)
	if cond == nil {
		cond = &status.Condition{
			Type:   cdv1.ApplicationConditionReady,
			Status: corev1.ConditionFalse,
		}
	}

	defer func() {
		instance.Status.Conditions.SetCondition(*cond)
		p := client.MergeFrom(original)
		if err := r.Client.Status().Patch(ctx, instance, p); err != nil {
			log.Error(err, "")
		}
	}()

	if err := r.handleFinalizer(instance); err != nil {
		return ctrl.Result{}, err
	}

	// Set secret
	secretChanged := r.setSecretString(instance) // for WebhookSecret

	// Set webhook registered
	webhookConditionChanged := r.setWebhookRegisteredCond(instance)

	// Set ready
	readyConditionChanged := r.setReadyCond(instance)

	// If conditions changed, update status
	if secretChanged || webhookConditionChanged || readyConditionChanged {
		p := client.MergeFrom(original)
		if err := r.Client.Status().Patch(ctx, instance, p); err != nil {
			log.Error(err, "")
			return ctrl.Result{}, err
		}
		instance.Status.Sync.Status = cdv1.SyncStatusCodeOutOfSync
		if err := sync.CheckSync(r.Client, instance, false); err != nil {
			log.Error(err, "")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ApplicationReconciler) handleFinalizer(instance *cdv1.Application) error {
	isAppMarkedToBeDeleted := instance.DeletionTimestamp != nil
	if isAppMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(instance, finalizer) {
			if err := r.finalizeApp(instance); err != nil {
				return err
			}
		}
		controllerutil.RemoveFinalizer(instance, finalizer)
		if err := r.Update(context.Background(), instance); err != nil {
			return err
		}
		return nil
	}

	if !controllerutil.ContainsFinalizer(instance, finalizer) {
		controllerutil.AddFinalizer(instance, finalizer)
		if err := r.Update(context.Background(), instance); err != nil {
			return err
		}
	}
	return nil
}

func (r *ApplicationReconciler) finalizeApp(instance *cdv1.Application) error {
	if err := deleteSyncFlag(instance); err != nil {
		r.Log.Error(err, "deleteSyncFlag error")
		return err
	}
	// if instance.Spec.Source.Token != nil {
	// 	gitCli, err := utils.GetGitCli(instance, r.Client)
	// 	if err != nil {
	// 		r.Log.Error(err, "")
	// 		return err
	// 	}
	// 	hookList, err := gitCli.ListWebhook()
	// 	if err != nil {
	// 		r.Log.Error(err, "")
	// 		return err
	// 	}
	// 	for _, h := range hookList {
	// 		if h.URL == instance.GetWebhookServerAddress() {
	// 			r.Log.Info("Deleting webhook " + h.URL)
	// 			if err := gitCli.DeleteWebhook(h.ID); err != nil {
	// 				r.Log.Error(err, "")
	// 				return err
	// 			}
	// 		}
	// 	}
	// }
	return nil
}

func (r *ApplicationReconciler) manageSyncRoutine(instance *cdv1.Application) error {
	instance.Status.Sync.Status = cdv1.SyncStatusCodeUnknown
	appKeyName := instance.Name + "/" + instance.Namespace

	initSyncFlags()

	if instance.Spec.SyncPolicy.SyncCheckPeriod != syncPeriods[appKeyName] {
		if err := deleteSyncFlag(instance); err != nil {
			return err
		}
	}
	if !existingSyncRoutine(appKeyName) {
		done, ticker := registerSyncFlag(instance)
		go sync.PeriodicSyncCheck(r.Client, instance, done, ticker)
	}
	return nil
}

func initSyncFlags() {
	if syncStopFlags == nil {
		syncStopFlags = make(map[string]chan bool)
	}
	if syncTickers == nil {
		syncTickers = make(map[string]*time.Ticker)
	}
	if syncPeriods == nil {
		syncPeriods = make(map[string]int64)
	}
}

func existingSyncRoutine(appKeyName string) bool {
	return syncStopFlags[appKeyName] != nil && syncTickers[appKeyName] != nil
}

func registerSyncFlag(instance *cdv1.Application) (chan bool, *time.Ticker) {
	appKeyName := instance.Name + "/" + instance.Namespace

	done := make(chan bool)
	syncStopFlags[appKeyName] = done

	ticker := time.NewTicker(time.Second * time.Duration(instance.Spec.SyncPolicy.SyncCheckPeriod))
	syncTickers[appKeyName] = ticker

	syncPeriods[appKeyName] = instance.Spec.SyncPolicy.SyncCheckPeriod

	return done, ticker
}

func deleteSyncFlag(instance *cdv1.Application) error {
	appKeyName := instance.Name + "/" + instance.Namespace

	if syncStopFlags[appKeyName] != nil && syncTickers[appKeyName] != nil {
		syncStopFlags[appKeyName] <- true
		delete(syncStopFlags, appKeyName)

		syncTickers[appKeyName].Stop()
		delete(syncTickers, appKeyName)

		delete(syncPeriods, appKeyName)

		return nil
	}
	return fmt.Errorf("No Sync Flags name" + appKeyName + "!")
}

// Set status.secrets, return if it's changed or not
func (r *ApplicationReconciler) setSecretString(instance *cdv1.Application) bool {
	secretChanged := false
	if instance.Status.Secrets == "" {
		instance.Status.Secrets = utils.RandomString(20)
		secretChanged = true
	}

	return secretChanged
}

// Set ready condition, return if it's changed or not
func (r *ApplicationReconciler) setReadyCond(instance *cdv1.Application) bool {
	ready := instance.Status.Conditions.GetCondition(cdv1.ApplicationConditionReady)
	if ready == nil {
		ready = &status.Condition{
			Type:   cdv1.ApplicationConditionReady,
			Status: corev1.ConditionFalse,
		}
	}

	// TODO
	// // For now, only checked is if webhook-registered is true & secrets are set
	// webhookRegistered := instance.Status.Conditions.GetCondition(cicdv1.IntegrationConfigConditionWebhookRegistered)
	// if instance.Status.Secrets != "" && webhookRegistered != nil && (webhookRegistered.Status == corev1.ConditionTrue || webhookRegistered.Reason == cicdv1.IntegrationConfigConditionReasonNoGitToken) {
	// 	ready.Status = corev1.ConditionTrue
	// }
	readyConditionChanged := instance.Status.Conditions.SetCondition(*ready)

	return readyConditionChanged
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cdv1.Application{}).
		Complete(r)
}

// Set webhook-registered condition, return if it's changed or not
func (r *ApplicationReconciler) setWebhookRegisteredCond(instance *cdv1.Application) bool {
	webhookRegistered := instance.Status.Conditions.GetCondition(cdv1.ApplicationConditionWebhookRegistered)
	if webhookRegistered == nil {
		webhookRegistered = &status.Condition{
			Type:   cdv1.ApplicationConditionWebhookRegistered,
			Status: corev1.ConditionFalse,
		}
	}

	// If token is empty, skip to register
	if instance.Spec.Source.Token == nil {
		webhookRegistered.Reason = cdv1.ApplicationConditionReasonNoGitToken
		webhookRegistered.Message = "Skipped to register webhook"
		webhookRegisteredCondChanged := instance.Status.Conditions.SetCondition(*webhookRegistered)
		return webhookRegisteredCondChanged
	}

	// Register only if the condition is false
	if webhookRegistered.IsFalse() {
		webhookRegistered.Status = corev1.ConditionFalse
		webhookRegistered.Reason = ""
		webhookRegistered.Message = ""

		gitCli, err := utils.GetGitCli(instance, r.Client)
		if err != nil {
			webhookRegistered.Reason = "gitCliErr"
			webhookRegistered.Message = err.Error()
		} else {
			addr := instance.GetWebhookServerAddress()
			isUnique := true
			r.Log.Info("Registering webhook " + addr)
			entries, err := gitCli.ListWebhook()
			if err != nil {
				webhookRegistered.Reason = "webhookRegisterFailed"
				webhookRegistered.Message = err.Error()
			}
			for _, e := range entries {
				if addr == e.URL {
					webhookRegistered.Reason = "webhookRegisterFailed"
					webhookRegistered.Message = "same webhook has already registered"
					isUnique = false
					break
				}
			}
			if isUnique {
				if err := gitCli.RegisterWebhook(addr); err != nil {
					webhookRegistered.Reason = "webhookRegisterFailed"
					webhookRegistered.Message = err.Error()
				} else {
					webhookRegistered.Status = corev1.ConditionTrue
					webhookRegistered.Reason = ""
					webhookRegistered.Message = ""
				}
			}
		}
		webhookRegisteredCondChanged := instance.Status.Conditions.SetCondition(*webhookRegistered)
		return webhookRegisteredCondChanged
	}
	webhookRegisteredCondChanged := instance.Status.Conditions.SetCondition(*webhookRegistered)
	return webhookRegisteredCondChanged
}
