FROM alpine:3.13 as builder

RUN apk add --no-cache ca-certificates curl

ARG ARCH=linux/amd64
ARG KUBECTL_VER=1.20.4

RUN curl -sL https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VER}/bin/${ARCH}/kubectl \
    -o /usr/local/bin/kubectl && chmod +x /usr/local/bin/kubectl && \
    kubectl version --client=true

FROM alpine:3.13 as flux-cli

# Create minimal nsswitch.conf file to prioritize the usage of /etc/hosts over DNS queries.
# https://github.com/gliderlabs/docker-alpine/issues/367#issuecomment-354316460
RUN [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf

RUN apk add --no-cache ca-certificates

COPY --from=builder /usr/local/bin/kubectl /usr/local/bin/
COPY --chmod=755 flux /usr/local/bin/

ENTRYPOINT [ "flux" ]
