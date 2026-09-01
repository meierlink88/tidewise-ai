import { describe, expect, it } from 'vitest';
import {
  parseReportEvidenceListWire,
  parseReportHomeWire,
  parseReportIndustryChainDetailWire,
  parseReportLayerDetailWire
} from './wire-contract';

const reportId = 'RPT11111111-1111-4111-8111-111111111111';
const report = {
  id: reportId,
  title: '当前事件如何从地缘政治与宏观经济传导至产业链',
  generated_at: '2026-09-01T04:39:03Z',
  published_at: '2026-09-01T04:40:00Z'
};
const warming = { code: 'warming', label: '升温' } as const;
const cooling = { code: 'cooling', label: '降温' } as const;
const confidence = { label: '中', score: null } as const;
const nature = { code: 'direct_evidence', label: '直接证据' } as const;

function impact(type: 'anchor' | 'industry_chain_node', key: string, name: string) {
  return {
    ref: { type, key },
    name,
    result: warming,
    confidence,
    time_window: '中期',
    has_evidence: true
  };
}

const cards = [
  {
    key: 'card-geopolitics',
    kind: 'geopolitics',
    display_order: 1,
    detail_ref: { type: 'layer', key: 'geopolitics' },
    title: '地缘政治',
    subtitle: '安全对抗与通道可用性',
    conclusion: '海湾安全对抗持续升温。',
    result: warming,
    confidence,
    time_window: '即时–中期',
    impact_items: [impact('anchor', 'geo-a01', '海湾安全对抗')],
    has_evidence: true
  },
  {
    key: 'card-macroeconomics',
    kind: 'macroeconomics',
    display_order: 2,
    detail_ref: { type: 'layer', key: 'macroeconomics' },
    title: '宏观经济',
    subtitle: '增长预期与政策利率',
    conclusion: '增长预期下降。',
    result: cooling,
    confidence,
    time_window: '中期',
    impact_items: [impact('anchor', 'macro-a01', '增长预期')],
    has_evidence: true
  },
  {
    key: 'card-chn-21',
    kind: 'industry_chain',
    display_order: 3,
    detail_ref: { type: 'industry_chain', key: 'chn-21' },
    title: '油品石化贸易服务产业链',
    subtitle: '海湾风险直接关联链',
    conclusion: '油品运输服务分化。',
    result: warming,
    confidence,
    time_window: '中期',
    impact_items: [impact('industry_chain_node', 'chn-21-n01', '油品运输服务')],
    has_evidence: true
  }
] as const;

function homeGroup(summary: typeof report = report) {
  return {
    report: summary,
    industry_chain_count: 54,
    cards,
    company: {
      key: 'company',
      display_order: 4,
      title: '企业',
      published: false,
      boundary: '本次推理尚未进入企业层。'
    }
  };
}

const anchor = {
  key: 'geo-a01',
  display_order: 1,
  name: '海湾安全对抗',
  current_state: '风险上升',
  result: warming,
  nature,
  reasoning: '通行量下降与袭击风险共同推高安全压力。',
  time_window: '即时–中期',
  confidence,
  scope: { type: 'anchor', key: 'geo-a01' },
  has_evidence: true
};

const relatedChain = {
  key: 'chn-21',
  display_order: 21,
  name: '油品石化贸易服务产业链',
  conclusion: '油品运输服务分化。',
  status: '运输节点已闭合',
  result: warming,
  confidence,
  time_window: '中期',
  scope: { type: 'industry_chain', key: 'chn-21' },
  has_evidence: true
};

