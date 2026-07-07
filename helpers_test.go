package main

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
)

func writeTempReport(content string) string {
	f, err := os.CreateTemp(GinkgoT().TempDir(), "report-*.json")
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()
	if _, err := f.WriteString(content); err != nil {
		panic(err)
	}
	return f.Name()
}
