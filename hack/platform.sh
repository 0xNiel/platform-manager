#!/usr/bin/env bash
# Bring the full local demo up or down with one command.
#
#   hack/platform.sh up      kind + LocalStack + ArgoCD + Crossplane + Prometheus,
#                            seeded tenants (healthy and broken), then the API and UI
#   hack/platform.sh stop    stop the API, UI, and Prometheus port-forward only
#   hack/platform.sh down    stop everything and delete the cluster and LocalStack
#   hack/platform.sh status  show what is running
#
# Every kubectl and helm call uses a kubeconfig for the kind cluster only, so
# your current kubectl context is never used or changed.
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
CLUSTER=${KIND_CLUSTER_NAME:-platform-manager}
API_PORT=${API_PORT:-9080}
UI_PORT=${UI_PORT:-9082}
PROBE_PORT=${PROBE_PORT:-8082}
PROM_PORT=${PROM_PORT:-9091}
LOCALSTACK_IMAGE=${LOCALSTACK_IMAGE:-localstack/localstack:4.12.0}
LOCALSTACK_CONTAINER=localstack-main
CROSSPLANE_VERSION=${CROSSPLANE_VERSION:-1.20.0}
TOOLBOX_IMG=${TOOLBOX_IMG:-platform-manager-toolbox:dev}
ARGOCD_MANIFEST=https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

STATE="$ROOT/.platform"
KCFG="$STATE/kubeconfig"

k() { kubectl --kubeconfig "$KCFG" "$@"; }
h() { helm --kubeconfig "$KCFG" "$@"; }
awslocal() { docker exec "$LOCALSTACK_CONTAINER" awslocal "$@"; }

