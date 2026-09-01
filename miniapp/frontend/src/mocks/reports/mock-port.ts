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
const mediumHigh: ReportConfidence = { label: '中–高', score: null };
const lowMedium: ReportConfidence = { label: '低–中', score: null };
const low060: ReportConfidence = { label: '低（0.60）', score: 0.6 };
const low062: ReportConfidence = { label: '低（0.62）', score: 0.62 };
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
    '地缘政治风险 UP；航运安全 DOWN；贸易通道可用性 DOWN',
    warming,
    high,
    '即时–中期',
    '美伊军事冲突、船只扣押、霍尔木兹袭击与通行量骤降直接推高海湾对抗风险，并压低航运安全与贸易通道可用性。'
  ),
  anchor(
    'geo-a02',
    2,
    '南海海洋权益与安全争端',
    '地缘政治风险 UP；航运安全 STABLE',
    diverging,
    medium,
    '短期–中期',
    '黄岩岛执法巡查强化了主权对抗，直接抬升争端风险；Event 未显示主要商业航线受阻，因此航运安全保持稳定。'
  )
];

const macroAnchors: ReportAnchor[] = [
  anchor(
    'macro-a01',
    1,
    '加息',
    '政策利率 UP/MEDIUM',
    warming,
    medium,
    '中期',
    '美联储主席的加息表态直接抬高未来政策利率路径预期，使“加息”宏观锚点升温。'
  ),
  anchor(
    'macro-a02',
    2,
    '增长预期修正',
    '经济增长预期 DOWN/MEDIUM',
    cooling,
    medium,
    '中期',
    '世界银行在能源供应持续中断情景下下调全球增长预测，直接压低经济增长预期。'
  )
];

