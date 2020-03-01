package collector_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"github-actions-exporter/pkg/server/collector"
	"io/ioutil"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/prometheus/client_golang/prometheus"
)

func TestRunsCollectorDescribe(t *testing.T) {
	tests := []struct {
		name         string
		receiver     *collector.RunsCollector
		in           chan *prometheus.Desc
		want         *prometheus.Desc
		optsFunction func(interface{}) cmp.Option
	}{
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			collector.NewRunsCollector(
				"",
				"",
				loggerMock{},
				&httpClientMock{},
			),
			make(chan *prometheus.Desc, 1),
			prometheus.NewDesc(
				"github_actions_runs",
				"List how many workflow runs each repository actions",
				[]string{"repository", "status"},
				nil,
			),
			func(got interface{}) cmp.Option {
				switch v := got.(type) {
				case *prometheus.Desc:
					return cmp.AllowUnexported(*v)
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		name := tt.name
		receiver := tt.receiver
		in := tt.in
		want := tt.want
		optsFunction := tt.optsFunction
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			receiver.Describe(in)
			got := <-in
			if diff := cmp.Diff(want, got, optsFunction(got)); diff != "" {
				t.Errorf("(-want +got):\n%s", diff)
			}
		})
	}
}

func TestRunsCollectorCollect(t *testing.T) {
	tests := []struct {
		name         string
		receiver     *collector.RunsCollector
		in           chan prometheus.Metric
		want         []prometheus.Metric
		optsFunction func(interface{}) cmp.Option
	}{
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			collector.NewRunsCollector(
				"fake",
				"",
				loggerMock{},
				&httpClientMock{
					fakeDo: func(request *http.Request) (*http.Response, error) {
						bytes, _ := json.Marshal(collector.WorkflowRunsResponse{
							TotalCount: 1,
						})
						return &http.Response{
							Body: ioutil.NopCloser(strings.NewReader(string(bytes))),
						}, nil
					},
				},
			),
			make(chan prometheus.Metric, 3),
			func() []prometheus.Metric {
				gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
					Namespace: "github_actions",
					Name:      "runs",
					Help:      "List how many workflow runs each repository actions",
				}, []string{"repository", "status"})
				var ret []prometheus.Metric
				statuses := []string{
					"queued",
					"in_progress",
					"completed",
				}
				for _, status := range statuses {
					labels := []string{
						"fake",
						status,
					}
					gaugeVec.WithLabelValues(labels...).Set(1)
					gauge, err := gaugeVec.GetMetricWithLabelValues(labels...)
					if err != nil {
						t.Fatal(err)
					}
					ret = append(ret, gauge)
				}
				return ret
			}(),
			func(got interface{}) cmp.Option {
				switch got.(type) {
				case prometheus.Metric:
					deref := func(v interface{}) interface{} {
						return reflect.ValueOf(v).Elem().Interface()
					}
					v := deref(got)
					switch reflect.TypeOf(v).Name() {
					case "gauge":
						var opts cmp.Options
						for _, rv := range getRecursiveStructReflectValue(reflect.ValueOf(v)) {
							switch rv.Type().Name() {
							case "selfCollector":
								opts = append(opts, cmpopts.IgnoreUnexported(rv.Interface()))
							default:
								opts = append(opts, cmp.AllowUnexported(rv.Interface()))
							}
						}
						return opts
					default:
						return nil
					}
				}
				return nil
			},
		},
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			collector.NewRunsCollector(
				"fa\nke",
				"",
				loggerMock{
					fakeErrorf: func(format string, v ...interface{}) {
						want := "Failed to fetch runs count: failed to create request object: parse \"https://api.github.com/repos/fa\\nke/actions/runs?status=queued\": net/url: invalid control character in URL\n"
						got := fmt.Sprintf(format, v...)
						if diff := cmp.Diff(want, got); diff != "" {
							t.Errorf("(-want +got):\n%s", diff)
						}
					},
				},
				&httpClientMock{},
			),
			func() chan prometheus.Metric {
				ch := make(chan prometheus.Metric, 1)
				close(ch)
				return ch
			}(),
			nil,
			func(got interface{}) cmp.Option {
				return nil
			},
		},
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			collector.NewRunsCollector(
				"fake",
				"",
				loggerMock{
					fakeErrorf: func(format string, v ...interface{}) {
						want := "Failed to fetch runs count: failed to request: fake\n"
						got := fmt.Sprintf(format, v...)
						if diff := cmp.Diff(want, got); diff != "" {
							t.Errorf("(-want +got):\n%s", diff)
						}
					},
				},
				&httpClientMock{
					fakeDo: func(request *http.Request) (*http.Response, error) {
						return nil, errors.New("fake")
					},
				},
			),
			func() chan prometheus.Metric {
				ch := make(chan prometheus.Metric, 1)
				close(ch)
				return ch
			}(),
			nil,
			func(got interface{}) cmp.Option {
				return nil
			},
		},
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			collector.NewRunsCollector(
				"fake",
				"",
				loggerMock{
					fakeErrorf: func(format string, v ...interface{}) {
						want := "Failed to fetch runs count: failed to read response: fake\n"
						got := fmt.Sprintf(format, v...)
						if diff := cmp.Diff(want, got); diff != "" {
							t.Errorf("(-want +got):\n%s", diff)
						}
					},
				},
				&httpClientMock{
					fakeDo: func(request *http.Request) (*http.Response, error) {
						return &http.Response{
							Body: readCloserMock{
								fakeRead: func(p []byte) (n int, err error) {
									return 0, errors.New("fake")
								},
							},
						}, nil
					},
				},
			),
			func() chan prometheus.Metric {
				ch := make(chan prometheus.Metric, 1)
				close(ch)
				return ch
			}(),
			nil,
			func(got interface{}) cmp.Option {
				return nil
			},
		},
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			collector.NewRunsCollector(
				"fake",
				"",
				loggerMock{
					fakeErrorf: func(format string, v ...interface{}) {
						want := "Failed to fetch runs count: failed to parse response: invalid character 'k' in literal false (expecting 'l')\n"
						got := fmt.Sprintf(format, v...)
						if diff := cmp.Diff(want, got); diff != "" {
							t.Errorf("(-want +got):\n%s", diff)
						}
					},
				},
				&httpClientMock{
					fakeDo: func(request *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       ioutil.NopCloser(strings.NewReader("fake")),
						}, nil
					},
				},
			),
			func() chan prometheus.Metric {
				ch := make(chan prometheus.Metric, 1)
				close(ch)
				return ch
			}(),
			nil,
			func(got interface{}) cmp.Option {
				return nil
			},
		},
	}
	for _, tt := range tests {
		name := tt.name
		receiver := tt.receiver
		in := tt.in
		want := tt.want
		optsFunction := tt.optsFunction
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			receiver.Collect(in)
			bruteForce := func(got prometheus.Metric) string {
				diff := ""
				for _, w := range want {
					result := cmp.Diff(w, got, optsFunction(got))
					if result == "" {
						return result
					} else {
						diff = result
					}
				}
				return diff
			}
			for i := 0; i < cap(in); i++ {
				got := <-in
				if diff := bruteForce(got); diff != "" {
					t.Errorf("(-want +got):\n%s", diff)
				}
			}
		})
	}
}
