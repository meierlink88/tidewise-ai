import type { ReportSummary } from '@/models/report';
import { request } from './request';

export function getLatestReportSummary(): Promise<ReportSummary> {
  return request({
    mock: () => ({
      id: 'report-001',
      title: '事件传导观察',
      summary: '后续由 Go API/BFF 与外部 Agent 平台集成后提供真实报告摘要。',
      generatedAt: '2026-07-05'
    })
  });
}
