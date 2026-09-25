package cfcmdtrace

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type specTrace struct {
	Name      string      `json:"name"`
	RunTimeNs int64       `json:"run_time_ns"`
	Attempts  int         `json:"attempts"`
	Cmds      []cmdRecord `json:"cmds"`
}

func dur(ns int64) string { return fmt.Sprintf("%.1fs", time.Duration(ns).Seconds()) }

// Display column widths for the summary tables. ArgsPreview is captured at 160
// chars (record.go); these govern how much of that is shown.
const (
	specNameColW = 60
	cmdColW      = 72
	sigColW      = 72
	cmdSpecColW  = 30
)

func cmdDurNs(r cmdRecord) int64 {
	if !r.Completed {
		return 0
	}
	return r.EndNs - r.StartNs
}

type flatCmd struct {
	rec  cmdRecord
	spec string
	sig  string
}

// agg accumulates count/total/distinct-specs for a group of commands keyed by
// verb or signature, so both aggregate tables share one grouping pass.
type agg struct {
	count int
	total int64
	specs map[string]bool
}

func aggregate(all []flatCmd, key func(flatCmd) string) map[string]*agg {
	m := map[string]*agg{}
	for _, f := range all {
		a := m[key(f)]
		if a == nil {
			a = &agg{specs: map[string]bool{}}
			m[key(f)] = a
		}
		a.count++
		a.total += cmdDurNs(f.rec)
		a.specs[f.spec] = true
	}
	return m
}

func formatReport(specs []specTrace, namePrefix string, topN int) string {
	var b strings.Builder
	b.WriteString("\n===CATS-TRACE===\n")

	b.WriteString("\nSlowest specs (runtime | cf | other | attempts):\n")
	type specRow struct {
		name               string
		runtime, cf, other int64
		attempts           int
	}
	rows := make([]specRow, 0, len(specs))
	for _, s := range specs {
		var cf int64
		for _, c := range s.Cmds {
			cf += cmdDurNs(c)
		}
		other := s.RunTimeNs - cf
		if other < 0 {
			other = 0
		}
		attempts := s.Attempts
		if attempts == 0 {
			attempts = 1
		}
		rows = append(rows, specRow{s.Name, s.RunTimeNs, cf, other, attempts})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].runtime > rows[j].runtime })
	for i, r := range rows {
		if i >= topN {
			break
		}
		b.WriteString(fmt.Sprintf("  %-*s %8s | %8s | %8s | x%d\n",
			specNameColW, trunc(r.name, specNameColW), dur(r.runtime), dur(r.cf), dur(r.other), r.attempts))
	}

	all := []flatCmd{}
	rePrefix := compilePrefix(namePrefix)
	for _, s := range specs {
		for _, c := range s.Cmds {
			all = append(all, flatCmd{c, s.Name, signatureWith(strings.Fields(c.ArgsPreview), rePrefix)})
		}
	}

	b.WriteString("\nTime by command verb (count | total | mean):\n")
	byVerb := aggregate(all, func(f flatCmd) string { return f.rec.Verb })
	verbs := make([]string, 0, len(byVerb))
	for k := range byVerb {
		verbs = append(verbs, k)
	}
	sort.Slice(verbs, func(i, j int) bool { return byVerb[verbs[i]].total > byVerb[verbs[j]].total })
	for i, v := range verbs {
		if i >= topN {
			break
		}
		a := byVerb[v]
		mean := int64(0)
		if a.count > 0 {
			mean = a.total / int64(a.count)
		}
		b.WriteString(fmt.Sprintf("  %-24s %5d | %8s | %8s\n", v, a.count, dur(a.total), dur(mean)))
	}

	b.WriteString("\nSlowest individual commands:\n")
	sort.Slice(all, func(i, j int) bool { return cmdDurNs(all[i].rec) > cmdDurNs(all[j].rec) })
	for i, f := range all {
		if i >= topN {
			break
		}
		b.WriteString(fmt.Sprintf("  %8s  %-*s  [%s]\n",
			dur(cmdDurNs(f.rec)), cmdColW, trunc(f.rec.ArgsPreview, cmdColW), trunc(f.spec, cmdSpecColW)))
	}

	b.WriteString("\nRepeated commands (count | total | mean | #specs):\n")
	bySig := aggregate(all, func(f flatCmd) string { return f.sig })
	sigs := make([]string, 0, len(bySig))
	for k, a := range bySig {
		if a.count >= 2 {
			sigs = append(sigs, k)
		}
	}
	sort.Slice(sigs, func(i, j int) bool { return bySig[sigs[i]].total > bySig[sigs[j]].total })
	for i, s := range sigs {
		if i >= topN {
			break
		}
		a := bySig[s]
		mean := a.total / int64(a.count)
		b.WriteString(fmt.Sprintf("  %5d | %8s | %8s | %3d  %s\n",
			a.count, dur(a.total), dur(mean), len(a.specs), trunc(s, sigColW)))
	}

	b.WriteString("\n===END-CATS-TRACE===\n")
	return b.String()
}

// trunc caps s to n display columns, marking a cut with a trailing ellipsis.
// The returned width is still exactly n, keeping table columns aligned.
func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
