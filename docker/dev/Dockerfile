# syntax=docker/dockerfile:1.7
ARG TOOLCHAIN_IMAGE=sf2-themes-toolchain:latest
FROM ${TOOLCHAIN_IMAGE}

ARG DEBIAN_FRONTEND=noninteractive

USER root
COPY docker/dev/apt-packages.txt /tmp/apt-packages.txt
RUN apt-get update -qq && \
    xargs -a /tmp/apt-packages.txt apt-get install -y --no-install-recommends && \
    rm -rf /var/lib/apt/lists/* /tmp/apt-packages.txt

RUN if ! id -u dev >/dev/null 2>&1; then useradd -m -s /bin/bash dev; fi && \
    mkdir -p /workspace/sf2-themes/web && \
    chown -R dev:dev /workspace

RUN mkdir -p /opt/sf2-themes/manifests/web && \
    chown -R dev:dev /opt/sf2-themes
WORKDIR /opt/sf2-themes/manifests
COPY --chown=dev:dev pyproject.toml uv.lock ./
COPY --chown=dev:dev web/package.json web/package-lock.json ./web/

USER dev
RUN mise exec -- uv sync --all-extras --no-install-project && \
    cd web && mise exec -- npm ci

USER root
RUN cd web && mise exec -- npx playwright install-deps chromium

USER dev
RUN cd web && mise exec -- npx playwright install chromium

USER root
RUN chmod -R a+rX /opt/sf2-themes/manifests /home/dev/.cache
COPY --chmod=0755 docker/dev/entrypoint.sh /usr/local/bin/sf2-themes-dev-entrypoint

WORKDIR /workspace/sf2-themes
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/sf2-themes-dev-entrypoint"]
