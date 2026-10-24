#!/usr/bin/env bash
# End-to-end test for Flux inter-controller SPIFFE mTLS.
#
# It installs SPIRE, Envoy Gateway and Flux with SPIFFE enabled on a kind
# cluster, and verifies that the controllers can consume and serve artifacts,
# send events, and receive webhooks over mTLS. The SPIRE chart and Envoy
# Gateway versions are set by the e2e-amd64-spiffe-mtls job in
# .github/workflows/e2e.yaml.
set -euo pipefail

FLUX_NAMESPACE="${FLUX_NAMESPACE:-flux-system}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MANIFESTS_DIR="${ROOT_DIR}/tests/inter-ctrl-spiffe-mtls/manifests"
FLUX_BIN="${FLUX_BIN:-${ROOT_DIR}/bin/flux}"
ENVOY_GATEWAY_VERSION="${ENVOY_GATEWAY_VERSION:-v1.9.2}"
SPIRE_CRDS_CHART_VERSION="${SPIRE_CRDS_CHART_VERSION:-0.6.1}"
SPIRE_CHART_VERSION="${SPIRE_CHART_VERSION:-0.30.3}"
SPIFFE_DOMAIN="example.org"
SA_PREFIX="spiffe://${SPIFFE_DOMAIN}/ns/${FLUX_NAMESPACE}/sa"

log()  { printf '\n\033[1;34m==> %s\033[0m\n' "$*"; }
info() { printf '    %s\n' "$*"; }
fail() { printf '\033[1;31mFAIL: %s\033[0m\n' "$*" >&2; exit 1; }

cleanup() {
    local code=$?
    if [[ $code -ne 0 ]]; then
        log "FAILURE (exit ${code}) — dumping diagnostics"
        kubectl -n "${FLUX_NAMESPACE}" get pods -o wide || true
        kubectl -n "${FLUX_NAMESPACE}" get events --sort-by=.lastTimestamp | tail -50 || true
        for dep in source-controller source-watcher kustomize-controller helm-controller notification-controller image-reflector-controller image-automation-controller; do
            echo "--- ${dep} ---"
            kubectl -n "${FLUX_NAMESPACE}" logs "deploy/${dep}" --tail=80 || true
        done
        kubectl -n "${FLUX_NAMESPACE}" logs deploy/event-sink --tail=80 || true
        kubectl -n envoy-gateway-system get pods -o wide || true
        kubectl -n envoy-gateway-system logs deploy/envoy-gateway --tail=80 || true
        kubectl -n envoy-gateway-system logs -l gateway.envoyproxy.io/owning-gateway-name=receiver --tail=80 || true
    fi
    if [[ -n "${PORT_FORWARD_PID:-}" ]]; then
        kill "${PORT_FORWARD_PID}" >/dev/null 2>&1 || true
    fi
    exit "$code"
}
trap cleanup EXIT

for bin in kubectl helm kind docker jq curl; do
    command -v "${bin}" >/dev/null 2>&1 || fail "required tool '${bin}' not found"
done
[[ -x "${FLUX_BIN}" ]] || fail "flux binary not found at ${FLUX_BIN}"

##############################################################################
log "Installing SPIRE"
##############################################################################
helm repo add spiffe https://spiffe.github.io/helm-charts-hardened >/dev/null 2>&1 || true
helm repo update spiffe >/dev/null
helm upgrade --install spire-crds spiffe/spire-crds \
    --version "${SPIRE_CRDS_CHART_VERSION}" \
    --namespace spire-crds \
    --create-namespace \
    --wait --timeout 5m
helm upgrade --install spire spiffe/spire \
    --version "${SPIRE_CHART_VERSION}" \
    --namespace spire-system \
    --create-namespace \
    --set global.spire.trustDomain="${SPIFFE_DOMAIN}" \
    --set global.spire.namespaces.create=true \
    --set spire-server.controllerManager.enabled=true \
    --set spire-agent.sds.enabled=true \
    --set spiffe-csi-driver.enabled=true \
    --set spiffe-oidc-discovery-provider.enabled=false \
    --wait --timeout 10m
