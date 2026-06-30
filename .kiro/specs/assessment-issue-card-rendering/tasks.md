# Implementation Plan

## Overview

Bugfix implementation for AssessmentPage.tsx Issue Card rendering. Follows the exploratory bugfix workflow: write tests first to confirm bugs exist, then implement fixes, then verify all tests pass. Covers 4 bugs: description newline rendering, multi-label-group table separation, per-group independent headers, and header row th highlights.

## Tasks

- [x] 1. Write bug condition exploration test
  - **Property 1: Bug Condition** - Issue Card Rendering Bugs (Description Newlines, Multi-Group Tables, Header th Highlights)
  - **CRITICAL**: This test MUST FAIL on unfixed code - failure confirms the bug exists
  - **DO NOT attempt to fix the test or the code when it fails**
  - **NOTE**: This test encodes the expected behavior - it will validate the fix when it passes after implementation
  - **GOAL**: Surface counterexamples that demonstrate the 4 rendering bugs exist
  - **Setup**: Install Vitest + React Testing Library + jsdom (`npm install -D vitest @testing-library/react @testing-library/jest-dom jsdom fast-check`), add vitest config with jsdom environment
  - **Scoped PBT Approach**: Scope the property to concrete failing cases for each bug condition:
    - Bug 1: description containing `\n` → assert list items (`<li>`) exist (currently only pre-line div)
    - Bug 2: examples with 2+ distinct labels → assert 2+ independent `<table>` elements exist
    - Bug 3: second group with `items[0].row_number == 1` → assert its cells appear as `<th>` in thead
    - Bug 4: example with `row_number == 1` and `highlights: [1]` → assert `<th>` at index 1 has red border style
  - Test file: `frontend/src/pages/__tests__/AssessmentPage.bugcondition.test.tsx`
  - Mock `apiClient.get` to return crafted assessment data triggering all 4 bug conditions
  - Run test on UNFIXED code
  - **EXPECTED OUTCOME**: Test FAILS (this is correct - it proves the bugs exist)
  - Document counterexamples found (e.g., "description renders as single div with pre-line instead of list items", "only 1 table element found despite 2 label groups", "th elements lack red border styling")
  - Mark task complete when test is written, run, and failure is documented
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 2. Write preservation property tests (BEFORE implementing fix)
  - **Property 2: Preservation** - Existing Non-Buggy Rendering Behavior Unchanged
  - **IMPORTANT**: Follow observation-first methodology
  - **Setup**: Reuse same test infrastructure from task 1
  - Observe behavior on UNFIXED code for non-buggy inputs:
    - Observe: single-line description renders as inline text in a div with `white-space: pre-line` (no list elements)
    - Observe: single label group renders exactly 1 `<table>` element with thead using `headers` field
    - Observe: `row_number > 1` with highlights renders `<td>` with red border `1.5px solid var(--rose, #dc2626)`
    - Observe: cells with merges render with `colspan` and blue background `rgba(59, 130, 246, 0.06)`
    - Observe: collapsed card shows title, severity badge, description, affected count without table content
  - Write property-based tests (using fast-check) capturing observed behavior:
    - Property: for all single-line descriptions (no `\n`), rendered DOM contains no `<li>` elements in description area
    - Property: for all single-label-group issues, exactly 1 `<table>` exists per issue card
    - Property: for all examples with `row_number > 1` and non-empty highlights, `<td>` at highlighted index has red border
    - Property: for all examples with merges, merged `<td>` has colspan matching merge span
  - Test file: `frontend/src/pages/__tests__/AssessmentPage.preservation.test.tsx`
  - Run tests on UNFIXED code
  - **EXPECTED OUTCOME**: Tests PASS (this confirms baseline behavior to preserve)
  - Mark task complete when tests are written, run, and passing on unfixed code
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_