const chainDetails: ReportIndustryChainDetailContent[] = [
  chain(
    'chn-01',
    1,
    '人形机器人产业链',
    '人形机器人、人形机器人传感器、人形机器人减速器、人形机器人电机出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    warming,
    medium,
    '中期–长期',
    [
      node(
        'chn-01-n01',
        1,
        '人形机器人',
        '商业化进度 UP/MEDIUM；渗透率 UP/MEDIUM',
        warming,
        direct,
        medium,
        '中期–长期',
        '中国2026年8月在北京举办第二届人形机器人比赛，展示技术进步并引发应用讨论；DIGITIMES评论指出人形机器人行业扩产，具身智能正实体化，AI扩展下人类智能与机器活动界限模糊，反映具身智能技术正从验证进入规模应用。对“人形机器人”环节而言，这意味着商业化进度上升、渗透率上升，因此本期判断为升温。'
      ),
      node(
        'chn-01-n02',
        2,
        '人形机器人传感器',
        '商业化进度 UP/LOW',
        warming,
        direct,
        medium,
        '长期',
        '中国2026年8月在北京举办第二届人形机器人比赛，展示技术进步并引发应用讨论。对“人形机器人传感器”环节而言，这意味着商业化进度上升，因此本期判断为升温。'
      ),
      node(
        'chn-01-n03',
        3,
        '人形机器人减速器',
        '商业化进度 UP/LOW',
        warming,
        direct,
        medium,
        '长期',
        '中国2026年8月在北京举办第二届人形机器人比赛，展示技术进步并引发应用讨论。对“人形机器人减速器”环节而言，这意味着商业化进度上升，因此本期判断为升温。'
      ),
      node(
        'chn-01-n04',
        4,
        '人形机器人电机',
        '商业化进度 UP/LOW',
        warming,
        direct,
        medium,
        '长期',
        '中国2026年8月在北京举办第二届人形机器人比赛，展示技术进步并引发应用讨论。对“人形机器人电机”环节而言，这意味着商业化进度上升，因此本期判断为升温。'
      ),
      node(
        'chn-01-n05',
        5,
        '机器人控制计算平台',
        '市场需求 UP/LOW',
        warming,
        inferred,
        low060,
        '中期–长期（传导滞后）',
        '人形机器人的商业化和渗透率上升，会增加整机对运动控制、感知融合和实时决策计算的需求。控制计算平台是整机的明确组成环节，因此推测其市场需求将随整机放量而上升。'
      )
    ],
    [
      edge('chn-01-edge-01', 1, 'chn-01-n04', 'chn-01-n01', '组成'),
      edge('chn-01-edge-02', 2, 'chn-01-n03', 'chn-01-n01', '组成'),
      edge('chn-01-edge-03', 3, 'chn-01-n05', 'chn-01-n01', '组成'),
      edge('chn-01-edge-04', 4, 'chn-01-n02', 'chn-01-n01', '组成')
    ],
    '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
    '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。'
  ),
  chain(
    'chn-02',
    2,
    'AI数据中心液冷服务器产业链',
    'AI芯片、液冷服务器、液冷系统出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证；本链新增 1 条动态传导假设。',
    warming,
    lowMedium,
    '长期–中期',
    [
      node(
        'chn-02-n01',
        1,
        'AI芯片',
        '商业化进度 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        warming,
        direct,
        lowMedium,
        '长期–中期',
        'Google与Marvell扩大定制芯片合作，涉及TPU等AI芯片领域；OpenAI在Hot Chips 2026披露自研AI加速器ASIC Jalapeño的基准测试结果，该芯片为通用AI负载从零设计，非GPU改造；该节点归属 7 条链，本链语境未确定。对“AI芯片”环节而言，这意味着商业化进度上升、技术成熟度上升，因此本期判断为升温。'
      ),
      node(
        'chn-02-n02',
        2,
        '液冷服务器',
        '商业化进度 UP/MEDIUM',
        warming,
        direct,
        medium,
        '长期',
        '中国于2026年8月29日发布首个国家级液冷标准，以应对AI机架功耗逼近1MW的散热挑战，该标准为液冷服务器确立了统一的规范性技术基准。对“液冷服务器”环节而言，这意味着商业化进度上升，因此本期判断为升温。'
      ),
      node(
        'chn-02-n03',
        3,
        '液冷系统',
        '商业化进度 UP/MEDIUM',
        warming,
        direct,
        medium,
        '长期',
        '中国于2026年8月29日发布首个国家级液冷标准，以应对AI机架功耗逼近1MW的散热挑战，该标准为液冷系统确立了统一的规范性技术基准。对“液冷系统”环节而言，这意味着商业化进度上升，因此本期判断为升温。'
      ),
      node(
        'chn-02-n04',
        4,
        '数据中心',
        '真实同链拓扑相邻，尚无直接 Signal',
        pending,
        validation,
        low,
        '后续周期',
        '“数据中心”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。'
      ),
      node(
        'chn-02-n05',
        5,
        '服务器冷板',
        '市场需求 UP/LOW',
        warming,
        inferred,
        low060,
        '长期（传导滞后）',
        '液冷服务器商业化进度上升，意味着冷板等核心散热部件的配套量会随整机交付增加。服务器冷板是液冷服务器的真实组成环节，因此推测其市场需求上升。'
      )
    ],
    [
      edge('chn-02-edge-01', 1, 'chn-02-n05', 'chn-02-n02', '组成'),
      edge('chn-02-edge-02', 2, 'chn-02-n01', 'chn-02-n02', '组成'),
      edge('chn-02-edge-03', 3, 'chn-02-n02', 'chn-02-n04', '投入'),
      edge('chn-02-edge-04', 4, 'chn-02-n02', 'chn-02-n03', '依赖')
    ],
    '共享节点的本链语境仍待解析；上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
    '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。'
  ),
  chain(
    'chn-03',
    3,
    'AI算力基础设施服务产业链',
    'AI芯片、算力供给出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    warming,
    lowMedium,
    '长期–中期',
    [
      node(
        'chn-03-n01',
        1,
        'AI芯片',
        '商业化进度 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        warming,
        direct,
        lowMedium,
        '长期–中期',
        'Google与Marvell扩大定制芯片合作，涉及TPU等AI芯片领域；OpenAI在Hot Chips 2026披露自研AI加速器ASIC Jalapeño的基准测试结果，该芯片为通用AI负载从零设计，非GPU改造；该节点归属 7 条链，本链语境未确定。对“AI芯片”环节而言，这意味着商业化进度上升、技术成熟度上升，因此本期判断为升温。'
      ),
      node(
        'chn-03-n02',
        2,
        '算力供给',
        '产能利用率 UP/MEDIUM；商业化进度 UP/LOW；市场供给 UP/MEDIUM；性价比 UP/MEDIUM；有效产能 UP/HIGH',
        warming,
        direct,
        lowMedium,
        '中期',
        'InferenceXv3在推理过程中实现95%以上KVCache命中率；截至2026年7月底，全国智算总规模达245万PFLOPS(FP16)，其中145万PFLOPS纳入国家级监测调度平台；该节点归属 5 条链，本链语境未确定。对“算力供给”环节而言，这意味着产能利用率上升、商业化进度上升、市场供给上升、性价比上升、有效产能上升，因此本期判断为升温。'
      ),
      node(
        'chn-03-n03',
        3,
        'AI服务器',
        '真实同链拓扑相邻，尚无直接 Signal',
        pending,
        validation,
        low,
        '后续周期',
        '“AI服务器”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。'
      ),
      node(
        'chn-03-n04',
        4,
        '数据中心',
        '真实同链拓扑相邻，尚无直接 Signal',
        pending,
        validation,
        low,
        '后续周期',
        '“数据中心”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。'
      ),
      node(
        'chn-03-n05',
        5,
        '算力调度平台',
        '真实同链拓扑相邻，尚无直接 Signal',
        pending,
        validation,
        low,
        '后续周期',
        '“算力调度平台”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。'
      )
    ],
    [
      edge('chn-03-edge-01', 1, 'chn-03-n01', 'chn-03-n03', '组成'),
      edge('chn-03-edge-02', 2, 'chn-03-n02', 'chn-03-n04', '依赖'),
      edge('chn-03-edge-03', 3, 'chn-03-n02', 'chn-03-n05', '依赖')
    ],
    '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
    '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。'
  ),
  chain(
    'chn-21',
    4,
    '油品石化贸易服务产业链',
    '油品运输服务出现直接 Signal，当前链级聚合结果为分化，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    diverging,
    mediumHigh,
    '中期',
    [
      node(
        'chn-21-n01',
        1,
        '油品运输服务',
        '交付周期 UP/HIGH；利润率 UP/HIGH；战略资源安全 DOWN/MEDIUM；销售价格 UP/HIGH',
        diverging,
        direct,
        mediumHigh,
        '中期',
        '美伊冲突扰乱霍尔木兹海峡，导致油品运输服务的交付周期延长；美伊冲突扰乱霍尔木兹海峡，油运高景气延续，TD3C-TCE超60万美元/天。对“油品运输服务”环节而言，这意味着交付周期上升、利润率上升、战略资源安全下降、销售价格上升，因此本期判断为分化。'
      ),
      node(
        'chn-21-n02',
        2,
        '成品油批发交付服务',
        '投入成本 UP/LOW；交付周期 UP/LOW',
        cooling,
        inferred,
        low062,
        '短期–中期（传导滞后）',
        '油品运输的交付周期和销售价格上升，会把更高的物流成本和更长的到货时间传导给批发交付环节。成品油批发服务明确依赖运输服务，因此推测其投入成本和交付周期上升，经营景气受压。'
      )
    ],
    [edge('chn-21-edge-01', 1, 'chn-21-n02', 'chn-21-n01', '依赖')],
    '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
    '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。'
  )
];

