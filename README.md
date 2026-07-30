# gomeleon

[![go.mod version](https://img.shields.io/github/go-mod/go-version/woodie/gomeleon)](https://github.com/woodie/gomeleon)
[![CI](https://github.com/woodie/gomeleon/actions/workflows/go.yml/badge.svg)](https://github.com/woodie/gomeleon/actions/workflows/go.yml)
[![Release](https://img.shields.io/github/v/release/woodie/gomeleon.svg)](https://github.com/woodie/gomeleon/releases/latest)
[![License](https://img.shields.io/github/license/woodie/gomeleon.svg)](LICENSE)

![Example Screenshot](docs/example.png)

The `gomeleon` command uses [Ginkgo](https://github.com/onsi/ginkgo) under the hood to emulate the style of [RSpec](https://github.com/rspec/rspec) "format documentation" output.

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

Sample output:

```
Gomeleon
  run
    with a passing report
      prints the suite name
      indents container hierarchy
      indents leaf nodes
      deduplicates shared hierarchy
      prints the summary
      appends xcbeautify's 'Test Succeeded' / 'Tests Passed' footer
      appends Ginkgo's own 'Ran X of Y Specs' / 'SUCCESS!' footer
    with a failing report
      annotates the failed spec
      prints the failures section
      prints the failed examples list
      prints the summary with failure count
      switches the xcbeautify-style footer to 'Test Failed'
      switches the Ginkgo-style footer to 'FAIL!' and counts the failure
    with a skipping report
      annotates the skipped spec
      prints the summary with skipped count
      folds the skip into the xcbeautify-style footer's skipped count, still 'Test Succeeded'
      excludes the skipped spec from 'Ran X of Y' in the Ginkgo-style footer
    when the report file is missing
      returns an error
  color output
    when not a TTY
      omits ANSI codes from passing leaf nodes
      omits ANSI codes from the summary
    when a TTY
      colors passing leaf nodes green
      colors the passing summary green
      colors failed leaf nodes red
      colors the failing summary red
      colors skipped leaf nodes cyan
  main routing
    when a .json argument is given
      formats the report file directly
    when runGinkgo writes a report
      uses a path outside the project directory

Finished in 0.0086 seconds
27 examples, 0 failures

Test Succeeded
Tests Passed: 0 failed, 0 skipped, 27 total (0.0086 seconds)

Ran 27 of 27 Specs in 0.0086 seconds
SUCCESS! -- 27 Passed | 0 Failed | 0 Pending | 0 Skipped
```
