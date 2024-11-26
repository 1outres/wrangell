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
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	wrangellv1alpha1 "github.com/1outres/wrangell/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var actionlog = logf.Log.WithName("action-resource")

// SetupActionWebhookWithManager registers the webhook for Action in the manager.
func SetupActionWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&wrangellv1alpha1.Action{}).
		WithValidator(&ActionCustomValidator{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-wrangell-loutres-me-v1alpha1-action,mutating=false,failurePolicy=fail,sideEffects=None,groups=wrangell.loutres.me,resources=actions,verbs=create;update,versions=v1alpha1,name=vaction-v1alpha1.kb.io,admissionReviewVersions=v1

// ActionCustomValidator struct is responsible for validating the Action resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ActionCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &ActionCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Action.
func (v *ActionCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	action, ok := obj.(*wrangellv1alpha1.Action)
	if !ok {
		return nil, fmt.Errorf("expected a Action object but got %T", obj)
	}
	actionlog.Info("Validation for Action upon creation", "name", action.GetName())

	return v.validate(ctx, action)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Action.
func (v *ActionCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	action, ok := newObj.(*wrangellv1alpha1.Action)
	if !ok {
		return nil, fmt.Errorf("expected a Action object for the newObj but got %T", newObj)
	}
	actionlog.Info("Validation for Action upon update", "name", action.GetName())

	return v.validate(ctx, action)
}

func (v *ActionCustomValidator) validate(ctx context.Context, action *wrangellv1alpha1.Action) (admission.Warnings, error) {
	_, err := url.Parse(action.Spec.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("endpoint is not a valid URL")
	}

	err = action.Spec.Data.Validate()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Action.
func (v *ActionCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	action, ok := obj.(*wrangellv1alpha1.Action)
	if !ok {
		return nil, fmt.Errorf("expected a Action object but got %T", obj)
	}
	actionlog.Info("Validation for Action upon deletion", "name", action.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
