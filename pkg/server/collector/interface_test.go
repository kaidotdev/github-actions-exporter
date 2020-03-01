package collector_test

import (
	"github-actions-exporter/pkg/server/collector"
	"io"
	"net/http"
	"reflect"
)

type loggerMock struct {
	collector.ILogger
	fakeErrorf func(format string, v ...interface{})
	fakeInfof  func(format string, v ...interface{})
	fakeDebugf func(format string, v ...interface{})
}

func (l loggerMock) Errorf(format string, v ...interface{}) {
	l.fakeErrorf(format, v...)
}

func (l loggerMock) Infof(format string, v ...interface{}) {
	l.fakeInfof(format, v...)
}

func (l loggerMock) Debugf(format string, v ...interface{}) {
	l.fakeDebugf(format, v...)
}

func getRecursiveStructReflectValue(rv reflect.Value) []reflect.Value {
	var values []reflect.Value
	switch rv.Kind() {
	case reflect.Ptr:
		values = append(values, getRecursiveStructReflectValue(rv.Elem())...)
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			values = append(values, getRecursiveStructReflectValue(rv.Index(i))...)
		}
	case reflect.Map:
		for _, k := range rv.MapKeys() {
			values = append(values, getRecursiveStructReflectValue(rv.MapIndex(k))...)
		}
	case reflect.Struct:
		values = append(values, reflect.New(rv.Type()).Elem())
		for i := 0; i < rv.NumField(); i++ {
			values = append(values, getRecursiveStructReflectValue(rv.Field(i))...)
		}
	default:
	}
	return values
}

type httpClientMock struct {
	collector.IHTTPClient
	fakeDo func(*http.Request) (*http.Response, error)
}

func (hc httpClientMock) Do(request *http.Request) (*http.Response, error) {
	return hc.fakeDo(request)
}

type readCloserMock struct {
	io.ReadCloser
	fakeRead func(p []byte) (n int, err error)
}

func (rc readCloserMock) Read(p []byte) (n int, err error) {
	return rc.fakeRead(p)
}

func (rc readCloserMock) Close() error {
	return nil
}
