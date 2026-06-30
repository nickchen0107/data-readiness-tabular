/**
 * Issue Card Rendering V2 — Bug Condition Exploration + Preservation Property Tests
 *
 * Task 2: Bug Condition Exploration Tests (Frontend)
 *   - Bug 1: Description first line rendered as paragraph (not <li>)
 *   - Bug 3: Empty header highlighted cells use amber background
 *   EXPECTED: These tests MUST FAIL on unfixed code (failure confirms bugs exist)
 *
 * Task 4: Preservation Property Tests (Frontend)
 *   - Descriptions without \n: single <div> with white-space: pre-line, no <ul>/<li>
 *   - Non-highlighted header cells: standard grey background (#f3f4f6)
 *   - Non-format-consistency issues: no format label spans rendered
 *   EXPECTED: These tests MUST PASS on unfixed code (confirms baseline behavior)
 *
 * **Validates: Requirements 1.1, 1.3, 2.1, 2.3, 3.1, 3.3, 3.4, 3.6**
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, cleanup, waitFor } from '@testing-library/react'
import * as fc from 'fast-check'
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
  useSearchParams: () => [new URLSearchParams('id=test-123'), vi.fn()],
}))

// Mock apiClient
vi.mock('../../api/client', () => ({
  default: {
    get: vi.fn(),
  },
}))

import AssessmentPage from '../AssessmentPage'
import apiClient from '../../api/client'

// --------------- Test Data Helpers ---------------

function makeAssessmentData(overrides: {
  description?: string
  indicator?: string
  examples?: any[]
} = {}) {
  return {
    id: 'test-123',
    total_score: 72.0,
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
        title: '測試問題',
        severity: 'High',
        description: overrides.description ?? '簡單描述',
        affected_rows: 5,
        unit: '列受影響',
        indicator: overrides.indicator ?? 'format_consistency',
        examples: overrides.examples ?? [
          {
            label: undefined,
            headers: ['姓名', '電話'],
            row_number: 2,
            cells: ['張三', '0912345678'],
            highlights: [],
            merges: [],
          },
        ],
      },
    ],
    row_distribution: { high: 50, medium: 30, low: 20 },
  }
}

// --------------- Task 2: Bug Condition Exploration Tests ---------------

describe('Task 2: Bug Condition Exploration Tests', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    cleanup()
  })

  /**
   * Bug 1 — Description First Line as Paragraph
   *
   * **Validates: Requirements 1.1, 2.1**
   *
   * When description contains "\n", the FIRST line should be rendered as a <div> or <p>
   * (paragraph), NOT as a <li>. Subsequent lines should be <li> elements.
   *
   * EXPECTED: FAIL on unfixed code (current code renders ALL lines as <li>)
   */
  it('Bug 1: description first line should be a paragraph (<div>/<p>), not a <li>', async () => {
    const description = '以下欄位格式不一致：\nTracking No.\nAmount'
    const assessmentData = makeAssessmentData({ description })
    vi.mocked(apiClient.get).mockResolvedValue({ data: assessmentData })

    const { container } = render(<AssessmentPage />)

    await waitFor(() => {
      expect(screen.queryByText('載入評估結果中...')).not.toBeInTheDocument()
    })

    // The first line "以下欄位格式不一致：" should NOT be inside a <li>
    const allListItems = container.querySelectorAll('li')
    const firstLineText = '以下欄位格式不一致：'

    // Check that the first line is NOT rendered as a <li>
    const firstLineInLi = Array.from(allListItems).some(
      li => li.textContent?.includes(firstLineText)
    )
    expect(firstLineInLi).toBe(false)

    // The first line should be rendered as a <div> or <p> paragraph element
    const allDivs = container.querySelectorAll('div, p')
    const firstLineInParagraph = Array.from(allDivs).some(
      el => el.textContent?.trim() === firstLineText
    )
    expect(firstLineInParagraph).toBe(true)

    // Subsequent lines ("Tracking No.", "Amount") should still be <li> elements
    const subsequentLineInLi = Array.from(allListItems).some(
      li => li.textContent?.includes('Tracking No.')
    )
    expect(subsequentLineInLi).toBe(true)
  })

  /**
   * Bug 3 — Empty Header Amber Background
   *
   * **Validates: Requirements 1.3, 2.3**
   *
   * When row_number=1 and highlights=[1], the highlighted <th> should have
   * amber/orange background (contains "245, 158, 11" or "#f59e0b"),
   * NOT red styling (220, 38, 38 or dc2626).
   *
   * EXPECTED: FAIL on unfixed code (current code uses red: "dc2626")
   */
  it('Bug 3: highlighted header <th> should use amber background, not red', async () => {
    const assessmentData = makeAssessmentData({
      description: '空白標題欄偵測',
      indicator: 'table_structure',
      examples: [
        {
          label: undefined,
          headers: ['姓名', '電話', '地址'],
          row_number: 1,
          cells: ['姓名', '', '地址'],
          highlights: [1],
          merges: [],
        },
        {
          label: undefined,
          headers: ['姓名', '電話', '地址'],
          row_number: 2,
          cells: ['張三', '0912', '台北'],
          highlights: [],
          merges: [],
        },
      ],
    })
    vi.mocked(apiClient.get).mockResolvedValue({ data: assessmentData })

    const { container } = render(<AssessmentPage />)

    await waitFor(() => {
      expect(screen.queryByText('載入評估結果中...')).not.toBeInTheDocument()
    })

    // Expand the issue card
    const issueTitle = screen.getByText('測試問題')
    fireEvent.click(issueTitle)

    // Find all <th> elements in the rendered table
    const allTh = container.querySelectorAll('th')

    // Find the highlighted <th> (the one at index 1 in the header, which is the second
    // data column th — first th is the "#" column)
    // Assert: highlighted <th> should have amber/orange background
    const hasAmber = Array.from(allTh).some(th => {
      const style = th.getAttribute('style') || ''
      return style.includes('245, 158, 11') || style.includes('f59e0b')
    })
    expect(hasAmber).toBe(true)
  })
})