const layer = {
  key: 'geopolitics',
  display_order: 1,
  title: '地缘政治',
  conclusion: '海湾安全对抗持续升温。',
  result: warming,
  confidence,
  time_window: '即时–中期',
  anchors: [anchor],
  reasoning_steps: [
    {
      key: 'geo-step-01',
      display_order: 1,
      input: '通行量下降',
      mechanism: '安全风险重定价',
      output: '航运压力上升',
      type: '证据 → 推理',
      confidence,
      scope: { type: 'reasoning_step', key: 'geo-step-01' },
      has_evidence: true
    }
  ],
  related_anchor_keys: ['macro-a01'],
  related_chain_keys: ['chn-21'],
  downward_transmission: {
    summary: '只发布有结构化目标的传导。',
    published_paths: [
      {
        key: 'geo-path-01',
        display_order: 1,
        source_conclusion: '海湾通道压力上升',
        target_refs: [
          { ref: { type: 'layer', key: 'macroeconomics' }, label: '宏观经济', result: cooling },
          { ref: { type: 'anchor', key: 'macro-a01' }, label: '增长预期', result: cooling },
          { ref: { type: 'industry_chain', key: 'chn-21' }, label: '油品贸易链', result: warming },
          {
            ref: { type: 'industry_chain_node', key: 'chn-21-n01' },
            label: '油品运输服务',
            result: warming
          }
        ],
        logic: '通行受阻抬高运输成本。',
        relation_nature: '推理假设',
        evidence_role: '来源与目标证据分离',
        confidence,
        status: '运输节点已闭合。',
        scope: { type: 'transmission_path', key: 'geo-path-01' },
        has_evidence: true
      }
    ],
    candidate_mechanisms: [],
    boundary_notes: ['不把同源信号改写为直接因果。']
  },
  uncertainty: {
    counterevidence: '商业航运仍局部稳定。',
    evidence_gap: '缺少连续船舶量。',
    boundary: '国家仅作背景。',
    reversal_condition: '若通行恢复则下调结论。',
    checkpoints: []
  },
  scope: { type: 'layer', key: 'geopolitics' },
  has_evidence: true
};

const chain = {
  key: 'chn-21',
  claim_key: 'chn-21-claim',
  display_order: 21,
  name: '油品石化贸易服务产业链',
  conclusion: '油品运输服务分化。',
  status: '运输节点已闭合',
  result: warming,
  confidence,
  time_window: '中期',
  path_summary: '运输成本向批发交付传导。',
  accepted_hypothesis_summary: null,
  nodes: [
    {
      key: 'chn-21-n01',
      display_order: 1,
      name: '油品运输服务',
      impact: '交付周期上升',
      result: warming,
      nature,
      reasoning: '通行受阻延长运输周期。',
      time_window: '中期',
      confidence,
      scope: { type: 'industry_chain_node', key: 'chn-21-n01' },
      has_evidence: true
    },
    {
      key: 'chn-21-n02',
      display_order: 2,
      name: '成品油批发交付服务',
      impact: '经营景气承压',
      result: cooling,
      nature: { code: 'reasoning_hypothesis', label: '推理假设' },
      reasoning: '更高物流成本继续向交付环节传导。',
      time_window: '短期–中期',
      confidence,
      scope: { type: 'industry_chain_node', key: 'chn-21-n02' },
      has_evidence: false
    }
  ],
  edges: [
    {
      key: 'chn-21-edge-01',
      display_order: 1,
      from_node_key: 'chn-21-n02',
      to_node_key: 'chn-21-n01',
      relation_label: '依赖'
    }
  ],
  uncertainty: {
    counterevidence_and_gap: '批发交付仍需经营数据。',
    stop_condition: '运输信号反转则停止。',
    checkpoints: []
  },
  scope: { type: 'industry_chain', key: 'chn-21' },
  has_evidence: true
};

