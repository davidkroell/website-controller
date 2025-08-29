package main

import (
	"os"
	wsapiv1 "website-operator/api/v1"
	"website-operator/internal"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	clientset, config, err := internal.GetLocalOrInClusterKubernetes()
	if err != nil {
		panic(err.Error())
	}

	ctrl.SetLogger(zap.New())

	scheme := runtime.NewScheme()
	log := ctrl.Log.WithName("setup website controller")
	utilruntime.Must(wsapiv1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Error(err, "unable to start manager")
		os.Exit(1)
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&wsapiv1.WebSite{}).
		Complete(internal.NewWebsiteReconciler(mgr, clientset))
	if err != nil {
		log.Error(err, "unable to create controller")
		os.Exit(1)
	}

	log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Error(err, "error running manager")
		os.Exit(1)
	}
}
