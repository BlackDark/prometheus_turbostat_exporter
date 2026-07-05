# Helm Chart & Release Design

Date: 2026-07-05
Status: Approved

## Context

`prometheus_turbostat_exporter` reads host hardware counters (MSRs) via
`turbostat`. Today the only supported container deployment is
`docker run --privileged`. This design adds a Helm chart for Kubernetes with
the minimum permissions that still work out of the box, plus a release
pipeline for the chart, as one part of a larger PR that also adds
golangci-lint, zizmor, and Dockerfile hardening.

## Deployment shape

- **DaemonSet**, one pod per node — turbostat measures host hardware state,
  so it must run once per physical/VM host, not scale with application
  replicas.
- `hostNetwork: false`. Metrics are exposed via a `ClusterIP` `Service`.
- An optional, disabled-by-default `ServiceMonitor` (prometheus-operator CRD)
  is included for clusters that use it; a plain `Service` with scrape
  annotations works without it.
- No `hostPID`. The app never inspects host processes.
- A `ServiceAccount` is created only to explicitly set
  `automountServiceAccountToken: false`. No `Role`/`ClusterRole` is created —
  the app never calls the Kubernetes API, so it needs zero Kubernetes RBAC
  permissions. This is called out explicitly in the README so it's clear the
  "permissions" this chart deals with are host/Linux capabilities, not
  Kubernetes RBAC.

## Host access / security context

MSR access is gated two ways on stock kernels: the device file
`/dev/cpu/*/msr` is normally `0600 root:root` (a DAC check), and the kernel's
msr driver additionally requires `CAP_SYS_RAWIO` on open regardless of UID.
turbostat's own error message when this fails suggests exactly these two
fixes: "try chown or chmod +r /dev/cpu/*/msr, or run as root."

Chart default, chosen to work unmodified on a stock kernel while dropping as
much as possible from today's `--privileged`:

- hostPath volume mounting `/dev/cpu` (type `Directory`), mounted read-only
  in the container.
- `securityContext`:
  - `privileged: false`
  - `runAsUser: 0` (required unless the host admin has chmod'd/chowned the
    msr device — see README)
  - `allowPrivilegeEscalation: false`
  - `readOnlyRootFilesystem: true`
  - `seccompProfile: { type: RuntimeDefault }`
  - `capabilities: { drop: [ALL], add: [SYS_RAWIO, SYS_ADMIN] }`

The README documents each capability's purpose, why root is still needed by
default, and how a host admin can go further (chmod the device / udev rule)
to eventually drop `runAsUser: 0` too — that path is not the chart default
because it requires a manual host-side prerequisite that would otherwise
silently break the chart on unmodified hosts.

## Values

- `image.repository` / `image.tag` (defaults to `.Chart.AppVersion`) /
  `image.pullPolicy`
- Env vars mirroring `configuration.go`: log level, default/background
  collect intervals, background mode toggle, basic auth (`enabled`,
  `username`, inline `password` or `existingSecret`/`existingSecretKey`)
- `resources`, `nodeSelector`, `tolerations`, `affinity`
- `service.type` / `service.port`
- `serviceMonitor.enabled` (default `false`) plus standard
  interval/labels/relabelings passthrough

## Versioning & release pipeline

The chart's `Chart.yaml` version is **independent** of the app's version
(starts at `0.1.0`), bumped only when chart templates/values change.
`appVersion` documents the exporter version the chart defaults to, updated
manually when convenient — it is not auto-synced.

A new GitHub Actions workflow (`.github/workflows/helm.yml`) triggers on
tags matching `helm-v*.*.*`:

1. Checkout, install `helm`.
2. `helm lint` the chart.
3. Package the chart (version taken from the tag).
4. `helm push` the packaged chart as an OCI artifact to
   `oci://ghcr.io/<owner>/charts/turbostat-exporter`.
5. Create a GitHub Release for the tag with the packaged `.tgz` attached.

This reuses the existing tag-triggered release pattern (`v*.*.*` already
triggers `docker.yml`) with a distinct prefix so chart and app releases never
collide. Cutting a chart release is: bump `Chart.yaml` version, tag
`helm-vX.Y.Z`, push the tag — no manual workflow dispatch inputs, no
Chart.yaml diffing logic in CI.

## Out of scope

- Publishing to a traditional Helm repo index (GitHub Pages + chart-releaser)
  — OCI is now the standard distribution mechanism and avoids maintaining a
  second branch/index.
- Automatically bumping `appVersion` when the app releases — left as a
  manual edit to keep the two release cadences fully decoupled.
