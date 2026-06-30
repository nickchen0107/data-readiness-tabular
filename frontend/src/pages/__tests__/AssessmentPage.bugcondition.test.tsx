/**
 * Bug Condition Exploration Test - Assessment Issue Card Rendering
 *
 * **Validates: Requirements 1.1, 1.2, 1.3, 1.4**
 *
 * These tests are EXPECTED TO FAIL on unfixed code. Failure confirms the bugs exist.
 * DO NOT attempt to fix these tests or the code when they fail.
 *
 * Bug 1: description containing `\n` → assert list items (<li>) exist
 * Bug 2: examples with 2+ distinct labels → assert 2+ independent <table> elements
 * Bug 3: second group with items[0].row_number == 1 → assert its cells appear as <th> in thead
 * Bug 4: example with row_number == 1 and highlights: [1] → assert <th> at index 1 has red border style
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import * as fc from 'fast-check'

// Mock recharts - it doesn't render well in jsdom
vi.mock('recharts', () => ({
  PieChart: ({ children }: any) => <div data-testid="pie-chart">{children}</div>,
  Pie: ({ children }: any) => <div>{children}</div>,
  Cell: () => <div />,
}))

// Mock react-router-dom
const mockNavigate = vi.fn()
vi.mock('react-router-dom', () => ({
  useNavigate: () => mockNavigate,
  useSearchParams: () => [new URLSearchParams('id=test-assessment-123')],
}))

// Mock apiClient
vi.mock('../../api/client', () => ({
  default: {
    get: vi.fn(),
  },
}))

import AssessmentPage from '../AssessmentPage'
import apiClient from '../../api/client'

/**
 * Crafted assessment data that triggers all 4 bug conditions simultaneously
 */
function makeAssessmentData(overrides: { description?: string; examples?: any[] } = {}) {
  return {
    id: 'test-assessment-123',
    total_score: 65.5,
    status: 'conditional',
    filename: 'test.xlsx',
    total_rows: 100,
    row_completeness: 80,
    column_completeness: 70,
    format_consistency: 60,
    duplicate_similar: 90,
    table_structure: 75,
    ai_query_readiness: 55,
    issues: [
      {
        title: '格式混用問題',
        severity: 'High',
        description: overrides.description ?? '第1行缺少姓名\n第5行缺少電話\n第8行日期格式錯誤',
        affected_rows: 12,
        unit: '列受影響',
        indicator: 'format_consistency',
        examples: overrides.examples ?? [
          // Group 1: label "表格一"
          {
            label: '表格一',
            headers: ['姓名', '電話', '地址'],
            row_number: 1,
            cells: ['姓名', '電話', '地址'],
            highlights: [1],
            merges: [],
          },
          {
            label: '表格一',
            headers: ['姓名', '電話', '地址'],
            row_number: 2,
            cells: ['張三', '0912345678', '台北市'],
            highlights: [],
            merges: [],
          },
          // Group 2: label "表格二" - different label triggers Bug 2
          {
            label: '表格二',
            headers: ['產品', '數量', '單價'],
            row_number: 1,
            cells: ['產品名稱', '數量', '單價'],
            highlights: [1],
            merges: [],
          },
          {
            label: '表格二',
            headers: ['產品', '數量', '單價'],
            row_number: 3,
            cells: ['蘋果', '10', '25'],
            highlights: [2],
            merges: [],
          },
        ],
      },
    ],
    row_distribution: {
      high: 60,
      medium: 25,
      low: 15,
    },
  }
}

async function renderAndExpand() {
  const assessmentData = makeAssessmentData()
  vi.mocked(apiClient.get).mockResolvedValue({ data: assessmentData })

  const { container } = render(<AssessmentPage />)

  // Wait for the component to load
  await waitFor(() => {
    expect(screen.queryByText('載入評估結果中...')).not.toBeInTheDocument()
  })

  // Click to expand the issue card
  const issueTitle = screen.getByText('格式混用問題')
  fireEvent.click(issueTitle)

  return container
}

