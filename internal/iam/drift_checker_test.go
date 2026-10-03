/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package iam

import (
	"reflect"
	"testing"
)

func TestDeclaredInlinePolicyNames(t *testing.T) {
	tests := []struct {
		name string
		spec map[string]interface{}
		want map[string]bool
	}{
		{
			name: "no inlinePolicy field",
			spec: map[string]interface{}{},
			want: map[string]bool{},
		},
		{
			name: "declared policies",
			spec: map[string]interface{}{
				"inlinePolicy": []interface{}{
					map[string]interface{}{"name": "s3-read", "policy": "{}"},
					map[string]interface{}{"name": "logs-write", "policy": "{}"},
				},
			},
			want: map[string]bool{"s3-read": true, "logs-write": true},
		},
		{
			name: "skips malformed and unnamed entries",
			spec: map[string]interface{}{
				"inlinePolicy": []interface{}{
					"not-a-map",
					map[string]interface{}{"policy": "{}"},
					map[string]interface{}{"name": ""},
					map[string]interface{}{"name": "kept"},
				},
			},
			want: map[string]bool{"kept": true},
		},
		{
			name: "wrong type for inlinePolicy",
			spec: map[string]interface{}{"inlinePolicy": "oops"},
			want: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := declaredInlinePolicyNames(tt.spec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("declaredInlinePolicyNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeclaredManagedPolicyArns(t *testing.T) {
	const readOnly = "arn:aws:iam::aws:policy/ReadOnlyAccess"

	tests := []struct {
		name string
		spec map[string]interface{}
		want map[string]bool
	}{
		{
			name: "no managedPolicyArns field",
			spec: map[string]interface{}{},
			want: map[string]bool{},
		},
		{
			name: "declared ARNs, skipping empty and non-string values",
			spec: map[string]interface{}{
				"managedPolicyArns": []interface{}{readOnly, "", 42},
			},
			want: map[string]bool{readOnly: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := declaredManagedPolicyArns(tt.spec); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("declaredManagedPolicyArns() = %v, want %v", got, tt.want)
			}
		})
	}
}
