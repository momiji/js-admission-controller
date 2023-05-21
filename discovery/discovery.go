package discovery

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"strings"
)

type Discovery struct {
	discoveryClient *discovery.DiscoveryClient
	discoveryMapper *restmapper.DeferredDiscoveryRESTMapper
	dynamicClient   dynamic.Interface
}

func NewDiscovery(config *rest.Config) (*Discovery, error) {

	disC, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	dynC, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	cacheC := memory.NewMemCacheClient(disC)
	cacheC.Invalidate()

	dm := restmapper.NewDeferredDiscoveryRESTMapper(cacheC)

	return &Discovery{disC, dm, dynC}, nil
}

func (d *Discovery) GetGVRFromResource(resource string) (schema.GroupVersionResource, error) {
	var gvr schema.GroupVersionResource
	if strings.Count(resource, "/") >= 2 {
		s := strings.SplitN(resource, "/", 3)
		gvr = schema.GroupVersionResource{Group: s[0], Version: s[1], Resource: s[2]}
	} else if strings.Count(resource, "/") == 1 {
		s := strings.SplitN(resource, "/", 2)
		gvr = schema.GroupVersionResource{Group: "", Version: s[0], Resource: s[1]}
	} else {
		gvr = schema.GroupVersionResource{Group: "", Version: "", Resource: resource}
	}

	gvrs, err := d.discoveryMapper.ResourcesFor(gvr)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	if len(gvrs) == 0 {
		return schema.GroupVersionResource{}, fmt.Errorf("unable to find resource %s", resource)
	}

	return gvrs[0], nil
}

func (d *Discovery) GetGVKFromResource(resource string) (schema.GroupVersionKind, error) {
	var gvr schema.GroupVersionResource
	if strings.Count(resource, "/") >= 2 {
		s := strings.SplitN(resource, "/", 3)
		gvr = schema.GroupVersionResource{Group: s[0], Version: s[1], Resource: s[2]}
	} else if strings.Count(resource, "/") == 1 {
		s := strings.SplitN(resource, "/", 2)
		gvr = schema.GroupVersionResource{Group: "", Version: s[0], Resource: s[1]}
	} else {
		gvr = schema.GroupVersionResource{Group: "", Version: "", Resource: resource}
	}

	gvrs, err := d.discoveryMapper.KindsFor(gvr)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}

	if len(gvrs) == 0 {
		return schema.GroupVersionKind{}, fmt.Errorf("unable to find resource %s", resource)
	}

	return gvrs[0], nil
}
