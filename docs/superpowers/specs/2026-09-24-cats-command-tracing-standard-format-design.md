# CATS command tracing: standardize on Ginkgo's JSON report

**Date:** 2026-09-24
**Branch:** cats-command-tracing
**Status:** superseded in part — see note below

> **Update (2026-09-25):** The capture mechanism described here (monkey-patching
> `cf.Cf`) has been replaced. Tracing now uses the opt-in command observer added
> to `cf-test-helpers` (`cf.RegisterObserver`), which covers every cf command the
> library runs — including `CfRedact`, `CfSilent`, `CfWithStdin`, and
> workflowhelpers auth/targeting — with redaction applied inside the library.
> References to the "`cf.Cf` wrapper" below are historical; the collector, record
> shape, normalization, and in-process summary are unchanged. See
> `helpers/cfcmdtrace/` on branch `cats-command-tracing-observer`.

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

What is being added is a **standard machine artifact** for these answers. The
PoC hand-formats four text tables in-process (~170 lines in `report.go`) and
prints them at suite end. That human summary is kept as-is — enabling tracing
must always yield a readable result. Alongside it, the suite will also emit
Ginkgo's standard JSON report so an AI or `jq`/pandas can analyze the raw
records and compare across runs without anyone reading raw JSON by hand.

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

### Change 2 — keep the in-process human summary (unchanged)

**Guiding principle: enabling `CATS_TRACE=1` must produce a human-readable
summary at the end of the run, automatically.** Humans are never expected to read
the JSON — that is the machine artifact. So the existing `ReportAfterSuite`
summary in `report.go` stays: the four tables, framed by the greppable
`===CATS-TRACE===` … `===END-CATS-TRACE===` markers, printed to stdout at suite
end whenever tracing is on.

- **`CATS_TRACE_TOP` is retained** — it sizes the summary tables (default 20).
  It is *not* removed.
- The table formatting, `flatCmd` flattening, and `aggregate` grouping stay as-is
  (post the earlier line-level cleanups: `rePrefix` hoisted, `decodePayload`
  inlined, `sanitize` redaction fixed).
- Optional, low-risk touch: append a one-line pointer to the JSON artifact path
  after the summary so a reader knows where the full data is. This is additive.

No standalone rendering tool is built. The summary is produced by one in-process
code path; the JSON serves AI/`jq`/pandas separately.

### Data flow

```
cf.Cf wrapper → collector (per-command records)
             → AddReportEntry per spec / suite-setup
             ├─(a)→ Ginkgo merged cfcmdtrace-*.json   (full data; AI / jq / pandas)
             └─(b)→ ReportAfterSuite summary → stdout  (humans, in the log; sized by CATS_TRACE_TOP)
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

- Keep the capture, normalize, and report unit tests as they are — including the
  `topN`/`CATS_TRACE_TOP` behavior test, since that env var is retained.
- Add one test asserting a `specTrace` round-trips through `json.Marshal` into
  the shape the JSON consumer depends on (guards the wire contract that the
  standard artifact exposes).

### README

Keep the existing description of the summary and `CATS_TRACE_TOP`. Add: that
enabling tracing now also writes a standard `cfcmdtrace-*.json` under the
artifacts directory, and the four `jq` one-liners that answer the questions from
it (for AI/script analysis). The existing admin-password redaction warning stays.

## Scope and trade-offs

- `CATS_TRACE_TOP` and the full in-process summary are **retained** — enabling
  tracing always yields a human-readable result with no extra step.
- The JSON report is purely additive: a standard machine artifact for AI/`jq`
  analysis and cross-run comparison, produced only when `CATS_TRACE=1`.
- Feature remains opt-in via `CATS_TRACE=1`; no behavior change for suites that
  don't set it.
- No standalone rendering tool — one in-process path produces the summary,
  avoiding a second binary to build and keep in sync with the record shape.
