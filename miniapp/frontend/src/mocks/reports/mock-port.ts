import type {
  ReportAnchor,
  ReportCard,
  ReportConfidence,
  ReportEvidence,
  ReportEvidenceScope,
  ReportHome,
  ReportIndustryChainDetail,
  ReportIndustryChainDetailContent,
  ReportIndustryChainNode,
  ReportLayerDetail,
  ReportNature,
  ReportPort,
  ReportResult,
  ReportSummary
} from '../../features/reports/contract';
import { ReportError } from '../../features/reports/contract';

const REPORT_ID = 'RPT11111111-1111-4111-8111-111111111111';

const warming: ReportResult = { code: 'warming', label: '升温' };
const cooling: ReportResult = { code: 'cooling', label: '降温' };
const diverging: ReportResult = { code: 'diverging', label: '分化' };
const pending: ReportResult = { code: 'pending', label: '待验证' };
const high: ReportConfidence = { label: '高', score: null };
const medium: ReportConfidence = { label: '中', score: null };
const low: ReportConfidence = { label: '低', score: null };
const direct: ReportNature = { code: 'direct_evidence', label: '直接证据' };
const inferred: ReportNature = { code: 'reasoning_hypothesis', label: '推理假设' };
const validation: ReportNature = { code: 'pending_validation', label: '待验证' };

const report: ReportSummary = {
  id: REPORT_ID,
  title: '当前事件如何从地缘政治与宏观经济传导至产业链（动态传导）',
  generatedAt: '2026-09-01T12:39:03+08:00',
  publishedAt: '2026-09-01T12:45:00+08:00'
};

const geoAnchors: ReportAnchor[] = [
  anchor(
    'geo-a01',
    1,
    '伊朗—美以及海湾安全对抗',
    '地缘政治风险上升；航运安全与贸易通道可用性下降',
    warming,
    high,
    '即时–中期',
    '军事冲突、船只扣押、霍尔木兹袭击与通行量骤降共同推高海湾对抗风险。'
  ),
  anchor(
    'geo-a02',
    2,
    '南海海洋权益与安全争端',
    '地缘政治风险上升；商业航运安全暂时稳定',
    diverging,
    medium,
    '短期–中期',
    '黄岩岛执法巡查强化主权对抗，但报告未显示主要商业航线受阻。'
  )
];

const macroAnchors: ReportAnchor[] = [
  anchor(
    'macro-a01',
    1,
    '加息',
    '政策利率预期上升',
    warming,
    medium,
    '中期',
    '政策沟通抬高了未来政策利率路径预期。'
  ),
  anchor(
    'macro-a02',
    2,
    '增长预期修正',
    '全球经济增长预期下降',
    cooling,
    medium,
    '中期',
    '能源供应持续中断情景下的增长预测被下调。'
  )
];

