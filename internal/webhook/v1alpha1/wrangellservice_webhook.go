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
var wrangellservicelog = logf.Log.WithName("wrangellservice-resource")

// SetupWrangellServiceWebhookWithManager registers the webhook for WrangellService in the manager.
func SetupWrangellServiceWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&wrangellv1alpha1.WrangellService{}).
		WithValidator(&WrangellServiceCustomValidator{}).
		WithDefaulter(&WrangellServiceCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-wrangell-loutres-me-v1alpha1-wrangellservice,mutating=true,failurePolicy=fail,sideEffects=None,groups=wrangell.loutres.me,resources=wrangellservices,verbs=create;update,versions=v1alpha1,name=mwrangellservice-v1alpha1.kb.io,admissionReviewVersions=v1

// WrangellServiceCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind WrangellService when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type WrangellServiceCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &WrangellServiceCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind WrangellService.
func (d *WrangellServiceCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	wrangellservice, ok := obj.(*wrangellv1alpha1.WrangellService)

	if !ok {
		return fmt.Errorf("expected an WrangellService object but got %T", obj)
	}
	wrangellservicelog.Info("Defaulting for WrangellService", "name", wrangellservice.GetName())

	// TODO(user): fill in your defaulting logic.

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-wrangell-loutres-me-v1alpha1-wrangellservice,mutating=false,failurePolicy=fail,sideEffects=None,groups=wrangell.loutres.me,resources=wrangellservices,verbs=create;update,versions=v1alpha1,name=vwrangellservice-v1alpha1.kb.io,admissionReviewVersions=v1

// WrangellServiceCustomValidator struct is responsible for validating the WrangellService resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type WrangellServiceCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &WrangellServiceCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type WrangellService.
func (v *WrangellServiceCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	wrangellservice, ok := obj.(*wrangellv1alpha1.WrangellService)
	if !ok {
		return nil, fmt.Errorf("expected a WrangellService object but got %T", obj)
	}
	wrangellservicelog.Info("Validation for WrangellService upon creation", "name", wrangellservice.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type WrangellService.
func (v *WrangellServiceCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	wrangellservice, ok := newObj.(*wrangellv1alpha1.WrangellService)
	if !ok {
		return nil, fmt.Errorf("expected a WrangellService object for the newObj but got %T", newObj)
	}
	wrangellservicelog.Info("Validation for WrangellService upon update", "name", wrangellservice.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type WrangellService.
func (v *WrangellServiceCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	wrangellservice, ok := obj.(*wrangellv1alpha1.WrangellService)
	if !ok {
		return nil, fmt.Errorf("expected a WrangellService object but got %T", obj)
	}
	wrangellservicelog.Info("Validation for WrangellService upon deletion", "name", wrangellservice.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}