# ADR-002: Go and Cobra for the CLI

## Status

Accepted

## Context

ShipHold is a command-line tool that will run in two places: a developer's terminal and a CI runner. It talks to the Docker daemon, a reverse proxy, a container registry, and the GitHub API, and it must be trivially installable by someone who does not want to install a language runtime to use it.

The language choice was effectively made when the project was conceived, but it was never written down, and the version floor it implies was never examined at all.

## Options Considered

**Go.** Single static binary, no runtime dependency, first-class Docker and Kubernetes ecosystem libraries, excellent concurrency primitives for health-probe polling, and a standard library that covers HTTP, JSON, crypto, and structured logging without third-party packages. It is also the language the surrounding ecosystem — Docker, Traefik, Kubernetes, Cosign — is written in, which matters more than it sounds: the reference implementations of everything ShipHold integrates with are readable.

**Python.** Faster to write, and a rich ecosystem. Rejected on distribution: shipping a Python CLI to a CI runner means a runtime, a virtualenv, and a dependency-resolution step in the critical path of a deployment gate. A gate that can fail to install is a poor gate.

**Rust.** Comparable distribution story and stronger correctness guarantees. Rejected on time budget. The Docker and GitHub client ecosystems are less mature, and a ten-week part-time schedule has no room for fighting a borrow checker while also learning a domain.

For the CLI framework: **Cobra** against **`flag`** (standard library) and **urfave/cli**. The standard library is sufficient for flags but has no subcommand model, and ShipHold has seven subcommands with nested flags. Cobra is the de facto choice across this ecosystem (Docker, Kubernetes, Hugo, GitHub CLI), which means its conventions are already familiar to the people most likely to use this tool.

## Decision

Go, with Cobra for command structure.

**Language version floor: Go 1.24.** Not 1.27.

The repository currently declares `go 1.27.0`. Go 1.27 was released in August 2026 and is real and current, but declaring the newest available language version in a tool intended for other people to build has a cost that was not weighed: every contributor and every CI runner must have a toolchain at least that new, and older CI images will silently trigger a toolchain download or fail outright. Nothing in ShipHold's code uses a 1.25-or-later language feature.

The floor should be the oldest version whose features the project actually uses. `log/slog` (ADR-014) requires 1.21. Allowing comfortable room for the standard library features this project relies on, and matching what CI images typically ship, **1.24** is the right declaration. It should be raised only when a specific feature requires it, and the raise should be noted here.

**Direct dependencies are kept few and boring.** Cobra for commands, `gopkg.in/yaml.v3` for configuration, the official Docker client, `pgx` for PostgreSQL. Everything else comes from the standard library unless a specific case justifies otherwise. Each new direct dependency is a supply-chain surface in a tool whose entire purpose is supply-chain discipline; the irony of a careless dependency tree here is worth avoiding.

## Consequences

**Makes easy.** A single binary that CI can download and run. Native access to the Docker SDK. Goroutines and `context` for health polling with timeouts and cancellation — which ADR-008's verification predicate needs. Cross-compilation to Linux and macOS from one machine.

**Makes harder.** Go's error handling is verbose; ADR-004 exists partly to keep that verbosity from becoming noise. Generics are available but thin, so some repetition across adapters is expected and should be tolerated rather than abstracted around.

**Rules out.** Plugins loaded at runtime. Extending ShipHold means recompiling it, which is an acceptable trade for a tool whose value is determinism.

**Immediate actions.**

- Change `go.mod` to `go 1.24`. Verify with `go build ./...` on a 1.24 toolchain.
- Run `go mod tidy`. `gopkg.in/yaml.v3` is imported directly by `internal/config` but is currently listed under the indirect require block.
- Pin the CI toolchain to the declared floor, not to `stable`, so that accidental use of a newer language feature fails in CI rather than on a contributor's machine.

**Known wart, accepted.** The module path is `github.com/GunithaR/ShipHold` — mixed case, while all documentation writes `shiphold`. Mixed-case module paths are legal but awkward: the module cache escapes capitals (`!s!hip!hold`), and they interact poorly with case-insensitive filesystems. Renaming would break every existing import path for no functional gain. The decision is to keep it and note it here so that it reads as a considered trade rather than an oversight. The binary, the command name, and all prose remain lowercase `shiphold`.
