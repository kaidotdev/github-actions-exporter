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

var (
	runnerStatuses = []string{
		"offline",
		"online",
	}
	runnersPerPage = 100
)

type Runner struct {
	ID     uint64 `json:"id"`
	Name   string `json:"name"`
	OS     string `json:"os"`
	Status string `json:"status"`
	Busy   bool   `json:"busy"`
}

type RunnersResponse struct {
	TotalCount int      `json:"total_count"`
	Runners    []Runner `json:"runners"`
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

func (c *RunnersCollector) fetchRunners(page int) ([]Runner, error) {
	fmt.Printf("page: %d\n", page)
	request, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/actions/runners?per_page=%d&page=%d", c.repository, runnersPerPage, page), nil)
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

	var runnersResponse RunnersResponse
	if err := json.Unmarshal(body, &runnersResponse); err != nil {
		return nil, xerrors.Errorf("failed to parse response: %w", err)
	}

	if runnersResponse.TotalCount > runnersPerPage*page {
		runners, err := c.fetchRunners(page + 1)
		if err != nil {
			return nil, xerrors.Errorf("failed to execute fetchRunners: %w", err)
		}
		runnersResponse.Runners = append(runnersResponse.Runners, runners...)
	}

	return runnersResponse.Runners, nil
}

func (c *RunnersCollector) scrapeRunners() {
	runners, err := c.fetchRunners(1)
	if err != nil {
		c.logger.Errorf("Failed to fetch runners count: %s\n", err.Error())
		return
	}
	m := make(map[string][]Runner)
	for _, runner := range runners {
		m[runner.Status] = append(m[runner.Status], runner)
	}
	for _, status := range runnerStatuses {
		labels := []string{
			c.repository,
			status,
		}
		c.runners.WithLabelValues(labels...).Set(float64(len(m[status])))
	}
}

func (c *RunnersCollector) StartLoop(ctx context.Context, interval time.Duration) {
	go func(ctx context.Context) {
		c.scrapeRunners()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				c.scrapeRunners()
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
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
	for _, collector := range c.collectors() {
		collector.Collect(ch)
	}
}
