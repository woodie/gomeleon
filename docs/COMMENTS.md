# Comments

Rationale, history, and design notes that used to live as multi-line comments
in the source. Organized by file, then by the function or code block each
note is attached to. The source itself now carries at most one short line at
any given spot -- anything longer that would previously have been a
multi-line `//` note lives here instead. When a code location kept its own
one-line comment, it's noted below so this stays a complete map of "why," not
a duplicate of what's already readable in the file.

## main.go

### `run`, sticky-error `fprintf`/`fprintln` closures
Kept a one-line comment in place: "Sticky-error: after the first write
failure, fprintf/fprintln become no-ops; that error is returned at the end."

Full history: `run` writes to an arbitrary `io.Writer` (`out`) many times
over the course of formatting a report, and a write failure partway through
shouldn't require a manual error check after every single call. The two
local closures capture the first error into `writeErr` and silently no-op on
every call after that, so the rest of `run`'s formatting logic keeps running
unconditionally instead of bailing out mid-format or writing anything
further to a writer that's already failed; the first error is then surfaced
through `run`'s own return value once formatting finishes.

### `run`, xcbeautify-style footer block
Kept a one-line comment in place: "Mirrors xcbeautify's real 'Test
Succeeded'/'Tests Passed' footer verbatim, folding Pending into its one
skipped count."

Full history: this footer's exact wording, punctuation, and structure are
lifted verbatim from a genuine `xcodebuild test` run through real
xcbeautify -- a green/red "Test Succeeded"/"Test Failed" headline, then a
"Tests Passed:" line that, despite the name, always reports all three counts
rather than just passing specs. xcbeautify/XCTest has no concept of a
separate "pending" bucket the way Ginkgo does, so this folds Ginkgo's
Pending and Skipped counts together into the one "skipped" number that line
expects.

### `run`, Ginkgo-style footer block
Kept a one-line comment in place: "Mirrors Ginkgo's own 'Ran X of Y Specs'
footer verbatim; X excludes Pending/Skipped specs."

Full history: likewise lifted verbatim from a genuine `ginkgo` run, for
parity alongside the xcbeautify-style footer above: "Ran X of Y Specs in N
seconds," where X deliberately excludes specs that never actually executed
(Pending and Skipped), followed by a "SUCCESS!"/"FAIL!" headline carrying
the full Passed/Failed/Pending/Skipped breakdown.

### `runGinkgo`, rename-error branch
Kept a one-line comment in place: "cmd already failed; that failure takes
precedence over this rename error below."

Full history: when both `cmd.Run()` and the subsequent `os.Rename` fail, the
rename error is already printed to stderr just above, but it isn't the error
`runGinkgo` should ultimately report -- `cmd`'s own failure (and its exit
code) is what the caller actually cares about, so control falls through to
the `err != nil` branch below instead of returning early on the rename
failure alone.
