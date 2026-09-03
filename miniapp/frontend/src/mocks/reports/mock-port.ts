import type {
  ReportAnchor,
  ReportCard,
  ReportCardPage,
  ReportCodedLabel,
  ReportConfidence,
  ReportEvidence,
  ReportHome,
  ReportIndustryChainDetail,
  ReportIndustryChainDetailContent,
  ReportIndustryChainNode,
  ReportLayerDetail,
  ReportLayerKey,
  ReportPort,
  ReportResult,
  ReportSummary,
  ReportTimeWindow
} from '../../features/reports/contract';
import { ReportError } from '../../features/reports/contract';

const REPORT_ID = 'RPT11111111-1111-4111-8111-111111111111';
const direct = code('direct_evidence', '直接证据');
const hypothesis = code('reasoning_hypothesis', '推理假设');
const confirmed = code('confirmed', '已确认');
const pendingValidation = code('pending_validation', '待验证');
const warming = code('warming', '升温');
const cooling = code('cooling', '降温');
const diverging = code('diverging', '分化');
const medium = confidence('medium', '中');
const high = confidence('high', '高');
const low = confidence('low', '低');

const report: ReportSummary = {
  id: REPORT_ID,
  generatedAt: '2026-09-01T04:39:03Z',
  publishedAt: '2026-09-01T04:45:00Z',
  industryChainCount: 54
};

const evidenceByToken = new Map<string, ReportEvidence[]>();
let tokenSequence = 1;

function evidenceToken(summary: string, keywords: string[]): string {
  const token = `RPE00000000-0000-4000-8000-${String(tokenSequence++).padStart(12, '0')}`;
  evidenceByToken.set(token, [{ publishedAt: '2026-08-31T02:30:06Z', summary, keywords }]);
  return token;
}

const geoAnchors: ReportAnchor[] = [
  anchor(
    'geo-a01',
    '伊朗—美以及海湾安全对抗',
    '地缘政治风险 UP；航运安全 DOWN；贸易通道可用性 DOWN',
    warming,
    '美伊军事冲突、船只扣押、霍尔木兹袭击与通行量骤降直接推高海湾对抗风险。'
  ),
  anchor(
    'geo-a02',
    '南海海洋权益与安全争端',
    '地缘政治风险 UP；航运安全 STABLE',
    diverging,
    '黄岩岛执法巡查强化主权对抗，但主要商业航线尚未显示受阻。'
  )
];

const macroAnchors: ReportAnchor[] = [
  anchor('macro-a01', '加息', '政策利率 UP/MEDIUM', warming, '政策沟通抬高未来政策利率路径预期。'),
  anchor(
    'macro-a02',
    '增长预期修正',
    '经济增长预期 DOWN/MEDIUM',
    cooling,
    '能源供应持续中断情景下的权威预测直接压低全球增长预期。'
  )
];

const chainNames = [
  '人形机器人产业链',
  'AI数据中心液冷服务器产业链',
  'AI算力基础设施服务产业链',
  'AI视频生成服务产业链',
  '油气勘探开发产业链',
  '生成式人工智能模型及应用服务产业链',
  '汽车线控制动执行器产业链',
  '企业AI智能体产业链',
  '智能语音技术服务产业链',
  'AI智能手机产业链',
  'AI计算芯片产业链',
  '电动汽车换电服务产业链'
];

const chains: ReportIndustryChainDetailContent[] = Array.from({ length: 54 }, (_, index) => {
  const number = index + 1;
  const key = `chn-${String(number).padStart(2, '0')}`;
  const name = chainNames[index] ?? `产业链分析 ${number}`;
  const nodes = number === 1 ? humanoidNodes(key) : standardNodes(key, name);
  return {
    key,
    name,
    conclusion: `${name}的直接信号与结构化传导已形成当前报告结论，相邻节点仍需后续验证。`,
    result: number % 9 === 0 ? diverging : warming,
    confidence: number === 1 ? medium : low,
    timeWindow: window('medium_long', '中期–长期'),
    pathSummary: nodes.map((graphNode) => graphNode.name).join(' → '),
    acceptedHypothesisSummary: null,
    topologyNodes: nodes.map((graphNode) => ({
      key: graphNode.key,
      name: graphNode.name
    })),
    nodes,
    edges: nodes.slice(1).map((graphNode) => ({
      fromNodeLocalKey: graphNode.key,
      toNodeLocalKey: nodes[0]!.key,
      relationLabel: '组成'
    })),
    counterevidenceAndGap: '仍需目标节点的订单、价格、产能或经营数据验证。',
    stopCondition: '若后续事件使节点信号失效、方向反转或链语境不成立，则修正本链结论。',
    evidenceScopeToken: evidenceToken(`${name}摘要依据`, [name.replace('产业链', ''), '产业链'])
  };
});

