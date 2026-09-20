# ADR-001: Layered architecture and the dependency rule

## Status

Accepted

## Context

ShipHold touches Git, GitHub Actions, a container registry, the Docker daemon, a reverse proxy, and PostgreSQL. That is six external systems, each with its own SDK, failure modes, and rate of change. A tool this size fails in a predictable way: vendor types leak into the core, the deployment logic becomes untestable without a Docker daemon, and swapping any one dependency turns into a rewrite.

The project also has a specific testability requirement that follows from what it is. A deployment safety engine whose safety rules can only be tested by performing real deployments cannot be trusted, because nobody will run the full suite often enough. The policy rules, the state machine, and the readiness logic must be testable in milliseconds with no infrastructure at all.

A third force is timeline. A ten-week part-time budget cannot absorb an elaborate architecture. Whatever structure is chosen has to be cheap to follow and obvious to violate.

## Options Considered

**Flat package layout.** Everything in a handful of packages, no enforced direction. Fastest to start, and for a genuinely small tool it is often correct. Rejected because the six external systems make the "small tool" assumption false, and because the boundary between evidence and evaluation — the project's central idea — would have nowhere to live.

**Full hexagonal architecture with an explicit ports layer.** A dedicated `ports/` package declaring every interface, with infrastructure implementing them and application consuming them. This is what the previous draft of `architecture.md` described: a five-layer stack with `Ports / Interfaces` sitting between domain and infrastructure.

This was rejected for a reason specific to Go. Go interfaces are satisfied implicitly, and the idiom is that **the consumer declares the interface it needs**, in its own package, as narrowly as possible. A central ports package inverts that: it produces wide interfaces shaped by what implementations offer rather than by what callers need, and it creates a package every layer must import, which is the opposite of decoupling. It is a faithful translation of a pattern from languages with explicit interface implementation, and it translates badly.

**Layered architecture with consumer-defined interfaces.** Four layers, one-way dependencies, and interfaces declared at the point of use.

## Decision

Four layers, with dependencies flowing in one direction only:

```text
cmd/            binary, exit codes
  │
  ▼
internal/cli/            parse, render, translate errors to exit codes
  │
  ▼
internal/application/    orchestration; declares the ports it needs
  │
  ▼
internal/domain/         types and rules; imports nothing but stdlib
  ▲
  │  (implements)
internal/infrastructure/ concrete adapters
```

Four rules:

1. **`internal/domain/` imports only the standard library.** No Docker SDK, no database driver, no HTTP client, no YAML library, no Cobra. If a domain type needs a struct tag to be serialised, the serialisation belongs elsewhere.
2. **Ports are declared by their consumers.** `application/check` declares the evidence-collector interface it needs. `application/deploy` declares the deployment-target interface it needs. The interfaces live next to the code that calls them, and they are as narrow as that call site permits.
3. **`internal/infrastructure/` depends on `internal/domain/` and on nothing else inside the project.** Adapters return domain types. An adapter that imports an application package is a layering violation.
4. **`internal/cli/` contains no business logic.** It parses flags, calls one application service, renders the result, and maps errors to exit codes. If a `cli` file contains a policy rule, a retry loop, or a state transition, it is in the wrong layer.

Dependencies are wired in exactly one place: a composition root at `internal/cli/deps.go`, written **before the second command exists**, not after the eighth.

## Consequences

**Makes easy.** Domain and policy tests run with no infrastructure. Application tests use fakes for the ports they declare. Replacing PostgreSQL, Traefik, or the registry touches one package. New commands wire dependencies through the composition root rather than reassembling the chain by hand.

**Makes harder.** Some data must be mapped between an SDK's types and domain types instead of passed straight through. This is the cost of the boundary and it is being paid on purpose.

**Rules out.** Domain types carrying `yaml:` or `db:` struct tags. Vendor types in application-service signatures. Any "just this once" import from domain into infrastructure — the first one deletes the rule.

**Enforcement.** The dependency rule is checkable and should be checked in CI rather than trusted:

```bash
# domain must not import anything outside the standard library
go list -deps ./internal/domain/... | grep -E '^(github|gopkg|golang.org/x)\.' && exit 1

# infrastructure must not import application
go list -f '{{.ImportPath}} {{join .Imports " "}}' ./internal/infrastructure/... \
  | grep 'internal/application' && exit 1
```

A rule that is only in a document is a rule that will be broken in month two.

**Note on `internal/`.** Placing all packages under `internal/` means nothing outside this module can import them. That is deliberate for v1: it keeps every package's API private while the design settles, and promoting a package to a public path later is a one-directional move that can be made when something is actually stable. It is also Go's compile-time enforced equivalent of the `src/` layout convention from the DiffSage playbook.
