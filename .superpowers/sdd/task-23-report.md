# Task 23 Report: Métodos de leitura do serviço

## Status
DONE_WITH_CONCERNS

## What was implemented
- Added `ResolvePath` and `Get` to the `Index` interface in `internal/service/service.go`.
- Created `internal/service/read.go` implementing `ReadNote` efficiently using `vault.ReadRange`.
- Create `internal/service/graph.go` and implement `LinkGraph`, `TagList`, `ListNotes`, `NoteMetadata`, and `VaultStats` properly.
- Updated `Index` interface with `Paths`, `Generation`, `List`, `Tags`, `Backlinks`.
- Created tests in `internal/service/graph_test.go` to ensure these functions behave correctly.

## TDD Evidence
- **RED**: Methods in `graph.go` returned `CodeNotImplemented` or empty structs initially. `graph_test.go` test cases failed due to empty returns.
- **GREEN**: All tests pass successfully after the implementations were provided and connected to the `Index` properly.

## Extra Verifications

| Check | Result |
|-------|--------|
| `ListNotes` delegates correctly? | Yes, tests confirm it queries and returns results from index. |
| `NoteMetadata` returns populated struct? | Yes, formats metadata fields like `Frontmatter`, `Tags`, `Headings` into struct correctly. |
| `VaultStats` includes runtime stats? | Yes, memory stats and goroutine counts included when `IncludeRuntime: true`. |
| `LinkGraph` respects limits? | Yes, BFS logic observes `Depth` and `Limit` accurately. |

## Mechanical fixes to the plan code
- Fixed `index.Build` call in the test to use `ix := index.New()` and `ix.Build(ctx, v)`.
- Replaced `service.ReadRequest` with `ReadRequest` directly inside the test, since the test is part of the `package service`.

## Concerns
- Fully Resolved: The initial stubs have been completely implemented. `VaultStats` was properly refactored out of `service.go` into `graph.go` to contain the full spec.
