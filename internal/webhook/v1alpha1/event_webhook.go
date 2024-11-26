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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	wrangellv1alpha1 "github.com/1outres/wrangell/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var eventlog = logf.Log.WithName("event-resource")

// SetupEventWebhookWithManager registers the webhook for Event in the manager.
func SetupEventWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&wrangellv1alpha1.Event{}).
		WithValidator(&EventCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-wrangell-loutres-me-v1alpha1-event,mutating=false,failurePolicy=fail,sideEffects=None,groups=wrangell.loutres.me,resources=events,verbs=create;update,versions=v1alpha1,name=vevent-v1alpha1.kb.io,admissionReviewVersions=v1

// EventCustomValidator struct is responsible for validating the Event resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type EventCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &EventCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Event.
func (v *EventCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	event, ok := obj.(*wrangellv1alpha1.Event)
	if !ok {
		return nil, fmt.Errorf("expected a Event object but got %T", obj)
	}
	eventlog.Info("Validation for Event upon creation", "name", event.GetName())

	return v.validate(ctx, event)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Event.
func (v *EventCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	event, ok := newObj.(*wrangellv1alpha1.Event)
	if !ok {
		return nil, fmt.Errorf("expected a Event object for the newObj but got %T", newObj)
	}
	eventlog.Info("Validation for Event upon update", "name", event.GetName())

	return v.validate(ctx, event)
}

func (v *EventCustomValidator) validate(ctx context.Context, event *wrangellv1alpha1.Event) (admission.Warnings, error) {
	err := event.Spec.Data.Validate()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Event.
func (v *EventCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	event, ok := obj.(*wrangellv1alpha1.Event)
	if !ok {
		return nil, fmt.Errorf("expected a Event object but got %T", obj)
	}
	eventlog.Info("Validation for Event upon deletion", "name", event.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
