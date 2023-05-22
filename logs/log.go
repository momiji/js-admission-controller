package logs

import (
	"k8s.io/klog/v2"
)

var (
	TraceMode = false
	DebugMode = false
)

func Tracef(fmt string, args ...interface{}) {
	if TraceMode {
		klog.InfofDepth(1, fmt, args...)
	}
}

func Debugf(fmt string, args ...interface{}) {
	if DebugMode || TraceMode {
		klog.InfofDepth(1, fmt, args...)
	}
}

func Infof(fmt string, args ...interface{}) {
	klog.InfofDepth(1, fmt, args...)
}

func Warnf(fmt string, args ...interface{}) {
	klog.WarningfDepth(1, fmt, args...)
}

func Errorf(fmt string, args ...interface{}) {
	klog.ErrorfDepth(1, fmt, args...)
}

func Fatalf(fmt string, args ...interface{}) {
	klog.FatalfDepth(1, fmt, args...)
}
