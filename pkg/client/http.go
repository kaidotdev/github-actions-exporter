package client

import (
	"net/http"
	"net/url"
	"time"

	"golang.org/x/xerrors"
)

type HTTPClient struct {
	RetryStrategy RetryStrategy
	Inner         *http.Client
}

func (c *HTTPClient) Do(request *http.Request) (*http.Response, error) {
	sleep, retry := c.RetryStrategy.Sleep()

	response, err := c.Inner.Do(request)
	if err != nil {
		if innerErr, ok := err.(*url.Error); ok && retry && innerErr.Temporary() {
			time.Sleep(sleep)
			return c.Do(request)
		}
		return nil, xerrors.Errorf("failed to request: %w", err)
	}

	return response, nil
}
