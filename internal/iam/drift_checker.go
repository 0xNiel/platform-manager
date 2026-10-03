/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DriftChecker performs drift detection on IAM resources
type DriftChecker struct {
	k8sClient client.Client
	awsClient *AWSClient
	logger    logr.Logger
}

// NewDriftChecker creates a new drift checker
func NewDriftChecker(k8sClient client.Client, awsClient *AWSClient, logger logr.Logger) *DriftChecker {
	return &DriftChecker{
		k8sClient: k8sClient,
		awsClient: awsClient,
		logger:    logger,
	}
}

// CheckRoleDrift checks drift for an IAM Role
func (dc *DriftChecker) CheckRoleDrift(ctx context.Context, roleResource *unstructured.Unstructured) (*DriftResult, error) {
	result := &DriftResult{
		ResourceType:  ResourceTypeRole,
		ResourceName:  roleResource.GetName(),
		Namespace:     roleResource.GetNamespace(),
		TenantName:    roleResource.GetLabels()["platform.io/tenant"],
		CheckedAt:     time.Now(),
		HasDrift:      false,
		Severity:      DriftSeverityNone,
		DesiredState:  make(map[string]interface{}),
		ObservedState: make(map[string]interface{}),
		ActualState:   make(map[string]interface{}),
	}

	// Extract spec.forProvider (desired state)
	specForProvider, found, err := unstructured.NestedMap(roleResource.Object, "spec", "forProvider")
	if err != nil || !found {
		result.Error = "failed to extract spec.forProvider"
		return result, fmt.Errorf("failed to extract spec.forProvider: %w", err)
	}
	result.DesiredState = specForProvider

	// Extract status.atProvider (observed state by Crossplane)
	statusAtProvider, found, err := unstructured.NestedMap(roleResource.Object, "status", "atProvider")
	if err != nil || !found {
		// Resource may not be reconciled yet
		result.Details = append(result.Details, DriftDetail{
			Type:     DriftTypeReconciliationLag,
			Severity: DriftSeverityWarning,
			Message:  "Resource has not been reconciled by Crossplane yet",
		})
		result.HasDrift = true
		result.Severity = DriftSeverityWarning
		return result, nil
	}
	result.ObservedState = statusAtProvider

	// Get role name from status (AWS role name)
	roleName, ok := statusAtProvider["roleName"].(string)
	if !ok {
		// Try to get from ARN
		arn, ok := statusAtProvider["arn"].(string)
		if ok {
			result.ResourceARN = arn
			// Extract role name from ARN (arn:aws:iam::account-id:role/role-name)
			parts := strings.Split(arn, "/")
			if len(parts) > 0 {
				roleName = parts[len(parts)-1]
			}
		}
	}

	if roleName == "" {
		result.Error = "could not determine AWS role name"
		return result, fmt.Errorf("could not determine AWS role name from status")
	}

	// Fetch actual state from AWS
	awsRole, err := dc.awsClient.GetRole(ctx, roleName)
	if err != nil {
		result.Error = fmt.Sprintf("failed to fetch from AWS: %v", err)
		result.Details = append(result.Details, DriftDetail{
			Type:     DriftTypeMissingPrivileges,
			Severity: DriftSeverityCritical,
			Message:  fmt.Sprintf("Role not found in AWS: %v", err),
		})
		result.HasDrift = true
		result.Severity = DriftSeverityCritical
		return result, nil
	}

	// Convert AWS role to map for comparison
	awsRoleMap := map[string]interface{}{
		"roleName": *awsRole.RoleName,
		"arn":      *awsRole.Arn,
		"path":     *awsRole.Path,
	}
	if awsRole.Description != nil {
		awsRoleMap["description"] = *awsRole.Description
	}
	result.ActualState = awsRoleMap
	result.ResourceARN = *awsRole.Arn

	// Compare assume role policy (trust policy)
	if err := dc.compareAssumeRolePolicy(result, specForProvider, awsRole); err != nil {
		dc.logger.Error(err, "failed to compare assume role policy", "role", roleName)
	}

	// Check for inline policies drift
	if err := dc.checkInlinePoliciesDrift(ctx, result, roleName, specForProvider); err != nil {
		dc.logger.Error(err, "failed to check inline policies", "role", roleName)
	}

	// Check for attached managed policies drift
	if err := dc.checkAttachedPoliciesDrift(ctx, result, roleName, specForProvider); err != nil {
		dc.logger.Error(err, "failed to check attached policies", "role", roleName)
	}

	// Determine overall severity
	if len(result.Details) > 0 {
		result.HasDrift = true
		result.Severity = dc.calculateOverallSeverity(result.Details)

		// Collect drift types
		driftTypeMap := make(map[DriftType]bool)
		for _, detail := range result.Details {
			driftTypeMap[detail.Type] = true
		}
		for dt := range driftTypeMap {
			result.DriftTypes = append(result.DriftTypes, dt)
		}
	}

	return result, nil
}

