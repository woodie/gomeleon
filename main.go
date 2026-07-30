package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"
)

var isTTY = term.IsTerminal(int(os.Stdout.Fd()))

func colorize(code, s string) string {
	if !isTTY {
		return s
	}
	return prefix + code + "m" + s + prefix + "0m"
}

const (
	prefix = "\033["
	red    = "31"
	green  = "32"
	yellow = "33"
	cyan   = "36"
	gray   = "90"
)

// Style picks the leaf-label rendering, matching gorderly's/xctidy's shared
// four-format surface exactly: Classic is the default (no flag), Fd/Fs/Fv
// are the RSpec/Mocha/Vitest looks. All four share one common footer below
// (the xcbeautify-style "Test Succeeded"/"Tests Passed" block) -- there's no
// per-style footer the way gorderly's -fv has its own Vitest-shaped one,
// and Ginkgo's own "Ran X of Y Specs"/"SUCCESS!" footer is dropped entirely
// rather than ported, in favor of that one shared block.
type Style int

const (
	StyleClassic Style = iota
	StyleFd
	StyleFs
	StyleFv
)

type SuiteReport struct {
	SuiteName   string       `json:"SuiteName"`
	RunTime     float64      `json:"RunTime"`
	SpecReports []SpecResult `json:"SpecReports"`
}

type SpecResult struct {
	ContainerHierarchyTexts []string `json:"ContainerHierarchyTexts"`
	LeafNodeText            string   `json:"LeafNodeText"`
	LeafNodeType            string   `json:"LeafNodeType"`
	State                   string   `json:"State"`
	RunTime                 float64  `json:"RunTime"`
	Failure                 *Failure `json:"Failure,omitempty"`
}

type Failure struct {
	Message  string   `json:"Message"`
	Location Location `json:"Location"`
}

type Location struct {
	FileName   string `json:"FileName"`
	LineNumber int    `json:"LineNumber"`
}

func formatSeconds(ns float64) string {
	d := time.Duration(ns)
	if d < time.Second {
		return fmt.Sprintf("%.4f", d.Seconds())
	}
	return fmt.Sprintf("%.2f", d.Seconds())
}

func formatDuration(ns float64) string {
	return formatSeconds(ns) + " seconds"
}

// colorizePass/colorizeFail/colorizePending/colorizeSkip each switch on style
// internally, matching gorderly's colorizePass/colorizeFail/colorizeSkip --
// Classic colors only the glyph and leaves the name mostly uncolored, Fs
// grays the name out, Fv keeps its glyph minimal with no trailing detail,
// and Fd (today's only style before this) is unchanged: the whole label as
// one colored block. Ginkgo's Pending has no Vitest/gorderly equivalent, so
// Fv folds it into the same dim treatment as Skipped.

func colorizePass(style Style, name string, runTime float64) string {
	switch style {
	case StyleClassic:
		return colorize(green, "✔") + " " + name + " (" + colorize(green, formatDuration(runTime)) + ")"
	case StyleFs:
		return colorize(green, "✔") + " " + colorize(gray, name)
	case StyleFv:
		return colorize(green, "✓") + " " + name
	default: // StyleFd
		return colorize(green, name)
	}
}

func colorizeFail(style Style, name string, n int) string {
	switch style {
	case StyleClassic:
		return colorize(red, "✗") + " " + colorize(red, fmt.Sprintf("%s (FAILED - %d)", name, n))
	case StyleFs:
		return colorize(red, "✗") + " " + colorize(gray, name) + " " + colorize(red, fmt.Sprintf("(FAILED - %d)", n))
	case StyleFv:
		return colorize(red, "×") + " " + name
	default: // StyleFd
		return colorize(red, fmt.Sprintf("%s (FAILED - %d)", name, n))
	}
}

