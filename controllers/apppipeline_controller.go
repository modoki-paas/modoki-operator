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

	"github.com/go-logr/logr"
	"golang.org/x/xerrors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	modokiv1alpha1 "github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"github.com/modoki-paas/modoki-operator/pkg/config"
	ghpipeline "github.com/modoki-paas/modoki-operator/pkg/pipeline/github"
)

// AppPipelineReconciler reconciles a AppPipeline object
type AppPipelineReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Config *config.Config
}

// +kubebuilder:rbac:groups=modoki.tsuzu.dev,resources=apppipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=modoki.tsuzu.dev,resources=apppipelines/status,verbs=get;update;patch

func (r *AppPipelineReconciler) Reconcile(req ctrl.Request) (res ctrl.Result, err error) {
	ctx := context.Background()
	logger := r.Log.WithValues("apppipeline", req.NamespacedName)

	// your logic here
	var pl modokiv1alpha1.AppPipeline
	if err := r.Get(ctx, req.NamespacedName, &pl); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defer func() {
		if err != nil {
			pl.Status.Message = err.Error()

			if err := r.Client.Status().Update(ctx, &pl); err != nil {
				logger.Error(err, "failed to update status", "status", pl.Status.Message)

				res.Requeue = true
			}
		}
	}()

	ghp := ghpipeline.NewGitHubPipeline(
		r, &pl, r.Config, r.Scheme, r.Log.WithName("ghpipeline"),
	)

	if err := ghp.Run(ctx); err != nil {
		return ctrl.Result{
			Requeue: true,
		}, xerrors.Errorf("failed to update AppPipeline: %w", err)
	}

	pl.Status.Message = ""

	if err := r.Client.Status().Update(ctx, &pl); err != nil {
		return ctrl.Result{
			Requeue: true,
		}, xerrors.Errorf("failed to update status of AppPipeline: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *AppPipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&modokiv1alpha1.AppPipeline{}).
		Owns(&modokiv1alpha1.Application{}).
		Owns(&modokiv1alpha1.RemoteSync{}).
		Complete(r)
}
