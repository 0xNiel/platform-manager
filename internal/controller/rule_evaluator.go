package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0xNiel/platform-manager/internal/rules"
	"github.com/0xNiel/platform-manager/internal/rules/builtin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// RuleEvaluator periodically evaluates rules against resources
type RuleEvaluator struct {
	client     client.Client
	apiReader  client.Reader
	engine     *rules.Engine
	interval   time.Duration
	mu         sync.RWMutex
	lastResult *rules.EvaluationResult
}

// NewRuleEvaluator creates a new rule evaluator
func NewRuleEvaluator(mgr manager.Manager, interval time.Duration) *RuleEvaluator {
	engine := rules.NewEngine()
	engine.RegisterRules(builtin.GetAllRules())

	return &RuleEvaluator{
		client:    mgr.GetClient(),
		apiReader: mgr.GetAPIReader(),
		engine:    engine,
		interval:  interval,
	}
}

// Start runs the rule evaluator
func (r *RuleEvaluator) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("starting rule evaluator", "interval", r.interval)

	// Run immediately on startup
	if err := r.evaluate(ctx); err != nil {
		logger.Error(err, "initial rule evaluation failed")
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping rule evaluator")
			return nil
		case <-ticker.C:
			if err := r.evaluate(ctx); err != nil {
				logger.Error(err, "rule evaluation failed")
			}
		}
	}
}

// evaluate runs all rules against all resources
func (r *RuleEvaluator) evaluate(ctx context.Context) error {
	logger := log.FromContext(ctx)
	startTime := time.Now()

	logger.Info("starting rule evaluation")

	// Get all resources to evaluate
	resources, tenantMap, err := r.getAllResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	logger.Info("resources collected", "count", len(resources))

	// Evaluate rules
	result := r.engine.EvaluateResources(ctx, resources, tenantMap)

	// Store result
	r.mu.Lock()
	r.lastResult = &result
	r.mu.Unlock()

	logger.Info("rule evaluation complete",
		"duration", time.Since(startTime),
		"resources", result.ResourcesChecked,
		"rules", result.RulesEvaluated,
		"findings", len(result.Findings),
		"errors", len(result.Errors),
	)

	// Log findings summary
	summary := r.engine.GetPlatformSummary()
	logger.Info("findings summary",
		"total", summary.TotalFindings,
		"critical", summary.Critical,
		"high", summary.High,
		"medium", summary.Medium,
		"low", summary.Low,
		"info", summary.Info,
	)

	return nil
}

// getAllResources collects all resources that should be evaluated
func (r *RuleEvaluator) getAllResources(ctx context.Context) ([]*unstructured.Unstructured, map[string]string, error) {
	var allResources []*unstructured.Unstructured
	tenantMap := make(map[string]string)

	// Resource types to scan
	resourceTypes := []schema.GroupVersionKind{
		// Pods
		{Group: "", Version: "v1", Kind: "Pod"},

		// Crossplane Providers
		{Group: "pkg.crossplane.io", Version: "v1", Kind: "Provider"},

		// ArgoCD Applications
		{Group: "argoproj.io", Version: "v1alpha1", Kind: "Application"},

		// Resource Quotas
		{Group: "", Version: "v1", Kind: "ResourceQuota"},

		// ResourceSummaries (our own CRD)
		{Group: "platform.io", Version: "v1alpha1", Kind: "ResourceSummary"},
	}

	// Also scan IAM resources
	iamResourceTypes := []schema.GroupVersionKind{
		{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "Role"},
		{Group: "iam.aws.upbound.io", Version: "v1beta1", Kind: "Policy"},
	}
	resourceTypes = append(resourceTypes, iamResourceTypes...)

	// Scan Crossplane managed resources (any XR or Claim)
	// This requires dynamic discovery, but for now we'll focus on specific types

	for _, gvk := range resourceTypes {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)

		if err := r.apiReader.List(ctx, list); err != nil {
			// Log but continue - some resource types may not exist
			log.FromContext(ctx).V(1).Info("skipping resource type",
				"gvk", gvk.String(),
				"error", err.Error(),
			)
			continue
		}

		for i := range list.Items {
			resource := &list.Items[i]
			allResources = append(allResources, resource)

			// Extract tenant from labels
			labels := resource.GetLabels()
			if labels != nil {
				if tenant, ok := labels["platform.io/tenant"]; ok {
					tenantMap[string(resource.GetUID())] = tenant
				}
			}
		}
	}

	return allResources, tenantMap, nil
}

// GetEngine returns the rule engine for API access
func (r *RuleEvaluator) GetEngine() *rules.Engine {
	return r.engine
}

// GetLastResult returns the most recent evaluation result
func (r *RuleEvaluator) GetLastResult() *rules.EvaluationResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastResult
}

// TriggerEvaluation triggers an immediate evaluation
func (r *RuleEvaluator) TriggerEvaluation(ctx context.Context) error {
	return r.evaluate(ctx)
}

// NeedsLeaderElection implements the LeaderElectionRunnable interface
func (r *RuleEvaluator) NeedsLeaderElection() bool {
	return true
}
