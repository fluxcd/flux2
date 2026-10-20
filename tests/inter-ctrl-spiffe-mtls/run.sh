#!/usr/bin/env bash
# End-to-end test for Flux inter-controller SPIFFE mTLS.
#
# It installs SPIRE, Flux with the controller images built from the
# rfc-0014-inter-ctrl-spiffe-mtls branches, and verifies that the controllers
# can consume and serve artifacts, send events, and receive webhooks over mTLS.
set -euo pipefail

CLUSTER="${CLUSTER:-flux-e2e-inter-ctrl-spiffe-mtls}"
FLUX_NAMESPACE="${FLUX_NAMESPACE:-flux-system}"

# TODO: Once the controller releases with SPIFFE mTLS are out, delete the
# temporary rc image tag variables and image overrides below. The `flux install`
# command will then use the semver-tagged images from manifests/bases.
SOURCE_CONTROLLER_TAG="${SOURCE_CONTROLLER_TAG:-rc-f099e63b}"
SOURCE_WATCHER_TAG="${SOURCE_WATCHER_TAG:-rc-c8b7007b}"
KUSTOMIZE_CONTROLLER_TAG="${KUSTOMIZE_CONTROLLER_TAG:-rc-b1b28ff5}"
HELM_CONTROLLER_TAG="${HELM_CONTROLLER_TAG:-rc-effd0af5}"
NOTIFICATION_CONTROLLER_TAG="${NOTIFICATION_CONTROLLER_TAG:-rc-47876e1e}"
IMAGE_REFLECTOR_CONTROLLER_TAG="${IMAGE_REFLECTOR_CONTROLLER_TAG:-rc-f5bfef9f}"
IMAGE_AUTOMATION_CONTROLLER_TAG="${IMAGE_AUTOMATION_CONTROLLER_TAG:-rc-20fe8a35}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MANIFESTS_DIR="${ROOT_DIR}/tests/inter-ctrl-spiffe-mtls/manifests"
FLUX_BIN="${FLUX_BIN:-${ROOT_DIR}/bin/flux}"
ENVOY_GATEWAY_VERSION="${ENVOY_GATEWAY_VERSION:-v1.9.2}"
ENVOY_GATEWAY_IMAGE="${ENVOY_GATEWAY_IMAGE:-docker.io/envoyproxy/gateway:${ENVOY_GATEWAY_VERSION}}"
ENVOY_PROXY_IMAGE="${ENVOY_PROXY_IMAGE:-docker.io/envoyproxy/envoy:distroless-v1.39.1}"
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

kubectl config use-context "kind-${CLUSTER}" >/dev/null

##############################################################################
log "Pre-loading images into kind"
##############################################################################
load_image() {
    local image="$1" archive node
    docker pull --platform linux/amd64 "${image}" >/dev/null
    archive="$(mktemp)"
    docker save "${image}" -o "${archive}"
    for node in $(kind get nodes --name "${CLUSTER}"); do
        docker cp "${archive}" "${node}:/var/tmp/kind-image.tar"
        docker exec "${node}" ctr -n k8s.io images import \
            --platform linux/amd64 --digests --snapshotter=overlayfs \
            /var/tmp/kind-image.tar >/dev/null
        docker exec "${node}" rm -f /var/tmp/kind-image.tar
    done
    rm -f "${archive}"
}
for image in \
    "ghcr.io/fluxcd/source-controller:${SOURCE_CONTROLLER_TAG}" \
    "ghcr.io/fluxcd/source-watcher:${SOURCE_WATCHER_TAG}" \
    "ghcr.io/fluxcd/kustomize-controller:${KUSTOMIZE_CONTROLLER_TAG}" \
    "ghcr.io/fluxcd/helm-controller:${HELM_CONTROLLER_TAG}" \
    "ghcr.io/fluxcd/notification-controller:${NOTIFICATION_CONTROLLER_TAG}" \
    "ghcr.io/fluxcd/image-reflector-controller:${IMAGE_REFLECTOR_CONTROLLER_TAG}" \
    "ghcr.io/fluxcd/image-automation-controller:${IMAGE_AUTOMATION_CONTROLLER_TAG}" \
    "${ENVOY_GATEWAY_IMAGE}" \
    "${ENVOY_PROXY_IMAGE}"; do
    load_image "${image}"
done

