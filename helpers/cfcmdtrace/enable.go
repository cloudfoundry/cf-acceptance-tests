package cfcmdtrace

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
)

const defaultTopN = 20

var coll *collector

func Enabled() bool {
	switch os.Getenv("CATS_TRACE") {
	case "1", "true":
		return true
	default:
		return false
	}
}

func topN() int {
	if v := os.Getenv("CATS_TRACE_TOP"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultTopN
}

func recoverPayload(entry types.ReportEntry) (specTrace, bool) {
	if entry.Value.AsJSON != "" {
		var p specTrace
		if err := json.Unmarshal([]byte(entry.Value.AsJSON), &p); err == nil {
			return p, true
		}
	}
	if p, ok := entry.Value.GetRawValue().(specTrace); ok {
		return p, true
	}
	return specTrace{}, false
}

func Enable(namePrefix string) {
	coll = newCollector()

	cf.RegisterObserver(coll)

	ginkgo.ReportAfterEach(func(report ginkgo.SpecReport) {
		ginkgo.AddReportEntry("cfcmdtrace", specTrace{
			Name:      report.FullText(),
			RunTimeNs: int64(report.RunTime),
			Attempts:  report.NumAttempts,
			Cmds:      coll.drain(),
		})
	})

	ginkgo.ReportAfterSuite("cfcmdtrace summary", func(report ginkgo.Report) {
		var specs []specTrace
		for _, spec := range report.SpecReports {
			for _, entry := range spec.ReportEntries {
				if entry.Name != "cfcmdtrace" && entry.Name != "cfcmdtrace-suite" {
					continue
				}
				if p, ok := recoverPayload(entry); ok {
					specs = append(specs, p)
				}
			}
		}

		// Write to stdout, not GinkgoWriter: in a parallel run only node 1's
		// ReportAfterSuite fires and its GinkgoWriter output is dropped without -v.
		fmt.Fprint(os.Stdout, formatReport(specs, namePrefix, topN()))
	})
}

func DrainSuiteSetup() specTrace {
	if coll == nil {
		return specTrace{}
	}
	return specTrace{Name: "<suite-setup>", Attempts: 1, Cmds: coll.drain()}
}
