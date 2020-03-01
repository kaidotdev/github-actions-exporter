package server

import (
	"context"
	"github-actions-exporter/pkg/client"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"go.opencensus.io/plugin/ochttp"
)

const (
	gracePeriod = 10
)

type Instance struct {
	processors []IProcessor
	httpClient IHTTPClient
	logger     ILogger
}

func NewInstance() *Instance {
	return &Instance{
		httpClient: &client.HTTPClient{
			RetryStrategy: &client.ExponentialBackOff{
				Base:       10 * time.Millisecond,
				RetryCount: 3,
			},
			Inner: &http.Client{
				Timeout: 3 * time.Second,
				Transport: &ochttp.Transport{
					Base: &http.Transport{
						Proxy: http.ProxyFromEnvironment,
						DialContext: (&net.Dialer{
							Timeout:   1 * time.Second,
							KeepAlive: 10 * time.Second,
						}).DialContext,
						DisableKeepAlives:     false,
						MaxIdleConns:          0,
						MaxIdleConnsPerHost:   100,
						IdleConnTimeout:       10 * time.Second,
						MaxConnsPerHost:       0,
						TLSHandshakeTimeout:   1 * time.Second,
						ExpectContinueTimeout: 1 * time.Second,
					},
				},
			},
		},
		logger: client.NewDefaultLogger(),
	}
}

func (i *Instance) HTTPClient() IHTTPClient {
	return i.httpClient
}

func (i *Instance) SetHTTPClient(httpClient IHTTPClient) {
	i.httpClient = httpClient
}

func (i *Instance) Logger() ILogger {
	return i.logger
}

func (i *Instance) SetLogger(logger ILogger) {
	i.logger = logger
}

func (i *Instance) AddProcessor(processor IProcessor) {
	i.processors = append(i.processors, processor)
}

func (i *Instance) Start() {
	for _, processor := range i.processors {
		go func(processor IProcessor) {
			defer func() {
				if err := recover(); err != nil {
					i.logger.Errorf("panic: %+v\n", err)
					i.logger.Debugf("%s\n", debug.Stack())
				}
			}()
			if err := processor.Start(); err != nil && err != http.ErrServerClosed {
				i.logger.Errorf("Failed to listen: %s\n", err.Error())
			}
		}(processor)
	}
}

func (i *Instance) Shutdown(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(gracePeriod)*time.Second)
	defer cancel()
	for _, p := range i.processors {
		if err := p.Stop(ctx); err != nil {
			i.logger.Errorf("Failed to shutdown: %+v\n", err)
		}
	}
	select {
	case <-ctx.Done():
		i.logger.Infof("Instance shutdown timed out in %d seconds\n", gracePeriod)
	default:
	}
	i.logger.Infof("Instance has been shutdown\n")
}
