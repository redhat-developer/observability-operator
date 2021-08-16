# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4

WORKDIR /

COPY manager .

USER 65532:65532

ENTRYPOINT ["/manager"]
