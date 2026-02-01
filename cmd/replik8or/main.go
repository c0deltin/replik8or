package main

import (
	"os"

	"github.com/c0deltin/replik8or/internal/replicator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/c0deltin/replik8or/internal/config"
	"github.com/c0deltin/replik8or/internal/controller/source"
)

func main() {
	ctx := signals.SetupSignalHandler()

	log.SetLogger(zap.New())
	setupLog := log.Log.WithName("replik8or")

	cfg, err := config.Read()
	if err != nil {
		setupLog.Error(err, "reading configuration")
		os.Exit(1)
	}

	ctrlCfg, err := ctrlconfig.GetConfig()
	if err != nil {
		setupLog.Error(err, "reading kubernetes configuration")
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	mgr, err := manager.New(ctrlCfg, manager.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: cfg.MetricsAddress,
		},
		HealthProbeBindAddress: cfg.HealthProbeAddress,
	})
	if err != nil {
		setupLog.Error(err, "setup controller manager")
		os.Exit(1)
	}

	configMapReconciler := source.NewReconciler[*corev1.ConfigMap](
		mgr.GetClient(),
		cfg,
		replicator.EmptyConfigMap,
		replicator.EmptyConfigMapList,
	)
	if err := configMapReconciler.SetupWithManager("source-configmap", mgr); err != nil {
		setupLog.Error(err, "setup source reconciler", "controller", "ConfigMap")
		os.Exit(1)
	}

	secretReconciler := source.NewReconciler[*corev1.Secret](
		mgr.GetClient(),
		cfg,
		replicator.EmptySecret,
		replicator.EmptySecretList,
	)
	if err := secretReconciler.SetupWithManager("source-secret", mgr); err != nil {
		setupLog.Error(err, "setup source reconciler", "controller", "Secret")
		os.Exit(1)
	}

	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "starting controller manager")
		os.Exit(1)
	}
}