// compareAssumeRolePolicy compares the assume role policy (trust policy)
func (dc *DriftChecker) compareAssumeRolePolicy(result *DriftResult, specForProvider map[string]interface{}, awsRole *iamtypes.Role) error {
	// Get desired assume role policy from spec
	desiredPolicy, ok := specForProvider["assumeRolePolicy"].(string)
	if !ok {
		return fmt.Errorf("assumeRolePolicy not found in spec")
	}

	// Parse desired policy
	desiredDoc, err := ParsePolicyDocument(desiredPolicy)
	if err != nil {
		return fmt.Errorf("failed to parse desired policy: %w", err)
	}

	// Get actual policy from AWS
	if awsRole.AssumeRolePolicyDocument == nil {
		result.Details = append(result.Details, DriftDetail{
			Type:     DriftTypeMissingPrivileges,
			Severity: DriftSeverityCritical,
			Message:  "Assume role policy missing in AWS",
		})
		return nil
	}

	// Decode and parse actual policy
	actualDoc, err := dc.awsClient.DecodeAssumeRolePolicy(*awsRole.AssumeRolePolicyDocument)
	if err != nil {
		return fmt.Errorf("failed to decode actual policy: %w", err)
	}

	// Compare policies
	if !dc.comparePolicyDocuments(desiredDoc, actualDoc) {
		result.Details = append(result.Details, DriftDetail{
			Type:     DriftTypePolicyMismatch,
			Severity: DriftSeverityWarning,
			Message:  "Assume role policy differs from spec",
			Path:     "assumeRolePolicy",
			Expected: desiredDoc,
			Actual:   actualDoc,
		})
	}

	return nil
}

// checkInlinePoliciesDrift compares the role's inline policies in AWS with
// spec.forProvider.inlinePolicy.
func (dc *DriftChecker) checkInlinePoliciesDrift(ctx context.Context, result *DriftResult, roleName string, specForProvider map[string]interface{}) error {
	// List inline policies from AWS
	awsPolicyNames, err := dc.awsClient.ListRolePolicies(ctx, roleName)
	if err != nil {
		return fmt.Errorf("failed to list inline policies: %w", err)
	}

	declared := declaredInlinePolicyNames(specForProvider)
	inAWS := make(map[string]bool, len(awsPolicyNames))

	for _, policyName := range awsPolicyNames {
		inAWS[policyName] = true
		if declared[policyName] {
			continue
		}
		result.Details = append(result.Details, DriftDetail{
			Type:     DriftTypeExtraPrivileges,
			Severity: DriftSeverityCritical,
			Message:  fmt.Sprintf("Inline policy %s exists in AWS but is not declared in the spec", policyName),
			Path:     fmt.Sprintf("inlinePolicies.%s", policyName),
			Actual:   policyName,
		})
	}

	for policyName := range declared {
		if inAWS[policyName] {
			continue
		}
		result.Details = append(result.Details, DriftDetail{
			Type:     DriftTypeMissingPrivileges,
			Severity: DriftSeverityCritical,
			Message:  fmt.Sprintf("Inline policy %s is declared in the spec but missing in AWS", policyName),
			Path:     fmt.Sprintf("inlinePolicies.%s", policyName),
			Expected: policyName,
		})
	}

	return nil
}

