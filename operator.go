package main

import (
	"context"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// LonghornBandwidthReconciler reconciles Longhorn pods
type LonghornBandwidthReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	BandwidthConfig BandwidthConfig
}

func main() {
	log.SetLogger(zap.New())
	logger := log.Log.WithName("longhorn-bandwidth-operator")

	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	config, err := loadConfig(configPath)
	if err != nil {
		logger.Error(err, "Failed to load configuration")
		os.Exit(1)
	}

	// Set up manager
	mgr, err := manager.New(getKubeConfig(), manager.Options{
		HealthProbeBindAddress: ":8081",
	})
	if err != nil {
		logger.Error(err, "Failed to set up manager")
		os.Exit(1)
	}

	// Set up reconciler
	reconciler := &LonghornBandwidthReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		BandwidthConfig: config,
	}

	// Set up controller
	err = ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(reconciler)

	if err != nil {
		logger.Error(err, "Failed to create controller")
		os.Exit(1)
	}

	// Add health check handler
	err = mgr.AddHealthzCheck("healthz", healthz.Ping)
	if err != nil {
		logger.Error(err, "Unable to set up health check")
		os.Exit(1)
	}

	// Add readiness check handler
	err = mgr.AddReadyzCheck("readyz", healthz.Ping)
	if err != nil {
		logger.Error(err, "Unable to set up ready check")
		os.Exit(1)
	}

	logger.Info("Starting manager")
	if err := mgr.Start(context.TODO()); err != nil {
		logger.Error(err, "Failed to start manager")
		os.Exit(1)
	}
}

func (r *LonghornBandwidthReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Pod instance
	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		logger.Error(err, "Failed to get Pod")
		return reconcile.Result{}, err
	}

	// Check if the pod matches our criteria
	if pod.Labels["longhorn.io/component"] == "instance-manager" {
		nodeName, ok := pod.Labels["longhorn.io/node"]
		if !ok {
			logger.Info("Pod doesn't have longhorn.io/node label", "pod", pod.Name)
			return reconcile.Result{}, nil
		}

		// Get node configuration
		nodeConfig, ok := r.BandwidthConfig.Nodes[nodeName]
		if !ok {
			logger.Info("No configuration found for node", "node", nodeName)
			return reconcile.Result{}, nil
		}

		// Update pod annotations
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations["kubernetes.io/ingress-bandwidth"] = nodeConfig.IngressLimit
		pod.Annotations["kubernetes.io/egress-bandwidth"] = nodeConfig.EgressLimit

		// Update the pod
		err = r.Update(ctx, pod)
		if err != nil {
			logger.Error(err, "Failed to update Pod", "pod", pod.Name)
			return reconcile.Result{}, err
		}

		logger.Info("Updated Pod annotations", "pod", pod.Name, "node", nodeName)
	}

	return reconcile.Result{}, nil
}

func getKubeConfig() *rest.Config {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}
	return config
}
