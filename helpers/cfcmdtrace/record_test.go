package cfcmdtrace

import (
	"testing"
	"time"
)

func TestVerbOf(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"push", "app"}, "push"},
		{[]string{"-v", "service", "x"}, "service"},
		{[]string{}, "<none>"},
	}
	for _, c := range cases {
		if got := verbOf(c.in); got != c.want {
			t.Fatalf("verbOf(%v)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestArgsPreviewTruncates(t *testing.T) {
	got := argsPreview([]string{"push", "reallylongappname"}, 8)
	if len([]rune(got)) > 8 {
		t.Fatalf("preview not truncated: %q", got)
	}
}

func TestCollectorRecordsCompletion(t *testing.T) {
	c := newCollector()
	start := time.Now()
	c.CommandStarted("push myapp", start)
	c.CommandCompleted("push myapp", 250*time.Millisecond, 0)
	recs := c.drain()
	if len(recs) != 1 {
		t.Fatalf("want 1 record, got %d", len(recs))
	}
	if !recs[0].Completed || recs[0].Verb != "push" || recs[0].EndNs <= recs[0].StartNs {
		t.Fatalf("bad record: %+v", recs[0])
	}
	if recs[0].ExitCode != 0 {
		t.Fatalf("want exit 0, got %d", recs[0].ExitCode)
	}
}

func TestCollectorMatchesCompletionByArgs(t *testing.T) {
	c := newCollector()
	now := time.Now()
	c.CommandStarted("push a", now)
	c.CommandStarted("delete b", now)
	c.CommandCompleted("delete b", 10*time.Millisecond, 1)
	recs := c.drain()
	var pushRec, deleteRec cmdRecord
	for _, r := range recs {
		switch r.Verb {
		case "push":
			pushRec = r
		case "delete":
			deleteRec = r
		}
	}
	if pushRec.Completed {
		t.Fatalf("push should still be open: %+v", pushRec)
	}
	if !deleteRec.Completed || deleteRec.ExitCode != 1 {
		t.Fatalf("delete completion mismatched: %+v", deleteRec)
	}
}
