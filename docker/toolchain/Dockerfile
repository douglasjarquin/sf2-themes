# syntax=docker/dockerfile:1.7
ARG UBUNTU_IMAGE=ubuntu:24.04@sha256:33ceb71981b602c1a7443a53469e4dba065f7503eab3078a2d7a57a2ab987517
FROM ${UBUNTU_IMAGE}

ARG MISE_VERSION=2026.9.1
ARG DEBIAN_FRONTEND=noninteractive

COPY docker/toolchain/apt-packages.txt /tmp/apt-packages.txt
RUN apt-get update -qq && \
    xargs -a /tmp/apt-packages.txt apt-get install -y --no-install-recommends && \
    rm -rf /var/lib/apt/lists/* /tmp/apt-packages.txt

SHELL ["/bin/bash", "-c"]

ENV MISE_DATA_DIR=/usr/local/share/mise \
    MISE_CONFIG_DIR=/usr/local/etc/mise \
    MISE_YES=1 \
    PATH="/usr/local/share/mise/shims:/usr/local/bin:${PATH}"

RUN MISE_VERSION="v${MISE_VERSION}" MISE_INSTALL_PATH=/usr/local/bin/mise \
    sh -c "curl -fsSL https://mise.jdx.dev/install.sh | sh" && \
    mise --version

WORKDIR /opt/sf2-themes/toolchain
COPY mise.toml mise.lock ./
# mise install --locked verifies GitHub artifact attestations over the
# network per tool; the Sigstore TUF fetch behind that has intermittently
# failed with a transport error unrelated to artifact integrity, reproduced
# and confirmed transient across repeated fresh-container attempts.
RUN mise trust --yes . && \
    for attempt in 1 2 3; do \
      mise install --locked && break; \
      [ "$attempt" -lt 3 ] || exit 1; \
      sleep 5; \
    done

RUN mkdir -p "$MISE_CONFIG_DIR" && \
    cp mise.toml "$MISE_CONFIG_DIR/config.toml" && \
    cp mise.lock "$MISE_CONFIG_DIR/mise.lock" && \
    mise trust --yes "$MISE_CONFIG_DIR/config.toml" && \
    chmod -R a+rX "$MISE_CONFIG_DIR" "$MISE_DATA_DIR"

LABEL org.opencontainers.image.source="https://github.com/douglasjarquin/sf2-themes" \
      org.opencontainers.image.title="sf2-themes-toolchain" \
      org.opencontainers.image.description="Prewarmed mise toolchain (Python, Node, aube, ruff, shellcheck) for sf2-themes, no app source"
