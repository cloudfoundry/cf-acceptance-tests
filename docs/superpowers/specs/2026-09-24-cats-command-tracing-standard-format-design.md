# CATS command tracing: standardize on Ginkgo's JSON report

**Date:** 2026-09-24
**Branch:** cats-command-tracing
**Status:** approved, pending implementation

## Background

An earlier proof-of-concept (commit `8819b081`, "feat: add opt-in cf command
tracing to CATS") added `helpers/cfcmdtrace/`, an opt-in profiler enabled with
`CATS_TRACE=1`. It monkey-patches `cf.Cf` to time every invocation and, at suite
end, prints four aligned-text tables to stdout.

The four questions the tables answer are settled and remain the goal:

1. **Slowest specs** — total runtime split into `cf` time vs. "other" (non-cf)
   time, with retry attempts.
2. **Time by command verb** — count / total / mean, grouped by `cf` verb.
3. **Slowest individual commands** — the worst single invocations.
4. **Repeated commands** — commands run many times, grouped by a normalized
   signature (GUIDs, generated names, hex, numeric suffixes collapsed).

What is *not* settled is the output mechanism. The PoC hand-formats four
20-row text tables (~170 lines in `report.go`). The preference is to use a
**standard solution** for the data, keeping only a short human-readable summary.

## Constraints discovered during investigation

- **Per-command timing has no standard substitute.** cf-test-helpers exposes a
  `Reporter` interface, but `Report(startTime, cmd)` fires only at command
  *start*, never end, and lives in an `internal/` package with no injection seam
  on `cf.Cf` (`vendor/.../cf-test-helpers/v2/internal/reporter.go`,
  `commandstarter/command_starter.go:27-33`). `CF_TRACE` emits per-HTTP-request
  traces, not per-command wall-clock, and cannot attribute a command to a spec.
  No stdlib mechanism (`runtime/trace`, `expvar`, `testing`) times subprocesses
  and aggregates. **Conclusion: the `cf.Cf` wrapper stays.**
- **Ginkgo's JSON report is the right standard artifact.** `rc.JSONReport`
  exists (`vendor/.../ginkgo/v2/types/config.go:100`); Ginkgo merges per-node
  reports into one monolithic file by default (opt-out at `config.go:544`); and
  each `ReportEntry`'s value is serialized into `Value.AsJSON`
  (`vendor/.../ginkgo/v2/types/report_entry.go:57`). So a `specTrace` passed to
  `AddReportEntry` appears as embedded JSON in the standard artifact, retrievable
  with a one-line `jq`. **No custom JSON emitter is needed.**

## Design

### Capture layer — unchanged

- The `cf.Cf` wrapper in `enable.go` and the `collector` / `cmdRecord` /
  `specTrace` records in `record.go` stay as-is.
- `AddReportEntry("cfcmdtrace", specTrace{…})` per spec and the
  `AddReportEntry("cfcmdtrace-suite", …)` for suite-setup calls stay — this is
  already how records cross parallel-node boundaries.
- `normalize.go` stays — the signature normalization is still needed for the
  repeated-commands rollup in the summary.

### Change 1 — enable Ginkgo's JSON report (standard artifact)

In `cats_suite_test.go`, alongside the existing `rc.JUnitReport` line, gated on a
configured artifacts directory **and** `cfcmdtrace.Enabled()`:

```go
if cfcmdtrace.Enabled() {
    rc.JSONReport = filepath.Join(
        Config.GetArtifactsDirectory(),
        fmt.Sprintf("cfcmdtrace-%d.json", GinkgoParallelProcess()))
}
```

The feature stays fully opt-in: the JSON is produced only when `CATS_TRACE=1`.
No CI command changes are required.

### Change 2 — shrink `report.go` to a compact human summary

Replace the four full tables with a compact summary: each of the four questions
as a few headline rows (top 5), still framed by the greppable
`===CATS-TRACE===` … `===END-CATS-TRACE===` markers, ending with a pointer to
the JSON file for full detail.

Removed:
- `CATS_TRACE_TOP` env var and `topN` (summary is fixed-small; full detail lives
  in the JSON).
- The display column-width constants and most of the `trunc` machinery.

Kept:
- The `flatCmd` flattening and the `aggregate` grouping helper (the summary still
  needs verb and signature rollups) — they simply print fewer rows.

### Data flow

```
cf.Cf wrapper → collector (per-command records)
             → AddReportEntry per spec / suite-setup
             ├─(a)→ Ginkgo merged cfcmdtrace-*.json   (full data; AI / jq / pandas)
             └─(b)→ ReportAfterSuite compact summary → stdout   (humans, in the log)
```

### What the JSON consumer sees

Per spec in `SpecReports[]`: the spec name, `RunTime`, `NumAttempts`, and
`ReportEntries[].Value.AsJSON` holding:

```json
{ "name": "...", "run_time_ns": 0, "attempts": 3,
  "cmds": [ { "verb": "push", "args_preview": "...", "start_ns": 0,
             "end_ns": 0, "exit_code": 0, "completed": true } ] }
```

All four questions are derivable from that. The README will carry a `jq`
one-liner for each.

### Testing

- Keep the capture and normalize unit tests unchanged.
- Adjust the report tests to the trimmed summary: the section-presence,
  repeated-command-collapse, and cf-vs-other-gap assertions stay; the `topN`
  test is removed with the feature.
- Add one test asserting a `specTrace` round-trips through `json.Marshal` into
  the shape the JSON consumer depends on (guards the wire contract).

### README

Replace the four-table description with: how to enable, where the
`cfcmdtrace-*.json` artifact lands, the four `jq` one-liners answering the
questions, and a note that a compact summary is also printed to the log. The
existing admin-password redaction warning stays.

## Scope and trade-offs

- Removing `CATS_TRACE_TOP` is a minor breaking change to an unreleased PoC —
  acceptable.
- Feature remains opt-in via `CATS_TRACE=1`; no behavior change for suites that
  don't set it.
- Analysis burden moves from Go code to a standard artifact, which is the
  explicit goal. The one cost is that the deepest views require a `jq`/AI step
  rather than being pre-rendered — mitigated by the compact in-log summary.
