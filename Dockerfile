# core-runtime carries ca-certificates-bundle, needed for TLS to Vault and GCS.
FROM quay.io/hummingbird/core-runtime:latest

ARG TARGETPLATFORM

COPY $TARGETPLATFORM/vault-snapshot-agent /usr/bin/vault-snapshot-agent

ENTRYPOINT ["/usr/bin/vault-snapshot-agent"]
