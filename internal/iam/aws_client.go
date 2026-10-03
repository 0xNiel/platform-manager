/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/go-logr/logr"
)

// AWSClient is a wrapper around the AWS IAM SDK client
type AWSClient struct {
	client *iam.Client
	logger logr.Logger
}

// NewAWSClient creates a new AWS IAM client
// For LocalStack, provide endpoint and static credentials
func NewAWSClient(ctx context.Context, logger logr.Logger, endpoint string, region string) (*AWSClient, error) {
	var cfg aws.Config
	var err error

	if endpoint != "" {
		// LocalStack mode
		logger.Info("Creating AWS IAM client for LocalStack", "endpoint", endpoint, "region", region)

		// Parse endpoint URL
		endpointURL, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("invalid endpoint URL: %w", err)
		}

		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}

		// Create IAM client with custom endpoint
		client := iam.NewFromConfig(cfg, func(o *iam.Options) {
			o.BaseEndpoint = aws.String(endpointURL.String())
		})

		return &AWSClient{
			client: client,
			logger: logger,
		}, nil
	}

	// Real AWS mode (using IRSA or instance profile)
	logger.Info("Creating AWS IAM client for real AWS", "region", region)
	cfg, err = config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSClient{
		client: iam.NewFromConfig(cfg),
		logger: logger,
	}, nil
}

// GetRole retrieves a role from AWS
func (c *AWSClient) GetRole(ctx context.Context, roleName string) (*iamtypes.Role, error) {
	input := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	output, err := c.client.GetRole(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get role %s: %w", roleName, err)
	}

	return output.Role, nil
}

// GetRolePolicy retrieves an inline policy for a role
func (c *AWSClient) GetRolePolicy(ctx context.Context, roleName, policyName string) (string, error) {
	input := &iam.GetRolePolicyInput{
		RoleName:   aws.String(roleName),
		PolicyName: aws.String(policyName),
	}

	output, err := c.client.GetRolePolicy(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get role policy %s/%s: %w", roleName, policyName, err)
	}

	// Decode URL-encoded policy document
	policyDoc, err := url.QueryUnescape(*output.PolicyDocument)
	if err != nil {
		return "", fmt.Errorf("failed to decode policy document: %w", err)
	}

	return policyDoc, nil
}

// ListRolePolicies lists inline policies for a role
func (c *AWSClient) ListRolePolicies(ctx context.Context, roleName string) ([]string, error) {
	input := &iam.ListRolePoliciesInput{
		RoleName: aws.String(roleName),
	}

	output, err := c.client.ListRolePolicies(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list role policies for %s: %w", roleName, err)
	}

	return output.PolicyNames, nil
}

// GetPolicy retrieves a managed policy
func (c *AWSClient) GetPolicy(ctx context.Context, policyArn string) (*iamtypes.Policy, error) {
	input := &iam.GetPolicyInput{
		PolicyArn: aws.String(policyArn),
	}

	output, err := c.client.GetPolicy(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy %s: %w", policyArn, err)
	}

	return output.Policy, nil
}

// GetPolicyVersion retrieves a specific version of a policy
func (c *AWSClient) GetPolicyVersion(ctx context.Context, policyArn, versionId string) (string, error) {
	input := &iam.GetPolicyVersionInput{
		PolicyArn: aws.String(policyArn),
		VersionId: aws.String(versionId),
	}

	output, err := c.client.GetPolicyVersion(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get policy version %s/%s: %w", policyArn, versionId, err)
	}

	// Decode URL-encoded policy document
	policyDoc, err := url.QueryUnescape(*output.PolicyVersion.Document)
	if err != nil {
		return "", fmt.Errorf("failed to decode policy document: %w", err)
	}

	return policyDoc, nil
}

// ListAttachedRolePolicies lists managed policies attached to a role
func (c *AWSClient) ListAttachedRolePolicies(ctx context.Context, roleName string) ([]iamtypes.AttachedPolicy, error) {
	input := &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(roleName),
	}

	output, err := c.client.ListAttachedRolePolicies(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list attached policies for %s: %w", roleName, err)
	}

	return output.AttachedPolicies, nil
}

// ListRoles lists all IAM roles (for detecting orphaned resources)
func (c *AWSClient) ListRoles(ctx context.Context) ([]iamtypes.Role, error) {
	var roles []iamtypes.Role
	paginator := iam.NewListRolesPaginator(c.client, &iam.ListRolesInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list roles: %w", err)
		}
		roles = append(roles, output.Roles...)
	}

	return roles, nil
}

// ListPolicies lists all IAM policies (for detecting orphaned resources)
func (c *AWSClient) ListPolicies(ctx context.Context, scope string) ([]iamtypes.Policy, error) {
	var policies []iamtypes.Policy
	var scopeType iamtypes.PolicyScopeType

	switch scope {
	case "Local":
		scopeType = iamtypes.PolicyScopeTypeLocal
	case "AWS":
		scopeType = iamtypes.PolicyScopeTypeAws
	default:
		scopeType = iamtypes.PolicyScopeTypeAll
	}

	paginator := iam.NewListPoliciesPaginator(c.client, &iam.ListPoliciesInput{
		Scope: scopeType,
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list policies: %w", err)
		}
		policies = append(policies, output.Policies...)
	}

	return policies, nil
}

// DecodeAssumeRolePolicy decodes a URL-encoded assume role policy document
func (c *AWSClient) DecodeAssumeRolePolicy(encoded string) (*PolicyDocument, error) {
	decoded, err := url.QueryUnescape(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode assume role policy: %w", err)
	}

	var doc PolicyDocument
	if err := json.Unmarshal([]byte(decoded), &doc); err != nil {
		return nil, fmt.Errorf("failed to parse assume role policy: %w", err)
	}

	return &doc, nil
}
