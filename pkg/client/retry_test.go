package client_test

import (
	"fmt"
	"github-actions-exporter/pkg/client"
	"runtime"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestRetrySleep(t *testing.T) {
	type want struct {
		first  time.Duration
		second bool
	}

	tests := []struct {
		name         string
		receiver     client.RetryStrategy
		wants        []want
		optsFunction func(interface{}) cmp.Option
	}{
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			&client.NoRetry{},
			[]want{
				{
					0,
					false,
				},
			},
			func(got interface{}) cmp.Option {
				return nil
			},
		},
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			&client.ExponentialBackOff{},
			[]want{
				{
					0,
					false,
				},
			},
			func(got interface{}) cmp.Option {
				return nil
			},
		},
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			&client.ExponentialBackOff{
				Base: 1 * time.Second,
			},
			[]want{
				{
					0,
					false,
				},
			},
			func(got interface{}) cmp.Option {
				return nil
			},
		},
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			&client.ExponentialBackOff{
				Base:       0,
				RetryCount: 3,
				Entropy: func(i int64) int64 {
					return i
				},
			},
			[]want{
				{
					0,
					true,
				},
				{
					0,
					true,
				},
				{
					0,
					true,
				},
			},
			func(got interface{}) cmp.Option {
				return nil
			},
		},
		{
			func() string {
				_, _, line, _ := runtime.Caller(1)
				return fmt.Sprintf("L%d", line)
			}(),
			&client.ExponentialBackOff{
				Base:       1 * time.Second,
				RetryCount: 3,
				Entropy: func(i int64) int64 {
					return i
				},
			},
			[]want{
				{
					1 * time.Second,
					true,
				},
				{
					2 * time.Second,
					true,
				},
				{
					4 * time.Second,
					true,
				},
				{
					0,
					false,
				},
			},
			func(got interface{}) cmp.Option {
				return nil
			},
		},
	}
	for _, tt := range tests {
		name := tt.name
		receiver := tt.receiver
		wants := tt.wants
		optsFunction := tt.optsFunction
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, want := range wants {
				gotFirst, gotSecond := receiver.Sleep()
				if diff := cmp.Diff(want.first, gotFirst, optsFunction(gotFirst)); diff != "" {
					t.Errorf("(-want +got):\n%s", diff)
				}
				if diff := cmp.Diff(want.second, gotSecond, optsFunction(gotSecond)); diff != "" {
					t.Errorf("(-want +got):\n%s", diff)
				}
			}
		})
	}
}