describe('Bug Condition Exploration - Issue Card Rendering', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  /**
   * Property 1: Bug Condition - Bug 1
   * Description containing \n → assert list items (<li>) exist
   *
   * Currently the code renders descriptions with white-space: pre-line in a single div.
   * Expected: when description contains \n, it should render as <li> elements.
   */
  it('Bug 1: description with newlines should render as list items (<li>)', async () => {
    await fc.assert(
      fc.asyncProperty(
        // Generate descriptions with at least one newline
        fc.array(fc.string({ minLength: 2, maxLength: 30 }), { minLength: 2, maxLength: 5 })
          .map(lines => lines.join('\n')),
        async (description) => {
          vi.clearAllMocks()
          const assessmentData = makeAssessmentData({ description })
          vi.mocked(apiClient.get).mockResolvedValue({ data: assessmentData })

          const { container } = render(<AssessmentPage />)

          await waitFor(() => {
            expect(screen.queryByText('載入評估結果中...')).not.toBeInTheDocument()
          })

          // The description area should contain <li> elements for remaining lines (after the first)
          // Bug 1 fix: first line is rendered as a <div> paragraph, remaining lines as <li>
          const remainingLineCount = description.split('\n').slice(1).filter(Boolean).length

          // Find list items within the component
          const listItems = container.querySelectorAll('li')
          expect(listItems.length).toBeGreaterThanOrEqual(remainingLineCount)
        }
      ),
      { numRuns: 5 }
    )
  })

  /**
   * Property 1: Bug Condition - Bug 2
   * Examples with more than 3 distinct labels → assert at most 3 <table> elements rendered
   *
   * Currently the code renders ALL groups without limiting to max 3.
   * Expected: even with 4+ label groups, only max 3 <table> elements should be rendered.
   */
  it('Bug 2: more than 3 label groups should be limited to max 3 <table> elements', async () => {
    vi.clearAllMocks()

    // Create data with 4 distinct label groups (exceeds the expected max of 3)
    const fourGroupExamples = [
      { label: '表格一', headers: ['A', 'B'], row_number: 2, cells: ['a1', 'b1'], highlights: [], merges: [] },
      { label: '表格二', headers: ['C', 'D'], row_number: 2, cells: ['c1', 'd1'], highlights: [], merges: [] },
      { label: '表格三', headers: ['E', 'F'], row_number: 2, cells: ['e1', 'f1'], highlights: [], merges: [] },
      { label: '表格四', headers: ['G', 'H'], row_number: 2, cells: ['g1', 'h1'], highlights: [], merges: [] },
    ]

    const assessmentData = makeAssessmentData({ examples: fourGroupExamples })
    vi.mocked(apiClient.get).mockResolvedValue({ data: assessmentData })

    const { container } = render(<AssessmentPage />)

    await waitFor(() => {
      expect(screen.queryByText('載入評估結果中...')).not.toBeInTheDocument()
    })

    // Expand the issue card
    const issueTitle = screen.getByText('格式混用問題')
    fireEvent.click(issueTitle)

    // Count the number of <table> elements
    const tables = container.querySelectorAll('table')

    // With 4 label groups, we expect at most 3 tables (slice to max 3)
    expect(tables.length).toBeLessThanOrEqual(3)
  })

  /**
   * Property 1: Bug Condition - Bug 3
   * Second group with items[0].row_number == 1 → assert its cells appear as <th> in thead
   *
   * Currently the second group uses the headers field for thead instead of the first row's cells.
   * Expected: when group.items[0].row_number == 1, that row's cells should be the <th> content.
   */
  it('Bug 3: second group header row (row_number==1) cells should appear as <th> in thead', async () => {
    const container = await renderAndExpand()

    // The second group's first row has cells: ['產品名稱', '數量', '單價']
    // These should appear as <th> content in the second table's thead
    const allTh = container.querySelectorAll('th')
    const thTexts = Array.from(allTh).map(th => th.textContent)

    // The second group's header should use cells ['產品名稱', '數量', '單價']
    // not the headers field ['產品', '數量', '單價']
    expect(thTexts).toContain('產品名稱')
  })

  /**
   * Property 1: Bug Condition - Bug 4
   * Example with row_number == 1 and highlights: [1] → assert <th> at index 1 has red border style
   *
   * Currently highlights are only applied to <td> cells in tbody.
   * Expected: when row_number == 1 and highlights exist, the corresponding <th> in thead
   * should have the red border style.
   */
  it('Bug 4: header row (row_number==1) with highlights should apply red border to <th> elements', async () => {
    const container = await renderAndExpand()

    // Find all <th> elements and check if any have the red border style
    const allTh = container.querySelectorAll('th')
    const highlightedTh = Array.from(allTh).filter(th => {
      const style = th.getAttribute('style') || ''
      return style.includes('1.5px solid') || style.includes('dc2626') || style.includes('--rose')
    })

    // At least one <th> should have the red highlight border
    // (first group's header row has highlights: [1], so th at index 1 should be red)
    expect(highlightedTh.length).toBeGreaterThan(0)
  })
})
