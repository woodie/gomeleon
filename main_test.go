package main

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const passingReport = `[{
  "SuiteName": "Something",
  "RunTime": 15000000,
  "SpecReports": [
    {
      "ContainerHierarchyTexts": ["checkAttachmentDir", "when the path is missing"],
      "LeafNodeText": "creates the directory",
      "LeafNodeType": "It",
      "State": "passed"
    },
    {
      "ContainerHierarchyTexts": ["checkAttachmentDir", "when the path is missing"],
      "LeafNodeText": "does not error",
      "LeafNodeType": "It",
      "State": "passed"
    },
    {
      "ContainerHierarchyTexts": ["checkAttachmentDir", "when the path is a symlink"],
      "LeafNodeText": "does not error",
      "LeafNodeType": "It",
      "State": "passed"
    }
  ]
}]`

const failingReport = `[{
  "SuiteName": "Something",
  "RunTime": 15000000,
  "SpecReports": [
    {
      "ContainerHierarchyTexts": ["checkAttachmentDir", "when the path is missing"],
      "LeafNodeText": "creates the directory",
      "LeafNodeType": "It",
      "State": "failed",
      "Failure": {
        "Message": "Expected file to exist",
        "Location": {"FileName": "main_test.go", "LineNumber": 42}
      }
    }
  ]
}]`

const skippingReport = `[{
  "SuiteName": "Something",
  "RunTime": 15000000,
  "SpecReports": [
    {
      "ContainerHierarchyTexts": ["checkAttachmentDir", "when the path is missing"],
      "LeafNodeText": "creates the directory",
      "LeafNodeType": "It",
      "State": "skipped"
    }
  ]
}]`

