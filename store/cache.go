package store

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sync"
)

type Cache struct {
	mux sync.RWMutex
	GVK map[string]*CacheGVK
}

type CacheGVK struct {
	Key        string
	Namespaces map[string]*CacheNamespace
}

type CacheNamespace struct {
	Namespace string
	Names     map[string]*CacheItem
}

type CacheItem struct {
	Obj *unstructured.Unstructured
}

func NewCache() *Cache {
	return &Cache{
		mux: sync.RWMutex{},
		GVK: make(map[string]*CacheGVK),
	}
}

func (c *Cache) Add(key string, namespace string, name string, obj *unstructured.Unstructured) {
	c.mux.Lock()
	defer c.mux.Unlock()

	gvk, ok := c.GVK[key]
	if !ok {
		gvk = c.newGVK(key)
		c.GVK[key] = gvk
	}

	ns, ok := gvk.Namespaces[namespace]
	if !ok {
		ns = c.newNamespace(namespace)
		gvk.Namespaces[namespace] = ns
	}

	ns.Names[name] = c.newItem(obj)
}

func (c *Cache) Remove(key string, namespace string, name string) {
	c.mux.Lock()
	defer c.mux.Unlock()

	gvk, ok := c.GVK[key]
	if !ok {
		return
	}

	ns, ok := gvk.Namespaces[namespace]
	if !ok {
		return
	}

	delete(ns.Names, name)
}

func (c *Cache) Find(key string, namespace string) []*unstructured.Unstructured {
	c.mux.RLock()
	defer c.mux.RUnlock()

	gvk, ok := c.GVK[key]
	if !ok {
		return nil
	}

	res := make([]*unstructured.Unstructured, 0)
	for _, ns := range gvk.Namespaces {
		if namespace == "" || namespace == ns.Namespace {
			for _, item := range ns.Names {
				res = append(res, item.Obj)
			}
		}
	}

	return res
}

func (c *Cache) newGVK(key string) *CacheGVK {
	return &CacheGVK{
		Key:        key,
		Namespaces: make(map[string]*CacheNamespace),
	}
}

func (c *Cache) newNamespace(namespace string) *CacheNamespace {
	return &CacheNamespace{
		Namespace: namespace,
		Names:     make(map[string]*CacheItem),
	}
}

func (c *Cache) newItem(obj *unstructured.Unstructured) *CacheItem {
	return &CacheItem{obj}
}
