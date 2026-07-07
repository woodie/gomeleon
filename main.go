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

func run(reportPath string, out io.Writer) error {
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

	// Sticky-error writers: once a write to out fails, subsequent calls are
	// no-ops and the first error is returned at the end of run().
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
				label = fmt.Sprintf("%s (FAILED - %d)", label, n)
				label = colorize(red, label)
				failures = append(failures, failureEntry{
					n:        n,
					full:     append(append([]string{report.SuiteName}, hierarchy...), spec.LeafNodeText),
					message:  spec.Failure.Message,
					location: fmt.Sprintf("%s:%d", spec.Failure.Location.FileName, spec.Failure.Location.LineNumber),
				})
			case "pending":
				label = colorize(yellow, fmt.Sprintf("%s (PENDING)", label))
			case "skipped":
				label = colorize(cyan, fmt.Sprintf("%s (SKIPPED)", label))
			default:
				label = colorize(green, label)
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

	// Real xcbeautify's own run-results footer, lifted verbatim from a
	// genuine `xcodebuild test` run: a green/red headline, then a
	// "Tests Passed:" line that -- despite the name -- always lists all
	// three counts, not just passes. xcbeautify/XCTest has no separate
	// "pending" bucket the way Ginkgo does, so Pending and Skipped specs
	// are folded together into its one "skipped" count here.
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

	// Real Ginkgo's own default-reporter footer, lifted verbatim from a
	// genuine `ginkgo` run, for good measure: "Ran X of Y Specs in N
	// seconds" -- X excludes specs that didn't actually execute (Pending
	// and Skipped) -- then a "SUCCESS!"/"FAIL!" headline with the full
	// Passed/Failed/Pending/Skipped breakdown.
	ranSpecs := totalSpecs - totalPending - totalSkipped
	passedSpecs := ranSpecs - totalFailed
	ginkgoVerdict := "SUCCESS!"
	if totalFailed > 0 {
		ginkgoVerdict = "FAIL!"
	}
	fprintln()
	fprintf("Ran %d of %d Specs in %s\n", ranSpecs, totalSpecs, formatDuration(totalRunTime))
	fprintln(colorize(verdictColor, fmt.Sprintf(
		"%s -- %d Passed | %d Failed | %d Pending | %d Skipped",
		ginkgoVerdict, passedSpecs, totalFailed, totalPending, totalSkipped,
	)))

	return writeErr
}

type failureEntry struct {
	n        int
	full     []string
	message  string
	location string
}

func ginkgoReportPath() string {
	return filepath.Join(os.TempDir(), "ginkgo-fd-report.json")
}

func runGinkgo(args []string) int {
	reportFile := "ginkgo-fd-report.json"
	reportPath := ginkgoReportPath()
	defer func() {
		if err := os.Remove(reportPath); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "ginkgo-fd: cleanup: %v\n", err)
		}
	}()

	ginkgoArgs := append([]string{"--json-report=" + reportFile}, args...)
	cmd := exec.Command("ginkgo", ginkgoArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if renameErr := os.Rename(reportFile, reportPath); renameErr != nil {
		fmt.Fprintf(os.Stderr, "ginkgo-fd: %v\n", renameErr)
		if err == nil {
			return 1
		}
		// cmd already failed; report that failure below rather than masking
		// it with the rename error.
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if _, statErr := os.Stat(reportPath); statErr == nil {
				_, _ = fmt.Fprintln(os.Stdout)
				if runErr := run(reportPath, os.Stdout); runErr != nil {
					fmt.Fprintf(os.Stderr, "ginkgo-fd: %v\n", runErr)
				}
			}
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "ginkgo-fd: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintln(os.Stdout)
	if err := run(reportPath, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "ginkgo-fd: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	args := os.Args[1:]

	if len(args) == 1 && strings.HasSuffix(args[0], ".json") {
		if err := run(args[0], os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "ginkgo-fd: %v\n", err)
			os.Exit(1)
		}
		return
	}

	os.Exit(runGinkgo(args))
}
