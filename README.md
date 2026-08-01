# gomeleon

[![go.mod version](https://img.shields.io/github/go-mod/go-version/woodie/gomeleon)](https://github.com/woodie/gomeleon)
[![CI](https://github.com/woodie/gomeleon/actions/workflows/go.yml/badge.svg)](https://github.com/woodie/gomeleon/actions/workflows/go.yml)
[![Release](https://img.shields.io/github/v/release/woodie/gomeleon.svg)](https://github.com/woodie/gomeleon/releases/latest)
[![License](https://img.shields.io/github/license/woodie/gomeleon.svg)](LICENSE)

![Example Screenshot](docs/example.png)

RSpec/Mocha/Vitest-style output for Go that works with
[Ginkgo](https://github.com/onsi/ginkgo)/[Gomega](https://github.com/onsi/gomega)
by wrapping the real `ginkgo` CLI, or reading its JSON report directly.
This is a great enhancement for teams already on Ginkgo/Gomega. Get that
same at-a-glance clarity without giving up Ginkgo's own assertions or
your existing suite.

## Installation

Install Ginkgo if you haven't already:

```
go install github.com/onsi/ginkgo/v2/ginkgo@latest
```

Then install gomeleon:

```
go install github.com/woodie/gomeleon@latest
```

Or build locally:

```
go mod tidy
go build -o gomeleon
mv gomeleon ~/go/bin/
```

## Usage

Run as a wrapper around `ginkgo`, passing any arguments through:

```
gomeleon
gomeleon ./...
gomeleon -v ./mypackage
```

Or format an existing report file directly:

```
gomeleon report.json
```

Flags: default (no flag) renders the classic style (glyph plus full detail);
`-fd` renders RSpec's documentation format; `-fs` renders Mocha/Jest's spec
format; `-fv` renders Vitest's tree format. `--format documentation`,
`--format spec`, and `--format vitest` are the long forms:

```
gomeleon -fd ./...
gomeleon -fs ./...
gomeleon -fv ./...
gomeleon --format spec ./...
```

### Version

```
gomeleon --version
```

Prints the installed version and exits immediately, without waiting on
stdin or running `ginkgo`. Long form only -- `-v` is already `ginkgo`'s
own verbose flag, forwarded straight through.

## Output styles

Four named styles, each matching a convention from a familiar test runner.
All four share one footer (the xcbeautify-style `Test Succeeded`/`Tests
Passed` block).

| Flag | Convention | Look |
|---|---|---|
|   | Our base formatter | Glyph + full detail, failures add `(FAILED - N)` |
| -fd | RSpec's doc format | Plain colored label, no glyph |
| -fs | Mocha's spec format | Glyph + grayed-out label |
| -fv | Vitest's own tree | Minimal glyph, no trailing detail |

Sample leaf lines for a passing, a failing, and a skipped spec, across all four formats:

```
# (none) — Classic
✔ creates the directory (0.0100 seconds)
✗ creates the directory (FAILED - 1)
○ creates the directory (SKIPPED)

# -fd — documentation
creates the directory
creates the directory (FAILED - 1)
creates the directory (SKIPPED)

# -fs — spec
✔ creates the directory
✗ creates the directory (FAILED - 1)
○ creates the directory (SKIPPED)

# -fv — vitest
✓ creates the directory
× creates the directory
↓ creates the directory
```

Full sample output (Classic):

```
Something
  checkAttachmentDir
    when the path is missing
      ✔ creates the directory (0.0100 seconds)
      ✔ does not error (0.0100 seconds)
    when the path is a symlink
      ✔ does not error (0.0100 seconds)

Finished in 0.0300 seconds
3 examples, 0 failures

Test Succeeded
Tests Passed: 0 failed, 0 skipped, 3 total (0.0300 seconds)
```

## Things to know

`gomeleon` is strictly a formatter over `ginkgo`'s own output, not a
testing framework, so it doesn't prescribe how to structure specs. For how
to write Ginkgo/Gomega tests, see
[Ginkgo's own docs](https://onsi.github.io/ginkgo/).

## Development

```
make build    # go build -o gomeleon
make install  # builds, then moves the binary to ~/go/bin/
make test     # verbose, dogfoods gomeleon on its own Ginkgo suite in -fs style
make lint     # golangci-lint
make check    # terse: silent on success, full log on failure
```

`make test`/`make check` shell out to the real `ginkgo` CLI (not `go test`
directly) since gomeleon's own purpose is reformatting that CLI's output --
install it with `go install github.com/onsi/ginkgo/v2/ginkgo@latest` if it
isn't already on your `PATH`.

Cutting a release: bump `gomeleonVersion` in `version.go` by hand before
tagging, matching `gorderly`'s own release process -- `gomeleon`'s primary
install path is `go install github.com/woodie/gomeleon@latest`, a
module-proxy fetch with no `.git` metadata to describe, so the version
string has to already be correct in the committed source at the tagged
commit.
