package client_test

import (
	"errors"
	"fmt"
	"github-actions-exporter/pkg/client"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type recordingTransport struct {
	request        *http.Request
	returnError    error
	returnResponse *http.Response
	callCount      int
}

func (t *recordingTransport) RoundTrip(request *http.Request) (response *http.Response, err error) {
	t.request = request
	t.callCount++
	if t.returnError == nil {
		return t.returnResponse, nil
	}
	return nil, t.returnError
}

type temporaryError struct {
	s string
}

func (te *temporaryError) Error() string {
	return te.s
}

func (te *temporaryError) Temporary() bool {
	return true
}

func TestHTTPClientDo(t *testing.T) {
	fakeReader := strings.NewReader("fake")

	type in struct {
		first *http.Request
	}

	type want struct {
		first *http.Response
	}

	type testcase struct {
		name            string
		receiver        *client.HTTPClient
		in              in
		want            want
		wantCallCount   int
		wantErrorString string
		optsFunction    func(interface{}) cmp.Option
	}

	tests := []testcase{
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			&client.HTTPClient{
				RetryStrategy: &client.ExponentialBackOff{
					Base:       1 * time.Millisecond,
					RetryCount: 3,
				},
				Inner: &http.Client{
					Transport: &recordingTransport{
						returnError: nil,
						returnResponse: &http.Response{
							StatusCode: http.StatusOK,
							Body:       ioutil.NopCloser(fakeReader),
						},
					},
				},
			},
			in{
				func() *http.Request {
					request, err := http.NewRequest("GET", "/", nil)
					if err != nil {
						t.Fatal()
					}
					return request
				}(),
			},
			want{
				&http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(fakeReader),
				},
			},
			1,
			"",
			func(got interface{}) cmp.Option {
				switch got.(type) {
				case *http.Response:
					return cmp.AllowUnexported(*fakeReader)
				default:
					return nil
				}
			},
		},
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			&client.HTTPClient{
				RetryStrategy: &client.ExponentialBackOff{
					Base:       1 * time.Millisecond,
					RetryCount: 3,
				},
				Inner: &http.Client{
					Transport: &recordingTransport{
						returnError:    errors.New("fake"),
						returnResponse: nil,
					},
				},
			},
			in{
				func() *http.Request {
					request, err := http.NewRequest("GET", "/", nil)
					if err != nil {
						t.Fatal()
					}
					return request
				}(),
			},
			want{
				nil,
			},
			1,
			"failed to request: Get \"/\": fake",
			func(got interface{}) cmp.Option {
				return nil
			},
		},
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			&client.HTTPClient{
				RetryStrategy: &client.ExponentialBackOff{
					Base:       1 * time.Millisecond,
					RetryCount: 3,
				},
				Inner: &http.Client{
					Transport: &recordingTransport{
						returnError: &temporaryError{
							s: "fake",
						},
						returnResponse: nil,
					},
				},
			},
			in{
				func() *http.Request {
					request, err := http.NewRequest("GET", "/", nil)
					if err != nil {
						t.Fatal()
					}
					return request
				}(),
			},
			want{
				nil,
			},
			4,
			"failed to request: Get \"/\": fake",
			func(got interface{}) cmp.Option {
				return nil
			},
		},
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			&client.HTTPClient{
				RetryStrategy: &client.NoRetry{},
				Inner: &http.Client{
					Transport: &recordingTransport{
						returnError: &temporaryError{
							s: "fake",
						},
						returnResponse: nil,
					},
				},
			},
			in{
				func() *http.Request {
					request, err := http.NewRequest("GET", "/", nil)
					if err != nil {
						t.Fatal()
					}
					return request
				}(),
			},
			want{
				nil,
			},
			1,
			"failed to request: Get \"/\": fake",
			func(got interface{}) cmp.Option {
				switch got.(type) {
				case *http.Response:
					return cmp.AllowUnexported(*fakeReader)
				default:
					return nil
				}
			},
		},
	}

	for _, tt := range tests {
		name := tt.name
		receiver := tt.receiver
		in := tt.in
		want := tt.want
		wantCallCount := tt.wantCallCount
		wantErrorString := tt.wantErrorString
		optsFunction := tt.optsFunction
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := receiver.Do(in.first)
			rt := receiver.Inner.Transport.(*recordingTransport)
			if diff := cmp.Diff(wantCallCount, rt.callCount); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(want.first, got, optsFunction(got)); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
			if err != nil {
				gotErrorString := err.Error()
				if diff := cmp.Diff(wantErrorString, gotErrorString); diff != "" {
					t.Errorf("(-want +got):\n%s", diff)
				}
				return
			}
		})
	}
}