// checkAttachedPoliciesDrift compares the role's attached managed policies in
// AWS with spec.forProvider.managedPolicyArns.
func (dc *DriftChecker) checkAttachedPoliciesDrift(ctx context.Context, result *DriftResult, roleName string, specForProvider map[string]interface{}) error {
	// List attached policies from AWS
	attachedPolicies, err := dc.awsClient.ListAttachedRolePolicies(ctx, roleName)
	if err != nil {
		return fmt.Errorf("failed to list attached policies: %w", err)
	}

	declared := declaredManagedPolicyArns(specForProvider)

	for _, policy := range attachedPolicies {
		arn := aws.ToString(policy.PolicyArn)
		if declared[arn] {
			continue
		}
		result.Details = append(result.Details, DriftDetail{
			Type:     DriftTypeExtraPrivileges,
			Severity: DriftSeverityCritical,
			Message:  fmt.Sprintf("Attached policy %s is not declared in the spec", aws.ToString(policy.PolicyName)),
			Path:     "attachedPolicies",
			Actual:   arn,
		})
	}

	return nil
}

// declaredInlinePolicyNames returns the names in spec.forProvider.inlinePolicy.
func declaredInlinePolicyNames(specForProvider map[string]interface{}) map[string]bool {
	names := map[string]bool{}
	items, _ := specForProvider["inlinePolicy"].([]interface{})
	for _, item := range items {
		policy, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := policy["name"].(string); ok && name != "" {
			names[name] = true
		}
	}
	return names
}

// declaredManagedPolicyArns returns the ARNs in spec.forProvider.managedPolicyArns.
func declaredManagedPolicyArns(specForProvider map[string]interface{}) map[string]bool {
	arns := map[string]bool{}
	items, _ := specForProvider["managedPolicyArns"].([]interface{})
	for _, item := range items {
		if arn, ok := item.(string); ok && arn != "" {
			arns[arn] = true
		}
	}
	return arns
}

// CheckPolicyDrift checks drift for an IAM Policy
func (dc *DriftChecker) CheckPolicyDrift(ctx context.Context, policyResource *unstructured.Unstructured) (*DriftResult, error) {
	result := &DriftResult{
		ResourceType:  ResourceTypePolicy,
		ResourceName:  policyResource.GetName(),
		Namespace:     policyResource.GetNamespace(),
		TenantName:    policyResource.GetLabels()["platform.io/tenant"],
		CheckedAt:     time.Now(),
		HasDrift:      false,
		Severity:      DriftSeverityNone,
		DesiredState:  make(map[string]interface{}),
		ObservedState: make(map[string]interface{}),
		ActualState:   make(map[string]interface{}),
	}

	// Extract spec.forProvider (desired state)
	specForProvider, found, err := unstructured.NestedMap(policyResource.Object, "spec", "forProvider")
	if err != nil || !found {
		result.Error = "failed to extract spec.forProvider"
		return result, fmt.Errorf("failed to extract spec.forProvider: %w", err)
	}
	result.DesiredState = specForProvider

	// Extract status.atProvider (observed state)
	statusAtProvider, found, err := unstructured.NestedMap(policyResource.Object, "status", "atProvider")
	if err != nil || !found {
		result.Details = append(result.Details, DriftDetail{
			Type:     DriftTypeReconciliationLag,
			Severity: DriftSeverityWarning,
			Message:  "Resource has not been reconciled by Crossplane yet",
		})
		result.HasDrift = true
		result.Severity = DriftSeverityWarning
		return result, nil
	}
	result.ObservedState = statusAtProvider

	// Get policy ARN from status
	policyArn, ok := statusAtProvider["arn"].(string)
	if !ok {
		result.Error = "could not determine AWS policy ARN"
		return result, fmt.Errorf("could not determine AWS policy ARN from status")
	}
	result.ResourceARN = policyArn

	// Fetch actual state from AWS
	awsPolicy, err := dc.awsClient.GetPolicy(ctx, policyArn)
	if err != nil {
		result.Error = fmt.Sprintf("failed to fetch from AWS: %v", err)
		result.Details = append(result.Details, DriftDetail{
			Type:     DriftTypeMissingPrivileges,
			Severity: DriftSeverityCritical,
			Message:  fmt.Sprintf("Policy not found in AWS: %v", err),
		})
		result.HasDrift = true
		result.Severity = DriftSeverityCritical
		return result, nil
	}

	result.ActualState = map[string]interface{}{
		"policyName": *awsPolicy.PolicyName,
		"arn":        *awsPolicy.Arn,
		"path":       *awsPolicy.Path,
	}

	// Get policy document from spec
	desiredPolicyStr, ok := specForProvider["policy"].(string)
	if !ok {
		result.Error = "policy document not found in spec"
		return result, fmt.Errorf("policy document not found in spec")
	}

	// Parse desired policy
	desiredDoc, err := ParsePolicyDocument(desiredPolicyStr)
	if err != nil {
		return result, fmt.Errorf("failed to parse desired policy: %w", err)
	}

	// Fetch actual policy document from AWS
	actualPolicyStr, err := dc.awsClient.GetPolicyVersion(ctx, policyArn, *awsPolicy.DefaultVersionId)
	if err != nil {
		return result, fmt.Errorf("failed to fetch policy version: %w", err)
	}

	// Parse actual policy
	actualDoc, err := ParsePolicyDocument(actualPolicyStr)
	if err != nil {
		return result, fmt.Errorf("failed to parse actual policy: %w", err)
	}

	// Compare policy documents
	if !dc.comparePolicyDocuments(desiredDoc, actualDoc) {
		result.Details = append(result.Details, DriftDetail{
			Type:     DriftTypePolicyMismatch,
			Severity: DriftSeverityHigh,
			Message:  "Policy document differs from spec",
			Path:     "policy",
			Expected: desiredDoc,
			Actual:   actualDoc,
		})
		result.HasDrift = true
		result.Severity = DriftSeverityHigh
	}

	return result, nil
}

