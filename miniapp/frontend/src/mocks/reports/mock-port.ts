import semanticFixture from './evidence-semantic.json';
import { parseReportEvidenceListWire } from '../../features/reports/wire-contract';
import normalized from './normalized.json';
import {
  type AnalysisKind,
  parseAnalysisGroups,
  parseAnalysisPage,
  parseAnalysisDetail,
  parseAnalysisChain
} from '../../features/reports/normalized-contract';
import type { ReportHome, ReportPort, ReportSummary } from '../../features/reports/contract';
import { ReportError } from '../../features/reports/contract';

const REPORT_ID = 'RPT11111111-1111-4111-8111-111111111111';
const report: ReportSummary = {
  id: REPORT_ID,
  generatedAt: '2026-09-01T04:39:03Z',
  publishedAt: '2026-09-01T04:45:00Z',
  industryChainCount: 54
};

export class MockReportPort implements ReportPort {
  async getAnalyses(reportId: string, kind: AnalysisKind, cursor?: string) {
    assertReport(reportId);
    if (cursor) throw new ReportError('invalidRequest');
    return parseAnalysisPage(normalized.groups.find((g) => g.kind === kind));
  }
  async getAnalysis(reportId: string, kind: AnalysisKind, key: string) {
    assertReport(reportId);
    const value: unknown = (normalized.details as Record<string, unknown>)[`${kind}/${key}`];
    if (!value) throw new ReportError('layerUnavailable');
    return parseAnalysisDetail(value, key);
  }
  async getAnalysisChain(reportId: string, kind: AnalysisKind, key: string, chainKey: string) {
    assertReport(reportId);
    const value: unknown = (normalized.chains as Record<string, unknown>)[
      `${kind}/${key}/${chainKey}`
    ];
    if (!value) throw new ReportError('chainUnavailable');
    return parseAnalysisChain(value, chainKey);
  }
  async getHome(): Promise<ReportHome> {
    return {
      selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
      reports: [
        {
          report: { ...report, schemaVersion: 'report-publication/v4' },
          cards: [],
          nextCursor: null,
          analysisGroups: parseAnalysisGroups(normalized.groups)
        }
      ]
    };
  }

  async getEvidences(reportId: string, scopeToken: string) {
    assertReport(reportId);
    const normalizedItems = (
      normalized.evidences as Record<
        string,
        { published_at: string; summary: string; keywords: string[] }[]
      >
    )[scopeToken];
    if (!normalizedItems) throw new ReportError('evidenceScopeUnavailable');
    return parseReportEvidenceListWire(
      {
        report_id: reportId,
        scope_token: scopeToken,
        items: normalizedItems.map((item, index) =>
          index === 0 && scopeToken === normalized.groups[0].items[0].summary.evidence_scope_token
            ? semanticFixture.items[0]
            : item
        )
      },
      reportId,
      scopeToken
    );
  }
}

export const mockReportPort = new MockReportPort();
export const normalizedMockReportPort = mockReportPort;

function assertReport(reportId: string): void {
  if (reportId !== REPORT_ID) throw new ReportError('reportUnavailable');
}
