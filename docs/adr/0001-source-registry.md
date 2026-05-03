# ADR-0001: Keep the `source.RegisterSource` extension point

- **Status:** Accepted
- **Date:** 2026-05-03

## Context

`pkg/source` ships three template sources: `local`, `github`, `toptal`.
The `SourceManager` was originally constructed by hardcoding those three
in `NewSourceManager`. During the May 2026 cleanup we extracted a small
factory registry (`SourceFactory`, `RegisterSource`, `buildSource`) so
`NewSourceManager` looks names up in a `map[string]SourceFactory`.

We have **no internal plan** to add a fourth source. The three current
ones cover every templating need we ship.

## Decision

Keep the registry. Do not inline the factories back into
`NewSourceManager`, even though it would simplify the call graph.

## Rationale

Downstream users — customers running their own private template
collections (internal GitLab, company HTTP endpoint, alternative GitHub
mirrors) — may want to register a source without forking the project.
The registry gives them a one-call extension point:

```go
source.RegisterSource("internal", func(_, _ string) (source.Source, error) {
    return myInternalSource(), nil
})
```

Cost of keeping it:

- One file (`pkg/source/registry.go`, ~25 LOC).
- One indirection inside `NewSourceManager` (`buildSource(name, ...)`).
- A package-level `var registry` map.

Cost of removing it: small now, but the moment a customer asks, we'd
either re-introduce the same pattern or push them to fork. Re-adding
later costs more than keeping (call sites change, tests change).

## Consequences

- New first-party sources still need to be added in two places: register
  in `init()` of their file, and add the name to the slice inside
  `NewSourceManager`. That's intentional — the manager controls
  *priority order*, which is a curated decision.
- The registry is package-global mutable state. Acceptable because
  registration happens at `init()` time and is monotonic (we never
  unregister).
- If a customer registers a source whose name collides with a built-in,
  their factory replaces ours. Document this if/when we publicize the
  extension point.

## Revisit if

- We pass two years with no third-party source registration. At that
  point the registry is dead weight and should be inlined.
- We need per-source configuration richer than `(localPath,
  templateURL)`. The factory signature would need rethinking, and the
  registry might be better replaced by a typed config struct.
