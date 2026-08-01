package main

// gomeleonVersion is bumped by hand at each tagged release, matching
// gorderly's own gomeleonVersion/gorderlyVersion pattern rather than
// xctidy's git-describe-derived Version.swift -- gomeleon's primary install
// path is `go install github.com/woodie/gomeleon@latest`, a module-proxy
// fetch with no .git metadata to describe, so the version has to already be
// the right string in the committed source at tag time.
const gomeleonVersion = "1.1.0"

// wantsVersion only recognizes the long flag, unlike gorderly's/xctidy's
// --version/-v pair -- gomeleon already forwards -v to the real `ginkgo`
// CLI as its own verbose flag (see parseStyle's "leaves unrelated args
// untouched" case), so claiming -v here would silently break `gomeleon -v
// ./mypackage`.
func wantsVersion(args []string) bool {
	for _, a := range args {
		if a == "--version" {
			return true
		}
	}
	return false
}
