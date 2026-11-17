# prometheus-turbostat-exporter

Prometheus exporter for turbostat C-state and package-state metrics, with binary, Docker, and Helm releases.

## Verify

- `mise run check`: gofmt, vet, golangci-lint, and all tests.
- `mise run zizmor` after changing workflows.
- Keep the separate binary, Docker, and Helm release tag formats unchanged.

CI uses one required `check` gate plus actionlint and strict Renovate validation.

## Stack

Follows https://github.com/BlackDark/agent-presets/blob/main/stack/core.md. Existing GoReleaser-style binary, Docker, and Helm release workflows remain specialized.

## Map

- `main.go`, `configuration.go`: process setup and configuration.
- `internal/`: parser, exporter, authentication, and tests.
- `helm/turbostat-exporter/`: Helm chart.
- `data/`: parser fixtures.

## Git

- Conventional Commits, one topic per PR.
- Squash unless commits need separate review.
- Force-push only with `--force-with-lease` on own branches.

## Boundaries

- Tools come from `mise.toml`; no global installs.
- Do not create tags, attach release binaries, or publish charts/images from a local session.
