package handlers

import (
	"net/http"

	"github.com/0xNiel/platform-manager/internal/controller"
	"github.com/0xNiel/platform-manager/internal/rules"
	"github.com/go-chi/chi/v5"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TroubleshootingHandler handles troubleshooting API endpoints
type TroubleshootingHandler struct {
	evaluator *controller.RuleEvaluator
}

// NewTroubleshootingHandler creates a new troubleshooting handler
func NewTroubleshootingHandler(evaluator *controller.RuleEvaluator) *TroubleshootingHandler {
	return &TroubleshootingHandler{
		evaluator: evaluator,
	}
}

// RegisterRoutes registers troubleshooting routes
func (h *TroubleshootingHandler) RegisterRoutes(r chi.Router) {
	r.Route("/troubleshooting", func(r chi.Router) {
		r.Get("/summary", h.GetSummary)
		r.Get("/findings", h.GetFindings)
		r.Get("/findings/{id}", h.GetFinding)
		r.Get("/tenants/{name}", h.GetTenantFindings)
		r.Post("/scan", h.TriggerScan)
		r.Post("/findings/{id}/resolve", h.ResolveFinding)
		r.Get("/rules", h.GetRules)
	})
}

// GetSummary returns a platform-wide summary of findings
func (h *TroubleshootingHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	engine := h.evaluator.GetEngine()
	summary := engine.GetPlatformSummary()

	WriteJSON(w, http.StatusOK, summary)
}

// GetFindings returns all active findings
func (h *TroubleshootingHandler) GetFindings(w http.ResponseWriter, r *http.Request) {
	tenantName := r.URL.Query().Get("tenant")
	severity := r.URL.Query().Get("severity")

	engine := h.evaluator.GetEngine()
	findings := engine.GetFindings(tenantName)

	// Filter by severity if specified
	if severity != "" {
		filtered := make([]rules.Finding, 0)
		for _, finding := range findings {
			if string(finding.Severity) == severity {
				filtered = append(filtered, finding)
			}
		}
		findings = filtered
	}

	response := map[string]interface{}{
		"findings": findings,
		"total":    len(findings),
	}

	WriteJSON(w, http.StatusOK, response)
}

// GetFinding returns a specific finding by ID
func (h *TroubleshootingHandler) GetFinding(w http.ResponseWriter, r *http.Request) {
	findingID := chi.URLParam(r, "id")

	engine := h.evaluator.GetEngine()
	findings := engine.GetFindings("")

	for _, finding := range findings {
		if finding.ID == findingID {
			WriteJSON(w, http.StatusOK, finding)
			return
		}
	}

	http.Error(w, "Finding not found", http.StatusNotFound)
}

// GetTenantFindings returns findings for a specific tenant
func (h *TroubleshootingHandler) GetTenantFindings(w http.ResponseWriter, r *http.Request) {
	tenantName := chi.URLParam(r, "name")

	engine := h.evaluator.GetEngine()
	findings := engine.GetFindings(tenantName)
	summary := engine.GetTenantSummary(tenantName)

	response := map[string]interface{}{
		"tenantName": tenantName,
		"summary":    summary,
		"findings":   findings,
	}

	WriteJSON(w, http.StatusOK, response)
}

// TriggerScan triggers an immediate rule evaluation
func (h *TroubleshootingHandler) TriggerScan(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := log.FromContext(ctx)

	// Trigger evaluation in background
	go func() {
		if err := h.evaluator.TriggerEvaluation(r.Context()); err != nil {
			logger.Error(err, "manual scan failed")
		}
	}()

	response := map[string]interface{}{
		"message": "Scan triggered successfully",
		"status":  "running",
	}

	WriteJSON(w, http.StatusAccepted, response)
}

// ResolveFinding marks a finding as resolved
func (h *TroubleshootingHandler) ResolveFinding(w http.ResponseWriter, r *http.Request) {
	findingID := chi.URLParam(r, "id")

	engine := h.evaluator.GetEngine()
	if !engine.MarkResolved(findingID) {
		http.Error(w, "Finding not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"message": "Finding marked as resolved",
		"id":      findingID,
	}

	WriteJSON(w, http.StatusOK, response)
}

// GetRules returns all available rules
func (h *TroubleshootingHandler) GetRules(w http.ResponseWriter, r *http.Request) {
	engine := h.evaluator.GetEngine()
	allRules := engine.GetRules()

	type RuleInfo struct {
		ID          string         `json:"id"`
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Severity    rules.Severity `json:"severity"`
	}

	ruleInfos := make([]RuleInfo, len(allRules))
	for i, rule := range allRules {
		ruleInfos[i] = RuleInfo{
			ID:          rule.ID(),
			Name:        rule.Name(),
			Description: rule.Description(),
			Severity:    rule.Severity(),
		}
	}

	response := map[string]interface{}{
		"rules": ruleInfos,
		"total": len(ruleInfos),
	}

	WriteJSON(w, http.StatusOK, response)
}