##############################################################################
log "Installing SPIRE"
##############################################################################
helm repo add spiffe https://spiffe.github.io/helm-charts-hardened >/dev/null 2>&1 || true
helm repo update spiffe >/dev/null
helm upgrade --install spire-crds spiffe/spire-crds \
    --version 0.6.1 \
    --namespace spire-crds \
    --create-namespace \
    --wait --timeout 5m
helm upgrade --install spire spiffe/spire \
    --version 0.30.3 \
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
kind load docker-image fluxcd/spiffe-testapp:e2e --name "${CLUSTER}" >/dev/null

##############################################################################
log "Installing Flux with the SPIFFE mTLS images"
##############################################################################
"${FLUX_BIN}" install --manifests "${ROOT_DIR}/manifests/test/" >/dev/null

# TODO: remove the rc image overrides below when the semver releases
# include SPIFFE mTLS.
kubectl -n "${FLUX_NAMESPACE}" set image deployment/source-controller manager="ghcr.io/fluxcd/source-controller:${SOURCE_CONTROLLER_TAG}"
kubectl -n "${FLUX_NAMESPACE}" set image deployment/source-watcher manager="ghcr.io/fluxcd/source-watcher:${SOURCE_WATCHER_TAG}"
kubectl -n "${FLUX_NAMESPACE}" set image deployment/kustomize-controller manager="ghcr.io/fluxcd/kustomize-controller:${KUSTOMIZE_CONTROLLER_TAG}"
kubectl -n "${FLUX_NAMESPACE}" set image deployment/helm-controller manager="ghcr.io/fluxcd/helm-controller:${HELM_CONTROLLER_TAG}"
kubectl -n "${FLUX_NAMESPACE}" set image deployment/notification-controller manager="ghcr.io/fluxcd/notification-controller:${NOTIFICATION_CONTROLLER_TAG}"
kubectl -n "${FLUX_NAMESPACE}" set image deployment/image-reflector-controller manager="ghcr.io/fluxcd/image-reflector-controller:${IMAGE_REFLECTOR_CONTROLLER_TAG}"
kubectl -n "${FLUX_NAMESPACE}" set image deployment/image-automation-controller manager="ghcr.io/fluxcd/image-automation-controller:${IMAGE_AUTOMATION_CONTROLLER_TAG}"

# HTTPS clients default to port 443 when the advertised host has no port.
add_https_port() {
    local svc="$1" target="$2"
    kubectl -n "${FLUX_NAMESPACE}" patch "service/${svc}" --type=json -p \
        "[{\"op\":\"add\",\"path\":\"/spec/ports/-\",\"value\":{\"name\":\"https\",\"port\":443,\"protocol\":\"TCP\",\"targetPort\":\"${target}\"}}]"
}
add_https_port source-controller http
add_https_port source-watcher http
add_https_port notification-controller http
add_https_port webhook-receiver http-webhook

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

# TODO: Remove this patch once source-controller and source-watcher releases
# include the /readyz readiness probe and flux2 pins those release manifests.
# The manifests installed here still point at the previous semver releases.
# The artifact server readiness probes cannot complete an mTLS handshake.
for dep in source-controller source-watcher; do
    kubectl -n "${FLUX_NAMESPACE}" patch "deployment/${dep}" --type=json -p \
        '[{"op":"replace","path":"/spec/template/spec/containers/0/readinessProbe","value":{"httpGet":{"path":"/readyz","port":"healthz"}}}]' >/dev/null
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
sed "s#__ENVOY_GATEWAY_IMAGE__#${ENVOY_GATEWAY_IMAGE}#" "${MANIFESTS_DIR}/envoy-gateway-config.yaml" | kubectl apply -f - >/dev/null
kubectl -n envoy-gateway-system rollout restart deployment/envoy-gateway >/dev/null
kubectl -n envoy-gateway-system rollout status deployment/envoy-gateway --timeout=10m

log "Deploying Envoy Gateway receiver"
# Envoy Gateway pulls its SPIFFE client SVID and trust bundle from SPIRE's SDS API.
sed "s#__ENVOY_PROXY_IMAGE__#${ENVOY_PROXY_IMAGE}#" "${MANIFESTS_DIR}/envoy-gateway.yaml" | kubectl apply -f - >/dev/null

log "Verifying SPIFFE mTLS startup logs"
assert_log() {
    local dep="$1" msg="$2" logs
    logs=$(kubectl -n "${FLUX_NAMESPACE}" logs "deploy/${dep}")
    grep -qF "${msg}" <<<"${logs}" || fail "missing log line for ${dep}: ${msg}"
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