const layerDetails: Record<ReportLayerKey, ReportLayerDetail> = {
  geopolitics: layerDetail('geopolitics', '地缘政治', geoAnchors, diverging),
  macroeconomics: layerDetail('macroeconomics', '宏观经济', macroAnchors, diverging)
};

const layerCards: ReportCard[] = [
  layerCard(layerDetails.geopolitics),
  layerCard(layerDetails.macroeconomics)
];
const chainCards = chains.map(chainCard);

const home: ReportHome = {
  selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
  reports: [{ report, cards: [...layerCards, ...chainCards.slice(0, 20)], nextCursor: '20' }]
};

export class MockReportPort implements ReportPort {
  async getHome(): Promise<ReportHome> {
    return home;
  }

  async getIndustryChains(reportId: string, cursor = '', limit = 20): Promise<ReportCardPage> {
    assertReport(reportId);
    const offset = cursor === '' ? 0 : Number(cursor);
    if (!Number.isSafeInteger(offset) || offset < 0 || limit < 1)
      throw new ReportError('invalidRequest');
    const items = chainCards.slice(offset, offset + limit);
    const nextOffset = offset + items.length;
    return { items, nextCursor: nextOffset < chainCards.length ? String(nextOffset) : null };
  }

  async getLayer(reportId: string, layerKey: ReportLayerKey): Promise<ReportLayerDetail> {
    assertReport(reportId);
    const detail = layerDetails[layerKey];
    if (!detail) throw new ReportError('layerUnavailable');
    return detail;
  }

  async getIndustryChain(reportId: string, chainKey: string): Promise<ReportIndustryChainDetail> {
    assertReport(reportId);
    const industryChain = chains.find((item) => item.key === chainKey);
    if (!industryChain) throw new ReportError('chainUnavailable');
    return { report, industryChain };
  }

  async getEvidences(reportId: string, scopeToken: string) {
    assertReport(reportId);
    const items = evidenceByToken.get(scopeToken);
    if (!items) throw new ReportError('evidenceScopeUnavailable');
    return { reportId, scopeToken, items };
  }
}

export const mockReportPort = new MockReportPort();

function layerDetail(
  key: ReportLayerKey,
  title: string,
  anchors: ReportAnchor[],
  result: ReportResult
): ReportLayerDetail {
  return {
    report,
    layer: {
      key,
      title,
      conclusion:
        key === 'geopolitics'
          ? '伊朗—美以及海湾安全对抗持续升温，霍尔木兹航运安全与贸易通道可用性下降。'
          : '全球增长预期受到能源中断情景压制，同时政策利率预期升温。',
      result,
      confidence: medium,
      timeWindow: window('medium', '中期'),
      anchors,
      reasoningSteps: [],
      transmissions: [
        {
          key: `${key}-path-01`,
          sourceConclusion: title,
          targets: [
            {
              ref: {
                type: key === 'geopolitics' ? 'layer' : 'industry_chain',
                localKey: key === 'geopolitics' ? 'macroeconomics' : 'chn-01'
              },
              name: key === 'geopolitics' ? '宏观经济' : '人形机器人产业链',
              result: diverging
            }
          ],
          logic: '结构化目标与传导机制已形成报告内闭环，仍需后续数据验证。',
          kind: hypothesis,
          confidence: medium,
          status: pendingValidation
        }
      ],
      uncertainty: {
        counterevidence: '局部指标仍保持稳定。',
        evidenceGap: '缺少连续经营与价格数据。',
        boundary: '不把同源信号改写为直接因果。',
        reversalCondition: '若关键事件信号反转则下调结论。'
      },
      evidenceScopeToken: evidenceToken(`${title}一句话结论依据`, [title, '报告结论'])
    },
    relatedIndustryChains: chains.map((chain) => ({
      key: chain.key,
      name: chain.name,
      result: chain.result
    }))
  };
}

