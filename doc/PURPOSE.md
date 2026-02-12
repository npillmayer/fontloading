# fontfind: Purpose, Status, and Direction

As of February 12, 2026.

## Purpose

`fontfind` is a Go library for discovering, matching, and loading scalable font files by descriptor:

- `pattern` (font family/name pattern)
- `style` (normal/italic/oblique)
- `weight` (light/normal/bold/etc.)

Core model types are in `font.go`:

- `Descriptor` describes what to search for.
- `ScalableFont` describes a found font with `fs.FS` + `Path` for reading bytes.

The package also includes:

- matching heuristics from filenames and variant names (`match.go`)
- a font registry/singleton cache abstraction (`fontregistry/registry.go`)
- resolver pipeline orchestration with async promise-like API (`locate/resolve.go`)

## Resolver Architecture

Typeface resolution is provider-based. Current resolver packages:

- `locate/fallbackfont`: embedded packaged fonts (`go:embed`)
- `locate/systemfont`: local/system font lookup
- `locate/googlefont`: Google Fonts API lookup + local caching

`ResolveTypeface` accepts one or more `FontLocator` functions and returns the first successful match.

## Current Status

This repository is early-stage and actively being refactored.

Evidence:

- very small history (4 commits total)
- commits are recent (February 10-11, 2026)
- no tags/releases yet
- no top-level `README.md`

Recent commit messages:

- `Initial commit`
- `First refactoring`
- `Renamed package`

## Build/Test Health Snapshot

From `go test ./...` on February 12, 2026 (after test refactoring):

- root package: no tests
- `fontregistry`: tests pass
- `locate`: tests pass
- `locate/googlefont`: tests pass
- `locate/systemfont`: no tests
- `locate/fallbackfont`: no tests

Test behavior notes:

- Google-font package tests are fully offline by default (no real API key, no real network).
- Google API response in tests is fixture-driven (`locate/googlefont/testdata/webfonts.json`).
- Cache download test is an offline unit test via fake I/O.

## Recent Progress

Completed in current refactor cycle:

- Added and wired `systemfont.IO` abstraction for fontconfig list lookup path.
- Reworked `TestFCFind` to use fixture-driven resolver flow via `systemfont.Find(..., testIO)`.
- Fixed system-font path handling bug to use descriptor path for loaded font.
- Converted Google API-key tests from fail-fast to skip-if-missing-key behavior.
- Refactored cache download test to deterministic offline unit-test style.
- Standardized API-key contract on `GOOGLE_FONTS_API_KEY` in code/comments/messages.
- Introduced `googlefont.IO` abstraction and service-based internals for host/network decoupling.
- Added `FindWithIO(...)` to inject fake I/O in tests while keeping existing public API intact.
- Migrated Google-font tests to fixture-driven fake I/O and removed default runtime network dependency.
- Added explicit fallback APIs and behavior:
  - `fallbackfont.Default()`
  - `fontregistry.FallbackTypeface()` cached under key `"fallback"`
  - `fontfind.FallbackFont()` now returns embedded `Go-Regular.otf` (no panic)
- Reconnected `ResolveTypeface` search flow with `fontregistry` cache (lookup + store + fallback on miss).
- Replaced first-variant Google selection with confidence-based variant selection.

## Known Gaps and Risks

The codebase explicitly marks several unfinished areas:

- TTC support not implemented (`*.ttc` skipped in comments and parser behavior)
- `locate` package comment calls current implementation a stand-in/quick hack
- async resolve API has TODO for context/cancellation integration
- `systemfont` fontconfig loader uses package-global cache state (`sync.Once` + globals), which may complicate test isolation for future cases
- `googlefont` default singleton service caches directory state (`sync.Once`), so cross-test/process behavior should be watched when adding more cases

Quality and consistency issues observed:

- typo/terminology/documentation drift in comments and package text (expected in early refactor phase)

## Implementation Notes (What Already Works)

- Embedded fallback fonts are packaged and can resolve basic matches.
- Fallback access is explicit and deterministic (`fallbackfont.Default`, registry key `"fallback"`).
- System font path can use a pre-generated fontconfig list, then fallback to `go-findfont`.
- Google Fonts lookup and caching pipeline is implemented with injectable host I/O for deterministic testing.
- Matching confidence model (`No/Low/High/Perfect`) is used by local/google matching paths, including Google variant choice.
- `ResolveTypeface` now reuses and fills `fontregistry` as intended by design.

## Suggested Direction (Starting Backlog)

1. Documentation baseline:
   - add `README.md` with usage examples, resolver order, config keys, and test modes
2. Complete TTC story: (NOT A NEAR-TERM GOAL)
   - decide parse/load strategy for `.ttc` and implement at least first readable variant support
3. Add more deterministic tests around resolver state:
   - ensure singleton/service cache behavior does not leak across tests
4. Resolve API cancellation:
   - add context-aware variant of `ResolveTypeface` (existing TODO)

## Practical Project Direction

The current direction appears to be:

- modular provider-based font resolution
- pragmatic fallback-first behavior
- gradual migration/refactor from earlier resource/location code layout

Near-term value is highest in reliability and contract clarity (tests, fallback behavior, config), then feature depth (TTC and richer matching).
