FROM --platform=$BUILDPLATFORM golang:1.26@sha256:670bb5aa5349ae03b8516764718486f4f16399134bdc9acd0c46a23f043ab05c AS build

ARG BUILDPLATFORM
ARG TARGETARCH
ARG VERSION

COPY . /src
RUN cd /src && CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -a -ldflags="-X 'github.com/hsiaoairplane/argocd-applicationset-namespaces-generator-plugin/cmd/version.Version=$VERSION'" -o /bin/argocd-applicationset-namespaces-generator-plugin

# Static binary on distroless: no shell, compiler or package manager at runtime.
FROM gcr.io/distroless/static-debian13:nonroot@sha256:18f4cc26c9ffa7f20a359298be7deadac5c607c8e03a02b8119f1a5fc2f82c99

COPY --from=build /bin/argocd-applicationset-namespaces-generator-plugin /usr/local/bin/argocd-applicationset-namespaces-generator-plugin

# Numeric, so runAsNonRoot can verify it (65532 is distroless "nonroot").
USER 65532:65532

ENTRYPOINT ["/usr/local/bin/argocd-applicationset-namespaces-generator-plugin", "server"]
