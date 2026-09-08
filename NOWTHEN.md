---
kind: tool
forge: github
tracker: github-issues
test: make test
---

A Go CLI that generates Kubernetes manifests for GitOps workflows, distributed as
release binaries (linux/darwin, amd64/arm64) plus `go install`. Alpha, and the
README says so; the internals are mid-rewrite. There is nothing to provision and
no service to deploy.

**Releasing is not merging, and it is exacting.** A release happens when a `v*`
tag is pushed, and `.github/workflows/release.yaml` fails the run unless two
things already match the tag: `internal/cmd/version.txt` contains the version,
and the first line of `Changes.md` is exactly `## v<version>  <date>` — where the
date is *today* in US Central at the moment the tag is pushed. A correct
changelog written yesterday fails today. `prepare.yaml` runs the same checks on
`release/*` branches as a dry run that publishes nothing.

The two workflows are kept byte-identical up to the publish steps, and
`test.yaml` enforces that with a diff — so editing one means editing the other or
CI goes red.

Every push runs golangci-lint and `go test ./...`. Changes land on `master`
through pull requests.

Docs are Material for MkDocs under `docs/`, built with `mkdocs build --strict`
and deployed to GitHub Pages by the same `v*` tag push that cuts a release.

To exercise the CLI, pass the directory rather than changing into it — e.g.
`go run ./ validate examples/guestbook`.
