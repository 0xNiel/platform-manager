/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package controller

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
	"github.com/0xNiel/platform-manager/internal/iam"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// IAMDriftScanner periodically scans for IAM drift
type IAMDriftScanner struct {
	client.Client
	apiReader    client.Reader // Non-cached reader for listing CRDs not in scheme
	logger       logr.Logger
	driftChecker *iam.DriftChecker

	// Drift results cache
	mu              sync.RWMutex
	platformSummary *iam.PlatformDriftSummary
	lastScanTime    time.Time
	scanInterval    time.Duration
}

// NewIAMDriftScanner creates a new IAM drift scanner
func NewIAMDriftScanner(k8sClient client.Client, scanInterval time.Duration) (*IAMDriftScanner, error) {
	logger := log.Log.WithName("iam-drift-scanner")

	// Get AWS configuration from environment
	awsEndpoint := os.Getenv("AWS_ENDPOINT")
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = "us-east-1"
	}

	// Create AWS client
	awsClient, err := iam.NewAWSClient(context.Background(), logger, awsEndpoint, awsRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Create drift checker
	driftChecker := iam.NewDriftChecker(k8sClient, awsClient, logger)

	return &IAMDriftScanner{
		Client:       k8sClient,
		logger:       logger,
		driftChecker: driftChecker,
		scanInterval: scanInterval,
		platformSummary: &iam.PlatformDriftSummary{
			TenantSummaries: make(map[string]*iam.TenantDriftSummary),
		},
	}, nil
}

// Start begins the periodic drift scanning
func (s *IAMDriftScanner) Start(ctx context.Context) error {
	s.logger.Info("Starting IAM drift scanner", "interval", s.scanInterval)

	// Run initial scan
	if err := s.ScanDrift(ctx); err != nil {
		s.logger.Error(err, "Initial drift scan failed")
	}

	// Start periodic scanning
	ticker := time.NewTicker(s.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping IAM drift scanner")
			return nil
		case <-ticker.C:
			if err := s.ScanDrift(ctx); err != nil {
				s.logger.Error(err, "Drift scan failed")
			}
		}
	}
}

// ScanDrift performs a full drift scan
func (s *IAMDriftScanner) ScanDrift(ctx context.Context) error {
	s.logger.Info("Starting drift scan")
	startTime := time.Now()

	// Create new platform summary
	platformSummary := &iam.PlatformDriftSummary{
		LastChecked:     startTime,
		TenantSummaries: make(map[string]*iam.TenantDriftSummary),
	}

	// Get all tenants
	tenantList := &platformv1alpha1.TenantList{}
	if err := s.List(ctx, tenantList); err != nil {
		return fmt.Errorf("failed to list tenants: %w", err)
	}

	platformSummary.TotalTenants = len(tenantList.Items)

	// Scan each tenant
	for _, tenant := range tenantList.Items {
		tenantSummary, err := s.scanTenantDrift(ctx, tenant.Name)
		if err != nil {
			s.logger.Error(err, "Failed to scan tenant drift", "tenant", tenant.Name)
			continue
		}

		platformSummary.TenantSummaries[tenant.Name] = tenantSummary
		platformSummary.TotalRoles += tenantSummary.TotalRoles
		platformSummary.TotalPolicies += tenantSummary.TotalPolicies
		platformSummary.RolesWithDrift += tenantSummary.RolesWithDrift
		platformSummary.PoliciesWithDrift += tenantSummary.PoliciesWithDrift
		platformSummary.CriticalDrifts += tenantSummary.CriticalDrifts
		platformSummary.HighDrifts += tenantSummary.HighDrifts
		platformSummary.WarningDrifts += tenantSummary.WarningDrifts
	}

	// Update cache
	s.mu.Lock()
	s.platformSummary = platformSummary
	s.lastScanTime = startTime
	s.mu.Unlock()

	duration := time.Since(startTime)
	s.logger.Info("Drift scan completed",
		"duration", duration,
		"tenants", platformSummary.TotalTenants,
		"totalRoles", platformSummary.TotalRoles,
		"totalPolicies", platformSummary.TotalPolicies,
		"rolesWithDrift", platformSummary.RolesWithDrift,
		"policiesWithDrift", platformSummary.PoliciesWithDrift,
	)

	return nil
}

