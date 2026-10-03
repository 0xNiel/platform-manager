/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package builtin

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/0xNiel/platform-manager/internal/rules"
)

func resourceSummary(created time.Time, lastSeen *time.Time) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]interface{}{}}
	u.SetAPIVersion("platform.platform.io/v1alpha1")
	u.SetKind("ResourceSummary")
	u.SetName("tenant-alpha-deployment-tenant-alpha-api")
	u.SetCreationTimestamp(metav1.NewTime(created))
	if lastSeen != nil {
		_ = unstructured.SetNestedField(u.Object, lastSeen.UTC().Format(time.RFC3339), "status", "lastSeen")
	}
	return u
}

func TestStaleResourceRule(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	recently := now.Add(-30 * time.Second)
	longAgo := now.Add(-3 * time.Hour)

	tests := []struct {
		name     string
		resource *unstructured.Unstructured
		want     int
	}{
		{
			// The bug: age was measured from creation, so this was flagged.
			name:     "old summary seen recently is not stale",
			resource: resourceSummary(now.Add(-16*time.Hour), &recently),
			want:     0,
		},
		{
			name:     "summary not seen for hours is stale",
			resource: resourceSummary(now.Add(-16*time.Hour), &longAgo),
			want:     1,
		},
		{
			name:     "old summary never seen is stale",
			resource: resourceSummary(now.Add(-2*time.Hour), nil),
			want:     1,
		},
		{
			name:     "new summary not seen yet is not stale",
			resource: resourceSummary(now.Add(-10*time.Minute), nil),
			want:     0,
		},
	}

	rule := &StaleResourceRule{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !rule.AppliesTo(tt.resource) {
				t.Fatal("rule should apply to ResourceSummary")
			}
			findings, err := rule.Evaluate(rules.RuleContext{Resource: tt.resource, Now: now})
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if len(findings) != tt.want {
				t.Errorf("got %d findings, want %d", len(findings), tt.want)
			}
		})
	}
}
