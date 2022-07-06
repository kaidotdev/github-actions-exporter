package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/xerrors"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "github_actions"
)

var (
	statuses = []string{
		"queued",
		"in_progress",
		"completed",
	}
)

type WorkflowRunsResponse struct {
	TotalCount   *int              `json:"total_count,omitempty"`
	WorkflowRuns []json.RawMessage `json:"workflow_runs,omitempty"`
}

type RunsCollector struct {
	repository string
	token      string
	logger     ILogger
	httpClient IHTTPClient
	runs       *prometheus.GaugeVec
}

func NewRunsCollector(
	repository string,
	token string,
	logger ILogger,
	httpClient IHTTPClient,
) *RunsCollector {
	return &RunsCollector{
		repository: repository,
		token:      token,
		logger:     logger,
		httpClient: httpClient,
		runs: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "runs",
			Help:      "List how many workflow runs each repository actions",
		}, []string{"repository", "status"}),
	}
}

func (c *RunsCollector) fetchRunsCount(status string) (*int, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/actions/runs?status=%s", c.repository, status), nil)
	if err != nil {
		return nil, xerrors.Errorf("failed to create request object: %w", err)
	}
	request.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, xerrors.Errorf("failed to request: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, xerrors.Errorf("failed to read response: %w", err)
	}

	var workflowRunsResponse WorkflowRunsResponse
	if err := json.Unmarshal(body, &workflowRunsResponse); err != nil {
		return nil, xerrors.Errorf("failed to parse response: %w", err)
	}

	if workflowRunsResponse.TotalCount == nil {
		return nil, xerrors.Errorf("bad response: %s", string(body))
	}

	return workflowRunsResponse.TotalCount, nil
}

func (c *RunsCollector) scrapeRuns() {
	for _, status := range statuses {
		count, err := c.fetchRunsCount(status)
		if err != nil {
			c.logger.Errorf("Failed to fetch runs count: %s\n", err.Error())
			return
		}
		if count != nil {
			labels := []string{
				c.repository,
				status,
			}
			c.runs.WithLabelValues(labels...).Set(float64(*count))
		}
	}
}

func (c *RunsCollector) StartLoop(ctx context.Context, interval time.Duration) {
	go func(ctx context.Context) {
		c.scrapeRuns()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				c.scrapeRuns()
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
}

func (c *RunsCollector) collectors() []prometheus.Collector {
	return []prometheus.Collector{
		c.runs,
	}
}

func (c *RunsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, collector := range c.collectors() {
		collector.Describe(ch)
	}
}

func (c *RunsCollector) Collect(ch chan<- prometheus.Metric) {
	for _, collector := range c.collectors() {
		collector.Collect(ch)
	}
}
