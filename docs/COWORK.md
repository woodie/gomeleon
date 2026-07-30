# Working with ginkgo-fd (and ginkgo)

Cross-project conventions (git locks, sandbox toolchain) are in `~/workspace/woodie/docs/COWORK.md`.

`ginkgo-fd` started as a standalone wrapper: a small Go CLI that shells out to
`ginkgo`, captures its JSON report output, and reformats it as RSpec-style
"format documentation" — nested `Describe`/`Context`/`It` text instead of dots.
It depends on the real `ginkgo` CLI being installed separately and never
touches Ginkgo's internals; it only consumes the report file Ginkgo already
produces.

We're now also working upstream, adding a native `-fd` flag directly to
`onsi/ginkgo` (clone at `../ginkgo`, branch `add_fd_flag_v2`, tracked as
[PR #1670](https://github.com/onsi/ginkgo/pull/1670)) so the same RSpec-style
output is built into Ginkgo itself, no wrapper required. So a session may
touch either repo, or both — e.g. comparing wrapper output against the native
`-fd` flag's output on the same test suite.

`../lambada` (a small SMTP relay with its own Ginkgo suite) is a third,
unrelated repo we use as a real-world test consumer — running its tests
through both the wrapper and the native flag to compare output.

## Repo roles

- **`ginkgo-fd`** (this repo) — the wrapper tool. `module github.com/woodie/ginkgo-fd`.
  Own test suite ("GinkgoFd" in `main_test.go`/`helpers_test.go`).
- **`../ginkgo`** — fork of `onsi/ginkgo`. `origin` = `git@github.com:woodie/ginkgo.git`,
  `upstream` = `git@github.com:onsi/ginkgo.git`. Work happens on `add_fd_flag_v2`.
- **`../lambada`** — unrelated SMTP relay project, useful here only as a test
  suite with real log output (`log.Printf`/`log.Println` in `BeforeEach`
  blocks) to sanity-check that `-fd` output stays clean under SUT logging.

## Build & test in Go

Each repo (`ginkgo-fd`, `ginkgo`, `lambada`) has its own `go.mod`. To test
wrapper changes against a **locally modified** `ginkgo` (rather than whatever
version is in `go.sum`), link it in with a `go.work` file instead of editing
`go.mod`:

```
go work init .
go work use ./ginkgo-fd ../ginkgo
```

`ginkgo-fd/go.work` and `lambada/go.work` already exist and point at
`/Users/woodie/workspace/ginkgo` — so local edits to `ginkgo` are picked up
immediately by both, no `go.mod`/`replace` edits, no `go mod tidy`. Delete or
ignore the `go.work` file (it's gitignored) when you want to build against the
published `ginkgo` module again.

**Building/installing the `ginkgo` CLI itself:**

```bash
go install -a ./...
```

Run from inside `ginkgo/`. The `-a` is load-bearing — `go install ./...`
without it can silently skip rebuilding and leave you testing a stale binary
that doesn't reflect your latest edits. If `-fd` behavior doesn't change after
an edit, suspect a stale binary before suspecting the code.

**Finding where that binary lands:** use `go env GOPATH`, not `echo $GOPATH`.
The shell's `$GOPATH` env var is often unset, which makes `echo $GOPATH/bin`
print something misleading like `/bin`. The real answer is `$(go env GOPATH)/bin`.

**Running tests:**

```bash
ginkgo -fd          # native flag, in a repo with the linked ginkgo
ginkgo-fd           # wrapper, anywhere ginkgo is separately installed
go test ./...       # plain go test, no Ginkgo CLI needed
```

I (Cowork) don't have a Go toolchain in my sandbox — I can read/edit Go source
directly, and inspect git state (status/diff/log/remotes), but anything that
actually builds, installs, or runs Go (`go build`, `go install`, `go test`,
`ginkgo ...`) has to be run by you in your own terminal. I'll give you the
exact command; you run it and paste back the result.

## Working on the `ginkgo` fork specifically

- Push with the remote spelled out: `git push origin add_fd_flag_v2`. Don't
  rely on a bare `git push` — the branch's tracking ref has previously ended up
  pointed at `upstream/master` instead of `origin/add_fd_flag_v2`, which would
  push to the wrong place. If you want to fix that permanently:
  `git branch --set-upstream-to=origin/add_fd_flag_v2 add_fd_flag_v2`.
- **Never edit `CHANGELOG.md`.** That's onsi's call as maintainer, not ours.
  Equivalent summary content goes in the PR description or a linked issue
  instead.
- Direct file edits + commits are fine for working on the feature branch;
  just keep commits scoped (e.g. code change separate from doc updates) so the
  PR history stays easy to review.

## Debugging approach

- Before concluding Ginkgo's `-fd` reporter is leaking SUT output, check
  `reporters/default_reporter.go`'s `didRunFd` — in fd mode it only prints the
  container hierarchy and leaf labels, never `CapturedStdOutErr` or
  `CapturedGinkgoWriterOutput`. If output still looks noisy, the more likely
  culprits, in order: a stale CLI binary (see `-a` above), or the suite under
  test simply not exercising the code paths you think it is. Trace
  `internal/output_interceptor.go` → `internal/group.go`'s `run()` to confirm
  interception fully wraps each spec attempt before assuming a real capture
  bug.
- `lambada` is useful here specifically because its `BeforeEach` blocks call
  `log.Printf`/`log.Println` synchronously (SMTP relay logging) — a
  reasonable stand-in for "real" noisy test output when comparing wrapper vs.
  native `-fd` behavior.

## Renamed to `gomeleon`

The upstream `onsi/ginkgo` `add_fd_flag_v2` effort above was abandoned in
favor of `gorderly`/`kotidy`/`xctidy` (plain `go test`/Kotest/Quick-Nimble
output, no dependency on a separate CLI or a merged upstream PR). This
wrapper still has a real, distinct purpose though: it's the only tool in
the family that consumes real `ginkgo`'s own JSON report, for whichever
suites in this account still use actual Ginkgo rather than `spec`+`expect`.
Renamed away from `ginkgo-fd` since the name read as an upstream Ginkgo
flag/fork rather than a standalone tool — `biloba` was considered and
rejected (already a real, unrelated package at `github.com/onsi/biloba`,
from the same `onsi` org that maintains Ginkgo itself). `module
github.com/woodie/ginkgo-fd` → `github.com/woodie/gomeleon`, `GinkgoFd` →
`Gomeleon` throughout code/tests, binary/temp-file names, and README —
this repo's own historical entries above are left as-is since they're
accurate to what the tool was called at the time.

**Update**: brought to the same four-format surface `gorderly`/`xctidy` share
— Classic (default, no flag), `-fd` (RSpec documentation, the original
ginkgo-fd look, now reachable via a flag instead of being the only option),
`-fs` (RSpec spec — grayed-out label), `-fv` (Vitest — minimal glyph, no
trailing detail). Ported gorderly's architecture directly: one `Style` enum
plus per-state `colorizePass`/`colorizeFail`/`colorizePending`/`colorizeSkip`
functions that switch on it, no separate `Formatter` interface. Per explicit
instruction, there's a single shared footer across all four styles (the
existing xcbeautify-style `Test Succeeded`/`Tests Passed` block) — Ginkgo's
own native `Ran X of Y Specs`/`SUCCESS!` footer was dropped entirely rather
than kept or ported, and there's no separate Vitest-specific footer the way
gorderly's `-fv` has one. Also added a `Makefile` (`lint`/`test`/`check`),
which this repo never had — `make check`/`make test`/`make link` all failed
with "No rule to make target" before this, confirming the repo really had
been abandoned mid-build-out.
