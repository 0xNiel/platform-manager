package metrics

import (
	"context"
	"fmt"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// PrometheusClient wraps the Prometheus API client
type PrometheusClient struct {
	api promv1.API
}

// NamespaceMetrics represents resource usage metrics for a namespace
type NamespaceMetrics struct {
	Namespace     string    `json:"namespace"`
	CPUUsageCores float64   `json:"cpuUsageCores"`
	MemoryUsageMB float64   `json:"memoryUsageMB"`
	PodCount      int       `json:"podCount"`
	Timestamp     time.Time `json:"timestamp"`
}

// PodMetrics represents resource usage metrics for a pod
type PodMetrics struct {
	Namespace     string    `json:"namespace"`
	PodName       string    `json:"podName"`
	CPUUsageCores float64   `json:"cpuUsageCores"`
	MemoryUsageMB float64   `json:"memoryUsageMB"`
	Timestamp     time.Time `json:"timestamp"`
}

// NewPrometheusClient creates a new Prometheus client
func NewPrometheusClient(prometheusURL string) (*PrometheusClient, error) {
	client, err := promapi.NewClient(promapi.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client: %w", err)
	}

	return &PrometheusClient{
		api: promv1.NewAPI(client),
	}, nil
}

// GetNamespaceMetrics retrieves resource usage metrics for a namespace
func (c *PrometheusClient) GetNamespaceMetrics(ctx context.Context, namespace string) (*NamespaceMetrics, error) {
	now := time.Now()
	metrics := &NamespaceMetrics{
		Namespace: namespace,
		Timestamp: now,
	}

	// Query CPU usage (rate over 5 minutes)
	cpuQuery := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s",container!="",container!="POD"}[5m]))`, namespace)
	cpuResult, _, err := c.api.Query(ctx, cpuQuery, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query CPU usage: %w", err)
	}
	metrics.CPUUsageCores = parseScalarOrVector(cpuResult)

	// Query memory usage
	memQuery := fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace="%s",container!="",container!="POD"})`, namespace)
	memResult, _, err := c.api.Query(ctx, memQuery, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query memory usage: %w", err)
	}
	metrics.MemoryUsageMB = parseScalarOrVector(memResult) / (1024 * 1024) // Convert to MB

	// Query pod count
	podQuery := fmt.Sprintf(`count(kube_pod_info{namespace="%s"})`, namespace)
	podResult, _, err := c.api.Query(ctx, podQuery, now)
	if err != nil {
		// Pod count is optional, continue without error
		metrics.PodCount = 0
	} else {
		metrics.PodCount = int(parseScalarOrVector(podResult))
	}

	return metrics, nil
}

// GetPodMetrics retrieves resource usage metrics for a specific pod
func (c *PrometheusClient) GetPodMetrics(ctx context.Context, namespace, podName string) (*PodMetrics, error) {
	now := time.Now()
	metrics := &PodMetrics{
		Namespace: namespace,
		PodName:   podName,
		Timestamp: now,
	}

	// Query CPU usage (rate over 5 minutes)
	cpuQuery := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s",pod="%s",container!="",container!="POD"}[5m]))`, namespace, podName)
	cpuResult, _, err := c.api.Query(ctx, cpuQuery, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query pod CPU usage: %w", err)
	}
	metrics.CPUUsageCores = parseScalarOrVector(cpuResult)

	// Query memory usage
	memQuery := fmt.Sprintf(`sum(container_memory_working_set_bytes{namespace="%s",pod="%s",container!="",container!="POD"})`, namespace, podName)
	memResult, _, err := c.api.Query(ctx, memQuery, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query pod memory usage: %w", err)
	}
	metrics.MemoryUsageMB = parseScalarOrVector(memResult) / (1024 * 1024) // Convert to MB

	return metrics, nil
}

// GetTenantMetrics retrieves aggregated metrics for all namespaces in a tenant
func (c *PrometheusClient) GetTenantMetrics(ctx context.Context, namespaces []string) (map[string]*NamespaceMetrics, error) {
	result := make(map[string]*NamespaceMetrics)

	for _, ns := range namespaces {
		metrics, err := c.GetNamespaceMetrics(ctx, ns)
		if err != nil {
			// Log error but continue with other namespaces
			continue
		}
		result[ns] = metrics
	}

	return result, nil
}

// parseScalarOrVector extracts a float64 value from a Prometheus query result
func parseScalarOrVector(val model.Value) float64 {
	switch v := val.(type) {
	case model.Vector:
		if len(v) > 0 {
			return float64(v[0].Value)
		}
	case *model.Scalar:
		return float64(v.Value)
	}
	return 0
}

// IsAvailable checks if Prometheus is available
func (c *PrometheusClient) IsAvailable(ctx context.Context) bool {
	_, _, err := c.api.LabelNames(ctx, nil, time.Now().Add(-time.Minute), time.Now())
	return err == nil
}
