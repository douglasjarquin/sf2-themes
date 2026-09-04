# syntax=docker/dockerfile:1.7
# Cursor Cloud can build only this Dockerfile and cannot consume a
# host-local toolchain image. Default TOOLCHAIN_IMAGE is Ubuntu so a
# no-arg build bootstraps mise here. CI still passes the published
# toolchain image to skip that download.
ARG TOOLCHAIN_IMAGE=ubuntu:24.04@sha256:33ceb71981b602c1a7443a53469e4dba065f7503eab3078a2d7a57a2ab987517
FROM ${TOOLCHAIN_IMAGE}

ARG MISE_VERSION=2026.9.1
ARG DEBIAN_FRONTEND=noninteractive

USER root
COPY docker/toolchain/apt-packages.txt /tmp/toolchain-apt-packages.txt
COPY docker/dev/apt-packages.txt /tmp/apt-packages.txt
RUN apt-get update -qq && \
    xargs -a /tmp/toolchain-apt-packages.txt apt-get install -y --no-install-recommends && \
    xargs -a /tmp/apt-packages.txt apt-get install -y --no-install-recommends && \
    rm -rf /var/lib/apt/lists/* /tmp/toolchain-apt-packages.txt /tmp/apt-packages.txt

SHELL ["/bin/bash", "-c"]

ENV MISE_DATA_DIR=/usr/local/share/mise \
    MISE_CONFIG_DIR=/usr/local/etc/mise \
    MISE_YES=1 \
    PATH="/usr/local/share/mise/shims:/usr/local/bin:${PATH}"

RUN if ! command -v mise >/dev/null 2>&1; then \
      MISE_VERSION="v${MISE_VERSION}" MISE_INSTALL_PATH=/usr/local/bin/mise \
      sh -c "curl -fsSL https://mise.jdx.dev/install.sh | sh"; \
    fi

WORKDIR /opt/sf2-themes/toolchain
COPY mise.toml mise.lock ./
# Skip when FROM the published toolchain image, which already ran this.
# mise install --locked verifies GitHub artifact attestations over the
# network per tool; the Sigstore TUF fetch behind that has intermittently
# failed with a transport error unrelated to artifact integrity, reproduced
# and confirmed transient across repeated fresh-container attempts.
RUN if [ ! -f "$MISE_CONFIG_DIR/config.toml" ]; then \
      mise trust --yes . && \
      for attempt in 1 2 3; do \
        mise install --locked && break; \
        [ "$attempt" -lt 3 ] || exit 1; \
        sleep 5; \
      done && \
      mkdir -p "$MISE_CONFIG_DIR" && \
      cp mise.toml "$MISE_CONFIG_DIR/config.toml" && \
      cp mise.lock "$MISE_CONFIG_DIR/mise.lock" && \
      mise trust --yes "$MISE_CONFIG_DIR/config.toml" && \
      chmod -R a+rX "$MISE_CONFIG_DIR" "$MISE_DATA_DIR"; \
    fi

RUN if ! id -u dev >/dev/null 2>&1; then useradd -m -s /bin/bash dev; fi && \
    mkdir -p /workspace/sf2-themes/web && \
    chown -R dev:dev /workspace && \
    echo "dev ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/dev && \
    chmod 0440 /etc/sudoers.d/dev

RUN mkdir -p /opt/sf2-themes/manifests/web && \
    chown -R dev:dev /opt/sf2-themes
WORKDIR /opt/sf2-themes/manifests
COPY --chown=dev:dev pyproject.toml uv.lock ./
COPY --chown=dev:dev web/package.json web/aube-lock.yaml web/.npmrc ./web/

# aube (xx::fslock) writes /tmp/fslock/<hash>. Root `mise install` in this
# file or the toolchain image leaves that directory 0755, so USER dev cannot
# create lockfiles. Same chmod as scripts/ci/run-in-dev-container.sh.
RUN mkdir -p /tmp/fslock && chmod 1777 /tmp/fslock

USER dev
RUN mise exec -- uv sync --all-extras --no-install-project && \
    cd web && mise exec -- aube ci

USER root
RUN cd web && mise exec -- aube exec playwright install-deps chromium

USER dev
RUN cd web && mise exec -- aube exec playwright install chromium

USER root
RUN chmod -R a+rX /opt/sf2-themes/manifests /home/dev/.cache
COPY --chmod=0755 docker/dev/entrypoint.sh /usr/local/bin/sf2-themes-dev-entrypoint

WORKDIR /workspace/sf2-themes
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/sf2-themes-dev-entrypoint"]