const chainDetails: ReportIndustryChainDetailContent[] = [
  chain(
    'chn-01',
    1,
    '人形机器人产业链',
    '人形机器人及关键部件出现直接升温信号，控制计算平台的需求上升仍属于传导假设。',
    warming,
    medium,
    '中期–长期',
    [
      node(
        'chn-01-n01',
        1,
        '人形机器人',
        '商业化进度与渗透率上升',
        warming,
        direct,
        medium,
        '中期–长期',
        '比赛与行业扩产信息显示具身智能正从验证走向规模应用。'
      ),
      node(
        'chn-01-n02',
        2,
        '人形机器人传感器',
        '商业化进度上升',
        warming,
        direct,
        medium,
        '长期',
        '整机应用进展为关键感知部件提供直接的商业化验证。'
      ),
      node(
        'chn-01-n03',
        3,
        '人形机器人减速器',
        '商业化进度上升',
        warming,
        direct,
        medium,
        '长期',
        '整机应用进展为关键执行部件提供直接的商业化验证。'
      ),
      node(
        'chn-01-n04',
        4,
        '人形机器人电机',
        '商业化进度上升',
        warming,
        direct,
        medium,
        '长期',
        '整机应用进展为核心动力部件提供直接的商业化验证。'
      ),
      node(
        'chn-01-n05',
        5,
        '机器人控制计算平台',
        '市场需求存在上升假设',
        warming,
        inferred,
        low,
        '中期–长期（传导滞后）',
        '整机放量会增加运动控制、感知融合和实时决策计算需求，仍需订单与经营数据验证。'
      )
    ],
    [
      edge('chn-01-edge-01', 1, 'chn-01-n04', 'chn-01-n01', '组成'),
      edge('chn-01-edge-02', 2, 'chn-01-n03', 'chn-01-n01', '组成'),
      edge('chn-01-edge-03', 3, 'chn-01-n05', 'chn-01-n01', '组成'),
      edge('chn-01-edge-04', 4, 'chn-01-n02', 'chn-01-n01', '组成')
    ],
    '传导假设仍需目标节点的订单、价格、产能或经营数据验证。',
    '若节点信号失效、方向反转或链语境不成立，则关闭或修正本链结论。'
  ),
  chain(
    'chn-02',
    2,
    'AI数据中心液冷服务器产业链',
    'AI芯片、液冷服务器与液冷系统升温，服务器冷板需求上升仍需订单验证。',
    warming,
    medium,
    '中期–长期',
    [
      node(
        'chn-02-n01',
        1,
        'AI芯片',
        '商业化与技术成熟度上升',
        warming,
        direct,
        medium,
        '中期–长期',
        '定制芯片合作与自研加速器测试推进。'
      ),
      node(
        'chn-02-n02',
        2,
        '液冷服务器',
        '商业化进度上升',
        warming,
        direct,
        medium,
        '长期',
        '国家级液冷标准建立统一技术基准。'
      ),
      node(
        'chn-02-n03',
        3,
        '服务器冷板',
        '市场需求存在上升假设',
        warming,
        inferred,
        low,
        '长期（传导滞后）',
        '液冷服务器交付增加可能带动核心冷板配套。'
      )
    ],
    [
      edge('chn-02-edge-01', 1, 'chn-02-n01', 'chn-02-n02', '组成'),
      edge('chn-02-edge-02', 2, 'chn-02-n03', 'chn-02-n02', '组成')
    ],
    '共享节点的本链语境与冷板订单仍待验证。',
    '若液冷服务器交付未增长或节点信号反转，则下调本链结论。'
  ),
  chain(
    'chn-03',
    3,
    'AI算力基础设施服务产业链',
    'AI芯片与算力供给升温，但对服务器、数据中心和调度平台的传导仍待验证。',
    warming,
    medium,
    '中期–长期',
    [
      node(
        'chn-03-n01',
        1,
        'AI芯片',
        '商业化与技术成熟度上升',
        warming,
        direct,
        medium,
        '中期–长期',
        '定制芯片合作与加速器测试推进。'
      ),
      node(
        'chn-03-n02',
        2,
        '算力供给',
        '有效产能与市场供给上升',
        warming,
        direct,
        medium,
        '中期',
        '全国智算规模与调度能力继续扩张。'
      ),
      node(
        'chn-03-n03',
        3,
        'AI服务器',
        '同链相邻，方向待确认',
        pending,
        validation,
        low,
        '后续周期',
        '缺少订单、价格、产能或经营数据。'
      )
    ],
    [edge('chn-03-edge-01', 1, 'chn-03-n01', 'chn-03-n03', '组成')],
    '同链相邻节点缺少直接观测与经营数据。',
    '若算力供给增量未转化为服务器需求，则停止向下传导。'
  ),
  chain(
    'chn-21',
    4,
    '油品石化贸易服务产业链',
    '油品运输服务直接分化，运输成本与到货周期向批发交付环节传导的假设偏降温。',
    diverging,
    high,
    '中期',
    [
      node(
        'chn-21-n01',
        1,
        '油品运输服务',
        '交付周期与销售价格上升，战略资源安全下降',
        diverging,
        direct,
        high,
        '中期',
        '海湾冲突扰乱霍尔木兹通行并推高油运景气，运输收益与交付风险同时上升。'
      ),
      node(
        'chn-21-n02',
        2,
        '成品油批发交付服务',
        '投入成本与交付周期上升，经营景气承压',
        cooling,
        inferred,
        low,
        '短期–中期（传导滞后）',
        '批发交付明确依赖运输服务，更高物流成本和更长到货时间可能向下传导。'
      )
    ],
    [edge('chn-21-edge-01', 1, 'chn-21-n02', 'chn-21-n01', '依赖')],
    '批发交付节点仍需订单、价格、库存和经营数据验证。',
    '若油运信号失效、方向反转或链语境不成立，则关闭或修正本链结论。'
  )
];

