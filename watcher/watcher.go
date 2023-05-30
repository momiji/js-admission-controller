package watcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/momiji/js-admissions-controller/logs"
	"github.com/momiji/js-admissions-controller/store"
	"github.com/momiji/js-admissions-controller/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

const (
	CREATED = iota
	UPDATED
	DELETED
)

type Watcher struct {
	mux       sync.Mutex
	locks     map[string]*sync.Mutex
	ctx       context.Context
	factory   dynamicinformer.DynamicSharedInformerFactory
	items     *store.Cache
	handler   cache.ResourceEventHandlerFuncs
	informers map[schema.GroupVersionResource]cache.SharedIndexInformer
}

func NewWatcher(ctx context.Context, client dynamic.Interface, action func(action int, obj *unstructured.Unstructured, old *unstructured.Unstructured)) *Watcher {
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(client, time.Minute, corev1.NamespaceAll, nil)
	items := store.NewCache()
	watcher := &Watcher{
		mux:       sync.Mutex{},
		locks:     make(map[string]*sync.Mutex),
		ctx:       ctx,
		factory:   factory,
		items:     items,
		informers: make(map[schema.GroupVersionResource]cache.SharedIndexInformer),
	}
	watcher.handler = cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			item, ok := obj.(*unstructured.Unstructured)
			if !ok {
				logs.Errorf("Cache: add item failed, item is not *unstructured.Unstructured")
				return
			}
			gvk := utils.GVKToString(item.GroupVersionKind())
			watcher.LockResource(gvk)
			defer watcher.UnlockResource(gvk)
			logs.Tracef("Cache: add item %s ns=%s name=%s", gvk, item.GetNamespace(), item.GetName())
			items.Add(gvk, item.GetNamespace(), item.GetName(), item)
			if action != nil {
				action(CREATED, item, nil)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newItem, ok := newObj.(*unstructured.Unstructured)
			if !ok {
				logs.Errorf("Cache: update new item failed, item is not *unstructured.Unstructured")
				return
			}
			oldItem, ok := oldObj.(*unstructured.Unstructured)
			if !ok {
				logs.Errorf("Cache: update old item failed, item is not *unstructured.Unstructured")
				return
			}
			// skip if object is unchanched
			oldVers, oldFound, _ := unstructured.NestedString(oldItem.Object, "metadata", "resourceVersion")
			newVers, newFound, _ := unstructured.NestedString(newItem.Object, "metadata", "resourceVersion")
			if oldFound && newFound && oldVers == newVers {
				return
			}
			gvk := utils.GVKToString(newItem.GroupVersionKind())
			watcher.LockResource(gvk)
			defer watcher.UnlockResource(gvk)
			logs.Tracef("Cache: update item %s ns=%s name=%s", gvk, newItem.GetNamespace(), newItem.GetName())
			items.Add(gvk, newItem.GetNamespace(), newItem.GetName(), newItem)
			if action != nil {
				action(UPDATED, newItem, oldItem)
			}
		},
		DeleteFunc: func(obj interface{}) {
			item, ok := obj.(*unstructured.Unstructured)
			if !ok {
				logs.Errorf("Cache: delete item failed, item is not *unstructured.Unstructured")
				return
			}
			gvk := utils.GVKToString(item.GroupVersionKind())
			watcher.LockResource(gvk)
			defer watcher.UnlockResource(gvk)
			logs.Tracef("Cache: delete item item %s ns=%s name=%s", gvk, item.GetNamespace(), item.GetName())
			items.Remove(gvk, item.GetNamespace(), item.GetName())
			if action != nil {
				action(DELETED, item, nil)
			}
		},
	}
	return watcher

}

func (w *Watcher) Add(gvr schema.GroupVersionResource) error {
	w.mux.Lock()
	defer w.mux.Unlock()

	// create new informer
	informer, ok := w.informers[gvr]
	if ok {
		return nil
	}

	informer = w.factory.ForResource(gvr).Informer()
	_, err := informer.AddEventHandler(w.handler)
	if err != nil {
		return fmt.Errorf("failed loading resources %s: %v", utils.GVRToString(gvr), err)
	}
	w.informers[gvr] = informer

	// start loading resources
	logs.Infof("Start loading resources %s", utils.GVRToString(gvr))
	go informer.Run(w.ctx.Done())

	// wait for resources to be synced
	isSynced := cache.WaitForCacheSync(w.ctx.Done(), informer.HasSynced)
	if !isSynced {
		return fmt.Errorf("failed loading resources %s", utils.GVRToString(gvr))
	}
	return nil
}

func (w *Watcher) LockResource(resource string) {
	w.mux.Lock()
	//defer w.mux.Unlock()
	//if _, ok := w.locks[resource]; !ok {
	//	w.locks[resource] = &sync.Mutex{}
	//}
	//w.locks[resource].Lock()
}
func (w *Watcher) UnlockResource(resource string) {
	//w.mux.Lock()
	defer w.mux.Unlock()
	//w.locks[resource].Unlock()
}

// GetResources return all resources with the same namespace or for all namespaces if namespace=""
//
// For a namespace resource (like pods), only resources for this namespace are returned if a namespace is provided, else all resources for all namespaces are returned.
// For a cluster resource (like clusterroles), all resources are returned, as the namespace should be empty.
func (w *Watcher) GetResources(resource string, namespace string) []*unstructured.Unstructured {
	return w.items.Find(resource, namespace)
}
