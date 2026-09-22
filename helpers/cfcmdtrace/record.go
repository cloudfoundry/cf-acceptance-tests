package cfcmdtrace

import (
	"strings"
	"sync"
	"time"
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

// CommandStarted / CommandCompleted implement cf.Observer; secrets are redacted by cf-test-helpers before we see them.
func (c *collector) CommandStarted(redactedArgs string, startTime time.Time) {
	args := cmdFields(redactedArgs)
	rec := &cmdRecord{
		Verb:        verbOf(args),
		ArgsPreview: argsPreview(args, 160),
		StartNs:     startTime.UnixNano(),
	}
	c.mu.Lock()
	c.records = append(c.records, rec)
	c.mu.Unlock()
}

// CommandCompleted matches completion to the oldest not-yet-completed record
// with the same redacted identity. Concurrent commands with identical args
// complete in start order, which is correct for the aggregate questions we
// answer even if an individual pairing is swapped.
func (c *collector) CommandCompleted(redactedArgs string, elapsed time.Duration, exitCode int) {
	preview := argsPreview(cmdFields(redactedArgs), 160)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, rec := range c.records {
		if rec.Completed || rec.ArgsPreview != preview {
			continue
		}
		rec.EndNs = rec.StartNs + elapsed.Nanoseconds()
		rec.ExitCode = exitCode
		rec.Completed = true
		return
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

// cmdFields strips the leading "cf" token: the observer delivers "cf push ...",
// but verb extraction and normalization expect args only ("push ...").
func cmdFields(redactedArgs string) []string {
	fields := strings.Fields(redactedArgs)
	if len(fields) > 0 && fields[0] == "cf" {
		return fields[1:]
	}
	return fields
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
