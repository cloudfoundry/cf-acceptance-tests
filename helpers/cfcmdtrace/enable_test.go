package cfcmdtrace

import (
	"encoding/json"
	"testing"

	"github.com/onsi/ginkgo/v2/types"
)

func TestDecodePayloadRoundTrip(t *testing.T) {
	p := specTrace{Name: "spec", RunTimeNs: 42, Attempts: 2, Cmds: []cmdRecord{{Verb: "push", Completed: true}}}
	asJSON, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodePayload(string(asJSON))
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "spec" || got.RunTimeNs != 42 || got.Attempts != 2 || len(got.Cmds) != 1 || got.Cmds[0].Verb != "push" {
		t.Fatalf("round-trip lost data: %+v", got)
	}
}

func TestRecoverPayloadFromAsJSON(t *testing.T) {
	p := specTrace{Name: "s", RunTimeNs: 7, Attempts: 1, Cmds: []cmdRecord{{Verb: "push"}}}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var entry types.ReportEntry
	entry.Value.AsJSON = string(b)
	got, ok := recoverPayload(entry)
	if !ok || got.Name != "s" || got.RunTimeNs != 7 || len(got.Cmds) != 1 {
		t.Fatalf("AsJSON recovery failed: ok=%v got=%+v", ok, got)
	}
}
