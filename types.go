package main

import (
	admission "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecFactory  = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecFactory.UniversalDeserializer()
)

func init() {
	//_ = corev1.AddToScheme(runtimeScheme)
	_ = admission.AddToScheme(runtimeScheme)
	//runtimeScheme.AddKnownTypes(
	//	schema.GroupVersion{Group: GroupCrd, Version: VersionCrd},
	//	&JsAdmission{},
	//	&ClusterJsAdmission{},
	//)
}

type JsAdmission struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              JsAdmissionSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type ClusterJsAdmission struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              JsAdmissionSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

type JsAdmissionSpec struct {
	Action string   `json:"type,omitempty" protobuf:"bytes,1,opt,name=action"`
	Kinds  []string `json:"kinds,omitempty" protobuf:"bytes,2,opt,name=kinds"`
	Js     string   `json:"js,omitempty" protobuf:"bytes,3,opt,name=js"`
}

func (in *JsAdmission) DeepCopyInto(out *JsAdmission) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

func (in *JsAdmission) DeepCopy() *JsAdmission {
	if in == nil {
		return nil
	}
	out := new(JsAdmission)
	in.DeepCopyInto(out)
	return out
}

func (in *JsAdmission) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *ClusterJsAdmission) DeepCopyInto(out *ClusterJsAdmission) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

func (in *ClusterJsAdmission) DeepCopy() *ClusterJsAdmission {
	if in == nil {
		return nil
	}
	out := new(ClusterJsAdmission)
	in.DeepCopyInto(out)
	return out
}

func (in *ClusterJsAdmission) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *JsAdmissionSpec) DeepCopyInto(out *JsAdmissionSpec) {
	*out = *in
	out.Action = in.Action
	out.Kinds = in.Kinds
	out.Js = in.Js
	return
}
