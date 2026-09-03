import { describe, expect, it } from 'vitest';
import {
  parseReportCardPageWire,
  parseReportEvidenceListWire,
  parseReportHomeWire,
  parseReportIndustryChainDetailWire,
  parseReportLayerDetailWire
} from './wire-contract';

const reportId = 'RPT11111111-1111-4111-8111-111111111111';
const scopeToken = 'RPE11111111-1111-4111-8111-111111111111';
const coded = { code: 'warming', label: '升温' };
const confidence = { code: 'medium', label: '中' };
const timeWindow = { code: 'medium', label: '中期' };
const report = {
  id: reportId,
  generated_at: '2026-09-01T04:39:03Z',
  published_at: '2026-09-01T04:40:00Z',
  industry_chain_count: 54
};

function card(localKey = 'card-chn-01') {
  return {
    local_key: localKey,
    kind: 'industry_chain',
    detail_ref: { type: 'industry_chain', local_key: 'chn-01' },
    title: '人形机器人产业链',
    subtitle: '产业链',
    conclusion: '链结论',
    result: coded,
    confidence,
    time_window: timeWindow,
    impact_items: [
      {
        ref: { type: 'industry_chain_node', local_key: 'chn-01-n01' },
        name: '人形机器人',
        result: coded,
        conclusion_basis: { code: 'direct_evidence', label: '直接证据' },
        validation_status: { code: 'confirmed', label: '已确认' },
        confidence,
        time_window: timeWindow,
        evidence_scope_token: scopeToken
      }
    ],
    evidence_scope_token: scopeToken
  };
}

describe('Report BFF wire contract', () => {
  it('parses the new home contract and preserves unknown future codes and labels', () => {
    const futureCard = card();
    futureCard.result = { code: 'future_direction', label: '未来方向' };
    const value = parseReportHomeWire({
      selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
      reports: [{ report, cards: [futureCard], next_cursor: 'opaque-cursor' }]
    });
    expect(value.reports[0]?.report.industryChainCount).toBe(54);
    expect(value.reports[0]?.cards[0]?.result).toEqual({
      code: 'future_direction',
      label: '未来方向'
    });
    expect(value.reports[0]?.nextCursor).toBe('opaque-cursor');
  });

  it('fails closed on retired home fields', () => {
    expect(() =>
      parseReportHomeWire({
        selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
        reports: [{ report, cards: [card()], next_cursor: null, industry_chain_count: 54 }]
      })
    ).toThrow('invalid Report wire response');
  });

  it('parses a report-bound industry-chain page', () => {
    expect(parseReportCardPageWire({ items: [card()], next_cursor: null }, reportId)).toEqual({
      items: [expect.objectContaining({ key: 'card-chn-01' })],
      nextCursor: null
    });
  });

  it('parses optional layer content with split basis and validation status', () => {
    const value = parseReportLayerDetailWire(
      {
        report,
        layer: {
          key: 'geopolitics',
          title: '地缘政治',
          conclusion: '一句话结论',
          result: coded,
          confidence,
          time_window: timeWindow,
          anchors: [
            {
              local_key: 'geo-a01',
              name: '海湾安全对抗',
              current_state: '风险上升',
              result: coded,
              conclusion_basis: { code: 'direct_evidence', label: '直接证据' },
              validation_status: { code: 'confirmed', label: '已确认' },
              reasoning: '直接事件形成节点信号。',
              time_window: timeWindow,
              confidence,
              evidence_scope_token: scopeToken
            }
          ],
          reasoning_steps: [],
          transmissions: [],
          uncertainty: {
            counterevidence: null,
            evidence_gap: null,
            boundary: null,
            reversal_condition: null
          },
          evidence_scope_token: scopeToken
        },
        related_industry_chains: [{ local_key: 'chn-01', name: '人形机器人产业链', result: coded }]
      },
      reportId,
      'geopolitics'
    );
    expect(value.layer.anchors[0]?.conclusionBasis?.code).toBe('direct_evidence');
    expect(value.layer.anchors[0]?.validationStatus.code).toBe('confirmed');
  });

  it('validates industry-chain topology against node local keys', () => {
    const payload = {
      report,
      industry_chain: {
        local_key: 'chn-01',
        name: '人形机器人产业链',
        conclusion: '一句话结论',
        result: coded,
        confidence,
        time_window: timeWindow,
        path_summary: '整机 → 控制平台',
        accepted_hypothesis_summary: null,
        topology_nodes: [
          { local_key: 'node-01', name: '人形机器人' },
          { local_key: 'node-02', name: '控制平台' },
          { local_key: 'node-03', name: '结构上下文节点' }
        ],
        nodes: [
          {
            local_key: 'node-01',
            name: '人形机器人',
            impact: '需求上升',
            result: coded,
            conclusion_basis: { code: 'direct_evidence', label: '直接证据' },
            validation_status: { code: 'confirmed', label: '已确认' },
            reasoning: '直接形成信号。',
            time_window: timeWindow,
            confidence,
            evidence_scope_token: scopeToken
          },
          {
            local_key: 'node-02',
            name: '控制平台',
            impact: '需求上升',
            result: coded,
            conclusion_basis: { code: 'reasoning_hypothesis', label: '推理假设' },
            validation_status: { code: 'pending_validation', label: '待验证' },
            reasoning: '同链传导。',
            time_window: timeWindow,
            confidence,
            evidence_scope_token: null
          }
        ],
        edges: [
          {
            from_node_local_key: 'node-02',
            to_node_local_key: 'node-01',
            relation_label: '组成'
          }
        ],
        counterevidence_and_gap: '仍需经营数据。',
        stop_condition: '信号反转则停止。',
        evidence_scope_token: scopeToken
      }
    };
    const parsed = parseReportIndustryChainDetailWire(payload, reportId, 'chn-01').industryChain;
    expect(parsed.edges).toHaveLength(1);
    expect(parsed.topologyNodes).toHaveLength(3);
    const broken = structuredClone(payload);
    broken.industry_chain.edges[0]!.to_node_local_key = 'missing-node';
    expect(() => parseReportIndustryChainDetailWire(broken, reportId, 'chn-01')).toThrow();
  });

  it('binds evidence responses to the opaque report scope token', () => {
    expect(
      parseReportEvidenceListWire(
        {
          report_id: reportId,
          scope_token: scopeToken,
          items: [{ published_at: null, summary: '证据摘要', keywords: ['海湾', '航运'] }]
        },
        reportId,
        scopeToken
      )
    ).toEqual({
      reportId,
      scopeToken,
      items: [{ publishedAt: null, summary: '证据摘要', keywords: ['海湾', '航运'] }]
    });
  });
});
