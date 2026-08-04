# Built by GoReleaser (dockers_v2). The build context contains the already
# compiled binaries laid out per platform, so there is no build stage here -
# the binary in the image is the same one published in the release archive.
#
# distroless/static rather than scratch: the agent talks TLS to Vault and to
# Google Cloud Storage and therefore needs the CA certificate bundle.
FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETPLATFORM

COPY $TARGETPLATFORM/vault-snapshot-agent /usr/bin/vault-snapshot-agent

USER 65532:65532

ENTRYPOINT ["/usr/bin/vault-snapshot-agent"]
