package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GVKToString(gvk schema.GroupVersionKind) string {
	api, kind := gvk.ToAPIVersionAndKind()
	return api + "/" + kind
}

func GVK1ToString(gvk metav1.GroupVersionKind) string {
	if gvk.Group == "" {
		return gvk.Version + "/" + gvk.Kind
	}
	return gvk.Group + "/" + gvk.Version + "/" + gvk.Kind
}

func GVRToString(gvr schema.GroupVersionResource) string {
	api := gvr.GroupVersion().String()
	return api + "/" + gvr.Resource
}

func GVR1ToString(gvr metav1.GroupVersionResource) string {
	if gvr.Group == "" {
		return gvr.Version + "/" + gvr.Resource
	}
	return gvr.Group + "/" + gvr.Version + "/" + gvr.Resource
}
