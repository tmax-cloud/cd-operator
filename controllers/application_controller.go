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

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/status"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdv1 "github.com/tmax-cloud/cd-operator/api/v1"
	"github.com/tmax-cloud/cd-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cd.tmax.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cd.tmax.io,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete

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
	original := instance.DeepCopy()

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

	/*
		exit, err := r.handleFinalizer(instance, original)
		if err != nil {
			log.Error(err, "")
			cond.Reason = "CannotHandleFinalizer"
			cond.Message = err.Error()
			return ctrl.Result{}, nil
		}
		if exit {
			return ctrl.Result{}, nil
		}
	*/

	// Set secret
	secretChanged := r.setSecretString(instance) // 뭐지?

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
	}

	return ctrl.Result{}, nil
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

// Set webhook-registered condition, return if it's changed or not
func (r *ApplicationReconciler) setWebhookRegisteredCond(instance *cdv1.Application) bool {
	webhookConditionChanged := false
	webhookRegistered := instance.Status.Conditions.GetCondition(cdv1.ApplicationConditionWebhookRegistered)
	if webhookRegistered == nil {
		webhookRegistered = &status.Condition{
			Type:   cdv1.ApplicationConditionWebhookRegistered,
			Status: corev1.ConditionFalse,
		}
	}

	// If token is empty, skip to register
	if instance.Spec.Git.Token == nil {
		webhookRegistered.Reason = cdv1.ApplicationConditionReasonNoGitToken
		webhookRegistered.Message = "Skipped to register webhook"
		webhookConditionChanged = instance.Status.Conditions.SetCondition(*webhookRegistered)
		return webhookConditionChanged
	}

	// Register only if the condition is false
	if webhookRegistered.IsFalse() {
		webhookRegistered.Status = corev1.ConditionFalse
		webhookRegistered.Reason = ""
		webhookRegistered.Message = ""

		gitCli, err := utils.GetGitCli(instance, r.Client)
		if err != nil {
			webhookRegistered.Reason = "invalidGitType"
			webhookRegistered.Message = fmt.Sprintf("git type %s is not supported", instance.Spec.Git.Type)
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
		webhookConditionChanged = instance.Status.Conditions.SetCondition(*webhookRegistered)
	}

	return webhookConditionChanged
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
	// webhookRegistered := instance.Status.Conditions.GetCondition(cdv1.ApplicationConditionWebhookRegistered)
	// if instance.Status.Secrets != "" && webhookRegistered != nil && (webhookRegistered.Status == corev1.ConditionTrue || webhookRegistered.Reason == cdv1.ApplicationConditionReasonNoGitToken) {
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