const cards: ReportCard[] = [
  card(
    'card-geopolitics',
    'geopolitics',
    1,
    '地缘政治',
    '安全对抗与通道可用性',
    '伊朗—美以及海湾安全对抗持续升温，霍尔木兹航运安全与贸易通道可用性下降；南海商业航运安全暂时稳定。',
    diverging,
    high,
    '即时–中期',
    'layer',
    'geopolitics',
    geoAnchors
  ),
  card(
    'card-macroeconomics',
    'macroeconomics',
    2,
    '宏观经济',
    '增长预期与政策利率',
    '全球增长预期受到能源中断情景压制，同时加息预期升温，构成偏紧的宏观组合。',
    diverging,
    medium,
    '中期',
    'layer',
    'macroeconomics',
    macroAnchors
  ),
  ...chainDetails.map((item, index) =>
    card(
      `card-${item.key}`,
      'industry_chain',
      index + 3,
      item.name,
      index === 3 ? '海湾风险直接关联链' : '产业链',
      item.conclusion,
      item.result,
      item.confidence,
      item.timeWindow,
      'industry_chain',
      item.key,
      item.nodes
    )
  )
];

const home: ReportHome = {
  selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
  reports: [
    {
      report,
      cards,
      company: {
        key: 'company',
        displayOrder: 4,
        title: '企业',
        published: false,
        boundary: '本次推理尚未进入企业层，不生成企业影响结论。'
      }
    }
  ]
};

const relatedIndustryChains = chainDetails.map((item) => ({
  key: item.key,
  displayOrder: item.displayOrder,
  name: item.name,
  result: item.result,
  detailRef: { type: 'industry_chain' as const, key: item.key }
}));

