package handlers

import (
	"fmt"
	"net/http"
	"strings"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
	"github.com/0xNiel/platform-manager/internal/metrics"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MetricsHandler handles metrics-related API requests
type MetricsHandler struct {
	Client           client.Client
	PrometheusClient *metrics.PrometheusClient
}

// NewMetricsHandler creates a new MetricsHandler
func NewMetricsHandler(c client.Client, promClient *metrics.PrometheusClient) *MetricsHandler {
	return &MetricsHandler{
		Client:           c,
		PrometheusClient: promClient,
	}
}

// GetNamespaceMetrics handles GET /api/v1/metrics/namespaces/{namespace}
func (h *MetricsHandler) GetNamespaceMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract namespace from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/metrics/namespaces/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "Namespace is required", http.StatusBadRequest)
		return
	}
	namespace := pathParts[0]

	// Check if Prometheus client is available
	if h.PrometheusClient == nil {
		http.Error(w, "Metrics service not available", http.StatusServiceUnavailable)
		return
	}

	// Get metrics from Prometheus
	nsMetrics, err := h.PrometheusClient.GetNamespaceMetrics(ctx, namespace)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get namespace metrics: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, nsMetrics)
}

// GetTenantMetrics handles GET /api/v1/metrics/tenants/{id}
func (h *MetricsHandler) GetTenantMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract tenant ID from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/metrics/tenants/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "Tenant ID is required", http.StatusBadRequest)
		return
	}
	tenantID := pathParts[0]

	// Get the tenant
	tenant := &platformv1alpha1.Tenant{}
	if err := h.Client.Get(ctx, types.NamespacedName{Name: tenantID}, tenant); err != nil {
		http.Error(w, fmt.Sprintf("Tenant not found: %v", err), http.StatusNotFound)
		return
	}

	// Check if Prometheus client is available
	if h.PrometheusClient == nil {
		http.Error(w, "Metrics service not available", http.StatusServiceUnavailable)
		return
	}

	// Get metrics for all tenant namespaces
	tenantMetrics, err := h.PrometheusClient.GetTenantMetrics(ctx, tenant.Spec.Namespaces)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get tenant metrics: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate aggregated metrics
	type AggregatedMetrics struct {
		TenantID         string                               `json:"tenantId"`
		TotalCPUCores    float64                              `json:"totalCpuCores"`
		TotalMemoryMB    float64                              `json:"totalMemoryMb"`
		TotalPods        int                                  `json:"totalPods"`
		NamespaceMetrics map[string]*metrics.NamespaceMetrics `json:"namespaceMetrics"`
	}

	aggregated := AggregatedMetrics{
		TenantID:         tenantID,
		NamespaceMetrics: tenantMetrics,
	}

	for _, nsMetrics := range tenantMetrics {
		aggregated.TotalCPUCores += nsMetrics.CPUUsageCores
		aggregated.TotalMemoryMB += nsMetrics.MemoryUsageMB
		aggregated.TotalPods += nsMetrics.PodCount
	}

	WriteJSON(w, http.StatusOK, aggregated)
}

// GetPodMetrics handles GET /api/v1/metrics/pods/{namespace}/{podName}
func (h *MetricsHandler) GetPodMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract namespace and pod name from path
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/metrics/pods/"), "/")
	if len(pathParts) < 2 || pathParts[0] == "" || pathParts[1] == "" {
		http.Error(w, "Namespace and pod name are required", http.StatusBadRequest)
		return
	}
	namespace := pathParts[0]
	podName := pathParts[1]

	// Check if Prometheus client is available
	if h.PrometheusClient == nil {
		http.Error(w, "Metrics service not available", http.StatusServiceUnavailable)
		return
	}

	// Get metrics from Prometheus
	podMetrics, err := h.PrometheusClient.GetPodMetrics(ctx, namespace, podName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get pod metrics: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJSON(w, http.StatusOK, podMetrics)
}
