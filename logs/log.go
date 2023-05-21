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
		klog.Infof(fmt, args...)
	}
}

func Debugf(fmt string, args ...interface{}) {
	if DebugMode || TraceMode {
		klog.Infof(fmt, args...)
	}
}

func Debug(args ...interface{}) {
	if DebugMode || TraceMode {
		klog.Info(args...)
	}
}

func Infof(fmt string, args ...interface{}) {
	klog.Infof(fmt, args...)
}

func Warnf(fmt string, args ...interface{}) {
	klog.Warningf(fmt, args...)
}

func Errorf(fmt string, args ...interface{}) {
	klog.Errorf(fmt, args...)
}

func Fatalf(fmt string, args ...interface{}) {
	klog.Fatalf(fmt, args...)
}
