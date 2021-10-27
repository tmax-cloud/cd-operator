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
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	"github.com/tmax-cloud/cd-operator/pkg/manifestmanager"
	"github.com/tmax-cloud/cd-operator/pkg/sync"

	v1 "k8s.io/api/core/v1"
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
	syncStopFlags map[string]chan bool    = make(map[string]chan bool)
	syncTickers   map[string]*time.Ticker = make(map[string]*time.Ticker)
	syncPeriods   map[string]int64        = make(map[string]int64)
)

//+kubebuilder:rbac:groups=cd.tmax.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cd.tmax.io,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cdapi.tmax.io,resources=applications/sync,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete

// Reconcile reconciles Application
func (r *ApplicationReconciler) Reconcile(c context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	if err := r.checkAppDestNamespace(instance); err != nil {
		log.Error(err, "check app destination namespace failed..")
		return ctrl.Result{}, err
	}

	if instance.Status.Sync.Status == "" {
		sync.SetDefaultSyncStatus(instance)
	}

	if instance.Spec.SyncPolicy.SyncCheckPeriod == 0 {
		sync.SetDefaultSyncCheckPeriod(instance)
	}

	original := instance.DeepCopy()

	r.manageSyncRoutine(instance)

	// New Condition default
	cond := meta.FindStatusCondition(instance.Status.Conditions, cdv1.ApplicationConditionReady)
	if cond == nil {
		cond = &metav1.Condition{
			Type:   cdv1.ApplicationConditionReady,
			Status: metav1.ConditionFalse,
		}
	}

	defer func() {
		meta.SetStatusCondition(&instance.Status.Conditions, *cond)
		p := client.MergeFrom(original)
		if err := r.Client.Status().Patch(ctx, instance, p); err != nil {
			log.Error(err, "")
		}
	}()

	if err := r.handleFinalizer(instance); err != nil {
		return ctrl.Result{}, err
	}

	// Set secret
	r.setSecretString(instance) // for WebhookSecret

	// Set webhook registered
	r.setWebhookRegisteredCond(instance)

	// Set ready
	r.setReadyCond(instance)

	instance.Status.Sync.Status = cdv1.SyncStatusCodeOutOfSync
	if instance.DeletionTimestamp == nil {
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
	deleteSyncFlag(instance)

	if err := r.clearDeployedResources(instance); err != nil {
		r.Log.Error(err, "Delete deployed resources failed..")
		return err
	}

	if err := r.clearWebhook(instance); err != nil {
		r.Log.Error(err, "Delete webhook failed..")
		return err
	}
	return nil
}

// TODO: Namespace 처리 방안
func (r *ApplicationReconciler) clearDeployedResources(instance *cdv1.Application) error {
	mgr := manifestmanager.ManifestManager{Client: r.Client, Context: context.Background()}

	deployedResourceList, err := mgr.GetDeployResourceList(instance)
	if err != nil {
		return err
	}
	for _, deployedResource := range deployedResourceList.Items {
		if err := mgr.DeleteDeployResource(&deployedResource); err != nil {
			return err
		}
	}

	return nil
}

func (r *ApplicationReconciler) clearWebhook(instance *cdv1.Application) error {
	if instance.Spec.Source.Token != nil {
		gitCli, err := utils.GetGitCli(instance, r.Client)
		if err != nil {
			return err
		}
		hookList, err := gitCli.ListWebhook()
		if err != nil {
			return err
		}
		for _, h := range hookList {
			if h.URL == instance.GetWebhookServerAddress() {
				r.Log.Info("Deleting webhook " + h.URL)
				if err := gitCli.DeleteWebhook(h.ID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *ApplicationReconciler) checkAppDestNamespace(instance *cdv1.Application) error {
	if err := r.Client.Get(context.Background(), types.NamespacedName{Name: instance.Spec.Destination.Namespace}, &v1.Namespace{}); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if err := r.Client.Create(context.Background(), &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: instance.Spec.Destination.Namespace}}); err != nil {
			return err
		}
	}
	return nil
}

func (r *ApplicationReconciler) manageSyncRoutine(instance *cdv1.Application) {
	instance.Status.Sync.Status = cdv1.SyncStatusCodeUnknown

	deleteSyncFlag(instance)
	done, ticker := registerSyncFlag(instance)
	go sync.PeriodicSyncCheck(r.Client, instance, done, ticker)
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

func deleteSyncFlag(instance *cdv1.Application) {
	appKeyName := instance.Name + "/" + instance.Namespace

	if syncStopFlags[appKeyName] != nil && syncTickers[appKeyName] != nil {
		syncStopFlags[appKeyName] <- true
		syncTickers[appKeyName].Stop()
	}
	delete(syncStopFlags, appKeyName)
	delete(syncTickers, appKeyName)
	delete(syncPeriods, appKeyName)
}

// Set status.secrets, return if it's changed or not
func (r *ApplicationReconciler) setSecretString(instance *cdv1.Application) {
	if instance.Status.Secrets == "" {
		instance.Status.Secrets = utils.RandomString(20)
	}
}

// Set ready condition, return if it's changed or not
func (r *ApplicationReconciler) setReadyCond(instance *cdv1.Application) {
	ready := meta.FindStatusCondition(instance.Status.Conditions, cdv1.ApplicationConditionReady)
	if ready == nil {
		ready = &metav1.Condition{
			Type:   cdv1.ApplicationConditionReady,
			Status: metav1.ConditionFalse,
		}
	}

	// TODO
	// // For now, only checked is if webhook-registered is true & secrets are set
	// webhookRegistered := instance.Status.Conditions.GetCondition(cicdv1.IntegrationConfigConditionWebhookRegistered)
	// if instance.Status.Secrets != "" && webhookRegistered != nil && (webhookRegistered.Status == corev1.ConditionTrue || webhookRegistered.Reason == cicdv1.IntegrationConfigConditionReasonNoGitToken) {
	// 	ready.Status = corev1.ConditionTrue
	// }
	meta.SetStatusCondition(&instance.Status.Conditions, *ready)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cdv1.Application{}).
		Complete(r)
}

// Set webhook-registered condition, return if it's changed or not
func (r *ApplicationReconciler) setWebhookRegisteredCond(instance *cdv1.Application) {
	webhookRegistered := meta.FindStatusCondition(instance.Status.Conditions, cdv1.ApplicationConditionWebhookRegistered)
	if webhookRegistered == nil {
		webhookRegistered = &metav1.Condition{
			Type:   cdv1.ApplicationConditionWebhookRegistered,
			Status: metav1.ConditionFalse,
		}
	}

	// If token is empty, skip to register
	if instance.Spec.Source.Token == nil {
		webhookRegistered.Reason = cdv1.ApplicationConditionReasonNoGitToken
		webhookRegistered.Message = "Skipped to register webhook"
		meta.SetStatusCondition(&instance.Status.Conditions, *webhookRegistered)
	}

	// Register only if the condition is false
	if webhookRegistered.Status == metav1.ConditionFalse {
		webhookRegistered.Status = metav1.ConditionFalse
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
					webhookRegistered.Status = metav1.ConditionTrue
					webhookRegistered.Reason = ""
					webhookRegistered.Message = ""
				}
			}
		}
		meta.SetStatusCondition(&instance.Status.Conditions, *webhookRegistered)
	}
	meta.SetStatusCondition(&instance.Status.Conditions, *webhookRegistered)
}
