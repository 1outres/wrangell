/*
Copyright 2024.

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

package v1alpha1

import (
	"context"
	"fmt"
	wrangellv1alpha1 "github.com/1outres/wrangell/api/v1alpha1"
	"github.com/expr-lang/expr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// nolint:unused
// log is for logging in this package.
var triggerlog = logf.Log.WithName("trigger-resource")

// SetupTriggerWebhookWithManager registers the webhook for Trigger in the manager.
func SetupTriggerWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&wrangellv1alpha1.Trigger{}).
		WithValidator(&TriggerCustomValidator{client: mgr.GetClient()}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-wrangell-loutres-me-v1alpha1-trigger,mutating=false,failurePolicy=fail,sideEffects=None,groups=wrangell.loutres.me,resources=triggers,verbs=create;update,versions=v1alpha1,name=vtrigger-v1alpha1.kb.io,admissionReviewVersions=v1

// +kubebuilder:rbac:groups=wrangell.loutres.me,resources=triggers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wrangell.loutres.me,resources=triggers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wrangell.loutres.me,resources=actions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wrangell.loutres.me,resources=events,verbs=get;list;watch;create;update;patch;delete

// TriggerCustomValidator struct is responsible for validating the Trigger resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type TriggerCustomValidator struct {
	client client.Client
}

var _ webhook.CustomValidator = &TriggerCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Trigger.
func (v *TriggerCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	trigger, ok := obj.(*wrangellv1alpha1.Trigger)
	if !ok {
		return nil, fmt.Errorf("expected a Trigger object but got %T", obj)
	}
	triggerlog.Info("Validation for Trigger upon creation", "name", trigger.GetName())

	return v.validate(ctx, trigger)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Trigger.
func (v *TriggerCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	trigger, ok := newObj.(*wrangellv1alpha1.Trigger)
	if !ok {
		return nil, fmt.Errorf("expected a Trigger object for the newObj but got %T", newObj)
	}
	triggerlog.Info("Validation for Trigger upon update", "name", trigger.GetName())

	return v.validate(ctx, trigger)
}

func (v *TriggerCustomValidator) validate(ctx context.Context, trigger *wrangellv1alpha1.Trigger) (admission.Warnings, error) {
	var event wrangellv1alpha1.Event
	err := v.client.Get(ctx, client.ObjectKey{Name: trigger.Spec.Event}, &event)
	if err != nil {
		return nil, fmt.Errorf("event %s not found", trigger.Spec.Event)
	}

	env := event.Spec.Data.CreateEmptyData()
	fmt.Println(env)
	for _, condition := range trigger.Spec.Conditions {
		_, err := expr.Compile(condition, expr.Env((map[string]interface{})(env)))
		if err != nil {
			return nil, fmt.Errorf("error compiling condition %s: %s", condition, err)
		}
	}

	for _, ta := range trigger.Spec.Actions {
		var action wrangellv1alpha1.Action
		err := v.client.Get(ctx, client.ObjectKey{Name: ta.Action}, &action)
		if err != nil {
			return nil, fmt.Errorf("action %s not found", ta.Action)
		}

		for _, param := range ta.Params {
			field, exists := action.Spec.Data.GetField(param.Name)
			if !exists {
				return nil, fmt.Errorf("field %s not found in action %s", param.Name, ta.Action)
			}

			if param.FromYaml == nil && param.FromData == nil {
				return nil, fmt.Errorf("field %s source is not specified", param.Name)
			}

			if param.FromData != nil {
				sourceField, exists := event.Spec.Data.GetField(param.FromData.Name)
				if !exists {
					return nil, fmt.Errorf("field %s not found in event data", param.FromData.Name)
				}

				if field.Type != sourceField.Type {
					return nil, fmt.Errorf("field %s type mismatch: expected %s, got %s", param.Name, field.Type, sourceField.Type)
				}
			}
		}
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Trigger.
func (v *TriggerCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	trigger, ok := obj.(*wrangellv1alpha1.Trigger)
	if !ok {
		return nil, fmt.Errorf("expected a Trigger object but got %T", obj)
	}
	triggerlog.Info("Validation for Trigger upon deletion", "name", trigger.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
