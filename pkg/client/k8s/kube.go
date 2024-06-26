package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

// Client is a help tool for accessing Kubernetes information from the
// Kubernetes API.
type Client struct {
	kube   *kubernetes.Clientset                   // typed api client
	dyn    dynamic.Interface                       // untyped api client
	mapper *restmapper.DeferredDiscoveryRESTMapper // gvr map
}

// New returns a new kubernetes client from local configuration or returns an
// error.
func New(context string) (*Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find HOME: %w", err)
	}

	config, restErr := rest.InClusterConfig()
	if restErr != nil {
		var localErr error
		kubeconfig := filepath.Join(home, ".kube", "config")

		lr := clientcmd.NewDefaultClientConfigLoadingRules()
		lr.ExplicitPath = kubeconfig

		co := new(clientcmd.ConfigOverrides)
		co.CurrentContext = context

		cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(lr, co)
		config, localErr = cfg.ClientConfig()

		if localErr != nil {
			return nil, fmt.Errorf("error loading k8s configs, REST (%w) and local (%w)", restErr, localErr)
		}
	}

	kube, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error loading typed k8s client: %w", err)
	}

	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error loading untyped k8s client: %w", err)
	}

	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error loading k8s discoverer: %w", err)
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(
		memory.NewMemCacheClient(dc),
	)

	return &Client{kube, dyn, mapper}, nil
}
