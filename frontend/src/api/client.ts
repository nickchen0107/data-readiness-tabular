import axios from 'axios'

const apiClient = axios.create({
  baseURL: '/data-readiness-tabular/api',
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor: attach Authorization header
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Response interceptor: handle 401 (only for non-auth endpoints)
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Don't redirect for login/register endpoints — 401 is expected for wrong credentials
      const url = error.config?.url || ''
      if (!url.includes('/auth/login') && !url.includes('/auth/register')) {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        window.location.href = '/data-readiness-tabular/login'
      }
    }
    return Promise.reject(error)
  }
)

export default apiClient

// --- Interactive Fix Types ---

export interface FlaggedCell {
  row_index: number
  col_index: number
  column_name: string
  row_number: number       // 1-based Excel row
  current_value: string
  issue_type: 'cell_reference_placeholder' | 'column_type_mismatch' | 'inline_remark' | 'empty_header'
  issue_description: string
}

export interface CellEditAction {
  row_index: number
  col_index: number
  action: 'replace' | 'keep' | 'delete_row' | 'remark_split' | 'header_rename'
  value?: string
}

export interface InteractiveFixRequest {
  assessment_id: string
  edits: CellEditAction[]
}

export interface InteractiveFixResponse {
  success: boolean
  rows_affected: number
  warnings: string[]
  log_entries: Array<{
    operation_type: string
    row_index?: number
    details?: string
  }>
}

/**
 * Submit interactive cell edits to the cleaning engine.
 * POST /api/clean/interactive
 */
export async function submitInteractiveEdits(
  req: InteractiveFixRequest
): Promise<InteractiveFixResponse> {
  const res = await apiClient.post<InteractiveFixResponse>('/clean/interactive', req)
  return res.data
}
