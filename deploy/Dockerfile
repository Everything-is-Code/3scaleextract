# Final image = Red Hat 3scale toolbox + threescale-export + nginx.
#
# Product YAML export uses the native toolbox binary (/usr/bin/3scale).
# There is NO container-in-container: docker/podman are not used at runtime.
#
# Stage "builder" only compiles the Go CLI; it is not part of the shipped image.

# --- Stage 1: compile threescale-export (not shipped) ---
FROM registry.access.redhat.com/ubi9/go-toolset:1.22 AS builder
USER root
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
ARG VERSION=dev
RUN mkdir -p /build/out && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -X github.com/Everything-is-Code/3scaleextract/internal/version.Version=${VERSION}" \
    -o /build/out/threescale-export ./cmd/threescale-export

# --- Stage 2: extend the official Red Hat toolbox (this is what runs on OpenShift) ---
FROM registry.redhat.io/3scale-amp2/toolbox-rhel9:3scale2.16

USER root
RUN dnf install -y nginx tar gzip && dnf clean all

COPY --from=builder /build/out/threescale-export /usr/local/bin/threescale-export
COPY deploy/entrypoint.sh /usr/local/bin/entrypoint.sh
COPY deploy/generate-index.sh /usr/local/bin/generate-index.sh
COPY deploy/nginx.conf /etc/nginx/nginx.conf

# OpenShift: random non-root UID, group 0 writable paths.
RUN sed -i 's/\r$//' /usr/local/bin/entrypoint.sh /usr/local/bin/generate-index.sh \
    && chmod +x /usr/local/bin/entrypoint.sh /usr/local/bin/generate-index.sh /usr/local/bin/threescale-export \
    && mkdir -p /export/data /export/www \
    /tmp/nginx/client_body /tmp/nginx/proxy /tmp/nginx/fastcgi /tmp/nginx/uwsgi /tmp/nginx/scgi \
    && chgrp -R 0 /export /tmp/nginx /usr/local/bin/threescale-export \
    && chmod -R g+rwX /export /tmp/nginx \
    && chmod g+rx /usr/local/bin/entrypoint.sh /usr/local/bin/generate-index.sh /usr/local/bin/threescale-export

# Force native toolbox — never spawn a sibling toolbox container.
ENV THREESCALE_TOOLBOX_BINARY=/opt/toolbox/bin/3scale \
    THREESCALE_OUTPUT_DIR=/export/data \
    EXPORT_HTTP_PORT=8080 \
    EXPORT_WWW_DIR=/export/www

EXPOSE 8080
USER 1001

# Override toolbox default ENTRYPOINT ["3scale"].
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
