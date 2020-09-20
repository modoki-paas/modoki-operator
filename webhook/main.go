package main

import (
	"net/http"
	"os"

	"github.com/go-logr/zapr"
	modokiv1alpha1 "github.com/modoki-paas/modoki-operator/api/v1alpha1"
	"github.com/modoki-paas/modoki-operator/pkg/webhook"
	remotesync "github.com/modoki-paas/modoki-operator/pkg/webhook/remoteSync"
	kpacktypes "github.com/pivotal/kpack/pkg/apis/build/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(kpacktypes.AddToScheme(scheme))

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(modokiv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func getListenAddr() string {
	port := os.Getenv("PORT")

	if len(port) == 0 {
		port = "80"
	}

	return ":" + port
}

func newK8sClient() client.Client {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})

	if err != nil {
		setupLog.Error(err, "failed to initialize manager")
	}

	return mgr.GetClient()
}

func main() {
	zrlogger := zap.NewRaw(zap.UseDevMode(true))
	logger := zapr.NewLogger(zrlogger)

	ctrl.SetLogger(logger)

	client := newK8sClient()
	remotesync.Register(
		client,
		ctrl.Log,
	)

	token := os.Getenv("GITHUB_WEBHOOK_TOKEN")

	http.Handle("/webhook", webhook.NewHandler(token))

	http.ListenAndServe(getListenAddr(), nil)
}