const layerDetails: Record<'geopolitics' | 'macroeconomics', ReportLayerDetail> = {
  geopolitics: {
    report,
    layer: {
      key: 'geopolitics',
      displayOrder: 1,
      scope: { type: 'layer', key: 'geopolitics' },
      title: '地缘政治',
      conclusion: cards[0].conclusion,
      result: diverging,
      confidence: high,
      timeWindow: '即时–中期',
      anchors: geoAnchors,
      reasoningSteps: [
        {
          key: 'geo-s01',
          displayOrder: 1,
          scope: { type: 'reasoning_step', key: 'geo-s01' },
          input: '霍尔木兹通行量骤降且油轮遭袭',
          mechanism: '安全威胁改变船东、保险与贸易通道定价',
          output: '海湾航运摩擦与通道压力上升',
          type: '证据 → 推理',
          confidence: high,
          hasEvidence: true
        },
        {
          key: 'geo-s02',
          displayOrder: 2,
          scope: { type: 'reasoning_step', key: 'geo-s02' },
          input: '南海执法巡查增强',
          mechanism: '主权对抗升温，但主要商业航线未显示受阻',
          output: '争端风险升温、航运安全局部稳定',
          type: '证据边界',
          confidence: medium,
          hasEvidence: true
        }
      ],
      downwardTransmission: {
        summary: '只有目标为报告内稳定对象且传导机制闭合时，才形成跨层入口。',
        publishedPaths: [
          {
            key: 'geo-x-macro-01',
            displayOrder: 1,
            scope: { type: 'transmission_path', key: 'geo-x-macro-01' },
            sourceConclusion: '海湾冲突与霍尔木兹通道压力上升',
            targetRefs: [
              {
                ref: { type: 'layer', key: 'macroeconomics' },
                label: '宏观经济',
                result: diverging
              }
            ],
            logic: '能源中断风险增强全球增长下行情景，同时运输摩擦进入成本预期。',
            relationNature: '推理假设（目标另有直接证据）',
            evidenceRole: '来源与目标证据分离',
            confidence: medium,
            status: '已形成解释闭环，仍需连续油运、能源价格和增长预测验证。',
            hasEvidence: true
          },
          {
            key: 'geo-x-chn-21',
            displayOrder: 2,
            scope: { type: 'transmission_path', key: 'geo-x-chn-21' },
            sourceConclusion: '霍尔木兹通行能力下降',
            targetRefs: [
              {
                ref: { type: 'industry_chain', key: 'chn-21' },
                label: '油品石化贸易服务产业链',
                result: diverging
              }
            ],
            logic: '通行受阻延长油品运输周期并抬高运价，成本和到货时间继续向批发交付传导。',
            relationNature: '同源直接证据 + 节点传导假设',
            evidenceRole: '运输节点直接关联，批发交付节点待验证',
            confidence: medium,
            status: '运输节点已闭合，批发交付节点仍需经营数据。',
            hasEvidence: true
          }
        ],
        candidateMechanisms: [],
        boundaryNotes: ['不把同一证据在两层分别产生的直接结果改写为跨层因果。']
      },
      uncertainty: {
        counterevidence: '南海商业航运安全暂时稳定，不能把执法巡查扩展为主要航线受阻。',
        evidenceGap: '缺霍尔木兹连续船舶量、港口装卸量、海湾出口量和冲突降级信号。',
        boundary: '地理国家信息只作为背景，不作为本层锚点。',
        reversalCondition: '若霍尔木兹通行恢复、袭击风险下降且冲突降级，则下调海湾对抗结论。',
        checkpoints: []
      },
      hasEvidence: true
    },
    relatedIndustryChains: relatedIndustryChains.filter((item) => item.key === 'chn-21')
  },
  macroeconomics: {
    report,
    layer: {
      key: 'macroeconomics',
      displayOrder: 2,
      scope: { type: 'layer', key: 'macroeconomics' },
      title: '宏观经济',
      conclusion: cards[1].conclusion,
      result: diverging,
      confidence: medium,
      timeWindow: '中期',
      anchors: macroAnchors,
      reasoningSteps: [
        {
          key: 'macro-s01',
          displayOrder: 1,
          scope: { type: 'reasoning_step', key: 'macro-s01' },
          input: '政策沟通暗示可能加息',
          mechanism: '未来政策利率路径预期上调',
          output: '加息锚点升温',
          type: '证据 → 结果',
          confidence: medium,
          hasEvidence: true
        },
        {
          key: 'macro-s02',
          displayOrder: 2,
          scope: { type: 'reasoning_step', key: 'macro-s02' },
          input: '能源持续中断情景下增长预测下调',
          mechanism: '权威预测压低全球增长预期',
          output: '增长预期降温',
          type: '证据 → 结果',
          confidence: medium,
          hasEvidence: true
        }
      ],
      downwardTransmission: {
        summary: '当前缺少宏观锚点到具体产业节点的完整闭环，因此不新增产业链方向结论。',
        publishedPaths: [],
        candidateMechanisms: [
          {
            key: 'macro-candidate-01',
            displayOrder: 1,
            scope: { type: 'candidate_mechanism', key: 'macro-candidate-01' },
            mechanism: '增长预期下降可能压低外需与周期品需求。',
            evidenceGap: '缺中国外需、工业需求和具体产业节点结果。',
            confidence: low,
            hasEvidence: false
          }
        ],
        boundaryNotes: ['未闭合的候选机制不进入产业链结论。']
      },
      uncertainty: {
        counterevidence: '能源替代来源、政策对冲和通胀回落可能削弱共同冲击。',
        evidenceGap: '缺中国进口成本、PPI、人民币汇率、工业增加值和风险偏好观测。',
        boundary: '没有宏观结果的证据不写成宏观锚点结论。',
        reversalCondition: '若能源运输恢复、增长预测回升或加息预期撤回，则下调宏观偏紧结论。',
        checkpoints: []
      },
      hasEvidence: true
    },
    relatedIndustryChains: []
  }
};

