package cfcmdtrace

import (
	"strings"
	"testing"
)

func sampleSpecs() []specTrace {
	return []specTrace{
		{Name: "slow spec", RunTimeNs: 10_000_000_000, Attempts: 3, Cmds: []cmdRecord{
			{Verb: "push", ArgsPreview: "push CATS-1-APP-aaaaaaaaaaaaaaaa", StartNs: 0, EndNs: 4_000_000_000, Completed: true},
			{Verb: "push", ArgsPreview: "push CATS-2-APP-bbbbbbbbbbbbbbbb", StartNs: 0, EndNs: 3_000_000_000, Completed: true},
		}},
		{Name: "fast spec", RunTimeNs: 1_000_000_000, Attempts: 1, Cmds: []cmdRecord{
			{Verb: "delete", ArgsPreview: "delete x", StartNs: 0, EndNs: 500_000_000, Completed: true},
		}},
	}
}

func TestFormatReportHasAllSections(t *testing.T) {
	out := formatReport(sampleSpecs(), "CATS", 20)
	for _, marker := range []string{
		"===CATS-TRACE===",
		"Slowest specs",
		"Time by command verb",
		"Slowest individual commands",
		"Repeated commands",
		"===END-CATS-TRACE===",
	} {
		if !strings.Contains(out, marker) {
			t.Fatalf("missing section %q in:\n%s", marker, out)
		}
	}
}

func TestFormatReportShowsAttempts(t *testing.T) {
	out := formatReport(sampleSpecs(), "CATS", 20)
	if !strings.Contains(out, "x3") {
		t.Fatalf("attempt count not surfaced:\n%s", out)
	}
}

func TestFormatReportComputesOtherGap(t *testing.T) {
	out := formatReport(sampleSpecs(), "CATS", 20)
	var specLine string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "slow spec") {
			specLine = line
			break
		}
	}
	if specLine == "" {
		t.Fatalf("no slow-spec row found in:\n%s", out)
	}
	// slow spec: 10s runtime, 7s cf → 3s other, on the spec's own row
	if !strings.Contains(specLine, "3.0s") {
		t.Fatalf("other-gap not on slow-spec row: %q\nfull:\n%s", specLine, out)
	}
}

func TestFormatReportCollapsesRepeatedPushes(t *testing.T) {
	out := formatReport(sampleSpecs(), "CATS", 20)
	if !strings.Contains(out, "push <PREFIX>-N-<RESOURCE>-<RAND>") {
		t.Fatalf("repeated pushes not collapsed:\n%s", out)
	}
}

func TestFormatReportRespectsTopN(t *testing.T) {
	out := formatReport(sampleSpecs(), "CATS", 1)
	if strings.Contains(out, "delete x") {
		t.Fatalf("topN=1 did not truncate individual commands:\n%s", out)
	}
}