// comparePolicyDocuments performs deep comparison of policy documents
func (dc *DriftChecker) comparePolicyDocuments(desired, actual *PolicyDocument) bool {
	// Compare versions
	if desired.Version != actual.Version {
		return false
	}

	// Normalize and compare statements
	if len(desired.Statement) != len(actual.Statement) {
		return false
	}

	// Create maps of statements for comparison
	desiredStmts := dc.normalizeStatements(desired.Statement)
	actualStmts := dc.normalizeStatements(actual.Statement)

	return reflect.DeepEqual(desiredStmts, actualStmts)
}

// normalizeStatements normalizes statements for comparison
func (dc *DriftChecker) normalizeStatements(statements []Statement) []map[string]interface{} {
	normalized := make([]map[string]interface{}, len(statements))

	for i, stmt := range statements {
		n := make(map[string]interface{})

		if stmt.Sid != "" {
			n["Sid"] = stmt.Sid
		}
		n["Effect"] = stmt.Effect

		if stmt.Principal != nil {
			n["Principal"] = dc.normalizeJSON(stmt.Principal)
		}
		if stmt.Action != nil {
			actions := NormalizeActions(stmt.Action)
			sort.Strings(actions)
			n["Action"] = actions
		}
		if stmt.Resource != nil {
			resources := NormalizeResources(stmt.Resource)
			sort.Strings(resources)
			n["Resource"] = resources
		}
		if stmt.Condition != nil {
			n["Condition"] = dc.normalizeJSON(stmt.Condition)
		}

		normalized[i] = n
	}

	return normalized
}

// normalizeJSON normalizes JSON data for comparison
func (dc *DriftChecker) normalizeJSON(data interface{}) interface{} {
	// Re-marshal and unmarshal to normalize structure
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return data
	}

	var normalized interface{}
	if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
		return data
	}

	return normalized
}

// calculateOverallSeverity determines the highest severity from drift details
func (dc *DriftChecker) calculateOverallSeverity(details []DriftDetail) DriftSeverity {
	severity := DriftSeverityNone

	for _, detail := range details {
		if detail.Severity == DriftSeverityCritical {
			return DriftSeverityCritical
		}
		if detail.Severity == DriftSeverityHigh && severity != DriftSeverityCritical {
			severity = DriftSeverityHigh
		}
		if detail.Severity == DriftSeverityWarning && severity == DriftSeverityNone {
			severity = DriftSeverityWarning
		}
	}

	return severity
}
