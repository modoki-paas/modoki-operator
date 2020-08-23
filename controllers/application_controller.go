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
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/labstack/gommon/log"
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

	"github.com/modoki-paas/modoki-operator/api/v1alpha1"
	modokiv1alpha1 "github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"github.com/modoki-paas/modoki-operator/generators"
	"github.com/modoki-paas/modoki-operator/pkg/yaml"
)

const (
	lastAppliedAnnotationsKey = "kubectl.kubernetes.io/last-applied-configuration"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Generator generators.Generator
}

// +kubebuilder:rbac:groups=modoki.tsuzu.dev,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=modoki.tsuzu.dev,resources=applications/status,verbs=get;update;patch

func (r *ApplicationReconciler) setMetadata(app *v1alpha1.Application, obj *unstructured.Unstructured) error {
	annotations := obj.GetAnnotations()
	buf := bytes.NewBuffer(nil)
	if err := unstructured.UnstructuredJSONScheme.Encode(obj, buf); err != nil {
		log.Error(err, "the object cannot be marshaled", "obj", obj)

		return err
	}

	annotations[lastAppliedAnnotationsKey] = buf.String()
	obj.SetAnnotations(annotations)

	if err := ctrl.SetControllerReference(app, obj, r.Scheme); err != nil {
		log.Error(err, "failed to set controller reference", "info", obj.GetKind()+"/"+obj.GetNamespace()+"/"+obj.GetName())

		return err
	}

	return nil
}

func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (res ctrl.Result, err error) {
	ctx := context.Background()
	log := r.Log.WithValues("application", req.NamespacedName)

	// your logic here
	var app modokiv1alpha1.Application
	if err := r.Get(ctx, req.NamespacedName, &app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defer func() {
		if err != nil {
			copied := app.DeepCopy()
			copied.Status.Status = modokiv1alpha1.ApplicationDeploymentFailed
			copied.Status.Message = err.Error()

			if err := r.Client.Patch(ctx, copied, client.MergeFrom(&app)); err != nil {
				log.Error(err, "failed to update status", "status", app.Status.Status)

				res.Requeue = true
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	log.Info("generator started")
	objs, err := r.Generator.Generate(ctx, &app)

	if err != nil {
		log.Error(err, "failed to generate yaml")

		return ctrl.Result{Requeue: true}, fmt.Errorf("failed to generate yaml: %w", err)
	}
	log.Info("generator returned", "items", len(objs))

	resources := make([]v1alpha1.ApplicationResource, 0, len(objs))
	for _, obj := range objs {
		if err := r.setMetadata(&app, obj); err != nil {
			return ctrl.Result{Requeue: false}, err
		}

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
					"failed to find a child resource",
					"type", gvk, "name", obj.GetName(), "namespace", obj.GetNamespace(),
				)

				return ctrl.Result{Requeue: true}, fmt.Errorf("failed to find a child resource: %w", err)
			}

			if err := r.Client.Create(ctx, obj); err != nil {
				log.Error(
					err,
					"failed to create a child resource",
					"type", gvk, "name", obj.GetName(), "namespace", obj.GetNamespace(),
				)

				return ctrl.Result{Requeue: true}, fmt.Errorf("failed to create a child resource: %w", err)
			}

			continue
		}

		var lastApplied *unstructured.Unstructured
		if lastAppliedJSON, ok := current.GetAnnotations()[lastAppliedAnnotationsKey]; ok {
			lastApplied, err = yaml.ParseUnstructured([]byte(lastAppliedJSON))

			if err != nil {
				return ctrl.Result{Requeue: false}, fmt.Errorf("last applied json is broken: %w", err)
			}

			if err := r.setMetadata(&app, lastApplied); err != nil {
				return ctrl.Result{Requeue: false}, fmt.Errorf("setting metadata to last applied json failed: %w", err)
			}
		} else {
			lastApplied = current
		}

		diff, err := client.MergeFrom(lastApplied).Data(obj)
		if err != nil {
			log.Error(err, "failed to generate diff", "lastApplied", lastApplied, "new", obj)

			return ctrl.Result{Requeue: false}, fmt.Errorf("failed to generate diff: %w", err)
		}

		if len(diff) > 2 { // != {}
			err := r.Client.Patch(ctx, obj, client.MergeFrom(lastApplied))

			if err != nil {
				log.Error(
					err,
					"failed to patch the child resource",
					"type", gvk, "name", obj.GetName(), "namespace", obj.GetNamespace(),
				)

				return ctrl.Result{Requeue: true}, fmt.Errorf("failed to patch the child resource: %w", err)
			}
		}

		resources = append(
			resources,
			v1alpha1.ApplicationResource{
				TypeMeta:  app.TypeMeta,
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		)
	}

	deleted := v1alpha1.FilterApplicationResource(app.Status.Resources, resources)

	for i := range deleted {
		gvk := schema.FromAPIVersionAndKind(
			deleted[i].APIVersion, deleted[i].Kind,
		)
		d := &unstructured.Unstructured{}
		d.SetGroupVersionKind(gvk)

		if err := r.Client.Delete(ctx, d); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "failed to delete an obsolete child resource", "gvk", gvk.String(), "name", d.GetName(), "namespace", d.GetNamespace())

			return ctrl.Result{Requeue: true}, fmt.Errorf("failed to delete an obsolete child resource: %w", err)
		}
	}

	{
		copied := app.DeepCopy()
		copied.Status.Domains = app.Spec.Domains
		copied.Status.Resources = resources
		copied.Status.Status = modokiv1alpha1.ApplicationDeployed

		if err := r.Client.Patch(ctx, copied, client.MergeFrom(&app)); err != nil {
			log.Error(err, "failed to update status", "status", app.Status.Status)

			return ctrl.Result{Requeue: true}, fmt.Errorf("failed to update status: %w", err)
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
