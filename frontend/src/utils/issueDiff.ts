/**
 * Issue diffing utilities for the Comparison Dashboard.
 * Pure functions to compute resolved vs remaining issues.
 */

export interface IssueExample {
  label?: string;
  headers: string[];
  row_number: number;
  cells: string[];
  highlights: number[];
  merges?: Array<{ start_col: number; span: number }>;
  format_labels?: string[];
}

export interface Issue {
  title: string;
  title_en?: string;
  severity: string;
  affected_rows: number;
  description?: string;
  description_en?: string;
  unit?: string;
  indicator?: string;
  examples?: IssueExample[];
}

/**
 * Returns issues present in `original` but absent in `postCleaning` (by exact title match).
 * These represent problems that were resolved during the cleaning process.
 */
export function getResolvedIssues(original: Issue[], postCleaning: Issue[]): Issue[] {
  const postTitles = new Set(postCleaning.map((issue) => issue.title));
  return original.filter((issue) => !postTitles.has(issue.title));
}

/**
 * Returns the post-cleaning issues list (alias for clarity in dashboard context).
 * These represent problems that still remain after cleaning.
 */
export function getRemainingIssues(postCleaning: Issue[]): Issue[] {
  return postCleaning;
}