kubectl -n spire-system wait --for=condition=Ready pods --all --timeout=5m
kubectl apply -f "${MANIFESTS_DIR}/spire.yaml"

##############################################################################
log "Building the event sink"
##############################################################################
docker build -t fluxcd/spiffe-testapp:e2e "${ROOT_DIR}/tests/inter-ctrl-spiffe-mtls/testapp" >/dev/null
kind load docker-image fluxcd/spiffe-testapp:e2e >/dev/null

##############################################################################
log "Installing Flux with the SPIFFE mTLS images"
##############################################################################
"${FLUX_BIN}" install --manifests "${ROOT_DIR}/manifests/test/" >/dev/null

COMMON_PATCH=$(cat <<EOF
spec:
  template:
    metadata:
      labels:
        spiffe.io/identity: "true"
    spec:
      containers:
        - name: manager
          env:
            - name: SPIFFE_ENDPOINT_SOCKET
              value: unix:///spiffe-workload-api/spire-agent.sock
          volumeMounts:
            - name: spiffe-workload-api
              mountPath: /spiffe-workload-api
              readOnly: true
      volumes:
        - name: spiffe-workload-api
          csi:
            driver: csi.spiffe.io
            readOnly: true
EOF
)

for dep in source-controller source-watcher kustomize-controller helm-controller notification-controller image-reflector-controller image-automation-controller; do
    kubectl -n "${FLUX_NAMESPACE}" patch "deployment/${dep}" --type=strategic -p "${COMMON_PATCH}" >/dev/null
done

add_arg() {
    local dep="$1" arg="$2"
    local patch
    patch=$(jq -n --arg value "${arg}" '[{op:"add",path:"/spec/template/spec/containers/0/args/-",value:$value}]')
    kubectl -n "${FLUX_NAMESPACE}" patch "deployment/${dep}" --type=json -p "${patch}" >/dev/null
}

SC_ID="${SA_PREFIX}/source-controller"
SW_ID="${SA_PREFIX}/source-watcher"
KC_ID="${SA_PREFIX}/kustomize-controller"
HC_ID="${SA_PREFIX}/helm-controller"
NC_ID="${SA_PREFIX}/notification-controller"
IRC_ID="${SA_PREFIX}/image-reflector-controller"
IAC_ID="${SA_PREFIX}/image-automation-controller"
EG_ID="spiffe://${SPIFFE_DOMAIN}/ns/envoy-gateway-system/sa/envoy-gateway-proxy"

for id in "${KC_ID}" "${HC_ID}" "${SW_ID}"; do
    add_arg source-controller "--authorized-artifact-client-spiffe-id=${id}"
done
add_arg source-controller "--authorized-event-server-spiffe-id=${NC_ID}"

for id in "${SC_ID}" "${SW_ID}"; do
    add_arg kustomize-controller "--authorized-artifact-server-spiffe-id=${id}"
    add_arg helm-controller "--authorized-artifact-server-spiffe-id=${id}"
done
add_arg kustomize-controller "--authorized-event-server-spiffe-id=${NC_ID}"
add_arg helm-controller "--authorized-event-server-spiffe-id=${NC_ID}"

for id in "${KC_ID}" "${HC_ID}"; do
    add_arg source-watcher "--authorized-artifact-client-spiffe-id=${id}"
done
add_arg source-watcher "--authorized-artifact-server-spiffe-id=${SC_ID}"
add_arg source-watcher "--authorized-event-server-spiffe-id=${NC_ID}"

for id in "${SC_ID}" "${KC_ID}" "${HC_ID}" "${SW_ID}" "${IRC_ID}" "${IAC_ID}"; do
    add_arg notification-controller "--authorized-notifier-spiffe-id=${id}"
done
add_arg notification-controller "--authorized-receiver-spiffe-id=${EG_ID}"

