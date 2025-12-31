/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
)

// mockLogger implements logr.Logger for testing
type mockLogger struct {
	infoCalls  []mockLogEntry
	errorCalls []mockLogEntry
}

type mockLogEntry struct {
	msg           string
	keysAndValues []interface{}
}

func (m *mockLogger) Init(info logr.RuntimeInfo) {}

func (m *mockLogger) Enabled(level int) bool {
	return true
}

func (m *mockLogger) Info(level int, msg string, keysAndValues ...interface{}) {
	m.infoCalls = append(m.infoCalls, mockLogEntry{
		msg:           msg,
		keysAndValues: keysAndValues,
	})
}

func (m *mockLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	m.errorCalls = append(m.errorCalls, mockLogEntry{
		msg:           msg,
		keysAndValues: keysAndValues,
	})
}

func (m *mockLogger) WithValues(keysAndValues ...interface{}) logr.LogSink {
	return m
}

func (m *mockLogger) WithName(name string) logr.LogSink {
	return m
}

func TestDefaultAuditLogger_LogAction_Success(t *testing.T) {
	mock := &mockLogger{}
	logger := NewAuditLoggerWithLogger(logr.New(mock))

	ctx := context.WithValue(context.Background(), userInfoKey, UserInfo{
		Username: "test-user",
		Role:     RoleAdmin,
	})

	logger.LogAction(ctx, "sync", "alpha-ml-platform", true, nil)

	// Should log to info
	if len(mock.infoCalls) != 1 {
		t.Errorf("Expected 1 info call, got %d", len(mock.infoCalls))
	}

	if len(mock.errorCalls) != 0 {
		t.Errorf("Expected 0 error calls, got %d", len(mock.errorCalls))
	}

	// Check log entry contains expected fields
	entry := mock.infoCalls[0]
	if entry.msg != "Audit: Action performed" {
		t.Errorf("Expected message 'Audit: Action performed', got '%s'", entry.msg)
	}

	// Check key-value pairs
	kvMap := kvToMap(entry.keysAndValues)
	if kvMap["user"] != "test-user" {
		t.Errorf("Expected user 'test-user', got '%v'", kvMap["user"])
	}
	if kvMap["userRole"] != "admin" {
		t.Errorf("Expected userRole 'admin', got '%v'", kvMap["userRole"])
	}
	if kvMap["action"] != "sync" {
		t.Errorf("Expected action 'sync', got '%v'", kvMap["action"])
	}
	if kvMap["resource"] != "alpha-ml-platform" {
		t.Errorf("Expected resource 'alpha-ml-platform', got '%v'", kvMap["resource"])
	}
	if kvMap["success"] != true {
		t.Errorf("Expected success true, got '%v'", kvMap["success"])
	}
}

func TestDefaultAuditLogger_LogAction_Failure(t *testing.T) {
	mock := &mockLogger{}
	logger := NewAuditLoggerWithLogger(logr.New(mock))

	ctx := context.WithValue(context.Background(), userInfoKey, UserInfo{
		Username: "test-user",
		Role:     RoleInfra,
	})

	testError := errors.New("application not found")
	logger.LogAction(ctx, "sync", "missing-app", false, testError)

	// Should log to error
	if len(mock.errorCalls) != 1 {
		t.Errorf("Expected 1 error call, got %d", len(mock.errorCalls))
	}

	if len(mock.infoCalls) != 0 {
		t.Errorf("Expected 0 info calls, got %d", len(mock.infoCalls))
	}

	// Check log entry
	entry := mock.errorCalls[0]
	if entry.msg != "Audit: Action failed" {
		t.Errorf("Expected message 'Audit: Action failed', got '%s'", entry.msg)
	}

	kvMap := kvToMap(entry.keysAndValues)
	if kvMap["error"] != "application not found" {
		t.Errorf("Expected error 'application not found', got '%v'", kvMap["error"])
	}
	if kvMap["success"] != false {
		t.Errorf("Expected success false, got '%v'", kvMap["success"])
	}
}

func TestDefaultAuditLogger_LogActionWithDetails(t *testing.T) {
	mock := &mockLogger{}
	logger := NewAuditLoggerWithLogger(logr.New(mock))

	ctx := context.WithValue(context.Background(), userInfoKey, UserInfo{
		Username: "infra-user",
		Role:     RoleInfra,
	})

	logger.LogActionWithDetails(ctx, "pause", "tenant-alpha-lambda-role", "iam.aws.upbound.io/v1beta1/Role", true, nil)

	if len(mock.infoCalls) != 1 {
		t.Fatalf("Expected 1 info call, got %d", len(mock.infoCalls))
	}

	kvMap := kvToMap(mock.infoCalls[0].keysAndValues)
	if kvMap["resourceGVK"] != "iam.aws.upbound.io/v1beta1/Role" {
		t.Errorf("Expected resourceGVK 'iam.aws.upbound.io/v1beta1/Role', got '%v'", kvMap["resourceGVK"])
	}
}

func TestDefaultAuditLogger_WithRequestID(t *testing.T) {
	mock := &mockLogger{}
	logger := NewAuditLoggerWithLogger(logr.New(mock))

	ctx := context.WithValue(context.Background(), userInfoKey, UserInfo{
		Username: "test-user",
		Role:     RoleAdmin,
	})
	ctx = context.WithValue(ctx, "requestID", "req-12345")

	logger.LogAction(ctx, "delete", "test-deployment", true, nil)

	if len(mock.infoCalls) != 1 {
		t.Fatalf("Expected 1 info call, got %d", len(mock.infoCalls))
	}

	kvMap := kvToMap(mock.infoCalls[0].keysAndValues)
	if kvMap["requestID"] != "req-12345" {
		t.Errorf("Expected requestID 'req-12345', got '%v'", kvMap["requestID"])
	}
}

func TestDefaultAuditLogger_WithoutUser(t *testing.T) {
	mock := &mockLogger{}
	logger := NewAuditLoggerWithLogger(logr.New(mock))

	// Context without user (should default to unknown/readonly)
	ctx := context.Background()

	logger.LogAction(ctx, "sync", "app", true, nil)

	if len(mock.infoCalls) != 1 {
		t.Fatalf("Expected 1 info call, got %d", len(mock.infoCalls))
	}

	kvMap := kvToMap(mock.infoCalls[0].keysAndValues)
	if kvMap["user"] != "unknown" {
		t.Errorf("Expected user 'unknown', got '%v'", kvMap["user"])
	}
	if kvMap["userRole"] != "readonly" {
		t.Errorf("Expected userRole 'readonly', got '%v'", kvMap["userRole"])
	}
}

func TestNoOpAuditLogger(t *testing.T) {
	logger := NewNoOpAuditLogger()

	ctx := context.WithValue(context.Background(), userInfoKey, UserInfo{
		Username: "test-user",
		Role:     RoleAdmin,
	})

	// Should not panic
	logger.LogAction(ctx, "sync", "app", true, nil)
	logger.LogActionWithDetails(ctx, "pause", "resource", "gvk", false, errors.New("test"))
}

func TestNewDefaultAuditLogger(t *testing.T) {
	logger := NewDefaultAuditLogger()
	if logger == nil {
		t.Error("NewDefaultAuditLogger returned nil")
	}

	// Should be usable without panic
	ctx := context.Background()
	logger.LogAction(ctx, "test", "resource", true, nil)
}

// Helper function to convert keysAndValues slice to map
func kvToMap(keysAndValues []interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if ok {
			m[key] = keysAndValues[i+1]
		}
	}
	return m
}
