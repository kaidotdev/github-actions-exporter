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
	workflowsPerPage = 100
)

type Workflow struct {
	ID        uint64 `json:"id"`
	NodeId    string `json:"node_id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	URL       string `json:"url"`
	HTMLURL   string `json:"html_url"`
	BadgeURL  string `json:"badge_url"`
}

type WorkflowsResponse struct {
	TotalCount *int       `json:"total_count,omitempty"`
	Workflows  []Workflow `json:"workflows,omitempty"`
}

type WorkflowsCollector struct {
	repository   string
	token        string
	logger       ILogger
	httpClient   IHTTPClient
	workflows    *prometheus.GaugeVec
	billableTime *prometheus.GaugeVec
}

func NewWorkflowsCollector(
	repository string,
	token string,
	logger ILogger,
	httpClient IHTTPClient,
) *WorkflowsCollector {
	return &WorkflowsCollector{
		repository: repository,
		token:      token,
		logger:     logger,
		httpClient: httpClient,
		workflows: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "workflows",
			Help:      "List how many workflows in a repository",
		}, []string{"repository", "state"}),
		billableTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "workflow_billable_time_seconds",
			Help:      "Total billable time of each workflows",
		}, []string{"repository", "name"}),
	}
}

func (c *WorkflowsCollector) fetchWorkflows(page int) ([]Workflow, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows?per_page=%d&page=%d", c.repository, workflowsPerPage, page), nil)
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

	var workflowsResponse WorkflowsResponse
	if err := json.Unmarshal(body, &workflowsResponse); err != nil {
		return nil, xerrors.Errorf("failed to parse response: %w", err)
	}
	if workflowsResponse.TotalCount == nil {
		return nil, xerrors.Errorf("bad response: %s", string(body))
	}

	if *workflowsResponse.TotalCount > runnersPerPage*page {
		workflows, err := c.fetchWorkflows(page + 1)
		if err != nil {
			return nil, xerrors.Errorf("failed to execute fetchWorkflows: %w", err)
		}
		workflowsResponse.Workflows = append(workflowsResponse.Workflows, workflows...)
	}

	return workflowsResponse.Workflows, nil
}

func (c *WorkflowsCollector) fetchBillableTime(id uint64) (*time.Duration, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%d/timing", c.repository, id), nil)
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

	var timing map[string]map[string]map[string]int64
	if err := json.Unmarshal(body, &timing); err != nil {
		return nil, xerrors.Errorf("failed to parse response: %w", err)
	}

	totalBillableMilliSeconds := int64(0)
	for _, m := range timing["billable"] {
		for _, i := range m {
			totalBillableMilliSeconds += i
		}
	}

	totalBillableTime := time.Duration(totalBillableMilliSeconds) * time.Millisecond
	return &totalBillableTime, nil
}

func (c *WorkflowsCollector) scrapeWorkflows() {
	workflows, err := c.fetchWorkflows(1)
	if err != nil {
		c.logger.Errorf("Failed to fetch workflows: %s\n", err.Error())
		return
	}
	workflowsMap := make(map[string][]Workflow)
	billableTimeMap := make(map[string]time.Duration)
	for _, workflow := range workflows {
		workflowsMap[workflow.State] = append(workflowsMap[workflow.State], workflow)

		billableTime, err := c.fetchBillableTime(workflow.ID)
		if err != nil {
			c.logger.Errorf("Failed to fetch billableTime: %s\n", err.Error())
			continue
		}
		billableTimeMap[workflow.Name] = *billableTime
	}
	for state, w := range workflowsMap {
		labels := []string{
			c.repository,
			state,
		}
		c.workflows.WithLabelValues(labels...).Set(float64(len(w)))
	}

	for name, billableTime := range billableTimeMap {
		labels := []string{
			c.repository,
			name,
		}
		c.billableTime.WithLabelValues(labels...).Set(float64(billableTime / time.Second))
	}
}

func (c *WorkflowsCollector) StartLoop(ctx context.Context, interval time.Duration) {
	go func(ctx context.Context) {
		c.scrapeWorkflows()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				c.scrapeWorkflows()
			case <-ctx.Done():
				return
			}
		}
	}(ctx)
}

func (c *WorkflowsCollector) collectors() []prometheus.Collector {
	return []prometheus.Collector{
		c.billableTime,
	}
}

func (c *WorkflowsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, collector := range c.collectors() {
		collector.Describe(ch)
	}
}

func (c *WorkflowsCollector) Collect(ch chan<- prometheus.Metric) {
	for _, collector := range c.collectors() {
		collector.Collect(ch)
	}
}
