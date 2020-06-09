FROM golang:1.14-alpine as build

RUN apk add --update make gcc musl-dev

ENV KUSTOMIZE_PLUGIN_HOME /kustomizer/plugins

WORKDIR /kustomizer/code

RUN mkdir -p ${KUSTOMIZE_PLUGIN_HOME}
ADD go.mod go.sum ./
ADD Makefile build_helpers.go ./
ADD SOPSGenerator.go ./
RUN make build
RUN make install

# Install kustomize. It is **required** to run this inside the plugin's source
# (until a better solution is found) due to a few issues in Go:
#
#   - https://github.com/golang/go/issues/17150
#   - https://github.com/golang/go/issues/24034
#
RUN go get sigs.k8s.io/kustomize/kustomize/v3

# We make it a multi-step build, because the go image, with all the other build dependencies
# ends up being quite large (1.5 GB)
# ---
FROM alpine:latest

ENV KUSTOMIZE_PLUGIN_HOME /kustomizer/plugins
COPY --from=build ${KUSTOMIZE_PLUGIN_HOME} ${KUSTOMIZE_PLUGIN_HOME}
COPY --from=build /go/bin/kustomize /usr/local/bin

WORKDIR /code/
ENTRYPOINT [ "kustomize", "build", "--enable_alpha_plugins" ]