add_arg image-reflector-controller "--authorized-event-server-spiffe-id=${NC_ID}"
add_arg image-automation-controller "--authorized-event-server-spiffe-id=${NC_ID}"

log "Waiting for the controllers to roll out"
for dep in source-controller source-watcher kustomize-controller helm-controller notification-controller image-reflector-controller image-automation-controller; do
    kubectl -n "${FLUX_NAMESPACE}" rollout status "deployment/${dep}" --timeout=10m
done

##############################################################################
log "Installing Envoy Gateway"
##############################################################################
kubectl apply --server-side -f \
    "https://github.com/envoyproxy/gateway/releases/download/${ENVOY_GATEWAY_VERSION}/install.yaml" >/dev/null
kubectl -n envoy-gateway-system rollout status deployment/envoy-gateway --timeout=10m
kubectl apply -f "${MANIFESTS_DIR}/envoy-gateway-config.yaml" >/dev/null
kubectl -n envoy-gateway-system rollout restart deployment/envoy-gateway >/dev/null
kubectl -n envoy-gateway-system rollout status deployment/envoy-gateway --timeout=10m

log "Deploying Envoy Gateway receiver"
# Envoy Gateway pulls its SPIFFE client SVID and trust bundle from SPIRE's SDS API.
kubectl apply -f "${MANIFESTS_DIR}/envoy-gateway.yaml" >/dev/null

log "Verifying SPIFFE mTLS startup logs"
assert_log() {
    local dep="$1" msg="$2" pod logs
    pod=$(kubectl -n "${FLUX_NAMESPACE}" get pods \
        -l "app.kubernetes.io/component=${dep}" \
        --sort-by=.metadata.creationTimestamp \
        -o jsonpath='{.items[-1].metadata.name}')
    logs=$(kubectl -n "${FLUX_NAMESPACE}" logs "${pod}")
    grep -qF "${msg}" <<<"${logs}" || fail "missing log line for ${dep} (pod ${pod}): ${msg}"
}
assert_log source-controller "artifact server will use inter-controller mTLS via SPIFFE"
assert_log source-controller "event server will use inter-controller mTLS via SPIFFE"
assert_log source-watcher "artifact server will use inter-controller mTLS via SPIFFE"
assert_log source-watcher "artifact fetcher will use inter-controller mTLS via SPIFFE"
assert_log source-watcher "event server will use inter-controller mTLS via SPIFFE"
assert_log kustomize-controller "artifact fetcher will use inter-controller mTLS via SPIFFE"
assert_log kustomize-controller "event server will use inter-controller mTLS via SPIFFE"
assert_log helm-controller "artifact fetcher will use inter-controller mTLS via SPIFFE"
assert_log helm-controller "event server will use inter-controller mTLS via SPIFFE"
assert_log notification-controller "event server will use inter-controller mTLS via SPIFFE"
assert_log notification-controller "receiver server will use inter-controller mTLS via SPIFFE"
assert_log image-reflector-controller "event server will use inter-controller mTLS via SPIFFE"
assert_log image-automation-controller "event server will use inter-controller mTLS via SPIFFE"

##############################################################################
log "Deploying the event sink"
##############################################################################
kubectl apply -f "${MANIFESTS_DIR}/event-sink.yaml"
kubectl -n "${FLUX_NAMESPACE}" rollout status deployment/event-sink --timeout=2m

##############################################################################
log "Running inter-controller traffic tests"
##############################################################################
kubectl apply -f "${MANIFESTS_DIR}/test-resources.yaml"

