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

package controller

import (
	"context"
	"github.com/1outres/wrangell/internal/gateway"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"net"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	wrangellv1alpha1 "github.com/1outres/wrangell/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// WrangellServiceReconciler reconciles a WrangellService object
type WrangellServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Server gateway.Server
}

// +kubebuilder:rbac:groups=wrangell.loutres.me,resources=wrangellservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wrangell.loutres.me,resources=wrangellservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wrangell.loutres.me,resources=wrangellservices/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WrangellService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *WrangellServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var wrangell wrangellv1alpha1.WrangellService
	err := r.Get(ctx, req.NamespacedName, &wrangell)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "unable to get WrangellService", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if !wrangell.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	err = r.updateStatus(ctx, &wrangell)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileService(ctx, wrangell)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcileDeployment(ctx, wrangell)
	if err != nil {
		return ctrl.Result{}, err
	}

	if wrangell.Spec.IdleTimeout == 0 {
		return ctrl.Result{}, nil
	} else {
		return ctrl.Result{
			RequeueAfter: time.Duration(wrangell.Spec.IdleTimeout) * time.Second,
		}, nil
	}
}

func (r *WrangellServiceReconciler) updateStatus(ctx context.Context, wrangell *wrangellv1alpha1.WrangellService) error {
	logger := log.FromContext(ctx)

	if wrangell.Spec.IdleTimeout == 0 {
		wrangell.Status.Replicas = 1
	} else if wrangell.Status.LatestRequest.IsZero() {
		wrangell.Status.Replicas = 0
	} else {
		if wrangell.Status.LatestRequest.Add(time.Second * time.Duration(wrangell.Spec.IdleTimeout)).Before(time.Now()) {
			wrangell.Status.Replicas = 0
		} else {
			wrangell.Status.Replicas = 1
		}
	}

	var svc corev1.Service
	err := r.Get(ctx, client.ObjectKey{Namespace: wrangell.Namespace, Name: "svc-" + wrangell.Name}, &svc)
	if err != nil {
		logger.Error(err, "unable to get Service")
	} else if len(svc.Status.LoadBalancer.Ingress) < 1 {
		wrangell.Status.LoadBalancerIP = ""
	} else {
		wrangell.Status.LoadBalancerIP = svc.Status.LoadBalancer.Ingress[0].IP

		r.Server.UpdateTarget(wrangell.Namespace, wrangell.Name, net.ParseIP(wrangell.Status.LoadBalancerIP), uint16(wrangell.Spec.Port), wrangell.Status.Replicas)
	}

	err = r.Status().Update(ctx, wrangell)
	return err
}

func (r *WrangellServiceReconciler) reconcileService(ctx context.Context, wrangell wrangellv1alpha1.WrangellService) error {
	logger := log.FromContext(ctx)

	svc := &corev1.Service{}
	svc.SetNamespace(wrangell.Namespace)
	svc.SetName("svc-" + wrangell.Name)

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Spec.Selector = map[string]string{"wrangell.loutres.me/name": wrangell.Name}
		svc.Spec.Type = corev1.ServiceTypeLoadBalancer

		var port, targetPort int32
		port = wrangell.Spec.Port
		if wrangell.Spec.TargetPort == 0 {
			targetPort = port
		} else {
			targetPort = wrangell.Spec.TargetPort
		}

		svc.Spec.Ports = []corev1.ServicePort{
			{
				Protocol:   corev1.ProtocolTCP,
				Port:       port,
				TargetPort: intstr.FromInt32(targetPort),
			},
		}

		return nil
	})

	if err != nil {
		logger.Error(err, "unable to create or update Service")
		return err
	}
	if op != controllerutil.OperationResultNone {
		logger.Info("reconcile Service successfully", "op", op)
	}

	return nil
}

func (r *WrangellServiceReconciler) reconcileDeployment(ctx context.Context, wrangell wrangellv1alpha1.WrangellService) error {
	logger := log.FromContext(ctx)

	deploy := &appsv1.Deployment{}
	deploy.SetNamespace(wrangell.Namespace)
	deploy.SetName(wrangell.Name)

	op, err := ctrl.CreateOrUpdate(ctx, r.Client, deploy, func() error {
		replicas := int32(wrangell.Status.Replicas)
		deploy.Spec.Replicas = &replicas
		if deploy.Spec.Selector == nil {
			deploy.Spec.Selector = &metav1.LabelSelector{}
		}
		deploy.Spec.Selector.MatchLabels = map[string]string{"wrangell.loutres.me/name": wrangell.Name}
		deploy.Spec.Template.Labels = map[string]string{"wrangell.loutres.me/name": wrangell.Name}

		deploy.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyAlways
		if len(deploy.Spec.Template.Spec.Containers) < 1 {
			deploy.Spec.Template.Spec.Containers = []corev1.Container{{}}
		} else if len(deploy.Spec.Template.Spec.Containers) > 1 {
			deploy.Spec.Template.Spec.Containers = deploy.Spec.Template.Spec.Containers[:1]
		}

		container := &deploy.Spec.Template.Spec.Containers[0]
		container.Name = "main"
		container.Image = wrangell.Spec.Image
		container.Ports = []corev1.ContainerPort{
			{
				ContainerPort: wrangell.Spec.TargetPort,
			},
		}
		container.ImagePullPolicy = corev1.PullIfNotPresent

		return nil
	})

	if err != nil {
		logger.Error(err, "unable to create or update Deployment")
		return err
	}
	if op != controllerutil.OperationResultNone {
		logger.Info("reconcile Deployment successfully", "op", op)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WrangellServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wrangellv1alpha1.WrangellService{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Named("wrangellservice").
		Complete(r)
}