// scanTenantDrift scans drift for a single tenant
func (s *IAMDriftScanner) scanTenantDrift(ctx context.Context, tenantName string) (*iam.TenantDriftSummary, error) {
	summary := &iam.TenantDriftSummary{
		TenantName:  tenantName,
		LastChecked: time.Now(),
		Drifts:      []iam.DriftResult{},
	}

	// Scan IAM Roles
	roleResults, err := s.scanRoles(ctx, tenantName)
	if err != nil {
		s.logger.Error(err, "Failed to scan roles", "tenant", tenantName)
	} else {
		summary.TotalRoles = len(roleResults)
		for _, result := range roleResults {
			if result.HasDrift {
				summary.RolesWithDrift++
				summary.Drifts = append(summary.Drifts, result)

				switch result.Severity {
				case iam.DriftSeverityCritical:
					summary.CriticalDrifts++
				case iam.DriftSeverityHigh:
					summary.HighDrifts++
				case iam.DriftSeverityWarning:
					summary.WarningDrifts++
				}
			}
		}
	}

	// Scan IAM Policies
	policyResults, err := s.scanPolicies(ctx, tenantName)
	if err != nil {
		s.logger.Error(err, "Failed to scan policies", "tenant", tenantName)
	} else {
		summary.TotalPolicies = len(policyResults)
		for _, result := range policyResults {
			if result.HasDrift {
				summary.PoliciesWithDrift++
				summary.Drifts = append(summary.Drifts, result)

				switch result.Severity {
				case iam.DriftSeverityCritical:
					summary.CriticalDrifts++
				case iam.DriftSeverityHigh:
					summary.HighDrifts++
				case iam.DriftSeverityWarning:
					summary.WarningDrifts++
				}
			}
		}
	}

	return summary, nil
}

// scanRoles scans all IAM roles for a tenant
func (s *IAMDriftScanner) scanRoles(ctx context.Context, tenantName string) ([]iam.DriftResult, error) {
	var results []iam.DriftResult

	// Define the GVK for Crossplane IAM Role
	roleGVK := schema.GroupVersionKind{
		Group:   "iam.aws.upbound.io",
		Version: "v1beta1",
		Kind:    "Role",
	}

	// List all roles with tenant label
	roleList := &unstructured.UnstructuredList{}
	roleList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   roleGVK.Group,
		Version: roleGVK.Version,
		Kind:    "RoleList",
	})

	listOpts := &client.ListOptions{}
	client.MatchingLabels{"platform.io/tenant": tenantName}.ApplyToList(listOpts)

	// Use apiReader (non-cached) to list CRDs not in the scheme
	if err := s.apiReader.List(ctx, roleList, listOpts); err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	s.logger.Info("Listed IAM roles for tenant",
		"tenant", tenantName,
		"roleCount", len(roleList.Items),
	)

	// Check drift for each role
	for _, item := range roleList.Items {
		result, err := s.driftChecker.CheckRoleDrift(ctx, &item)
		if err != nil {
			s.logger.Error(err, "Failed to check role drift",
				"role", item.GetName(),
				"tenant", tenantName,
			)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// scanPolicies scans all IAM policies for a tenant
func (s *IAMDriftScanner) scanPolicies(ctx context.Context, tenantName string) ([]iam.DriftResult, error) {
	var results []iam.DriftResult

	// Define the GVK for Crossplane IAM Policy
	policyGVK := schema.GroupVersionKind{
		Group:   "iam.aws.upbound.io",
		Version: "v1beta1",
		Kind:    "Policy",
	}

	// List all policies with tenant label
	policyList := &unstructured.UnstructuredList{}
	policyList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   policyGVK.Group,
		Version: policyGVK.Version,
		Kind:    "PolicyList",
	})

	listOpts := &client.ListOptions{}
	client.MatchingLabels{"platform.io/tenant": tenantName}.ApplyToList(listOpts)

	// Use apiReader (non-cached) to list CRDs not in the scheme
	if err := s.apiReader.List(ctx, policyList, listOpts); err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	s.logger.Info("Listed IAM policies for tenant",
		"tenant", tenantName,
		"policyCount", len(policyList.Items),
	)

	// Check drift for each policy
	for _, item := range policyList.Items {
		result, err := s.driftChecker.CheckPolicyDrift(ctx, &item)
		if err != nil {
			s.logger.Error(err, "Failed to check policy drift",
				"policy", item.GetName(),
				"tenant", tenantName,
			)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// GetPlatformSummary returns the current platform drift summary
func (s *IAMDriftScanner) GetPlatformSummary() *iam.PlatformDriftSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.platformSummary
}

// GetTenantSummary returns the drift summary for a specific tenant
func (s *IAMDriftScanner) GetTenantSummary(tenantName string) (*iam.TenantDriftSummary, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary, ok := s.platformSummary.TenantSummaries[tenantName]
	return summary, ok
}

// GetLastScanTime returns when the last scan was performed
func (s *IAMDriftScanner) GetLastScanTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastScanTime
}

// SetupWithManager sets up the scanner with the controller manager
func (s *IAMDriftScanner) SetupWithManager(mgr ctrl.Manager) error {
	// Set the API reader (non-cached) for listing CRDs not in the scheme
	s.apiReader = mgr.GetAPIReader()
	// Add the scanner as a runnable to the manager
	return mgr.Add(s)
}
