# turbostat-exporter

Helm chart for [prometheus_turbostat_exporter](https://github.com/BlackDark/prometheus_turbostat_exporter),
deployed as a DaemonSet since turbostat reads host hardware counters and
therefore only makes sense running once per node.

## Install

```bash
helm install turbostat-exporter oci://ghcr.io/blackdark/charts/turbostat-exporter --version <chart-version>
```

## Why this chart needs elevated permissions

turbostat reads CPU model-specific registers (MSRs) through
`/dev/cpu/*/msr`. On a stock kernel that device file is `root:root` mode
`0600`, and the kernel's msr driver *additionally* requires the
`CAP_SYS_RAWIO` capability to open it at all, regardless of UID. This is
also why the plain-Docker instructions in the main README use
`docker run --privileged`.

This chart does **not** use `privileged: true`. Instead its default
`securityContext` is the narrowest configuration that still works
unmodified on a stock kernel:

| Setting | Value | Why |
|---|---|---|
| `runAsUser` | `0` | The msr device is owned `root:root`; a non-root UID gets `EACCES` at the filesystem permission check before the capability check is even reached. |
| `capabilities.drop` | `[ALL]` | Start from nothing. |
| `capabilities.add` | `[SYS_RAWIO, SYS_ADMIN]` | `SYS_RAWIO` is what the msr driver itself checks for. `SYS_ADMIN` is needed for some of turbostat's other hardware queries (varies by CPU generation). |
| `privileged` | `false` | Everything else (seccomp, AppArmor, the rest of the capability set, device access beyond the one hostPath below) stays locked down. |
| `readOnlyRootFilesystem` | `true` | The app writes nothing to disk. |
| `allowPrivilegeEscalation` | `false` | Not needed once the above capabilities are granted directly. |

It also mounts the node's `/dev/cpu` directory read-only into the
container (`msrHostPath`, default `/dev/cpu`) — there is no way to expose
specific host hardware devices to a container other than a hostPath mount.

**Kubernetes RBAC is a separate concern from the above.** This chart
creates a `ServiceAccount` only to set `automountServiceAccountToken: false`
on it; it does **not** create any `Role`, `ClusterRole`, or binding, because
the exporter never calls the Kubernetes API.

### Going further: fully non-root

If your hosts have already loosened access to the msr device (e.g. a udev
rule that `chmod`s `/dev/cpu/*/msr` to be group- or world-readable, or
`chown`s it to a non-root user/group), you can override:

```yaml
securityContext:
  runAsUser: 1000
  runAsGroup: 1000
```

This is not the chart default because, on an unmodified host, it silently
fails (`permission denied` reading every MSR) rather than falling back —
turbostat's own error message when this happens is "try chown or chmod +r
/dev/cpu/*/msr, or run as root."

## Values

See [values.yaml](./values.yaml) for the full list. Notable ones:

- `image.tag` — defaults to `.Chart.AppVersion`. Override to pin an exact
  exporter version or track `main`.
- `background.enabled` / `background.intervalSeconds` /
  `defaultCollectSeconds` — how often and how long turbostat samples for.
- `basicAuth.*` — enable HTTP basic auth on `/metrics`. Use
  `basicAuth.existingSecret` in anything beyond local testing rather than
  the inline `basicAuth.password` value.
- `serviceMonitor.enabled` — set `true` if you run prometheus-operator.
- `msrHostPath` — override if your MSR devices live somewhere other than
  `/dev/cpu` (unusual).

## Releases

This chart is versioned independently from the exporter application. See
the repository root README's "Releasing" section for how chart releases
are cut and published.
