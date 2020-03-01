package server

import (
	"context"
	"net/http"

	"k8s.io/client-go/kubernetes"
)

type IProcessor interface {
	Start() error
	Stop(context.Context) error
}

type IKubernetesClient interface {
	kubernetes.Interface
}

type IHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type ILogger interface {
	Errorf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}
