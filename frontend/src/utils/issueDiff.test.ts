import { describe, it, expect } from 'vitest';
import { getResolvedIssues, getRemainingIssues, Issue } from './issueDiff';

describe('issueDiff', () => {
  const issue1: Issue = { title: '缺失值過多', severity: 'high', affected_rows: 42 };
  const issue2: Issue = { title: '格式不一致', severity: 'medium', affected_rows: 10 };
  const issue3: Issue = { title: '重複資料', severity: 'low', affected_rows: 5 };

  describe('getResolvedIssues', () => {
    it('returns issues in original but not in postCleaning', () => {
      const original = [issue1, issue2, issue3];
      const postCleaning = [issue2];

      const resolved = getResolvedIssues(original, postCleaning);

      expect(resolved).toEqual([issue1, issue3]);
    });

    it('returns empty array when all issues remain', () => {
      const original = [issue1, issue2];
      const postCleaning = [issue1, issue2];

      const resolved = getResolvedIssues(original, postCleaning);

      expect(resolved).toEqual([]);
    });

    it('returns all original issues when postCleaning is empty', () => {
      const original = [issue1, issue2];
      const postCleaning: Issue[] = [];

      const resolved = getResolvedIssues(original, postCleaning);

      expect(resolved).toEqual([issue1, issue2]);
    });

    it('returns empty array when original is empty', () => {
      const original: Issue[] = [];
      const postCleaning = [issue1];

      const resolved = getResolvedIssues(original, postCleaning);

      expect(resolved).toEqual([]);
    });

    it('compares by exact title match', () => {
      const original = [{ title: 'Issue A', severity: 'high', affected_rows: 10 }];
      const postCleaning = [{ title: 'Issue A ', severity: 'low', affected_rows: 1 }];

      const resolved = getResolvedIssues(original, postCleaning);

      // 'Issue A' !== 'Issue A ' (trailing space), so it's resolved
      expect(resolved).toEqual([original[0]]);
    });

    it('preserves original issue metadata in results', () => {
      const original = [{ title: 'Test', severity: 'high', affected_rows: 99 }];
      const postCleaning: Issue[] = [];

      const resolved = getResolvedIssues(original, postCleaning);

      expect(resolved[0].severity).toBe('high');
      expect(resolved[0].affected_rows).toBe(99);
    });
  });

  describe('getRemainingIssues', () => {
    it('returns the postCleaning list as-is', () => {
      const postCleaning = [issue1, issue3];

      const remaining = getRemainingIssues(postCleaning);

      expect(remaining).toEqual([issue1, issue3]);
    });

    it('returns empty array when postCleaning is empty', () => {
      const remaining = getRemainingIssues([]);

      expect(remaining).toEqual([]);
    });
  });
});
