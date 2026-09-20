# ADR-008: Docker and reverse-proxy traffic switching

## Status

**Proposed** — pending the Week-1 spike. Do not build load-bearing work on this until it is promoted to `Accepted`.

This ADR is deliberately left open. The decision below is the intended approach and the criteria by which it will be judged; the spike either confirms it or replaces it. An honest "this is the plan and here is how we will know if it is wrong" is worth more than a confident decision made without evidence.

## Context

Health-gated traffic switching is ShipHold's central mechanism. Everything else — readiness, provenance, rollback — arranges itself around the moment traffic moves from the old version to the new one. It is also the single riskiest implementation detail in the project, and the one most capable of invalidating the architecture if it turns out to be harder than assumed.

The requirement: a candidate container starts alongside the active one, is health-checked in isolation, and only then receives traffic — with zero dropped requests during the switch, and with the active version untouched if the candidate fails.

The difficulty is specific. Traefik's Docker provider derives routing configuration from **container labels**, and Docker labels are fixed at container creation. A running container's labels cannot be changed. So the obvious mental model — "relabel the container to move the route" — does not work, and the switch must happen some other way.

**Confidence note.** The above is my understanding of the constraint, drawn from Traefik's documented Docker-provider behaviour and community practice. It is the premise of this whole ADR, and it is the first thing the spike should verify, because if it is wrong the simplest option becomes viable again.

## Options Considered

**1. File provider with a weighted service.** Traefik runs with both a Docker provider (for service discovery) and a file provider watching a dynamic-configuration file. A weighted round-robin service names both versions; ShipHold rewrites the file to move weights from `blue=100, green=0` to `blue=0, green=100`. Traefik reloads the file without restarting.

Advantages: the switch is a file write, so it is fast and revertible; Traefik reloads dynamic configuration without dropping connections; and intermediate weights come free, so canary deployment later (see `extended-scope.md` §2.2) is a scheduling problem rather than a re-architecture. This appears to be the approach the Traefik community most commonly uses for blue/green and canary.

Costs: ShipHold must own a file on disk that Traefik watches, which is shared mutable state between two processes and needs atomic writes (write to a temp file, then `rename`) and a defined ownership story.

**2. Two routers with a priority flip.** Both versions expose routers matching the same rule; the higher-priority router wins. Switching means changing priority — which, with the Docker provider, means recreating a container, or, with the file provider, means writing a file. It largely collapses into option 1 with extra steps.

**3. Recreate the proxy-facing container.** Straightforward, and it is a restart rather than a switch. It drops connections, which defeats the purpose.

**4. Write directly to a Traefik API.** Traefik's dynamic configuration is not designed to be mutated through an API in the way this would require. Not pursued.

## Decision (provisional)

**Option 1: file provider with a weighted service**, subject to the spike below.

### The routing abstraction

The deployment domain never mentions Traefik. It depends on:

```go
type TrafficRouter interface {
    // Route the named service's traffic to this deployment.
    Activate(ctx context.Context, svc ServiceID, target DeploymentTarget) error

    // Which deployment currently receives traffic.
    Active(ctx context.Context, svc ServiceID) (DeploymentTarget, error)

    // Confirm the router is healthy and its configuration is loaded.
    Verify(ctx context.Context, svc ServiceID) error
}
```

`TraefikFileRouter` implements it. A second implementation — even a trivial one for tests — keeps the interface honest, per the DiffSage playbook's two-implementations rule. Nothing in `internal/domain/` or `internal/application/` may reference a label, a weight, or a file path.

### The switch sequence

```text
1. Start candidate container, not routed, on the shared network
2. Health-probe the candidate DIRECTLY (not through the proxy)
      endpoint, interval, failure_threshold, timeout — from policy
3. If unhealthy → remove candidate, active version untouched,
                  record FAILED, exit 5. Never switch.
4. Write router config atomically: active=0, candidate=100
5. Wait for the router to report the new configuration loaded
6. VERIFY THROUGH THE PROXY  (see below)
7. If verification fails → revert the file, re-verify the old version,
                           remove candidate, record FAILED
8. Stop the previous container
9. Record SUCCEEDED
```

Step 2 probing the container directly, and step 6 probing through the proxy, are deliberately different checks. The first asks *is this build healthy*; the second asks *did the routing change actually take effect*. Conflating them hides the class of bug where the app is fine and the proxy is pointing at the wrong place.

### What "verify routed traffic" means

`architecture.md` previously said "verify routed traffic" without defining it, which makes it unimplementable and untestable. The predicate:

```text
Send N requests (default 10) to the public endpoint through the proxy,
over a window of at most T (default 10s), with concurrency 1.

PASS   if all N return < 500 AND a response header or body marker
       identifies the candidate deployment.
FAIL   otherwise.
```

The identifying marker is the important half. Without it, verification passes when the proxy is still serving the *old* version perfectly well — the requests succeed and nothing has actually switched. ShipHold injects a `X-ShipHold-Deployment: <id>` response header via router middleware, or reads an application-provided build identifier. All values configurable; all recorded in provenance.

### Spike pass criteria

The Week-1 spike must demonstrate both, and record the results in this ADR:

```text
SUCCESS PATH
  Given  Traefik + sample-app:v1, v1 serving traffic
  And    sustained load through the public endpoint (e.g. hey -z 30s)
  When   v2 starts, passes health checks, and the route switches
  Then   ZERO 5xx and ZERO connection failures across the switch
  And    all responses after the switch identify v2
  And    v1 stops cleanly with no further errors

FAILURE PATH
  Given  the same setup
  When   v2 starts and fails its health check
  Then   the router config is never written
  And    v1 serves 100% of traffic throughout
  And    v2 is removed
  And    the failure is recorded with a reason
```

If option 1 fails these, try option 2 before changing anything else in the architecture. If both fail, the blue/green model itself needs revisiting — and finding that out in Week 1 costs a week, while finding it out in Week 6 costs the project.

## Consequences

**Makes easy.** Switching is a file write: fast, atomic, revertible. Canary weights are already expressible. The deployment domain stays proxy-agnostic, so substituting Caddy or nginx is one new adapter.

**Makes harder.** ShipHold owns a file that another process watches. That needs atomic writes, a defined owner, and a recovery story for a file left half-written by a killed process — which ADR-011's startup reconciliation must cover. The proxy's configuration becomes partly ShipHold-managed, and that boundary must be documented for operators.

**Rules out.** Any design where the deployment domain knows about Traefik. Switching by relabelling or recreating the active container. Declaring a deployment successful without verifying through the proxy.

**On promotion.** When the spike passes, change the status to `Accepted` and append a "Spike results" section with what was actually observed — including anything surprising. If it fails, supersede this ADR rather than quietly editing it; the record of a plan that did not survive contact with reality is worth keeping.

## References

- [Traefik: canary deployments with weighted load balancing](https://iximiuz.com/en/posts/traefik-canary-deployments-with-weighted-load-balancing/)
- [Docker Compose Tip #15: Blue-green deployments with Traefik](https://lours.me/posts/compose-tip-015-blue-green-deployments/)
- [Blue-Green-Deployment with Traefik — Traefik Labs community forum](https://community.traefik.io/t/blue-green-deployment-with-traefik/3737)
- [Simple Blue/Green Deployments with Traefik and Docker](https://frustrated.blog/2021/03/16/traefik_blue_green.html)

These are community sources, not primary documentation. Verify current behaviour against the Traefik documentation for the version actually being used before relying on any of it.
