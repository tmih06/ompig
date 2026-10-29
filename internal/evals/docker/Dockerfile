# Builds the two documentation-variant images from the repository (build context = repository root).
# Both variants run the same binaries; the harness removes the Ompig documentation from the without_docs run's home
# (internal/evals/harness_run.go) and the without_docs prompt loses its documentation section.
ARG GO_VERSION=1.27.3
FROM golang:${GO_VERSION}-bookworm AS builder
WORKDIR /src
# Build against the repository mount; only the staged binaries persist.
RUN --mount=type=bind,source=.,target=/src,rw \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /opt/ompig-evals/ompig ./cmd/ompig \
    && CGO_ENABLED=0 go build -o /opt/ompig-evals/ompig-eval-extension ./cmd/ompig-eval-extension \
    && CGO_ENABLED=0 go build -o /opt/ompig-evals/ompig-evals ./cmd/ompig-evals

FROM debian:bookworm-slim AS evaluator
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tmux coreutils \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /opt/ompig-evals /opt/ompig-evals
# The runner starts as root so it can hold the evaluators. Before prompting the model, the harness starts the agent as
# UID 65532. Keep the runner (judges, fixtures, prompts) root-only so model tools and generated code cannot read it.
RUN chmod 0755 /opt/ompig-evals /opt/ompig-evals/ompig /opt/ompig-evals/ompig-eval-extension \
    && chmod 0700 /opt/ompig-evals/ompig-evals \
    && mkdir -p /run/pi-eval-secrets /repo/packages/evals \
    && chmod 0700 /run/pi-eval-secrets
ENV PI_EVAL_OMPIG=/opt/ompig-evals/ompig PI_EVAL_EXTENSION=/opt/ompig-evals/ompig-eval-extension
WORKDIR /repo/packages/evals
ENTRYPOINT ["/opt/ompig-evals/ompig-evals", "entrypoint"]

FROM evaluator AS without_docs
ENV PI_EVAL_VARIANT=without_docs

FROM evaluator AS with_docs
ENV PI_EVAL_VARIANT=with_docs