// --------------- Task 4: Preservation Property Tests ---------------

describe('Task 4: Preservation Property Tests', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    cleanup()
  })

  // --- Generators ---

  /** Generate descriptions WITHOUT newline characters */
  const singleLineDescArb = fc.stringOf(
    fc.char().filter(c => c !== '\n' && c !== '\r'),
    { minLength: 3, maxLength: 40 }
  ).filter(s => s.trim().length > 2 && !s.includes('\n'))

  /**
   * Preservation: Descriptions WITHOUT "\n" render as single <div> with white-space: pre-line
   * and produce NO <ul> or <li> elements.
   *
   * **Validates: Requirements 3.1**
   *
   * EXPECTED: PASS on unfixed code (this is existing correct behavior)
   */
  it('Preservation: descriptions without \\n produce a <div> with pre-line, no <ul>/<li>', async () => {
    await fc.assert(
      fc.asyncProperty(singleLineDescArb, async (desc) => {
        cleanup()
        vi.clearAllMocks()

        const assessmentData = makeAssessmentData({
          description: desc,
          examples: [
            {
              headers: ['A', 'B'],
              row_number: 2,
              cells: ['v1', 'v2'],
              highlights: [],
              merges: [],
            },
          ],
        })
        vi.mocked(apiClient.get).mockResolvedValue({ data: assessmentData })

        const { container } = render(<AssessmentPage />)

        await waitFor(() => {
          expect(screen.queryByText('載入評估結果中...')).not.toBeInTheDocument()
        })

        // No <ul> or <li> should be present for single-line descriptions
        const ulElements = container.querySelectorAll('ul')
        const liElements = container.querySelectorAll('li')
        expect(ulElements.length).toBe(0)
        expect(liElements.length).toBe(0)

        // The description should be in a <div> with white-space: pre-line
        const descDivs = container.querySelectorAll('div')
        const preLineDiv = Array.from(descDivs).find(div => {
          const style = div.getAttribute('style') || ''
          return style.includes('pre-line') && div.textContent?.includes(desc)
        })
        expect(preLineDiv).toBeTruthy()
      }),
      { numRuns: 5 }
    )
  }, 30000)

  /**
   * Preservation: Header cells NOT in highlights array use standard grey background (#f3f4f6)
   *
   * **Validates: Requirements 3.3**
   *
   * EXPECTED: PASS on unfixed code (non-highlighted headers already use grey)
   */
  it('Preservation: non-highlighted header <th> cells use grey background #f3f4f6', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 2, max: 4 }).chain(colCount => {
          const headers = Array.from({ length: colCount }, (_, i) => `Col${i}`)
          return fc.record({
            headers: fc.constant(headers),
            colCount: fc.constant(colCount),
          })
        }),
        async ({ headers }) => {
          cleanup()
          vi.clearAllMocks()

          // Create example with row_number=1 but NO highlights (empty array)
          const assessmentData = makeAssessmentData({
            description: '標題測試',
            examples: [
              {
                headers,
                row_number: 1,
                cells: headers.map(h => h),
                highlights: [], // NO highlights
                merges: [],
              },
              {
                headers,
                row_number: 2,
                cells: headers.map((_, i) => `data${i}`),
                highlights: [],
                merges: [],
              },
            ],
          })
          vi.mocked(apiClient.get).mockResolvedValue({ data: assessmentData })

          const { container } = render(<AssessmentPage />)

          await waitFor(() => {
            expect(screen.queryByText('載入評估結果中...')).not.toBeInTheDocument()
          })

          // Expand the issue card
          const issueTitle = screen.getByText('測試問題')
          fireEvent.click(issueTitle)

          // All <th> elements should have standard grey background
          const allTh = container.querySelectorAll('th')
          for (const th of Array.from(allTh)) {
            const style = th.getAttribute('style') || ''
            // Should contain grey background (hex #f3f4f6 or rgb(243, 244, 246))
            const hasGrey = style.includes('#f3f4f6') || style.includes('243, 244, 246')
            expect(hasGrey).toBe(true)
            // Should NOT contain amber or red highlighting
            expect(style).not.toContain('245, 158, 11')
            expect(style).not.toContain('f59e0b')
            expect(style).not.toContain('220, 38, 38')
          }
        }
      ),
      { numRuns: 5 }
    )
  }, 30000)

  /**
   * Preservation: Non-format-consistency issue examples do NOT render format label spans
   *
   * **Validates: Requirements 3.4**
   *
   * EXPECTED: PASS on unfixed code (format labels don't exist yet)
   */
  it('Preservation: non-format-consistency issues have no format label spans', async () => {
    const nonFormatIndicators = ['row_completeness', 'duplicate_similar', 'table_structure']

    for (const indicator of nonFormatIndicators) {
      cleanup()
      vi.clearAllMocks()

      const assessmentData = makeAssessmentData({
        description: `問題: ${indicator}`,
        indicator,
        examples: [
          {
            headers: ['Name', 'Phone', 'Email'],
            row_number: 2,
            cells: ['Alice', '0912', 'a@b.com'],
            highlights: [1],
            merges: [],
          },
          {
            headers: ['Name', 'Phone', 'Email'],
            row_number: 5,
            cells: ['Bob', '', 'b@c.com'],
            highlights: [1],
            merges: [],
          },
        ],
      })
      vi.mocked(apiClient.get).mockResolvedValue({ data: assessmentData })

      const { container } = render(<AssessmentPage />)

      await waitFor(() => {
        expect(screen.queryByText('載入評估結果中...')).not.toBeInTheDocument()
      })

      // Expand the issue card
      const issueTitle = screen.getByText('測試問題')
      fireEvent.click(issueTitle)

      // No format type label spans should exist
      // Format labels would contain: 日期, 數字, 布林, 文字
      const formatLabels = ['日期', '數字', '布林', '文字']
      const tbody = container.querySelector('tbody')
      if (tbody) {
        const spans = tbody.querySelectorAll('span')
        for (const span of Array.from(spans)) {
          const text = span.textContent?.trim() || ''
          const isFormatLabel = formatLabels.includes(text)
          expect(isFormatLabel).toBe(false)
        }
      }
    }
  }, 30000)
})
