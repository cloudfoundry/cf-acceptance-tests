package cfcmdtrace

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/onsi/gomega/gexec"
)

const defaultTopN = 20

// coll is the per-node command collector; set by Enable.
var coll *collector

// Enabled reports whether command tracing should run. Tracing is opt-in: it is
// off unless CATS_TRACE is set to a truthy value ("1" or "true").
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

func decodePayload(asJSON string) (specTrace, error) {
	var p specTrace
	err := json.Unmarshal([]byte(asJSON), &p)
	return p, err
}

// recoverPayload extracts a specTrace from a report entry. Parallel non-node-1
// entries are JSON-rehydrated (decode AsJSON); in-process ones carry the raw struct.
func recoverPayload(entry types.ReportEntry) (specTrace, bool) {
	if entry.Value.AsJSON != "" {
		if p, err := decodePayload(entry.Value.AsJSON); err == nil {
			return p, true
		}
	}
	if p, ok := entry.Value.GetRawValue().(specTrace); ok {
		return p, true
	}
	return specTrace{}, false
}

// Enable patches cf.Cf and registers Ginkgo report hooks. Call during tree
// construction, guarded by Enabled(). workflowhelpers auth/targeting is not traced.
func Enable(namePrefix string) {
	coll = newCollector()

	orig := cf.Cf
	cf.Cf = func(args ...string) *gexec.Session {
		hook := coll.record(args)
		sess := orig(args...)
		hook(sess)
		return sess
	}

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

// DrainSuiteSetup returns a specTrace for suite-level cf calls on this node.
// Call from the all-nodes SynchronizedAfterSuite body with AddReportEntry("cfcmdtrace-suite", ...).
func DrainSuiteSetup() specTrace {
	if coll == nil {
		return specTrace{}
	}
	return specTrace{Name: "<suite-setup>", Attempts: 1, Cmds: coll.drain()}
}