- [ ] 3. Fix for Issue Card rendering bugs in AssessmentPage.tsx

  - [x] 3.1 Bug 1 — Description text: detect `\n` and render as list items
    - In the description rendering section (~line 230), check if `issue.description` contains `\n`
    - If yes: split by `\n`, filter empty strings, render as `<ul>` with `<li>` per line segment
    - If no: keep existing single `<div>` with `whiteSpace: 'pre-line'` rendering
    - Style `<ul>` with `margin: 0`, `paddingLeft: 18px`, `listStyleType: 'disc'`
    - Style `<li>` with `marginBottom: 2px`, `fontSize: 13`, `color: var(--ink-soft)`
    - _Bug_Condition: isBugCondition(issue) where issue.description CONTAINS '\n'_
    - _Expected_Behavior: DOM has `<li>` elements count == description.split('\n').filter(Boolean).length_
    - _Preservation: Single-line descriptions continue to render as div with pre-line_
    - _Requirements: 2.1, 3.1_

  - [x] 3.2 Bug 2 — Multiple label groups: slice to max 3 with independent tables
    - After the grouping logic, add `.slice(0, 3)` to limit rendered groups
    - Ensure each group's wrapper `<div>` has its own `border`, `borderRadius: 8`, and `marginTop: 12`
    - Verify existing group rendering already produces separate `<table>` elements (confirm from code reading)
    - _Bug_Condition: isBugCondition(issue) where countDistinctLabels(issue.examples) > 1_
    - _Expected_Behavior: DOM has min(distinctLabels, 3) separate `<table>` elements_
    - _Preservation: Single label group still renders one table as before_
    - _Requirements: 2.2, 3.2_

  - [x] 3.3 Bug 3 — Per-group independent header from row_number == 1
    - In each group's rendering block, check if `group.items[0].row_number === 1`
    - If true: use `group.items[0].cells` as thead `<th>` content (instead of `group.items[0].headers`)
    - Exclude that first item from tbody rendering (use `group.items.slice(1)` for tbody rows)
    - If false: keep existing behavior using `group.items[0].headers` for thead
    - Support merges in thead: read `group.items[0].merges` and apply `colspan` to corresponding `<th>` elements
    - _Bug_Condition: isBugCondition(group) where group.items[0].row_number == 1_
    - _Expected_Behavior: thead th content matches group.items[0].cells, not headers field_
    - _Preservation: Groups where first item row_number > 1 continue using headers field_
    - _Requirements: 2.3, 3.2_

  - [x] 3.4 Bug 4 — Header row highlights applied to th elements
    - When rendering thead from `group.items[0]` (the header row with `row_number == 1`):
    - Read `group.items[0].highlights` array
    - For each `<th>` at an index in highlights: apply `border: '1.5px solid var(--rose, #dc2626)'` and `background: 'rgba(220, 38, 38, 0.06)'`
    - For non-highlighted `<th>`: keep existing grey background `#f3f4f6` and border `1px solid #e5e7eb`
    - _Bug_Condition: isBugCondition(example) where example.row_number == 1 AND example.highlights.length > 0_
    - _Expected_Behavior: th[highlightedIndex] has red border style, not td_
    - _Preservation: row_number > 1 highlights still apply to td cells as before_
    - _Requirements: 2.4, 3.3_

  - [x] 3.5 Verify bug condition exploration test now passes
    - **Property 1: Expected Behavior** - Issue Card Rendering Bugs Fixed
    - **IMPORTANT**: Re-run the SAME test from task 1 - do NOT write a new test
    - The test from task 1 encodes the expected behavior for all 4 bug conditions
    - Run bug condition exploration test from step 1
    - **EXPECTED OUTCOME**: Test PASSES (confirms all 4 bugs are fixed)
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [x] 3.6 Verify preservation tests still pass
    - **Property 2: Preservation** - Existing Non-Buggy Rendering Behavior Unchanged
    - **IMPORTANT**: Re-run the SAME tests from task 2 - do NOT write new tests
    - Run preservation property tests from step 2
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions to existing behavior)
    - Confirm all preservation properties hold: single-line descriptions, single-group tables, data row highlights, merge styling
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_

- [x] 4. Checkpoint - Ensure all tests pass
  - Run full test suite: `cd frontend && npx vitest --run`
  - Ensure both bug condition tests and preservation tests pass
  - Verify no TypeScript compilation errors: `cd frontend && npx tsc --noEmit`
  - Ensure all tests pass, ask the user if questions arise

## Task Dependency Graph

```json
{
  "waves": [
    {"tasks": ["1", "2"]},
    {"tasks": ["3.1", "3.2", "3.3", "3.4"]},
    {"tasks": ["3.5", "3.6"]},
    {"tasks": ["4"]}
  ]
}
```

## Notes

- Vitest, React Testing Library, and fast-check are NOT yet installed in the frontend — task 1 includes setup
- All changes are scoped to a single file: `frontend/src/pages/AssessmentPage.tsx`
- Test files go in `frontend/src/pages/__tests__/`
- The bug condition test (task 1) is expected to FAIL initially — this confirms the bugs exist
- The preservation test (task 2) is expected to PASS initially — this confirms baseline behavior
- After fix (tasks 3.1–3.4), bug condition test should PASS and preservation test should still PASS
