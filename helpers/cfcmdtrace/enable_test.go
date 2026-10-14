package cfcmdtrace

import (
	"encoding/json"
	"testing"

	"github.com/onsi/ginkgo/v2/types"
)

func TestRecoverPayloadFromAsJSON(t *testing.T) {
	p := specTrace{Name: "s", RunTimeNs: 7, Attempts: 2, Cmds: []cmdRecord{{Verb: "push", Completed: true}}}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var entry types.ReportEntry
	entry.Value.AsJSON = string(b)
	got, ok := recoverPayload(entry)
	if !ok || got.Name != "s" || got.RunTimeNs != 7 || got.Attempts != 2 || len(got.Cmds) != 1 || got.Cmds[0].Verb != "push" {
		t.Fatalf("AsJSON recovery failed: ok=%v got=%+v", ok, got)
	}
}
