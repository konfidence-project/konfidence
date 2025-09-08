# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM ubuntu:latest
WORKDIR /
COPY bin/manager .

# install update-ca-certificates
RUN apt-get update && apt-get install -y ca-certificates

# USER 65532:65532

ENTRYPOINT ["/manager"]
