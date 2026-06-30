/**
 * Preservation Property Tests for AssessmentPage Issue Card Rendering
 *
 * These tests verify that NON-BUGGY inputs render correctly with the CURRENT (unfixed) code.
 * They capture baseline behavior that must remain unchanged after the bugfix.
 *
 * **Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.6**
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import fc from 'fast-check'
import React from 'react'

// Mock recharts - it doesn't render well in jsdom
vi.mock('recharts', () => ({
  PieChart: ({ children }: { children: React.ReactNode }) => <div data-testid="pie-chart">{children}</div>,
  Pie: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Cell: () => <div />,
}))

// Mock react-router-dom
const mockNavigate = vi.fn()
vi.mock('react-router-dom', () => ({
  useNavigate: () => mockNavigate,
  useSearchParams: () => [new URLSearchParams('id=test-assessment-123'), vi.fn()],
}))

// Mock apiClient
vi.mock('../../api/client', () => ({
  default: {
    get: vi.fn(),
  },
}))

import AssessmentPage from '../AssessmentPage'
import apiClient from '../../api/client'

/** Build a minimal valid assessment response with a single issue */
function buildAssessment(issue: {
  description: string
  examples?: Array<{
    label?: string
    headers: string[]
    row_number: number
    cells: string[]
    highlights: number[]
    merges?: Array<{ start_col: number; span: number }>
  }>
}) {
  return {
    id: 'test-assessment-123',
    total_score: 75.0,
    status: 'conditional',
    filename: 'test.xlsx',
    total_rows: 100,
    row_completeness: 80,
    column_completeness: 70,
    format_consistency: 65,
    duplicate_similar: 90,
    table_structure: 85,
    ai_query_readiness: 60,
    issues: [
      {
        title: 'Test Issue',
        severity: 'Medium',
        description: issue.description,
        affected_rows: 10,
        unit: '列受影響',
        indicator: 'format_consistency',
        examples: issue.examples || [],
      },
    ],
    row_distribution: { high: 50, medium: 30, low: 20 },
  }
}

/** Helper to render AssessmentPage and expand the first issue card */
async function renderAndExpand(assessmentData: ReturnType<typeof buildAssessment>) {
  cleanup() // Clean up previous renders
  const getMock = vi.mocked(apiClient.get)
  getMock.mockResolvedValueOnce({ data: assessmentData })

  const { container } = render(<AssessmentPage />)
  await screen.findByText('品質評估結果')

  // Click the issue card to expand it
  const issueTitle = screen.getByText('Test Issue')
  fireEvent.click(issueTitle)

  return container
}

// --- Generators ---

const singleLineDescArb = fc.stringOf(
  fc.char().filter(c => c !== '\n' && c !== '\r'),
  { minLength: 1, maxLength: 40 }
).filter(s => s.trim().length > 0)

const headerArb = fc.array(
  fc.constantFrom('Name', 'Phone', 'Email', 'Address', 'Date', 'ID'),
  { minLength: 2, maxLength: 5 }
)

const dataRowNumberArb = fc.integer({ min: 2, max: 100 })

// --- Property Tests ---

describe('Preservation Property Tests', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('Property: single-line descriptions contain no <li> elements in description area', async () => {
    await fc.assert(
      fc.asyncProperty(singleLineDescArb, async (desc) => {
        cleanup() // Clean up previous renders
        const assessment = buildAssessment({
          description: desc,
          examples: [{ headers: ['A', 'B'], row_number: 2, cells: ['v1', 'v2'], highlights: [] }],
        })

        const getMock = vi.mocked(apiClient.get)
        getMock.mockResolvedValueOnce({ data: assessment })
        const { container } = render(<AssessmentPage />)
        await screen.findByText('品質評估結果')

        const allLis = container.querySelectorAll('li')
        expect(allLis.length).toBe(0)
      }),
      { numRuns: 5 }
    )
  }, 30000)

  it('Property: single-label-group issues render exactly 1 <table> per issue card expanded area', async () => {
    await fc.assert(
      fc.asyncProperty(headerArb, dataRowNumberArb, async (headers, rowNum) => {
        const cells = headers.map((_, i) => `cell${i}`)
        const assessment = buildAssessment({
          description: 'Single line desc',
          examples: [
            { headers, row_number: rowNum, cells, highlights: [] },
            { headers, row_number: rowNum + 1, cells, highlights: [] },
          ],
        })

        const container = await renderAndExpand(assessment)
        const tables = container.querySelectorAll('table')
        expect(tables.length).toBe(1)
      }),
      { numRuns: 5 }
    )
  }, 30000)

  it('Property: data rows (row_number > 1) with highlights have <td> with red border', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 2, max: 4 }).chain(colCount => {
          const headers = Array.from({ length: colCount }, (_, i) => `H${i}`)
          return fc.record({
            headers: fc.constant(headers),
            rowNumber: dataRowNumberArb,
            cells: fc.constant(headers.map((_, i) => `c${i}`)),
            highlightIdx: fc.integer({ min: 0, max: colCount - 1 }),
          })
        }),
        async ({ headers, rowNumber, cells, highlightIdx }) => {
          const assessment = buildAssessment({
            description: 'Test desc',
            examples: [{ headers, row_number: rowNumber, cells, highlights: [highlightIdx] }],
          })

          const container = await renderAndExpand(assessment)
          const tbody = container.querySelector('tbody')
          expect(tbody).not.toBeNull()

          const tds = tbody!.querySelectorAll('tr td')
          // td at index highlightIdx + 1 (first td is row number column)
          const targetTd = tds[highlightIdx + 1] as HTMLElement | undefined
          if (targetTd) {
            const style = targetTd.getAttribute('style') || ''
            expect(style).toContain('1.5px solid')
          }
        }
      ),
      { numRuns: 5 }
    )
  }, 30000)

  it('Property: merged cells have colspan matching merge span', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 4, max: 5 }).chain(colCount => {
          const headers = Array.from({ length: colCount }, (_, i) => `Col${i}`)
          const cells = headers.map((_, i) => `v${i}`)
          const span = Math.min(3, colCount - 1)
          return fc.record({
            headers: fc.constant(headers),
            cells: fc.constant(cells),
            rowNumber: dataRowNumberArb,
            merge: fc.constant({ start_col: 0, span }),
          })
        }),
        async ({ headers, cells, rowNumber, merge }) => {
          const assessment = buildAssessment({
            description: 'Merge test',
            examples: [{ headers, row_number: rowNumber, cells, highlights: [], merges: [merge] }],
          })

          const container = await renderAndExpand(assessment)
          const tbody = container.querySelector('tbody')
          expect(tbody).not.toBeNull()

          const row = tbody!.querySelector('tr')
          expect(row).not.toBeNull()

          // Find td with matching colspan
          const allTds = row!.querySelectorAll('td')
          let foundMerge = false
          for (let i = 0; i < allTds.length; i++) {
            const colspan = allTds[i].getAttribute('colspan')
            if (colspan && parseInt(colspan) === merge.span) {
              foundMerge = true
              break
            }
          }
          expect(foundMerge).toBe(true)
        }
      ),
      { numRuns: 5 }
    )
  }, 30000)
})
