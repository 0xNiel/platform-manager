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

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// ActionMetrics tracks Platform Manager action operations
type ActionMetrics struct {
	actionsTotal   *prometheus.CounterVec
	authDenials    *prometheus.CounterVec
	actionDuration *prometheus.HistogramVec
}

var (
	// ActionsMetrics is the global metrics instance
	ActionsMetrics *ActionMetrics
)

func init() {
	ActionsMetrics = NewActionMetrics()
	ActionsMetrics.Register()
}

// NewActionMetrics creates a new ActionMetrics instance
func NewActionMetrics() *ActionMetrics {
	return &ActionMetrics{
		actionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "platform_manager_actions_total",
				Help: "Total number of action operations performed",
			},
			[]string{"action", "status", "user_role"},
		),
		authDenials: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "platform_manager_auth_denials_total",
				Help: "Total number of authorization denials",
			},
			[]string{"action", "user_role"},
		),
		actionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "platform_manager_action_duration_seconds",
				Help:    "Duration of action operations in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"action"},
		),
	}
}

// Register registers metrics with Prometheus
func (m *ActionMetrics) Register() {
	metrics.Registry.MustRegister(
		m.actionsTotal,
		m.authDenials,
		m.actionDuration,
	)
}

// RecordAction records an action execution
func (m *ActionMetrics) RecordAction(action string, userRole string, success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "failure"
	}

	m.actionsTotal.With(prometheus.Labels{
		"action":    action,
		"status":    status,
		"user_role": userRole,
	}).Inc()

	m.actionDuration.With(prometheus.Labels{
		"action": action,
	}).Observe(duration.Seconds())
}

// RecordAuthDenial records an authorization denial
func (m *ActionMetrics) RecordAuthDenial(action string, userRole string) {
	m.authDenials.With(prometheus.Labels{
		"action":    action,
		"user_role": userRole,
	}).Inc()
}

// Action name constants for consistency
const (
	ActionArgoSync            = "argo_sync"
	ActionArgoRefresh         = "argo_refresh"
	ActionCrossplanePause     = "crossplane_pause"
	ActionCrossplaneUnpause   = "crossplane_unpause"
	ActionCrossplaneReconcile = "crossplane_reconcile"
	ActionResourceDelete      = "resource_delete"
)