func colorizePending(style Style, name string) string {
	switch style {
	case StyleClassic:
		return colorize(yellow, "○") + " " + colorize(yellow, fmt.Sprintf("%s (PENDING)", name))
	case StyleFs:
		return colorize(yellow, "○") + " " + colorize(gray, name) + " " + colorize(yellow, "(PENDING)")
	case StyleFv:
		return colorize(yellow, "↓") + " " + name
	default: // StyleFd
		return colorize(yellow, fmt.Sprintf("%s (PENDING)", name))
	}
}

func colorizeSkip(style Style, name string) string {
	switch style {
	case StyleClassic:
		return colorize(cyan, "○") + " " + colorize(cyan, fmt.Sprintf("%s (SKIPPED)", name))
	case StyleFs:
		return colorize(cyan, "○") + " " + colorize(gray, name) + " " + colorize(cyan, "(SKIPPED)")
	case StyleFv:
		return colorize(cyan, "↓") + " " + name
	default: // StyleFd
		return colorize(cyan, fmt.Sprintf("%s (SKIPPED)", name))
	}
}

func run(reportPath string, out io.Writer, style Style) error {
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", reportPath, err)
	}

	var reports []SuiteReport
	if err := json.Unmarshal(data, &reports); err != nil {
		return fmt.Errorf("cannot parse JSON: %w", err)
	}

	totalSpecs := 0
	totalFailed := 0
	totalPending := 0
	totalSkipped := 0
	var totalRunTime float64
	var failures []failureEntry

	// Sticky-error: after the first write failure, fprintf/fprintln become no-ops; that error is returned at the end.
	var writeErr error
	fprintf := func(format string, a ...any) {
		if writeErr != nil {
			return
		}
		_, writeErr = fmt.Fprintf(out, format, a...)
	}
	fprintln := func(a ...any) {
		if writeErr != nil {
			return
		}
		_, writeErr = fmt.Fprintln(out, a...)
	}

	for _, report := range reports {
		fprintln(report.SuiteName)
		totalRunTime += report.RunTime

		var prevHierarchy []string

		for _, spec := range report.SpecReports {
			if spec.LeafNodeType != "It" {
				continue
			}

			totalSpecs++
			switch spec.State {
			case "failed", "panicked", "interrupted":
				totalFailed++
			case "pending":
				totalPending++
			case "skipped":
				totalSkipped++
			}

			hierarchy := spec.ContainerHierarchyTexts
			divergeAt := 0
			for divergeAt < len(prevHierarchy) && divergeAt < len(hierarchy) &&
				prevHierarchy[divergeAt] == hierarchy[divergeAt] {
				divergeAt++
			}

			for i := divergeAt; i < len(hierarchy); i++ {
				fprintf("%s%s\n", strings.Repeat("  ", i), hierarchy[i])
			}

			depth := len(hierarchy)
			indent := strings.Repeat("  ", depth)
			label := spec.LeafNodeText

			switch spec.State {
			case "failed", "panicked":
				n := len(failures) + 1
				label = colorizeFail(style, label, n)
				failures = append(failures, failureEntry{
					n:        n,
					full:     append(append([]string{report.SuiteName}, hierarchy...), spec.LeafNodeText),
					message:  spec.Failure.Message,
					location: fmt.Sprintf("%s:%d", spec.Failure.Location.FileName, spec.Failure.Location.LineNumber),
				})
			case "pending":
				label = colorizePending(style, label)
			case "skipped":
				label = colorizeSkip(style, label)
			default:
				label = colorizePass(style, label, spec.RunTime)
			}

			fprintf("%s%s\n", indent, label)
			prevHierarchy = hierarchy
		}

		fprintln()
	}

	if len(failures) > 0 {
		fprintln("Failures:")
		for _, f := range failures {
			fprintf("\n  %d) %s\n", f.n, strings.Join(f.full, " "))
			for _, line := range strings.Split(strings.TrimSpace(f.message), "\n") {
				fprintf("     %s\n", line)
			}
			fprintf("     # %s\n", f.location)
		}
		fprintln()
	}

	fprintf("Finished in %s\n", formatDuration(totalRunTime))

	parts := []string{fmt.Sprintf("%d examples", totalSpecs)}
	if totalFailed > 0 {
		parts = append(parts, fmt.Sprintf("%d failure", totalFailed))
	} else {
		parts = append(parts, "0 failures")
	}
	if totalPending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", totalPending))
	}
	if totalSkipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", totalSkipped))
	}
	summary := strings.Join(parts, ", ")
	if totalFailed > 0 {
		summary = colorize(red, summary)
	} else {
		summary = colorize(green, summary)
	}
	fprintln(summary)

	if len(failures) > 0 {
		fprintln("\nFailed examples:")
		for _, f := range failures {
			fprintf("\n  # %s\n", strings.Join(f.full, " "))
		}
	}

	// Mirrors xcbeautify's real "Test Succeeded"/"Tests Passed" footer verbatim, folding Pending into its one skipped count.
	verdict := "Test Succeeded"
	verdictColor := green
	if totalFailed > 0 {
		verdict = "Test Failed"
		verdictColor = red
	}
	fprintln()
	fprintln(colorize(verdictColor, verdict))
	testsPassedLine := fmt.Sprintf(
		"Tests Passed: %d failed, %d skipped, %d total (%s seconds)",
		totalFailed, totalPending+totalSkipped, totalSpecs, formatSeconds(totalRunTime),
	)
	fprintln(colorize(verdictColor, testsPassedLine))

	return writeErr
}