const cards: ReportCard[] = [
  card(
    'card-geopolitics',
    'geopolitics',
    1,
    '地缘政治',
    '安全对抗与通道可用性',
    '伊朗—美以及海湾安全对抗持续升温，霍尔木兹航运安全与贸易通道可用性下降；南海海洋权益与安全争端风险小幅升温，但商业航运安全暂时稳定。',
    diverging,
    mediumHigh,
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
    '全球增长预期受到能源中断情景压制，同时美联储加息预期升温；增长降温与利率上行构成偏紧的宏观组合，但当前仅有两个真实 MacroEconomic 锚点，向中国价格、汇率和需求端的传导仍待补证。',
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
      industryChainCount: 54,
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
      confidence: mediumHigh,
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
      evidence('2026-08-31T02:30:06+00:00', '霍尔木兹海峡油轮交通量下降并遭袭击警告', [
        '霍尔木兹海峡',
        '油轮交通',
        '袭击警告'
      ]),
      evidence('2026-08-30T03:26:27+00:00', '中国海警在黄岩岛开展执法巡查', [
        '黄岩岛',
        '海警巡查',
        '执法行动'
      ]),
      evidence('2026-08-29T16:00:00+00:00', '美国政府准备使用捕获法处置扣押的伊朗石油和船只', [
        '捕获法',
        '伊朗石油',
        '扣押船只'
      ]),
      evidence('2026-08-29T16:00:00+00:00', '美伊在霍尔木兹海峡附近发生军事冲突', [
        '美伊冲突',
        '霍尔木兹海峡',
        '军事行动'
      ]),
      evidence('2026-08-29T08:11:41+00:00', '美伊冲突扰乱霍尔木兹海峡油运', [
        '美伊冲突',
        '海峡油运',
        '航运扰动'
      ])
    ]
  ],
  [
    'report_card:card-macroeconomics',
    [
      evidence('2026-08-29T13:46:38+00:00', '世界银行警告全球经济增长可能放缓至1.3%', [
        '世界银行',
        '全球增长',
        '增长预测'
      ]),
      evidence('2026-08-28T16:00:00+00:00', '美联储主席Warsh在杰克逊霍尔会议上暗示可能加息', [
        '美联储',
        '加息预期',
        '杰克逊霍尔'
      ])
    ]
  ],
  [
    'report_card:card-chn-01',
    [
      evidence('2026-08-28T22:40:51+00:00', '中国举办第二届人形机器人比赛', [
        '人形机器人',
        '机器人比赛',
        '具身智能'
      ]),
      evidence('2026-08-28T16:00:00+00:00', 'DIGITIMES发表关于具身智能实体化的评论', [
        '具身智能',
        '人形机器人',
        '产业化'
      ])
    ]
  ],
  [
    'report_card:card-chn-02',
    [
      evidence('2026-08-28T16:00:00+00:00', '中国发布首个国家级液冷标准', [
        '液冷标准',
        '数据中心',
        '液冷系统'
      ]),
      evidence(
        '2026-08-27T22:14:16+00:00',
        'OpenAI在Hot Chips 2026披露自研AI加速器Jalapeño基准测试结果',
        ['OpenAI', 'AI加速器', '基准测试']
      ),
      evidence('2026-08-27T16:00:00+00:00', 'Google与Marvell扩大定制芯片合作', [
        'Google',
        '定制芯片',
        '合作扩展'
      ])
    ]
  ],
  [
    'report_card:card-chn-03',
    [
      evidence('2026-08-31T02:11:00+00:00', '国家数据局披露全国智算总规模数据', [
        '智算规模',
        '算力供给',
        '数据中心'
      ]),
      evidence(
        '2026-08-27T22:14:16+00:00',
        'OpenAI在Hot Chips 2026披露自研AI加速器Jalapeño基准测试结果',
        ['OpenAI', 'AI加速器', '基准测试']
      ),
      evidence('2026-08-27T16:00:00+00:00', 'Google与Marvell扩大定制芯片合作', [
        'Google',
        '定制芯片',
        '合作扩展'
      ]),
      evidence('2026-08-23T16:00:00+00:00', 'InferenceXv3实现95%以上KVCache命中率', [
        '推理引擎',
        '缓存命中',
        '推理效率'
      ])
    ]
  ],
  [
    'report_card:card-chn-21',
    [
      evidence('2026-08-31T02:30:06+00:00', '霍尔木兹海峡油轮交通量下降并遭袭击警告', [
        '霍尔木兹海峡',
        '油轮交通',
        '袭击警告'
      ]),
      evidence('2026-08-29T08:11:41+00:00', '美伊冲突扰乱霍尔木兹海峡油运', [
        '美伊冲突',
        '海峡油运',
        '航运扰动'
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
