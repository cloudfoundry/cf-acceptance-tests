package cfcmdtrace

import (
	"strings"
	"sync"
	"time"

	"github.com/onsi/gomega/gexec"
)

type cmdRecord struct {
	Verb        string `json:"verb"`
	ArgsPreview string `json:"args_preview"`
	StartNs     int64  `json:"start_ns"`
	EndNs       int64  `json:"end_ns"`
	ExitCode    int    `json:"exit_code"`
	Completed   bool   `json:"completed"`
}

type collector struct {
	mu      sync.Mutex
	records []*cmdRecord
}

func newCollector() *collector {
	return &collector{}
}

// sanitize redacts the CF admin password from create-service-broker args.
func sanitize(args []string) []string {
	if verbOf(args) == "create-service-broker" && len(args) >= 4 {
		out := make([]string, len(args))
		copy(out, args)
		out[3] = "<REDACTED>"
		return out
	}
	return args
}

func (c *collector) record(args []string) func(*gexec.Session) {
	rec := &cmdRecord{
		Verb:        verbOf(args),
		ArgsPreview: argsPreview(sanitize(args), 160),
		StartNs:     time.Now().UnixNano(),
	}
	c.mu.Lock()
	c.records = append(c.records, rec)
	c.mu.Unlock()

	return func(sess *gexec.Session) {
		if sess == nil {
			return
		}
		go func() {
			<-sess.Exited
			c.mu.Lock()
			rec.EndNs = time.Now().UnixNano()
			rec.ExitCode = sess.ExitCode()
			rec.Completed = true
			c.mu.Unlock()
		}()
	}
}

func (c *collector) drain() []cmdRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]cmdRecord, 0, len(c.records))
	for _, r := range c.records {
		out = append(out, *r)
	}
	c.records = nil
	return out
}

func verbOf(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return "<none>"
}

func argsPreview(args []string, max int) string {
	s := strings.Join(args, " ")
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}
	return s
}