step() { printf '\n\033[1;34m==>\033[0m \033[1m%s\033[0m\n' "$*"; }
info() { printf '    %s\n' "$*"; }
die() { printf '\n\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

port_busy() { lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1; }

# wait_for <description> <timeout seconds> <command...>
wait_for() {
	local desc=$1 timeout=$2
	shift 2
	local end=$((SECONDS + timeout))
	until "$@" >/dev/null 2>&1; do
		[ "$SECONDS" -ge "$end" ] && die "timed out after ${timeout}s waiting for $desc"
		sleep 3
	done
}

http_ok() { curl -sf -o /dev/null "$1"; }
localstack_iam_ready() { curl -s localhost:4566/_localstack/health | grep -Eq '"iam": "(available|running)"'; }
provider_crds_ready() {
	k get crd roles.iam.aws.upbound.io buckets.s3.aws.upbound.io \
		tables.dynamodb.aws.upbound.io providerconfigs.aws.upbound.io
}
seeded_roles_in_aws() {
	awslocal iam get-role --role-name tenant-alpha-lambda-role &&
		awslocal iam get-role --role-name tenant-beta-data-role
}

# start_bg <name> <command...>: run in the background, log to .platform/<name>.log
start_bg() {
	local name=$1
	shift
	stop_bg "$name"
	nohup "$@" >"$STATE/$name.log" 2>&1 &
	echo $! >"$STATE/$name.pid"
}

stop_bg() {
	local pidfile="$STATE/$1.pid"
	[ -f "$pidfile" ] || return 0
	local pid
	pid=$(cat "$pidfile")
	if kill -0 "$pid" 2>/dev/null; then
		kill "$pid" 2>/dev/null || true
		local end=$((SECONDS + 10))
		while kill -0 "$pid" 2>/dev/null && [ "$SECONDS" -lt "$end" ]; do sleep 1; done
		kill -9 "$pid" 2>/dev/null || true
	fi
	rm -f "$pidfile"
}

running() {
	local pidfile="$STATE/$1.pid"
	[ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null
}

cluster_exists() { kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; }

# ---------------------------------------------------------------------------

preflight() {
	step "Checking prerequisites"
	local missing=()
	for cmd in docker kind kubectl helm go npm curl lsof; do
		command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
	done
	[ ${#missing[@]} -eq 0 ] || die "missing tools: ${missing[*]}"
	docker info >/dev/null 2>&1 || die "Docker is not running"
	mkdir -p "$STATE"

	# Free our own ports from a previous run before checking them.
	stop_app
	for port in "$API_PORT" "$UI_PORT" "$PROBE_PORT" "$PROM_PORT"; do
		if port_busy "$port"; then
			local owner
			owner=$(lsof -nP -iTCP:"$port" -sTCP:LISTEN | awk 'NR==2 {print $1}')
			die "port $port is in use by '$owner'. Free it, or pick other ports, e.g.
       make platform-up API_PORT=9090 UI_PORT=9083
       (A kind cluster created from an older kind-config.yaml maps 9080 and 9082.
        'make platform-down' deletes it so the next 'make platform-up' starts clean.)"
		fi
	done
	info "ok"
}

create_cluster() {
	step "kind cluster '$CLUSTER'"
	if cluster_exists; then
		info "already exists, reusing it"
	else
		# --kubeconfig keeps kind from adding a context to ~/.kube/config and switching to it.
		kind create cluster --name "$CLUSTER" --config "$ROOT/hack/kind-config.yaml" --kubeconfig "$KCFG"
	fi
	kind get kubeconfig --name "$CLUSTER" >"$KCFG"
	wait_for "the cluster API" 120 k get nodes
}

start_localstack() {
	step "LocalStack ($LOCALSTACK_IMAGE)"
	local current
	current=$(docker inspect -f '{{.Config.Image}} {{.State.Running}}' "$LOCALSTACK_CONTAINER" 2>/dev/null || true)
	if [ "$current" = "$LOCALSTACK_IMAGE true" ]; then
		info "already running"
	else
		docker rm -f "$LOCALSTACK_CONTAINER" >/dev/null 2>&1 || true
		docker run -d --name "$LOCALSTACK_CONTAINER" -p 4566:4566 -p 4510-4559:4510-4559 "$LOCALSTACK_IMAGE" >/dev/null
	fi
	wait_for "LocalStack IAM" 180 localstack_iam_ready
	info "ready on http://localhost:4566"
}

install_argocd() {
	step "ArgoCD"
	k create namespace argocd --dry-run=client -o yaml | k apply -f - >/dev/null
	# Server-side apply: the ArgoCD CRDs are too large for client-side apply annotations.
	k apply -n argocd --server-side --force-conflicts -f "$ARGOCD_MANIFEST" >/dev/null
	k -n argocd rollout status deploy/argocd-server --timeout=300s >/dev/null
	k -n argocd rollout status deploy/argocd-repo-server --timeout=300s >/dev/null
	k -n argocd rollout status statefulset/argocd-application-controller --timeout=300s >/dev/null
	info "ready"
}

install_crossplane() {
	step "Crossplane $CROSSPLANE_VERSION and AWS providers"
	h repo add crossplane-stable https://charts.crossplane.io/stable >/dev/null 2>&1 || true
	h repo update crossplane-stable >/dev/null
	h upgrade --install crossplane crossplane-stable/crossplane \
		--namespace crossplane-system --create-namespace \
		--version "$CROSSPLANE_VERSION" --wait --timeout 10m >/dev/null
	k apply -f "$ROOT/hack/crossplane/provider-aws.yaml" >/dev/null
	info "waiting for providers to become healthy (first run pulls images, a few minutes)"
	wait_for "Crossplane provider CRDs" 900 provider_crds_ready
	k wait --for=condition=Healthy provider.pkg.crossplane.io --all --timeout=900s >/dev/null
	k apply -f "$ROOT/hack/crossplane/providerconfig-localstack.yaml" >/dev/null
	info "ready"
}

install_prometheus() {
	step "Prometheus (CPU and memory metrics)"
	h repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null 2>&1 || true
	h repo update prometheus-community >/dev/null
	h upgrade --install prometheus prometheus-community/prometheus \
		--namespace monitoring --create-namespace \
		-f "$ROOT/hack/prometheus-values.yaml" --wait --timeout 10m >/dev/null
	info "ready"
}

seed() {
	step "Platform Manager CRDs and tenants"
	k apply -k "$ROOT/config/crd" >/dev/null
	wait_for "Tenant CRD" 60 k get crd tenants.platform.platform.io
	k apply -f "$ROOT/hack/seed-tenants/namespaces.yaml" >/dev/null
	k apply -f "$ROOT/config/samples/tenants.yaml" >/dev/null
	info "tenants: alpha, beta, gamma"

	step "Workloads (healthy and broken)"
	k apply -f "$ROOT/hack/seed-tenants/tenant-alpha.yaml" -f "$ROOT/hack/seed-tenants/tenant-beta.yaml" >/dev/null
	# A bad image, a crash-looping pod, and a deployment that can't be scheduled.
	k apply -f "$ROOT/hack/seed-tenants/failed-resources.yaml" >/dev/null
	info "nginx services, ImagePullBackOff, CrashLoopBackOff, Pending"

	step "ArgoCD applications"
	k apply -f "$ROOT/hack/seed-tenants/argo-apps.yaml" >/dev/null
	info "alpha syncs automatically; beta stays OutOfSync until you press Sync"

	step "Crossplane resources (IAM, S3, DynamoDB, MLPlatform claim)"
	k apply -f "$ROOT/hack/seed-tenants/iam-resources.yaml" -f "$ROOT/hack/crossplane/additional-resources.yaml" >/dev/null
	# One resource that can never reconcile (its AWS account is gone).
	k apply -f "$ROOT/hack/seed-tenants/failed-crossplane-resources.yaml" >/dev/null
	# The claim needs the XRD's CRDs, which appear a few seconds after the XRD.
	k apply -f "$ROOT/hack/crossplane/composite-resources.yaml" >/dev/null 2>&1 || true
	wait_for "MLPlatform claim CRD" 120 k get crd mlplatformclaims.platform.io
	k apply -f "$ROOT/hack/crossplane/composite-resources.yaml" >/dev/null

	info "waiting for Crossplane to create the IAM roles in LocalStack"
	wait_for "IAM roles in LocalStack" 600 seeded_roles_in_aws

	step "Drift and paused resources"
	# Changes made directly in AWS, behind Crossplane's back. Crossplane doesn't
	# manage policies the spec leaves out, so these stay and the scanner flags them.
	awslocal iam put-role-policy --role-name tenant-alpha-lambda-role \
		--policy-name unauthorized-s3-full-access \
		--policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:*","Resource":"*"}]}' >/dev/null
	awslocal iam attach-role-policy --role-name tenant-beta-data-role \
		--policy-arn arn:aws:iam::aws:policy/AdministratorAccess >/dev/null
	info "tenant-alpha-lambda-role: hand-added inline s3:* policy"
	info "tenant-beta-data-role: hand-attached AdministratorAccess"
	k annotate --overwrite bucket.s3.aws.upbound.io tenant-beta-analytics crossplane.io/paused=true >/dev/null
	info "paused bucket tenant-beta-analytics"
}

setup_terminal() {
	step "Web terminal (toolbox image and RBAC)"
	if ! docker image inspect "$TOOLBOX_IMG" >/dev/null 2>&1; then
		info "building $TOOLBOX_IMG (first run only)"
		docker build -q -t "$TOOLBOX_IMG" -f "$ROOT/Dockerfile.toolbox" "$ROOT" >/dev/null
	fi
	if ! docker exec "$CLUSTER-control-plane" crictl images 2>/dev/null | grep -q "${TOOLBOX_IMG%%:*}"; then
		kind load docker-image "$TOOLBOX_IMG" --name "$CLUSTER" >/dev/null
	fi
	k apply -k "$ROOT/config/terminal" >/dev/null
	info "ready"
}

start_app() {
	step "Platform Manager API and UI"
	(cd "$ROOT" && go build -o bin/manager ./cmd/main.go)
	start_bg prometheus-port-forward kubectl --kubeconfig "$KCFG" -n monitoring \
		port-forward svc/prometheus-server "$PROM_PORT:80"
	start_bg manager env KUBECONFIG="$KCFG" DEV_MODE=true AWS_ENDPOINT=http://localhost:4566 \
		"$ROOT/bin/manager" \
		--api-bind-address=":$API_PORT" \
		--health-probe-bind-address=":$PROBE_PORT" \
		--prometheus-url="http://localhost:$PROM_PORT" \
		--enable-terminal --terminal-image="$TOOLBOX_IMG" \
		--iam-drift-scan-interval=1m --rule-eval-interval=1m
	wait_for "the API on :$API_PORT (see .platform/manager.log)" 120 http_ok "http://localhost:$API_PORT/healthz"

	[ -d "$ROOT/web/node_modules" ] || (cd "$ROOT/web" && npm ci --no-audit --no-fund)
	start_bg web bash -c "cd '$ROOT/web' && VUE_APP_API_URL='http://localhost:$API_PORT/api/v1' \
		exec ./node_modules/.bin/vue-cli-service serve --port $UI_PORT"
	wait_for "the UI on :$UI_PORT (see .platform/web.log)" 300 http_ok "http://localhost:$UI_PORT"

	# Run the scanners now instead of waiting for their first interval.
	curl -s -X POST "http://localhost:$API_PORT/api/v1/iam/drift/scan" >/dev/null || true
	curl -s -X POST "http://localhost:$API_PORT/api/v1/troubleshooting/scan" >/dev/null || true
}

stop_app() {
	stop_bg web
	stop_bg manager
	stop_bg prometheus-port-forward
}

summary() {
	cat <<EOF

$(printf '\033[1;32m')Platform is up.$(printf '\033[0m')

  UI          http://localhost:$UI_PORT
  API         http://localhost:$API_PORT/api/v1/health/platform
  LocalStack  http://localhost:4566
  Logs        .platform/manager.log, .platform/web.log

  kubectl     kubectl --kubeconfig .platform/kubeconfig get tenants
  ArgoCD      kubectl --kubeconfig .platform/kubeconfig -n argocd port-forward svc/argocd-server 8443:443

Drift scans and troubleshooting rules run every minute. CPU and memory fill in
after Prometheus has a few minutes of samples. The broken pods become
troubleshooting findings after the rules' grace periods: CrashLoopBackOff after
5 restarts, ImagePullBackOff after 10 minutes, Pending after 15.

Stop the app:       make platform-stop
Tear it all down:   make platform-down
EOF
}

# ---------------------------------------------------------------------------

cmd_up() {
	preflight
	create_cluster
	start_localstack
	install_argocd
	install_crossplane
	install_prometheus
	seed
	setup_terminal
	start_app
	summary
}

cmd_stop() {
	step "Stopping the API, UI, and Prometheus port-forward"
	stop_app
	info "done (the cluster and LocalStack are still running; make platform-up restarts the app)"
}

cmd_down() {
	step "Stopping the API, UI, and Prometheus port-forward"
	stop_app
	info "done"
	step "Deleting kind cluster '$CLUSTER'"
	if cluster_exists; then kind delete cluster --name "$CLUSTER"; else info "not found"; fi
	step "Removing LocalStack"
	docker rm -f "$LOCALSTACK_CONTAINER" >/dev/null 2>&1 && info "removed" || info "not running"
	rm -rf "$STATE"
}

cmd_status() {
	printf '%-26s %s\n' "kind cluster '$CLUSTER'" "$(cluster_exists && echo running || echo absent)"
	printf '%-26s %s\n' "LocalStack" "$(docker inspect -f '{{.State.Status}} ({{.Config.Image}})' "$LOCALSTACK_CONTAINER" 2>/dev/null || echo absent)"
	for name in manager web prometheus-port-forward; do
		printf '%-26s %s\n' "$name" "$(running "$name" && echo "running (pid $(cat "$STATE/$name.pid"))" || echo stopped)"
	done
}

case "${1:-}" in
up) cmd_up ;;
stop) cmd_stop ;;
down) cmd_down ;;
status) cmd_status ;;
*)
	echo "usage: $0 {up|stop|down|status}" >&2
	exit 2
	;;
esac
