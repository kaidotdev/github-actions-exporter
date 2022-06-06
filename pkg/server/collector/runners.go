package collector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/xerrors"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	runnerStatuses = []string{
		"queued",
		"in_progress",
		"completed",
	}
)

type WorkflowRunnersResponse struct {
	TotalCount   int               `json:"total_count"`
	WorkflowRuns []json.RawMessage `json:"workflow_runs"`
}

type RunnersCollector struct {
	repository string
	token      string
	logger     ILogger
	httpClient IHTTPClient
	runners    *prometheus.GaugeVec
}

func NewRunnersCollector(
	repository string,
	token string,
	logger ILogger,
	httpClient IHTTPClient,
) *RunnersCollector {
	return &RunnersCollector{
		repository: repository,
		token:      token,
		logger:     logger,
		httpClient: httpClient,
		runners: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "runners",
			Help:      "List how many workflow runners each repository actions",
		}, []string{"repository", "status"}),
	}
}

func (c *RunnersCollector) fetchRunnersCount(status string) (int, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/actions/runners?status=%s", c.repository, status), nil)
	if err != nil {
		return 0, xerrors.Errorf("failed to create request object: %w", err)
	}
	request.Header.Set("Authorization", fmt.Sprintf("token %s", c.token))
	response, err := c.httpClient.Do(request)
	if err != nil {
		return 0, xerrors.Errorf("failed to request: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0, xerrors.Errorf("failed to read response: %w", err)
	}

	var workflowRunsResponse WorkflowRunnersResponse
	if err := json.Unmarshal(body, &workflowRunsResponse); err != nil {
		return 0, xerrors.Errorf("failed to parse response: %w", err)
	}

	return workflowRunsResponse.TotalCount, nil
}

func (c *RunnersCollector) scrapeRunners() {
	for _, status := range runnerStatuses {
		count, err := c.fetchRunnersCount(status)
		if err != nil {
			c.logger.Errorf("Failed to fetch runners count: %s\n", err.Error())
			return
		}
		labels := []string{
			c.repository,
			status,
		}
		c.runners.WithLabelValues(labels...).Set(float64(count))
	}
}

func (c *RunnersCollector) collectors() []prometheus.Collector {
	return []prometheus.Collector{
		c.runners,
	}
}

func (c *RunnersCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, collector := range c.collectors() {
		collector.Describe(ch)
	}
}

func (c *RunnersCollector) Collect(ch chan<- prometheus.Metric) {
	c.scrapeRunners()

	for _, collector := range c.collectors() {
		collector.Collect(ch)
	}
}