function anchor(
  key: string,
  name: string,
  currentState: string,
  result: ReportResult,
  reasoning: string
): ReportAnchor {
  return {
    key,
    name,
    currentState,
    result,
    conclusionBasis: direct,
    validationStatus: confirmed,
    reasoning,
    timeWindow: window('medium', '即时–中期'),
    confidence: high,
    evidenceScopeToken: evidenceToken(`${name}的直接依据`, [name, '直接证据'])
  };
}

function humanoidNodes(chainKey: string): ReportIndustryChainNode[] {
  const names = [
    '人形机器人',
    '人形机器人传感器',
    '人形机器人减速器',
    '人形机器人电机',
    '机器人控制计算平台'
  ];
  return names.map((name, index) => createNode(chainKey, index + 1, name, index < 4));
}

function standardNodes(chainKey: string, chainName: string): ReportIndustryChainNode[] {
  return [
    createNode(chainKey, 1, chainName.replace('产业链', '核心节点'), true),
    createNode(chainKey, 2, '相邻传导节点', false)
  ];
}

function createNode(
  chainKey: string,
  number: number,
  name: string,
  isDirect: boolean
): ReportIndustryChainNode {
  const key = `${chainKey}-n${String(number).padStart(2, '0')}`;
  return {
    key,
    name,
    impact: isDirect ? '商业化进度 UP/MEDIUM' : '市场需求 UP/LOW',
    result: warming,
    conclusionBasis: isDirect ? direct : hypothesis,
    validationStatus: isDirect ? confirmed : pendingValidation,
    reasoning: isDirect
      ? '公开事件直接形成节点信号。'
      : '同链结构关系支持传导假设，仍需目标节点经营数据验证。',
    timeWindow: window('medium_long', isDirect ? '中期–长期' : '中期–长期（传导滞后）'),
    confidence: isDirect ? medium : low,
    evidenceScopeToken: isDirect ? evidenceToken(`${name}的直接依据`, [name, '直接证据']) : null
  };
}

function layerCard(detail: ReportLayerDetail): ReportCard {
  const layer = detail.layer;
  return {
    key: `card-${layer.key}`,
    kind: layer.key,
    detailRef: { type: 'layer', localKey: layer.key },
    title: layer.title,
    subtitle: layer.key === 'geopolitics' ? '安全对抗与通道可用性' : '增长预期与政策利率',
    conclusion: layer.conclusion,
    result: layer.result,
    confidence: layer.confidence,
    timeWindow: layer.timeWindow,
    impactItems: layer.anchors.map((item) => ({
      ref: { type: 'anchor', localKey: item.key },
      name: item.name,
      result: item.result,
      conclusionBasis: item.conclusionBasis,
      validationStatus: item.validationStatus,
      confidence: item.confidence,
      timeWindow: item.timeWindow,
      evidenceScopeToken: item.evidenceScopeToken
    })),
    evidenceScopeToken: layer.evidenceScopeToken
  };
}

function chainCard(chain: ReportIndustryChainDetailContent): ReportCard {
  return {
    key: `card-${chain.key}`,
    kind: 'industry_chain',
    detailRef: { type: 'industry_chain', localKey: chain.key },
    title: chain.name,
    subtitle: '产业链',
    conclusion: chain.conclusion,
    result: chain.result,
    confidence: chain.confidence,
    timeWindow: chain.timeWindow,
    impactItems: chain.nodes.map((item) => ({
      ref: { type: 'industry_chain_node', localKey: item.key },
      name: item.name,
      result: item.result,
      conclusionBasis: item.conclusionBasis,
      validationStatus: item.validationStatus,
      confidence: item.confidence,
      timeWindow: item.timeWindow,
      evidenceScopeToken: item.evidenceScopeToken
    })),
    evidenceScopeToken: chain.evidenceScopeToken
  };
}

function code(codeValue: string, label: string): ReportCodedLabel {
  return { code: codeValue, label };
}

function confidence(codeValue: string, label: string): ReportConfidence {
  return { code: codeValue, label };
}

function window(codeValue: string, label: string): ReportTimeWindow {
  return { code: codeValue, label };
}

function assertReport(reportId: string): void {
  if (reportId !== REPORT_ID) throw new ReportError('reportUnavailable');
}