const evidenceByScope = new Map<string, ReportEvidence[]>([
  [
    'report_card:card-geopolitics',
    [
      evidence(
        '2026-08-31T07:25:00+08:00',
        '霍尔木兹海峡商业油轮通行显著下降，油轮遇袭风险上升。',
        ['霍尔木兹', '油轮通行', '航运安全']
      ),
      evidence('2026-08-31T07:20:00+08:00', '海湾油运保险与运输定价继续上行。', [
        '海湾油运',
        '保险定价'
      ])
    ]
  ],
  [
    'report_card:card-macroeconomics',
    [
      evidence('2026-08-31T08:10:00+08:00', '能源持续中断情景下，全球增长预测被下调。', [
        '能源供应',
        '增长预期'
      ]),
      evidence('2026-08-31T08:00:00+08:00', '政策沟通抬高未来加息路径预期。', [
        '政策利率',
        '加息预期'
      ])
    ]
  ],
  [
    'report_card:card-chn-01',
    [
      evidence('2026-08-31T09:20:00+08:00', '人形机器人比赛展示技术进步与应用进展。', [
        '人形机器人',
        '技术进步'
      ])
    ]
  ],
  [
    'report_card:card-chn-02',
    [
      evidence(
        '2026-08-29T18:00:00+08:00',
        '首个国家级液冷标准发布，为高功率AI机架建立统一技术基准。',
        ['液冷标准', 'AI机架']
      )
    ]
  ],
  [
    'report_card:card-chn-03',
    [
      evidence('2026-08-31T09:40:00+08:00', '全国智算规模与国家级监测调度能力继续扩张。', [
        '智算规模',
        '算力调度'
      ])
    ]
  ],
  [
    'report_card:card-chn-21',
    [
      evidence('2026-08-31T07:25:00+08:00', '霍尔木兹通行扰动延长油品运输周期并推高油运景气。', [
        '油品运输',
        '霍尔木兹'
      ])
    ]
  ],
  [
    'layer:geopolitics',
    [
      evidence(
        '2026-08-31T07:25:00+08:00',
        '霍尔木兹海峡商业油轮通行显著下降，油轮遇袭风险上升。',
        ['霍尔木兹', '航运安全']
      )
    ]
  ],
  [
    'layer:macroeconomics',
    [
      evidence('2026-08-31T08:10:00+08:00', '能源持续中断情景下，全球增长预测被下调。', [
        '增长预测',
        '能源供应'
      ])
    ]
  ],
  [
    'anchor:geo-a01',
    [
      evidence('2026-08-31T07:25:00+08:00', '霍尔木兹通行量下降并出现袭击警告。', [
        '霍尔木兹',
        '袭击警告'
      ])
    ]
  ],
  [
    'anchor:geo-a02',
    [evidence('2026-08-31T06:50:00+08:00', '黄岩岛周边执法巡查继续加强。', ['黄岩岛', '执法巡查'])]
  ],
  [
    'anchor:macro-a01',
    [
      evidence('2026-08-31T08:00:00+08:00', '政策沟通抬高未来加息路径预期。', [
        '政策利率',
        '加息预期'
      ])
    ]
  ],
  [
    'anchor:macro-a02',
    [
      evidence('2026-08-31T08:10:00+08:00', '能源中断情景下全球增长预测下调。', [
        '增长预测',
        '能源供应'
      ])
    ]
  ],
  [
    'transmission_path:geo-x-chn-21',
    [
      evidence('2026-08-31T07:25:00+08:00', '霍尔木兹通行扰动进入油品运输周期与价格。', [
        '霍尔木兹',
        '油品运输'
      ])
    ]
  ],
  ...chainDetails.flatMap((item) => [
    [`industry_chain:${item.key}`, evidenceForChain(item)] as [string, ReportEvidence[]],
    ...item.nodes.map(
      (itemNode) =>
        [`industry_chain_node:${itemNode.key}`, evidenceForNode(itemNode)] as [
          string,
          ReportEvidence[]
        ]
    )
  ])
]);

export class MockReportPort implements ReportPort {
  async getHome(): Promise<ReportHome> {
    return home;
  }

  async getLayer(
    reportId: string,
    layerKey: 'geopolitics' | 'macroeconomics'
  ): Promise<ReportLayerDetail> {
    assertReport(reportId);
    const detail = layerDetails[layerKey];
    if (!detail) throw new ReportError('layerUnavailable');
    return detail;
  }

  async getIndustryChain(reportId: string, chainKey: string): Promise<ReportIndustryChainDetail> {
    assertReport(reportId);
    const industryChain = chainDetails.find((item) => item.key === chainKey);
    if (!industryChain) throw new ReportError('chainUnavailable');
    return { report, industryChain };
  }