describe('Report BFF wire contract', () => {
  it('parses all same-day Report groups without merging cards', () => {
    const second = {
      ...report,
      id: 'RPT22222222-2222-4222-8222-222222222222',
      published_at: '2026-09-01T04:30:00Z'
    };
    const home = parseReportHomeWire({
      selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
      reports: [homeGroup(), homeGroup(second)]
    });

    expect(home.reports).toHaveLength(2);
    expect(home.reports[0].cards.map((card) => card.key)).toEqual([
      'card-geopolitics',
      'card-macroeconomics',
      'card-chn-21'
    ]);
    expect(home.reports[1].report.id).toBe(second.id);
    expect(home.reports[0].industryChainCount).toBe(54);
  });

  it('accepts today empty and fails closed on legacy/surplus/fallback-empty shapes', () => {
    expect(
      parseReportHomeWire({
        selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
        reports: []
      }).reports
    ).toEqual([]);
    expect(() =>
      parseReportHomeWire({
        selection: {
          mode: 'latest_fallback',
          date: '2026-09-01',
          timezone: 'Asia/Shanghai'
        },
        reports: []
      })
    ).toThrow('invalid Report wire contract');
    expect(() => parseReportHomeWire({ report: homeGroup() })).toThrow(
      'invalid Report wire contract'
    );
    const { industry_chain_count: _missingIndustryChainCount, ...missingCount } = homeGroup();
    expect(() =>
      parseReportHomeWire({
        selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
        reports: [missingCount]
      })
    ).toThrow('invalid Report wire contract');
    expect(() =>
      parseReportHomeWire({
        selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
        reports: [
          {
            ...homeGroup(),
            report: { ...report, unexpected_legacy_field: 'private-value' }
          }
        ]
      })
    ).toThrow('invalid Report wire contract');
  });

  it('accepts provider-valid display strings beyond legacy frontend limits', () => {
    const legalLongText = '长'.repeat(501);
    const longCards = cards.map((card, index) =>
      index === 0
        ? {
            ...card,
            title: legalLongText,
            time_window: legalLongText,
            impact_items: [{ ...card.impact_items[0], name: legalLongText }]
          }
        : card
    );
    const home = parseReportHomeWire({
      selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
      reports: [
        {
          ...homeGroup({ ...report, title: legalLongText }),
          cards: longCards,
          company: { ...homeGroup().company, title: legalLongText }
        }
      ]
    });

    expect(home.reports[0].report.title).toBe(legalLongText);
    expect(home.reports[0].cards[0].impactItems[0].name).toBe(legalLongText);
    expect(home.reports[0].company.title).toBe(legalLongText);
    expect(() =>
      parseReportHomeWire({
        selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
        reports: [homeGroup({ ...report, title: '超'.repeat(10_001) })]
      })
    ).toThrow('invalid Report wire contract');
  });

  it('parses layer scopes, full target reference types and gapped related-chain order', () => {
    const providerValidBoundaryNotes = Array.from(
      { length: 101 },
      (_, index) => `边界说明 ${index + 1}`
    );
    const detail = parseReportLayerDetailWire(
      {
        report,
        layer: {
          ...layer,
          downward_transmission: {
            ...layer.downward_transmission,
            boundary_notes: providerValidBoundaryNotes
          }
        },
        related_industry_chains: [relatedChain]
      },
      reportId,
      'geopolitics'
    );

    expect(detail.layer.downwardTransmission.publishedPaths[0].targetRefs).toHaveLength(4);
    expect(detail.layer.downwardTransmission.boundaryNotes).toEqual(providerValidBoundaryNotes);
    expect(detail.relatedIndustryChains[0]).toMatchObject({
      key: 'chn-21',
      displayOrder: 21,
      detailRef: { type: 'industry_chain', key: 'chn-21' }
    });
    expect(() =>
      parseReportLayerDetailWire(
        {
          report,
          layer: { ...layer, scope: { type: 'layer', key: 'macroeconomics' } },
          related_industry_chains: [relatedChain]
        },
        reportId,
        'geopolitics'
      )
    ).toThrow('invalid Report wire contract');
  });

  it('preserves only explicit chain edges and rejects unknown graph endpoints', () => {
    const detail = parseReportIndustryChainDetailWire(
      { report, industry_chain: chain },
      reportId,
      'chn-21'
    );

    expect(detail.industryChain.edges).toEqual([
      {
        key: 'chn-21-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-21-n02',
        toNodeKey: 'chn-21-n01',
        relationLabel: '依赖'
      }
    ]);
    expect(() =>
      parseReportIndustryChainDetailWire(
        {
          report,
          industry_chain: {
            ...chain,
            edges: [{ ...chain.edges[0], to_node_key: 'missing-node' }]
          }
        },
        reportId,
        'chn-21'
      )
    ).toThrow('invalid Report wire contract');
  });

  it('parses only Evidence display fields and preserves BFF order including duplicates', () => {
    const duplicate = {
      published_at: '2026-09-01T03:00:00Z',
      summary: '霍尔木兹通行扰动进入油品运输周期。',
      keywords: ['霍尔木兹', '油品运输']
    };
    const evidences = parseReportEvidenceListWire(
      {
        report_id: reportId,
        scope: { type: 'industry_chain', key: 'chn-21' },
        items: [duplicate, duplicate, { ...duplicate, published_at: null }]
      },
      reportId,
      { type: 'industry_chain', key: 'chn-21' }
    );

    expect(evidences.items).toEqual([
      {
        publishedAt: '2026-09-01T03:00:00Z',
        summary: duplicate.summary,
        keywords: duplicate.keywords
      },
      {
        publishedAt: '2026-09-01T03:00:00Z',
        summary: duplicate.summary,
        keywords: duplicate.keywords
      },
      { publishedAt: null, summary: duplicate.summary, keywords: duplicate.keywords }
    ]);
    expect(() =>
      parseReportEvidenceListWire(
        {
          report_id: reportId,
          scope: { type: 'industry_chain', key: 'chn-21' },
          items: [{ ...duplicate, evidence_id: 'EVD11111111-1111-4111-8111-111111111111' }]
        },
        reportId,
        { type: 'industry_chain', key: 'chn-21' }
      )
    ).toThrow('invalid Report wire contract');
  });
});