type failureEntry struct {
	n        int
	full     []string
	message  string
	location string
}

func ginkgoReportPath() string {
	return filepath.Join(os.TempDir(), "gomeleon-report.json")
}

// parseStyle pulls gomeleon's own format flags out of args, returning
// whatever's left untouched -- everything else (test paths, -v, etc.) gets
// forwarded straight through to the real ginkgo binary.
func parseStyle(args []string) (Style, []string) {
	style := StyleClassic
	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-fd":
			style = StyleFd
		case "-fs":
			style = StyleFs
		case "-fv":
			style = StyleFv
		case "--format":
			if i+1 < len(args) {
				switch args[i+1] {
				case "documentation":
					style = StyleFd
				case "spec":
					style = StyleFs
				case "vitest":
					style = StyleFv
				}
				i++
			}
		default:
			remaining = append(remaining, args[i])
		}
	}
	return style, remaining
}

func runGinkgo(args []string, style Style) int {
	reportFile := "gomeleon-report.json"
	reportPath := ginkgoReportPath()
	defer func() {
		if err := os.Remove(reportPath); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "gomeleon: cleanup: %v\n", err)
		}
	}()

	ginkgoArgs := append([]string{"--json-report=" + reportFile}, args...)
	cmd := exec.Command("ginkgo", ginkgoArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if renameErr := os.Rename(reportFile, reportPath); renameErr != nil {
		fmt.Fprintf(os.Stderr, "gomeleon: %v\n", renameErr)
		if err == nil {
			return 1
		}
		// cmd already failed; that failure takes precedence over this rename error below.
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if _, statErr := os.Stat(reportPath); statErr == nil {
				_, _ = fmt.Fprintln(os.Stdout)
				if runErr := run(reportPath, os.Stdout, style); runErr != nil {
					fmt.Fprintf(os.Stderr, "gomeleon: %v\n", runErr)
				}
			}
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "gomeleon: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintln(os.Stdout)
	if err := run(reportPath, os.Stdout, style); err != nil {
		fmt.Fprintf(os.Stderr, "gomeleon: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	style, args := parseStyle(os.Args[1:])

	if len(args) == 1 && strings.HasSuffix(args[0], ".json") {
		if err := run(args[0], os.Stdout, style); err != nil {
			fmt.Fprintf(os.Stderr, "gomeleon: %v\n", err)
			os.Exit(1)
		}
		return
	}

	os.Exit(runGinkgo(args, style))
}
