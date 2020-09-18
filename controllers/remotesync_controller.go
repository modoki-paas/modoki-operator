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
	"github.com/labstack/gommon/log"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	modokiv1alpha1 "github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"github.com/modoki-paas/modoki-operator/pkg/config"
	"github.com/modoki-paas/modoki-operator/pkg/kpackbuilder"
	kpacktypes "github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// RemoteSyncReconciler reconciles a RemoteSync object
type RemoteSyncReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Config *config.Config
}

// +kubebuilder:rbac:groups=modoki.tsuzu.dev,resources=remotesyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=modoki.tsuzu.dev,resources=remotesyncs/status,verbs=get;update;patch

func (r *RemoteSyncReconciler) Reconcile(req ctrl.Request) (res ctrl.Result, err error) {
	ctx := context.Background()
	_ = r.Log.WithValues("remotesync", req.NamespacedName)

	// your logic here
	var rs modokiv1alpha1.RemoteSync
	if err := r.Get(ctx, req.NamespacedName, &rs); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	defer func() {
		if err != nil {
			rs.Status.Message = err.Error()

			if err := r.Client.Status().Update(ctx, &rs); err != nil {
				log.Error(err, "failed to update status", "status", rs.Status.Message)

				res.Requeue = true
			}
		}
	}()

	builder := kpackbuilder.NewKpackBuilder(r.Client, &rs, r.Config, r.Scheme)

	builder.Run(ctx)

	return ctrl.Result{}, nil
}

func (r *RemoteSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&modokiv1alpha1.RemoteSync{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&kpacktypes.Image{}).
		Complete(r)
}
