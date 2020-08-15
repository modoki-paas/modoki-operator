/*


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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	modokiv1alpha1 "github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"github.com/modoki-paas/modoki-operator/generator"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Generator generator.Generator
}

// +kubebuilder:rbac:groups=modoki.tsuzu.dev,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=modoki.tsuzu.dev,resources=applications/status,verbs=get;update;patch

func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("application", req.NamespacedName)

	// your logic here
	var app modokiv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	objs, err := r.Generator.Generate(ctx, &app.Spec)

	if err != nil {
		log.Error(err, "failed to generate yaml")

		return ctrl.Result{Requeue: true}, err
	}

	for _, obj := range objs {
		gvk := schema.FromAPIVersionAndKind(
			obj.GetAPIVersion(), obj.GetKind(),
		)

		current := &unstructured.Unstructured{}
		current.SetGroupVersionKind(gvk)

		err := r.Client.Get(ctx, client.ObjectKey{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		}, current)

		if err != nil {
			if !apierrors.IsNotFound(err) {
				log.Error(
					err,
					"failed to find an child resource",
					"type", gvk, "name", obj.GetName(), "namespace", obj.GetNamespace(),
				)

				return ctrl.Result{Requeue: true}, err
			}

			if err := r.Client.Create(ctx, obj); err != nil {
				log.Error(
					err,
					"failed to create a child resource",
					"type", gvk, "name", obj.GetName(), "namespace", obj.GetNamespace(),
				)

				return ctrl.Result{Requeue: true}, err
			}

			continue
		}

		diff, err := client.MergeFrom(current).Data(obj)
		if err != nil {
			log.Error(err, "merge from error")
		} else {
			log.Info("diff is ...", "diff", string(diff))
		}

		if err := r.Client.Patch(ctx, obj, client.MergeFrom(current)); err != nil {
			log.Error(
				err,
				"failed to patch the child resource",
				"type", gvk, "name", obj.GetName(), "namespace", obj.GetNamespace(),
			)

			return ctrl.Result{Requeue: true}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager, ch <-chan event.GenericEvent) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&modokiv1alpha1.Application{}, builder.WithPredicates()).
		Watches(&source.Channel{Source: ch}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}
