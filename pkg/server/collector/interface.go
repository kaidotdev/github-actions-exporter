package collector

import (
	"net/http"
)

type ILogger interface {
	Errorf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}

type IHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}
