package main

import (
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"time"
)

const (
	resyncPeriod = time.Duration(1) * time.Minute
)

type Watcher struct {
	informer informers.GenericInformer
}

func newWatcher() (*Watcher, error) {
	kubeconfig, err := newKubeConfig()
	if err != nil {
		return nil, err
	}

	client, err := newClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	informer := dynamicinformer.NewDynamicSharedInformerFactory(cient, resyncPeriod)

	return &Watcher{informer}, nil
}

func (w *Watcher) Run(stopCh chan struct{}) {
	w.informer.Start(stopCh)
}

func newKubeConfig() (*rest.Config, error) {
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if err != nil {
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}
	return config, nil
}
