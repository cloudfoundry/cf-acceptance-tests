package cfcmdtrace

import (
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func execTrue() *exec.Cmd { return exec.Command("true") }

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
	RegisterTestingT(t)
	c := newCollector()
	sess, err := gexec.Start(execTrue(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	hook := c.record([]string{"push", "myapp"})
	hook(sess)
	sess.Wait(5 * time.Second)
	// allow the Exited goroutine to run
	time.Sleep(50 * time.Millisecond)
	recs := c.drain()
	if len(recs) != 1 {
		t.Fatalf("want 1 record, got %d", len(recs))
	}
	if !recs[0].Completed || recs[0].Verb != "push" || recs[0].EndNs <= recs[0].StartNs {
		t.Fatalf("bad record: %+v", recs[0])
	}
}