  async getEvidences(reportId: string, scope: ReportEvidenceScope) {
    assertReport(reportId);
    const items = evidenceByScope.get(`${scope.type}:${scope.key}`);
    if (!items) throw new ReportError('evidenceScopeUnavailable');
    return { reportId, scope, items };
  }
}

export const mockReportPort = new MockReportPort();

function assertReport(reportId: string): void {
  if (reportId !== REPORT_ID) throw new ReportError('reportUnavailable');
}

function anchor(
  key: string,
  displayOrder: number,
  name: string,
  currentState: string,
  result: ReportResult,
  confidence: ReportConfidence,
  timeWindow: string,
  reasoning: string
): ReportAnchor {
  return {
    key,
    displayOrder,
    scope: { type: 'anchor', key },
    name,
    currentState,
    result,
    nature: direct,
    reasoning,
    timeWindow,
    confidence,
    hasEvidence: true
  };
}

function node(
  key: string,
  displayOrder: number,
  name: string,
  impact: string,
  result: ReportResult,
  nature: ReportNature,
  confidence: ReportConfidence,
  timeWindow: string,
  reasoning: string
): ReportIndustryChainNode {
  return {
    key,
    displayOrder,
    scope: { type: 'industry_chain_node', key },
    name,
    impact,
    result,
    nature,
    reasoning,
    timeWindow,
    confidence,
    hasEvidence: nature.code !== 'pending_validation'
  };
}

function chain(
  key: string,
  displayOrder: number,
  name: string,
  conclusion: string,
  result: ReportResult,
  confidence: ReportConfidence,
  timeWindow: string,
  nodes: ReportIndustryChainNode[],
  edges: ReportIndustryChainDetailContent['edges'],
  counterevidenceAndGap: string,
  stopCondition: string
): ReportIndustryChainDetailContent {
  return {
    key,
    claimKey: `${key}-claim`,
    displayOrder,
    scope: { type: 'industry_chain', key },
    name,
    conclusion,
    status: '已发布',
    result,
    confidence,
    timeWindow,
    pathSummary: null,
    acceptedHypothesisSummary: null,
    nodes,
    edges,
    uncertainty: { counterevidenceAndGap, stopCondition, checkpoints: [] },
    hasEvidence: true
  };
}

function edge(
  key: string,
  displayOrder: number,
  fromNodeKey: string,
  toNodeKey: string,
  relationLabel: string
) {
  return { key, displayOrder, fromNodeKey, toNodeKey, relationLabel };
}

function card(
  key: string,
  kind: ReportCard['kind'],
  displayOrder: number,
  title: string,
  subtitle: string,
  conclusion: string,
  result: ReportResult,
  confidence: ReportConfidence,
  timeWindow: string,
  targetType: ReportCard['detailRef']['type'],
  targetKey: string,
  items: Array<ReportAnchor | ReportIndustryChainNode>
): ReportCard {
  return {
    key,
    kind,
    displayOrder,
    detailRef: { type: targetType, key: targetKey },
    title,
    subtitle,
    conclusion,
    result,
    confidence,
    timeWindow,
    impactItems: items.map((item) => ({
      ref: {
        type: kind === 'industry_chain' ? ('industry_chain_node' as const) : ('anchor' as const),
        key: item.key
      },
      name: item.name,
      result: item.result,
      confidence: item.confidence,
      timeWindow: item.timeWindow,
      hasEvidence: item.hasEvidence
    })),
    hasEvidence: true
  };
}

function evidence(publishedAt: string | null, summary: string, keywords: string[]): ReportEvidence {
  return { publishedAt, summary, keywords };
}

function evidenceForChain(item: ReportIndustryChainDetailContent): ReportEvidence[] {
  return [
    evidence('2026-08-31T09:20:00+08:00', item.conclusion, [
      item.name.replace('产业链', ''),
      item.result.label
    ])
  ];
}

function evidenceForNode(item: ReportIndustryChainNode): ReportEvidence[] {
  return item.hasEvidence
    ? [evidence('2026-08-31T09:20:00+08:00', item.impact, [item.name, item.result.label])]
    : [];
}
