# ADR-0004: Binary distribution via GoReleaser

- **Status:** Accepted (2026-06-20)
- **Supersedes:** —
- **Related:** [ADR-0001 (interface + registry architecture)](0001-interface-registry-architecture.md),
  [README.md](../../README.md)

## Context

selo ships a CLI (`cmd/selo`), but until now the only install path was `go install
github.com/inovacc/selo/cmd/selo@latest` — which requires a Go toolchain on the user's machine and
builds from source. Non-Go users, CI runners without Go, and anyone who just wants a binary had no
turnkey option, and `selo version` reported nothing meaningful for source-installed builds (no
embedded version/commit/date).

The repository already delegates releases to the `inovacc/workflows` reusable release workflow, which
runs GoReleaser on a `v*` tag — but it ran GoReleaser in *binary* mode with **no `.goreleaser.yaml`
config**, so no archives, checksums, or per-platform binaries were actually published.

We wanted installable, prebuilt binaries for the common OS/arch combinations, published automatically
on tag, with version metadata baked in — without adding a bespoke build/release script to maintain.

## Decision

Add a **`.goreleaser.yaml`** config and let the existing `inovacc/workflows` reusable release
workflow publish on every `v*` tag:

- **GoReleaser builds the matrix** linux/darwin/windows × amd64/arm64, producing per-platform
  archives, a `checksums.txt`, and a source archive, attached to the GitHub Release for the tag.
- **Version metadata via ldflags.** The build injects version/commit/date into `main.build*`
  variables (`-ldflags "-X main.buildVersion=… -X main.buildCommit=… -X main.buildDate=…"`), so
  `selo version` reports the real tag/commit/date in release builds (source/`go install` builds fall
  back to their existing defaults).
- **Local parity tasks.** New `Taskfile` targets — `release:check` (validate the config),
  `release:snapshot` (build the full matrix locally without publishing), and `release` (cut a real
  release) — let a maintainer dry-run the pipeline before tagging.
- **Trigger stays the tag.** Pushing `vX.Y.Z` is the single entry point; the reusable workflow owns
  auth and upload, so the repo carries only the declarative config.

## Consequences

**Positive**
- Users can download a prebuilt binary for their platform from the GitHub Releases page — no Go
  toolchain required.
- `selo version` is meaningful in distributed builds (auditable version/commit/date).
- Releases are reproducible and automated from a tag; `release:snapshot` allows a local rehearsal.
- The reusable org workflow keeps release credentials and upload logic out of this repo.

**Negative / trade-offs**
- The build matrix and archive layout are now a maintained artifact (`.goreleaser.yaml`); changing
  targets or packaging requires editing it.
- Release output depends on the `inovacc/workflows` reusable workflow contract — a breaking change
  upstream can break releases here.
- ldflags variable names (`main.build*`) couple the config to `cmd/selo`'s `main` package; renaming
  them requires updating both places in lockstep.

## Alternatives considered

- **Keep `go install`-only** — rejected: excludes non-Go users and CI without Go, and leaves
  `selo version` blank in distributed builds.
- **A hand-written release script (build loop + `gh release upload`)** — rejected: reinvents what
  GoReleaser already does (matrix, archives, checksums, changelog) and is more to maintain.
- **A separate GitHub Actions release job in this repo** — rejected: duplicates the org's
  `inovacc/workflows` reusable release and would re-implement its auth/upload handling.
