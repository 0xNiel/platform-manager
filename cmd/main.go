/*
Copyright (c) 2025 0xNiel

SPDX-License-Identifier: MIT
*/

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"os"
	"path/filepath"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	platformv1alpha1 "github.com/0xNiel/platform-manager/api/v1alpha1"
	"github.com/0xNiel/platform-manager/internal/api"
	"github.com/0xNiel/platform-manager/internal/controller"
	"github.com/0xNiel/platform-manager/internal/metrics"
	"github.com/0xNiel/platform-manager/internal/terminal"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(platformv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// nolint:gocyclo
func main() {
	var metricsAddr string
	var metricsCertPath, metricsCertName, metricsCertKey string
	var webhookCertPath, webhookCertName, webhookCertKey string
	var enableLeaderElection bool
	var probeAddr string
	var apiAddr string
	var prometheusURL string
	var secureMetrics bool
	var enableHTTP2 bool
	var iamDriftScanInterval string
	var ruleEvalInterval string
	var enableTerminal bool
	var terminalNamespace string
	var terminalImage string
	var terminalIdleTimeout string
	var tlsOpts []func(*tls.Config)
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&apiAddr, "api-bind-address", ":9080", "The address the API server binds to.")
	flag.StringVar(&prometheusURL, "prometheus-url",
		"http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090", "The URL of the Prometheus server.")
	flag.StringVar(&iamDriftScanInterval, "iam-drift-scan-interval", "5m",
		"The interval for IAM drift scanning (e.g., 5m, 1h)")
	flag.StringVar(&ruleEvalInterval, "rule-eval-interval", "2m", "The interval for rule evaluation (e.g., 2m, 5m, 10m)")
	flag.BoolVar(&enableTerminal, "enable-terminal", false, "Enable web terminal feature")
	flag.StringVar(&terminalNamespace, "terminal-namespace", "toolbox-sessions", "Namespace for terminal toolbox pods")
	flag.StringVar(&terminalImage, "terminal-image", "platform-manager-toolbox:latest", "Toolbox container image")
	flag.StringVar(&terminalIdleTimeout, "terminal-idle-timeout", "10m", "Terminal session idle timeout (e.g., 10m, 30m)")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.StringVar(&webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
	flag.StringVar(&webhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
	flag.StringVar(&webhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
	flag.StringVar(&metricsCertPath, "metrics-cert-path", "",
		"The directory that contains the metrics server certificate.")
	flag.StringVar(&metricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
	flag.StringVar(&metricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	// Create watchers for metrics and webhooks certificates
	var metricsCertWatcher, webhookCertWatcher *certwatcher.CertWatcher

	// Initial webhook TLS options
	webhookTLSOpts := tlsOpts

	if len(webhookCertPath) > 0 {
		setupLog.Info("Initializing webhook certificate watcher using provided certificates",
			"webhook-cert-path", webhookCertPath, "webhook-cert-name", webhookCertName, "webhook-cert-key", webhookCertKey)

		var err error
		webhookCertWatcher, err = certwatcher.New(
			filepath.Join(webhookCertPath, webhookCertName),
			filepath.Join(webhookCertPath, webhookCertKey),
		)
		if err != nil {
			setupLog.Error(err, "Failed to initialize webhook certificate watcher")
			os.Exit(1)
		}

		webhookTLSOpts = append(webhookTLSOpts, func(config *tls.Config) {
			config.GetCertificate = webhookCertWatcher.GetCertificate
		})
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: webhookTLSOpts,
	})

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	// If the certificate is not specified, controller-runtime will automatically
	// generate self-signed certificates for the metrics server. While convenient for development and testing,
	// this setup is not recommended for production.
	//
	// TODO(user): If you enable certManager, uncomment the following lines:
	// - [METRICS-WITH-CERTS] at config/default/kustomization.yaml to generate and use certificates
	// managed by cert-manager for the metrics server.
	// - [PROMETHEUS-WITH-CERTS] at config/prometheus/kustomization.yaml for TLS certification.
	if len(metricsCertPath) > 0 {
		setupLog.Info("Initializing metrics certificate watcher using provided certificates",
			"metrics-cert-path", metricsCertPath, "metrics-cert-name", metricsCertName, "metrics-cert-key", metricsCertKey)

		var err error
		metricsCertWatcher, err = certwatcher.New(
			filepath.Join(metricsCertPath, metricsCertName),
			filepath.Join(metricsCertPath, metricsCertKey),
		)
		if err != nil {
			setupLog.Error(err, "to initialize metrics certificate watcher", "error", err)
			os.Exit(1)
		}

		metricsServerOptions.TLSOpts = append(metricsServerOptions.TLSOpts, func(config *tls.Config) {
			config.GetCertificate = metricsCertWatcher.GetCertificate
		})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "ff8281bd.platform.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Create health aggregator
	healthAggregator := controller.NewHealthAggregator(mgr.GetClient())

	// Create resource scanner
	resourceScanner := controller.NewResourceScanner(mgr.GetClient())

	// Initialize IAM drift scanner (before tenant reconciler)
	scanInterval, err := time.ParseDuration(iamDriftScanInterval)
	if err != nil {
		setupLog.Error(err, "invalid IAM drift scan interval", "interval", iamDriftScanInterval)
		os.Exit(1)
	}

	iamScanner, err := controller.NewIAMDriftScanner(mgr.GetClient(), scanInterval)
	if err != nil {
		setupLog.Info("Failed to initialize IAM drift scanner, IAM drift detection will be unavailable", "error", err)
	} else {
		setupLog.Info("IAM drift scanner initialized", "interval", scanInterval)
		// Connect scanner to health aggregator
		healthAggregator.SetIAMScanner(iamScanner)
		if err := iamScanner.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to add IAM drift scanner to manager")
			os.Exit(1)
		}
	}

	// Initialize Rule Evaluator for troubleshooting
	evalInterval, err := time.ParseDuration(ruleEvalInterval)
	if err != nil {
		setupLog.Error(err, "invalid rule evaluation interval", "interval", ruleEvalInterval)
		os.Exit(1)
	}

	ruleEvaluator := controller.NewRuleEvaluator(mgr, evalInterval)
	setupLog.Info("Rule evaluator initialized",
		"interval", evalInterval, "rules", len(ruleEvaluator.GetEngine().GetRules()))
	if err := mgr.Add(ruleEvaluator); err != nil {
		setupLog.Error(err, "unable to add rule evaluator to manager")
		os.Exit(1)
	}

	if err := (&controller.TenantReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		HealthAggregator: healthAggregator,
		ResourceScanner:  resourceScanner,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Tenant")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	// Initialize Prometheus client (optional)
	var prometheusClient *metrics.PrometheusClient
	if prometheusURL != "" {
		var err error
		prometheusClient, err = metrics.NewPrometheusClient(prometheusURL)
		if err != nil {
			setupLog.Info("Failed to initialize Prometheus client, metrics will be unavailable", "error", err)
		} else {
			setupLog.Info("Prometheus client initialized", "url", prometheusURL)
		}
	} else {
		setupLog.Info("Prometheus URL not provided, metrics will be unavailable")
	}

	// Initialize terminal manager if enabled
	var terminalMgr *terminal.Manager
	if enableTerminal {
		idleTimeout, err := time.ParseDuration(terminalIdleTimeout)
		if err != nil {
			setupLog.Error(err, "invalid terminal idle timeout", "timeout", terminalIdleTimeout)
			os.Exit(1)
		}

		terminalConfig := terminal.Config{
			Enabled:        true,
			Namespace:      terminalNamespace,
			ToolboxImage:   terminalImage,
			IdleTimeout:    idleTimeout,
			MaxSessions:    20,
			AllowedTenants: []string{}, // Empty = all tenants allowed
			ServiceAccount: "toolbox-session",
		}

		terminalMgr = terminal.NewManager(mgr.GetClient(), terminalConfig, ctrl.Log.WithName("terminal"))
		setupLog.Info("Terminal feature enabled",
			"namespace", terminalNamespace,
			"image", terminalImage,
			"idleTimeout", idleTimeout)
	} else {
		setupLog.Info("Terminal feature disabled")
	}

	// Add the HTTP API server as a runnable (iamScanner and ruleEvaluator initialized earlier)
	apiServer := api.NewServer(apiAddr, mgr.GetClient(), prometheusClient, iamScanner, ruleEvaluator, terminalMgr)
	if err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		return apiServer.Start(ctx)
	})); err != nil {
		setupLog.Error(err, "unable to add API server to manager")
		os.Exit(1)
	}

	if metricsCertWatcher != nil {
		setupLog.Info("Adding metrics certificate watcher to manager")
		if err := mgr.Add(metricsCertWatcher); err != nil {
			setupLog.Error(err, "unable to add metrics certificate watcher to manager")
			os.Exit(1)
		}
	}

	if webhookCertWatcher != nil {
		setupLog.Info("Adding webhook certificate watcher to manager")
		if err := mgr.Add(webhookCertWatcher); err != nil {
			setupLog.Error(err, "unable to add webhook certificate watcher to manager")
			os.Exit(1)
		}
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