var _ = Describe("Gomeleon", func() {
	runReportStyle := func(raw string, style Style) string {
		path := writeTempReport(raw)
		var buf strings.Builder
		Expect(run(path, &buf, style)).To(Succeed())
		return buf.String()
	}
	runReport := func(raw string) string {
		return runReportStyle(raw, StyleClassic)
	}

	Describe("run", func() {
		var output string

		Context("with a passing report", func() {
			BeforeEach(func() { output = runReport(passingReport) })

			It("prints the suite name", func() {
				Expect(output).To(ContainSubstring("Something"))
			})

			It("indents container hierarchy", func() {
				Expect(output).To(ContainSubstring("checkAttachmentDir\n  when the path is missing"))
			})

			It("indents leaf nodes", func() {
				Expect(output).To(ContainSubstring("creates the directory"))
			})

			It("deduplicates shared hierarchy", func() {
				Expect(strings.Count(output, "when the path is missing")).To(Equal(1))
			})

			It("prints the summary", func() {
				Expect(output).To(ContainSubstring("3 examples, 0 failures"))
			})

			It("appends xcbeautify's 'Test Succeeded' / 'Tests Passed' footer", func() {
				Expect(output).To(ContainSubstring("Test Succeeded"))
				Expect(output).To(ContainSubstring("Tests Passed: 0 failed, 0 skipped, 3 total (0.0150 seconds)"))
			})
		})

		Context("with a failing report", func() {
			BeforeEach(func() { output = runReport(failingReport) })

			It("annotates the failed spec", func() {
				Expect(output).To(ContainSubstring("creates the directory (FAILED - 1)"))
			})

			It("prints the failures section", func() {
				Expect(output).To(ContainSubstring("Failures:"))
				Expect(output).To(ContainSubstring("Expected file to exist"))
				Expect(output).To(ContainSubstring("main_test.go:42"))
			})

			It("prints the failed examples list", func() {
				Expect(output).To(ContainSubstring("Failed examples:"))
			})

			It("prints the summary with failure count", func() {
				Expect(output).To(ContainSubstring("1 examples, 1 failure"))
			})

			It("switches the xcbeautify-style footer to 'Test Failed'", func() {
				Expect(output).To(ContainSubstring("Test Failed"))
				Expect(output).To(ContainSubstring("Tests Passed: 1 failed, 0 skipped, 1 total (0.0150 seconds)"))
			})
		})

		Context("with a skipping report", func() {
			BeforeEach(func() { output = runReport(skippingReport) })

			It("annotates the skipped spec", func() {
				Expect(output).To(ContainSubstring("creates the directory (SKIPPED)"))
			})

			It("prints the summary with skipped count", func() {
				Expect(output).To(ContainSubstring("1 examples, 0 failures, 1 skipped"))
			})

			It("folds the skip into the xcbeautify-style footer's skipped count", func() {
				Expect(output).To(ContainSubstring("Test Succeeded"))
				Expect(output).To(ContainSubstring("Tests Passed: 0 failed, 1 skipped, 1 total (0.0150 seconds)"))
			})
		})

		Context("when the report file is missing", func() {
			It("returns an error", func() {
				var buf strings.Builder
				Expect(run("/nonexistent/report.json", &buf, StyleClassic)).To(HaveOccurred())
			})
		})
	})

	Describe("color output", func() {
		Context("when not a TTY", func() {
			BeforeEach(func() { isTTY = false })
			AfterEach(func() { isTTY = false })

			It("omits ANSI codes from passing leaf nodes", func() {
				Expect(runReport(passingReport)).NotTo(ContainSubstring(prefix))
			})

			It("omits ANSI codes from the summary", func() {
				Expect(runReport(failingReport)).NotTo(ContainSubstring(prefix))
			})
		})

		Context("when a TTY", func() {
			BeforeEach(func() { isTTY = true })
			AfterEach(func() { isTTY = false })

			It("colors passing leaf nodes green", func() {
				Expect(runReport(passingReport)).To(ContainSubstring(prefix + green))
			})

			It("colors the passing summary green", func() {
				Expect(runReport(passingReport)).To(ContainSubstring(prefix + green))
			})

			It("colors failed leaf nodes red", func() {
				Expect(runReport(failingReport)).To(ContainSubstring(prefix + red))
			})

			It("colors the failing summary red", func() {
				Expect(runReport(failingReport)).To(ContainSubstring(prefix + red))
			})

			It("colors skipped leaf nodes cyan", func() {
				Expect(runReport(skippingReport)).To(ContainSubstring(prefix + cyan))
			})
		})
	})

	Describe("main routing", func() {
		Context("when a .json argument is given", func() {
			It("formats the report file directly", func() {
				path := writeTempReport(passingReport)
				var buf strings.Builder
				Expect(run(path, &buf, StyleClassic)).To(Succeed())
				Expect(buf.String()).To(ContainSubstring("Something"))
			})
		})

		Context("when runGinkgo writes a report", func() {
			It("uses a path outside the project directory", func() {
				Expect(ginkgoReportPath()).To(HavePrefix(os.TempDir()))
			})
		})
	})

	Describe("wantsVersion", func() {
		It("matches the long flag", func() {
			Expect(wantsVersion([]string{"--version"})).To(BeTrue())
		})

		It("does not match the short flag", func() {
			Expect(wantsVersion([]string{"-v"})).To(BeFalse())
		})

		It("matches regardless of position among other args", func() {
			Expect(wantsVersion([]string{"./...", "--version"})).To(BeTrue())
		})

		It("does not match on an empty argument list", func() {
			Expect(wantsVersion(nil)).To(BeFalse())
		})
	})

	Describe("parseStyle", func() {
		It("defaults to StyleClassic with no flags", func() {
			style, remaining := parseStyle([]string{"./..."})
			Expect(style).To(Equal(StyleClassic))
			Expect(remaining).To(Equal([]string{"./..."}))
		})

		It("recognizes -fd", func() {
			style, remaining := parseStyle([]string{"-fd", "./...", "-v"})
			Expect(style).To(Equal(StyleFd))
			Expect(remaining).To(Equal([]string{"./...", "-v"}))
		})

		It("recognizes -fs", func() {
			style, _ := parseStyle([]string{"-fs"})
			Expect(style).To(Equal(StyleFs))
		})

		It("recognizes -fv", func() {
			style, _ := parseStyle([]string{"-fv"})
			Expect(style).To(Equal(StyleFv))
		})

		It("recognizes --format documentation/spec/vitest", func() {
			style, remaining := parseStyle([]string{"--format", "spec", "./..."})
			Expect(style).To(Equal(StyleFs))
			Expect(remaining).To(Equal([]string{"./..."}))

			style, _ = parseStyle([]string{"--format", "documentation"})
			Expect(style).To(Equal(StyleFd))

			style, _ = parseStyle([]string{"--format", "vitest"})
			Expect(style).To(Equal(StyleFv))
		})

		It("leaves unrelated args untouched", func() {
			_, remaining := parseStyle([]string{"-v", "--race", "./..."})
			Expect(remaining).To(Equal([]string{"-v", "--race", "./..."}))
		})
	})

	Describe("style-specific rendering", func() {
		BeforeEach(func() { isTTY = false })

		Context("-fd (documentation, the pre-existing default look)", func() {
			It("prints the plain label with no glyph", func() {
				output := runReportStyle(passingReport, StyleFd)
				Expect(output).To(ContainSubstring("creates the directory"))
				Expect(output).NotTo(ContainSubstring("✔"))
			})

			It("annotates failures with (FAILED - n)", func() {
				output := runReportStyle(failingReport, StyleFd)
				Expect(output).To(ContainSubstring("creates the directory (FAILED - 1)"))
			})
		})

		Context("-fs (RSpec-style)", func() {
			It("prefixes passing specs with a checkmark", func() {
				output := runReportStyle(passingReport, StyleFs)
				Expect(output).To(ContainSubstring("✔"))
				Expect(output).To(ContainSubstring("creates the directory"))
			})

			It("prefixes failing specs with an X and keeps the FAILED detail", func() {
				output := runReportStyle(failingReport, StyleFs)
				Expect(output).To(ContainSubstring("✗"))
				Expect(output).To(ContainSubstring("(FAILED - 1)"))
			})

			It("prefixes skipped specs with a circle and SKIPPED detail", func() {
				output := runReportStyle(skippingReport, StyleFs)
				Expect(output).To(ContainSubstring("○"))
				Expect(output).To(ContainSubstring("(SKIPPED)"))
			})
		})

		Context("-fv (Vitest-style)", func() {
			It("uses a checkmark plus the millisecond duration, no FAILED-style detail", func() {
				output := runReportStyle(passingReport, StyleFv)
				// Check the leaf line itself, not the whole output -- the
				// shared footer legitimately contains "(... seconds)" too.
				Expect(output).To(ContainSubstring("✓ creates the directory 15ms\n"))
			})

			It("uses an × plus the millisecond duration for failures, no FAILED marker", func() {
				output := runReportStyle(failingReport, StyleFv)
				Expect(output).To(ContainSubstring("× creates the directory 15ms"))
				Expect(output).NotTo(ContainSubstring("(FAILED"))
			})

			It("uses a down-arrow for skipped specs", func() {
				output := runReportStyle(skippingReport, StyleFv)
				Expect(output).To(ContainSubstring("↓ creates the directory"))
			})
		})

		Context("all four styles", func() {
			It("still share the one xcbeautify-style footer", func() {
				for _, style := range []Style{StyleClassic, StyleFd, StyleFs, StyleFv} {
					output := runReportStyle(passingReport, style)
					Expect(output).To(ContainSubstring("Test Succeeded"))
					Expect(output).To(ContainSubstring("Tests Passed: 0 failed, 0 skipped, 3 total (0.0150 seconds)"))
				}
			})
		})
	})
})