kubectl -n "${FLUX_NAMESPACE}" wait --for=condition=Ready gitrepository/podinfo --timeout=5m
kubectl -n "${FLUX_NAMESPACE}" wait --for=condition=Ready kustomization/podinfo --timeout=5m
kubectl -n "${FLUX_NAMESPACE}" wait --for=condition=Ready helmrepository/podinfo --timeout=5m
kubectl -n "${FLUX_NAMESPACE}" wait --for=condition=Ready helmrelease/podinfo --timeout=5m
kubectl -n "${FLUX_NAMESPACE}" wait --for=condition=Ready artifactgenerator/podinfo --timeout=5m
kubectl -n "${FLUX_NAMESPACE}" wait --for=condition=Ready externalartifact/podinfo-split --timeout=5m
kubectl -n "${FLUX_NAMESPACE}" wait --for=condition=Ready externalartifact/podinfo-chart-split --timeout=5m
kubectl -n "${FLUX_NAMESPACE}" wait --for=condition=Ready kustomization/podinfo-external --timeout=5m
kubectl -n "${FLUX_NAMESPACE}" wait --for=condition=Ready helmrelease/podinfo-external --timeout=5m
kubectl -n "${FLUX_NAMESPACE}" wait --for=condition=Ready imagerepository/podinfo --timeout=5m
kubectl -n "${FLUX_NAMESPACE}" wait --for=condition=Ready receiver/spiffe-e2e --timeout=5m

WEBHOOK_PATH=$(kubectl -n "${FLUX_NAMESPACE}" get receiver/spiffe-e2e -o jsonpath='{.status.webhookPath}')
[[ -n "${WEBHOOK_PATH}" ]] || fail "receiver webhook path is empty"

log "Calling the receiver through Envoy Gateway"
for _ in $(seq 1 60); do
    kubectl -n envoy-gateway-system get deployment \
        -l gateway.envoyproxy.io/owning-gateway-name=receiver -o name 2>/dev/null | grep -q . && break
    sleep 2
done
kubectl -n envoy-gateway-system rollout status deployment \
    -l gateway.envoyproxy.io/owning-gateway-name=receiver --timeout=10m

ENVOY_SERVICE=$(kubectl -n envoy-gateway-system get service \
    -l gateway.envoyproxy.io/owning-gateway-name=receiver -o jsonpath='{.items[0].metadata.name}')
[[ -n "${ENVOY_SERVICE}" ]] || fail "Envoy Gateway service not found"

kubectl -n envoy-gateway-system port-forward "service/${ENVOY_SERVICE}" 8080:80 >/dev/null 2>&1 &
PORT_FORWARD_PID=$!
for _ in $(seq 1 30); do
    curl -s -o /dev/null "http://127.0.0.1:8080/" && break
    sleep 1
done
curl -fsS -X POST "http://127.0.0.1:8080${WEBHOOK_PATH}" \
    -H 'Content-Type: application/json' -d '{}' >/dev/null

WEBHOOK_DIGEST="${WEBHOOK_PATH##*/}"
for _ in $(seq 1 30); do
    kubectl -n "${FLUX_NAMESPACE}" logs deploy/notification-controller --since=5m 2>/dev/null \
        | grep -q "handling request: ${WEBHOOK_DIGEST}" && break
    sleep 1
done
kubectl -n "${FLUX_NAMESPACE}" logs deploy/notification-controller --since=5m 2>/dev/null \
    | grep -q "handling request: ${WEBHOOK_DIGEST}" \
    || fail "notification-controller did not handle the receiver request"
info "receiver call through Envoy Gateway succeeded"

log "Waiting for an event from every controller"
expected_controllers=(
    source-controller
    source-watcher
    kustomize-controller
    helm-controller
    image-reflector-controller
    image-automation-controller
)
for _ in $(seq 1 120); do
    logs=$(kubectl -n "${FLUX_NAMESPACE}" logs deploy/event-sink --tail=-1 2>/dev/null || true)
    missing=()
    for controller in "${expected_controllers[@]}"; do
        grep -q "reportingController=${controller}$" <<<"${logs}" || missing+=("${controller}")
    done
    if [[ ${#missing[@]} -eq 0 ]]; then
        info "event sink received events from all controllers"
        break
    fi
    sleep 2
done
[[ ${#missing[@]} -eq 0 ]] || fail "event sink is missing events from: ${missing[*]}"

log "SUCCESS: all inter-controller SPIFFE mTLS checks passed"
