// Generated from the approved report prototype data. Do not edit manually.
// Run: npm run generate:report-mock
import type { ReportIndustryChainDetailContent } from '../../features/reports/contract';

export const generatedIndustryChainDetails: ReportIndustryChainDetailContent[] = [
  {
    key: 'chn-01',
    claimKey: 'chn-01-claim',
    displayOrder: 1,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-01'
    },
    name: '人形机器人产业链',
    conclusion:
      '人形机器人、人形机器人传感器、人形机器人减速器、人形机器人电机出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 1 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '中期–长期',
    pathSummary:
      '人形机器人、人形机器人传感器、人形机器人减速器、人形机器人电机（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: '人形机器人 → 机器人控制计算平台',
    nodes: [
      {
        key: 'chn-01-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-01-n01'
        },
        name: '人形机器人',
        impact: '商业化进度 UP/MEDIUM；渗透率 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国2026年8月在北京举办第二届人形机器人比赛，展示技术进步并引发应用讨论；DIGITIMES评论指出人形机器人行业扩产，具身智能正实体化，AI扩展下人类智能与机器活动界限模糊，反映具身智能技术正从验证进入规模应用。对“人形机器人”环节而言，这意味着商业化进度上升、渗透率上升，因此本期判断为升温。',
        timeWindow: '中期–长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-01-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-01-n02'
        },
        name: '人形机器人传感器',
        impact: '商业化进度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国2026年8月在北京举办第二届人形机器人比赛，展示技术进步并引发应用讨论。对“人形机器人传感器”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-01-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-01-n03'
        },
        name: '人形机器人减速器',
        impact: '商业化进度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国2026年8月在北京举办第二届人形机器人比赛，展示技术进步并引发应用讨论。对“人形机器人减速器”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-01-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-01-n04'
        },
        name: '人形机器人电机',
        impact: '商业化进度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国2026年8月在北京举办第二届人形机器人比赛，展示技术进步并引发应用讨论。对“人形机器人电机”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-01-n05',
        displayOrder: 5,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-01-n05'
        },
        name: '机器人控制计算平台',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '人形机器人的商业化和渗透率上升，会增加整机对运动控制、感知融合和实时决策计算的需求。控制计算平台是整机的明确组成环节，因此推测其市场需求将随整机放量而上升。',
        timeWindow: '中期–长期（传导滞后）',
        confidence: {
          label: '低（0.60）',
          score: 0.6
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-01-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-01-n04',
        toNodeKey: 'chn-01-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-01-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-01-n03',
        toNodeKey: 'chn-01-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-01-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-01-n05',
        toNodeKey: 'chn-01-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-01-edge-04',
        displayOrder: 4,
        fromNodeKey: 'chn-01-n02',
        toNodeKey: 'chn-01-n01',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-02',
    claimKey: 'chn-02-claim',
    displayOrder: 2,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-02'
    },
    name: 'AI数据中心液冷服务器产业链',
    conclusion:
      'AI芯片、液冷服务器、液冷系统出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证；本链新增 1 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 1 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: 'AI芯片、液冷服务器、液冷系统（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: '液冷服务器 → 服务器冷板',
    nodes: [
      {
        key: 'chn-02-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-02-n01'
        },
        name: 'AI芯片',
        impact: '商业化进度 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Google与Marvell扩大定制芯片合作，涉及TPU等AI芯片领域；OpenAI在Hot Chips 2026披露自研AI加速器ASIC Jalapeño的基准测试结果，该芯片为通用AI负载从零设计，非GPU改造；该节点归属 7 条链，本链语境未确定。对“AI芯片”环节而言，这意味着商业化进度上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-02-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-02-n02'
        },
        name: '液冷服务器',
        impact: '商业化进度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国于2026年8月29日发布首个国家级液冷标准，以应对AI机架功耗逼近1MW的散热挑战，该标准为液冷服务器确立了统一的规范性技术基准。对“液冷服务器”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-02-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-02-n03'
        },
        name: '液冷系统',
        impact: '商业化进度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国于2026年8月29日发布首个国家级液冷标准，以应对AI机架功耗逼近1MW的散热挑战，该标准为液冷系统确立了统一的规范性技术基准。对“液冷系统”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-02-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-02-n04'
        },
        name: '数据中心',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“数据中心”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-02-n05',
        displayOrder: 5,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-02-n05'
        },
        name: '服务器冷板',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '液冷服务器商业化进度上升，意味着冷板等核心散热部件的配套量会随整机交付增加。服务器冷板是液冷服务器的真实组成环节，因此推测其市场需求上升。',
        timeWindow: '长期（传导滞后）',
        confidence: {
          label: '低（0.60）',
          score: 0.6
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-02-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-02-n05',
        toNodeKey: 'chn-02-n02',
        relationLabel: '组成'
      },
      {
        key: 'chn-02-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-02-n01',
        toNodeKey: 'chn-02-n02',
        relationLabel: '组成'
      },
      {
        key: 'chn-02-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-02-n02',
        toNodeKey: 'chn-02-n04',
        relationLabel: '投入'
      },
      {
        key: 'chn-02-edge-04',
        displayOrder: 4,
        fromNodeKey: 'chn-02-n02',
        toNodeKey: 'chn-02-n03',
        relationLabel: '依赖'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的本链语境仍待解析；上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-03',
    claimKey: 'chn-03-claim',
    displayOrder: 3,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-03'
    },
    name: 'AI算力基础设施服务产业链',
    conclusion:
      'AI芯片、算力供给出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: 'AI芯片、算力供给（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-03-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-03-n01'
        },
        name: 'AI芯片',
        impact: '商业化进度 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Google与Marvell扩大定制芯片合作，涉及TPU等AI芯片领域；OpenAI在Hot Chips 2026披露自研AI加速器ASIC Jalapeño的基准测试结果，该芯片为通用AI负载从零设计，非GPU改造；该节点归属 7 条链，本链语境未确定。对“AI芯片”环节而言，这意味着商业化进度上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-03-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-03-n02'
        },
        name: '算力供给',
        impact:
          '产能利用率 UP/MEDIUM；商业化进度 UP/LOW；市场供给 UP/MEDIUM；性价比 UP/MEDIUM；有效产能 UP/HIGH',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'InferenceXv3在推理过程中实现95%以上KVCache命中率；截至2026年7月底，全国智算总规模达245万PFLOPS(FP16)，其中145万PFLOPS纳入国家级监测调度平台；该节点归属 5 条链，本链语境未确定。对“算力供给”环节而言，这意味着产能利用率上升、商业化进度上升、市场供给上升、性价比上升、有效产能上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-03-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-03-n03'
        },
        name: 'AI服务器',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“AI服务器”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-03-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-03-n04'
        },
        name: '数据中心',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“数据中心”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-03-n05',
        displayOrder: 5,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-03-n05'
        },
        name: '算力调度平台',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“算力调度平台”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-03-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-03-n01',
        toNodeKey: 'chn-03-n03',
        relationLabel: '组成'
      },
      {
        key: 'chn-03-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-03-n02',
        toNodeKey: 'chn-03-n04',
        relationLabel: '依赖'
      },
      {
        key: 'chn-03-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-03-n02',
        toNodeKey: 'chn-03-n05',
        relationLabel: '依赖'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-04',
    claimKey: 'chn-04-claim',
    displayOrder: 4,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-04'
    },
    name: 'AI视频生成服务产业链',
    conclusion:
      'AI语料、算力供给出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: 'AI语料、算力供给（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-04-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-04-n01'
        },
        name: 'AI语料',
        impact: '商业化进度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'AgentX于2026年8月24日发布开源数据集，价值300万美元，支持长上下文、多轮和子代理特性；该节点归属 2 条链，本链语境未确定。对“AI语料”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-04-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-04-n02'
        },
        name: '算力供给',
        impact:
          '产能利用率 UP/MEDIUM；商业化进度 UP/LOW；市场供给 UP/MEDIUM；性价比 UP/MEDIUM；有效产能 UP/HIGH',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'InferenceXv3在推理过程中实现95%以上KVCache命中率；截至2026年7月底，全国智算总规模达245万PFLOPS(FP16)，其中145万PFLOPS纳入国家级监测调度平台；该节点归属 5 条链，本链语境未确定。对“算力供给”环节而言，这意味着产能利用率上升、商业化进度上升、市场供给上升、性价比上升、有效产能上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-04-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-04-n03'
        },
        name: '视频生成基础模型',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“视频生成基础模型”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-04-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-04-n03',
        toNodeKey: 'chn-04-n02',
        relationLabel: '依赖'
      },
      {
        key: 'chn-04-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-04-n01',
        toNodeKey: 'chn-04-n03',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-05',
    claimKey: 'chn-05-claim',
    displayOrder: 5,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-05'
    },
    name: '油气勘探开发产业链',
    conclusion:
      '商品原油、油气开采出现直接 Signal，当前链级聚合结果为分化，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 1 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'diverging',
      label: '分化'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '中期–短期',
    pathSummary: '商品原油、油气开采（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: '油气开采 → 油气集输与初步处理服务',
    nodes: [
      {
        key: 'chn-05-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-05-n01'
        },
        name: '商品原油',
        impact:
          '市场供给 STABLE/LOW；市场供给 DOWN/LOW；市场供给 UP/MEDIUM；库存水平 DOWN/MEDIUM；投入成本 UP/MEDIUM',
        result: {
          code: 'diverging',
          label: '分化'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '亚洲炼油商转向南美生产商可能导致供应来源增加，但商品原油的市场供给整体未变；委内瑞拉2026年7月原油出口总量环比下降至116万桶/日，对美出口升至高位。对“商品原油”环节而言，这意味着市场供给保持稳定、市场供给下降、市场供给上升、库存水平下降、投入成本上升，因此本期判断为分化。',
        timeWindow: '中期–短期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-05-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-05-n02'
        },
        name: '油气开采',
        impact: '市场供给 DOWN/LOW',
        result: {
          code: 'cooling',
          label: '降温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '委内瑞拉2026年7月原油出口总量环比下降至116万桶/日。对“油气开采”环节而言，这意味着市场供给下降，因此本期判断为降温。',
        timeWindow: '短期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-05-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-05-n03'
        },
        name: '具备产能的油气生产井',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“具备产能的油气生产井”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-05-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-05-n04'
        },
        name: '油气集输与初步处理服务',
        impact: '产能利用率 DOWN/LOW',
        result: {
          code: 'cooling',
          label: '降温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '油气开采供给下降会减少进入集输和初处理环节的原料量。该服务是开采产出后的明确下游环节，因此推测其产能利用率可能下降。',
        timeWindow: '短期–中期（传导滞后）',
        confidence: {
          label: '低（0.54）',
          score: 0.54
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-05-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-05-n04',
        toNodeKey: 'chn-05-n01',
        relationLabel: '投入'
      },
      {
        key: 'chn-05-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-05-n02',
        toNodeKey: 'chn-05-n03',
        relationLabel: '依赖'
      },
      {
        key: 'chn-05-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-05-n02',
        toNodeKey: 'chn-05-n04',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-06',
    claimKey: 'chn-06-claim',
    displayOrder: 6,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-06'
    },
    name: '生成式人工智能模型及应用服务产业链',
    conclusion:
      '生成式AI基础模型、算力供给出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证；本链新增 2 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 2 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: '生成式AI基础模型、算力供给（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary:
      '生成式AI基础模型 → 模型训练服务；生成式AI基础模型 → 生成式AI应用服务',
    nodes: [
      {
        key: 'chn-06-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-06-n01'
        },
        name: '生成式AI基础模型',
        impact: '商业化进度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'DeepMind发布Gemini Omni 1.1 Flash，宣称让开发者构建时拥有更多控制力。对“生成式AI基础模型”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-06-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-06-n02'
        },
        name: '算力供给',
        impact:
          '产能利用率 UP/MEDIUM；商业化进度 UP/LOW；市场供给 UP/MEDIUM；性价比 UP/MEDIUM；有效产能 UP/HIGH',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'InferenceXv3在推理过程中实现95%以上KVCache命中率；截至2026年7月底，全国智算总规模达245万PFLOPS(FP16)，其中145万PFLOPS纳入国家级监测调度平台；该节点归属 5 条链，本链语境未确定。对“算力供给”环节而言，这意味着产能利用率上升、商业化进度上升、市场供给上升、性价比上升、有效产能上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-06-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-06-n03'
        },
        name: '模型训练服务',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '生成式AI基础模型商业化加快，会带来持续训练、微调和迭代需求。模型训练服务是基础模型的明确投入环节，因此推测其市场需求上升。',
        timeWindow: '中期（传导滞后）',
        confidence: {
          label: '低（0.60）',
          score: 0.6
        },
        hasEvidence: false
      },
      {
        key: 'chn-06-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-06-n04'
        },
        name: '生成式AI应用服务',
        impact: '商业化进度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '基础模型商业化加快会提高可用模型供给，并降低下游应用的开发与部署门槛。基础模型是生成式AI应用服务的明确投入，因此推测应用服务的商业化进度上升。',
        timeWindow: '中期（传导滞后）',
        confidence: {
          label: '低（0.48）',
          score: 0.48
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-06-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-06-n03',
        toNodeKey: 'chn-06-n02',
        relationLabel: '依赖'
      },
      {
        key: 'chn-06-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-06-n03',
        toNodeKey: 'chn-06-n01',
        relationLabel: '投入'
      },
      {
        key: 'chn-06-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-06-n01',
        toNodeKey: 'chn-06-n04',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的本链语境仍待解析；上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-07',
    claimKey: 'chn-07-claim',
    displayOrder: 7,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-07'
    },
    name: '汽车线控制动执行器产业链',
    conclusion:
      '传感器、新能源车整车出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期–长期',
    pathSummary: '传感器、新能源车整车（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-07-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-07-n01'
        },
        name: '传感器',
        impact: '政策支持力度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '2026年8月29日，北京怀柔发布2026年第一批采购机遇清单，采购额约4亿元，包含超过10项采购机遇，针对高端科学仪器与传感器产业；该节点归属 7 条链，本链语境未确定。对“传感器”环节而言，这意味着政策支持力度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-07-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-07-n02'
        },
        name: '新能源车整车',
        impact: '商业化进度 UP/MEDIUM；市场需求 UP/MEDIUM；性价比 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国国家市场监管总局宣布分阶段实施GB1589-2026标准，新车型需于2027年7月1日起符合，现有车型需于2028年1月1日起符合，并给予车企技术升级和过渡时间；吉利汽车发布混动SUV车型Monjaro EM-i，开启首款混动D级SUV全球推广，直接增加新能源车（混动SUV）的供给与可得性；该节点归属 5 条链，本链语境未确定。对“新能源车整车”环节而言，这意味着商业化进度上升、市场需求上升、性价比上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-07-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-07-n03'
        },
        name: '线控制动执行器',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“线控制动执行器”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-07-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-07-n01',
        toNodeKey: 'chn-07-n03',
        relationLabel: '组成'
      },
      {
        key: 'chn-07-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-07-n03',
        toNodeKey: 'chn-07-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-08',
    claimKey: 'chn-08-claim',
    displayOrder: 8,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-08'
    },
    name: '企业AI智能体产业链',
    conclusion:
      'AI智能体、基础模型API服务出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 2 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 2 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: 'AI智能体、基础模型API服务（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: 'AI智能体 → 企业应用软件；AI智能体 → 企业知识库',
    nodes: [
      {
        key: 'chn-08-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-08-n01'
        },
        name: 'AI智能体',
        impact: '商业化进度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'AgentX于2026年8月24日发布开源数据集，价值300万美元，支持长上下文、多轮和子代理特性。对“AI智能体”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-08-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-08-n02'
        },
        name: '基础模型API服务',
        impact: '性价比 UP/HIGH；渗透率 UP/MEDIUM；渗透率 UP/HIGH',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '开源模型竞争力增强，Fireworks每日处理超40万亿token，是OpenAI API在3月底交易量的2倍；DeepMind发布Gemini Omni 1.1 Flash，宣称让开发者构建时拥有更多控制力。对“基础模型API服务”环节而言，这意味着性价比上升、渗透率上升、渗透率上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-08-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-08-n03'
        },
        name: '企业应用软件',
        impact: '商业化进度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          'AI智能体商业化进度上升，会使企业应用软件更快集成自动执行、知识检索和流程协同能力。企业应用软件明确依赖AI智能体，因此推测其商业化进度上升。',
        timeWindow: '中期（传导滞后）',
        confidence: {
          label: '低（0.42）',
          score: 0.42
        },
        hasEvidence: false
      },
      {
        key: 'chn-08-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-08-n04'
        },
        name: '企业知识库',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          'AI智能体进入商业应用后，需要可检索、可更新且权限可控的企业知识。AI智能体明确依赖企业知识库，因此推测企业知识库的市场需求上升。',
        timeWindow: '中期（传导滞后）',
        confidence: {
          label: '低（0.45）',
          score: 0.45
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-08-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-08-n03',
        toNodeKey: 'chn-08-n01',
        relationLabel: '依赖'
      },
      {
        key: 'chn-08-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-08-n01',
        toNodeKey: 'chn-08-n02',
        relationLabel: '依赖'
      },
      {
        key: 'chn-08-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-08-n01',
        toNodeKey: 'chn-08-n04',
        relationLabel: '依赖'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-09',
    claimKey: 'chn-09-claim',
    displayOrder: 9,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-09'
    },
    name: '智能语音技术服务产业链',
    conclusion:
      '智能语音基础模型、智能语音算法SDK与云API出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 2 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 2 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: '智能语音基础模型、智能语音算法SDK与云API（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary:
      '智能语音算法SDK与云API → 智能终端语音交互应用；智能语音基础模型 → 智能语音基础模型训练服务',
    nodes: [
      {
        key: 'chn-09-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-09-n01'
        },
        name: '智能语音基础模型',
        impact: '商业化进度 UP/MEDIUM；技术成熟度 UP/MEDIUM；渗透率 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'DeepMind于2026年8月26日发布Gemini 3.5 Transcribe，一款提供更智能语音转文字转录功能的产品，标志公司语音转录技术进入商业化发布阶段；DeepMind于2026年8月26日发布Gemini 3.5 Transcribe，其更智能的语音转文字功能有望提升语音转录产品的技术性能和体验。对“智能语音基础模型”环节而言，这意味着商业化进度上升、技术成熟度上升、渗透率上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-09-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-09-n02'
        },
        name: '智能语音算法SDK与云API',
        impact: '商业化进度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'DeepMind于2026年8月26日发布Gemini 3.5 Transcribe，提供更智能的语音转文字转录功能，直接提升了DeepMind在语音转录应用中的产品供给。对“智能语音算法SDK与云API”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-09-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-09-n03'
        },
        name: '智能终端语音交互应用',
        impact: '商业化进度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '智能语音SDK与云API商业化加快，会降低终端应用集成语音能力的成本和时间。该能力是智能终端语音交互应用的明确投入，因此推测应用端商业化进度上升。',
        timeWindow: '中期（传导滞后）',
        confidence: {
          label: '低（0.48）',
          score: 0.48
        },
        hasEvidence: false
      },
      {
        key: 'chn-09-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-09-n04'
        },
        name: '智能语音基础模型训练服务',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '智能语音基础模型的技术成熟度和商业化进度上升，通常需要持续训练、评测和场景微调。训练服务是基础模型的明确投入环节，因此推测其市场需求上升。',
        timeWindow: '中期（传导滞后）',
        confidence: {
          label: '低（0.57）',
          score: 0.57
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-09-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-09-n02',
        toNodeKey: 'chn-09-n03',
        relationLabel: '投入'
      },
      {
        key: 'chn-09-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-09-n01',
        toNodeKey: 'chn-09-n02',
        relationLabel: '投入'
      },
      {
        key: 'chn-09-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-09-n04',
        toNodeKey: 'chn-09-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-10',
    claimKey: 'chn-10-claim',
    displayOrder: 10,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-10'
    },
    name: 'AI智能手机产业链',
    conclusion:
      'AI芯片、传感器出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: 'AI芯片、传感器（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-10-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-10-n01'
        },
        name: 'AI芯片',
        impact: '商业化进度 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Google与Marvell扩大定制芯片合作，涉及TPU等AI芯片领域；OpenAI在Hot Chips 2026披露自研AI加速器ASIC Jalapeño的基准测试结果，该芯片为通用AI负载从零设计，非GPU改造；该节点归属 7 条链，本链语境未确定。对“AI芯片”环节而言，这意味着商业化进度上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-10-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-10-n02'
        },
        name: '传感器',
        impact: '政策支持力度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '2026年8月29日，北京怀柔发布2026年第一批采购机遇清单，采购额约4亿元，包含超过10项采购机遇，针对高端科学仪器与传感器产业；该节点归属 7 条链，本链语境未确定。对“传感器”环节而言，这意味着政策支持力度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-10-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-10-n03'
        },
        name: 'AI手机',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“AI手机”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-10-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-10-n01',
        toNodeKey: 'chn-10-n03',
        relationLabel: '组成'
      },
      {
        key: 'chn-10-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-10-n02',
        toNodeKey: 'chn-10-n03',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-11',
    claimKey: 'chn-11-claim',
    displayOrder: 11,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-11'
    },
    name: 'AI计算芯片产业链',
    conclusion:
      'AI芯片、集成电路制造出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: 'AI芯片、集成电路制造（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-11-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-11-n01'
        },
        name: 'AI芯片',
        impact: '商业化进度 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Google与Marvell扩大定制芯片合作，涉及TPU等AI芯片领域；OpenAI在Hot Chips 2026披露自研AI加速器ASIC Jalapeño的基准测试结果，该芯片为通用AI负载从零设计，非GPU改造；该节点归属 7 条链，本链语境未确定。对“AI芯片”环节而言，这意味着商业化进度上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-11-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-11-n02'
        },
        name: '集成电路制造',
        impact: '产能利用率 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '美国商务部推进CHIPS法案制造业激励项目，多数项目计划2028年前完成；该节点归属 4 条链，本链语境未确定。对“集成电路制造”环节而言，这意味着产能利用率上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-11-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-11-n03'
        },
        name: 'AI加速卡',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“AI加速卡”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-11-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-11-n04'
        },
        name: 'AI服务器',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“AI服务器”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-11-n05',
        displayOrder: 5,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-11-n05'
        },
        name: 'EDA',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“EDA”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-11-n06',
        displayOrder: 6,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-11-n06'
        },
        name: '先进封装',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“先进封装”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-11-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-11-n02',
        toNodeKey: 'chn-11-n06',
        relationLabel: '投入'
      },
      {
        key: 'chn-11-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-11-n01',
        toNodeKey: 'chn-11-n04',
        relationLabel: '组成'
      },
      {
        key: 'chn-11-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-11-n01',
        toNodeKey: 'chn-11-n06',
        relationLabel: '依赖'
      },
      {
        key: 'chn-11-edge-04',
        displayOrder: 4,
        fromNodeKey: 'chn-11-n01',
        toNodeKey: 'chn-11-n05',
        relationLabel: '依赖'
      },
      {
        key: 'chn-11-edge-05',
        displayOrder: 5,
        fromNodeKey: 'chn-11-n01',
        toNodeKey: 'chn-11-n03',
        relationLabel: '组成'
      },
      {
        key: 'chn-11-edge-06',
        displayOrder: 6,
        fromNodeKey: 'chn-11-n01',
        toNodeKey: 'chn-11-n02',
        relationLabel: '依赖'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-12',
    claimKey: 'chn-12-claim',
    displayOrder: 12,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-12'
    },
    name: '电动汽车换电服务产业链',
    conclusion:
      '换电运营调度平台、标准化换电电池包资产出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 1 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: '换电运营调度平台、标准化换电电池包资产（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: '标准化换电电池包资产 + 换电运营调度平台 → 换电服务',
    nodes: [
      {
        key: 'chn-12-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-12-n01'
        },
        name: '换电运营调度平台',
        impact: '商业化进度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Epicland考虑采用换电模式，意味着对换电运营调度平台的需求可能提升，从而推动其商业化进度。对“换电运营调度平台”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-12-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-12-n02'
        },
        name: '标准化换电电池包资产',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Epicland考虑采用换电模式，将增加对标准化换电电池包资产的市场需求。对“标准化换电电池包资产”环节而言，这意味着市场需求上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-12-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-12-n03'
        },
        name: '换电服务',
        impact: '商业化进度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '标准化换电电池包需求上升，同时换电运营调度平台商业化提速，表明资产与运营系统都在准备扩张。换电服务同时依赖这两个环节，因此推测其商业化进度上升。',
        timeWindow: '中期（传导滞后）',
        confidence: {
          label: '低（0.45）',
          score: 0.45
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-12-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-12-n03',
        toNodeKey: 'chn-12-n02',
        relationLabel: '依赖'
      },
      {
        key: 'chn-12-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-12-n03',
        toNodeKey: 'chn-12-n01',
        relationLabel: '依赖'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-13',
    claimKey: 'chn-13-claim',
    displayOrder: 13,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-13'
    },
    name: '量子信息技术系统产业链',
    conclusion:
      '超导量子计算机、量子计算控制软件出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 3 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 3 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '超导量子计算机、量子计算控制软件（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary:
      '超导量子计算机 → 超导量子芯片；超导量子计算机 → 稀释制冷机；超导量子计算机 → 量子测控电子系统',
    nodes: [
      {
        key: 'chn-13-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-13-n01'
        },
        name: '超导量子计算机',
        impact: '政策支持力度 UP/HIGH',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '美国商务部取消CHIPS法案下78亿美元研发拨款，并转向量子计算投资，向xLight等公司授予9亿美元、承诺20亿美元量子计算初步拨款，要求以股权换取资金。对“超导量子计算机”环节而言，这意味着政策支持力度上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-13-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-13-n02'
        },
        name: '量子计算控制软件',
        impact: '技术路线竞争 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '美国商务部取消CHIPS法案研发资金并转向量子计算投资，涉及NSTC框架调整。对“量子计算控制软件”环节而言，这意味着技术路线竞争上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-13-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-13-n03'
        },
        name: '稀释制冷机',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '超导量子计算机获得更强政策支持后，整机项目投入可能带动关键配套设备。稀释制冷机是超导量子计算机的明确组成环节，因此推测其采购需求上升。',
        timeWindow: '长期（传导滞后）',
        confidence: {
          label: '低（0.44）',
          score: 0.44
        },
        hasEvidence: false
      },
      {
        key: 'chn-13-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-13-n04'
        },
        name: '超导量子芯片',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '超导量子计算机获得更强政策支持后，研发和项目投入通常会向核心部件传导。超导量子芯片是整机的明确组成环节，因此推测其研发与采购需求上升。',
        timeWindow: '长期（传导滞后）',
        confidence: {
          label: '低（0.44）',
          score: 0.44
        },
        hasEvidence: false
      },
      {
        key: 'chn-13-n05',
        displayOrder: 5,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-13-n05'
        },
        name: '量子测控电子系统',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '超导量子计算机获得更强政策支持后，整机研发和部署将增加对测控环节的需求。量子测控电子系统是整机的明确组成环节，因此推测其市场需求上升。',
        timeWindow: '长期（传导滞后）',
        confidence: {
          label: '低（0.44）',
          score: 0.44
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-13-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-13-n04',
        toNodeKey: 'chn-13-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-13-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-13-n03',
        toNodeKey: 'chn-13-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-13-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-13-n05',
        toNodeKey: 'chn-13-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-13-edge-04',
        displayOrder: 4,
        fromNodeKey: 'chn-13-n02',
        toNodeKey: 'chn-13-n01',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-14',
    claimKey: 'chn-14-claim',
    displayOrder: 14,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-14'
    },
    name: '自动驾驶系统产业链',
    conclusion:
      '自动驾驶系统出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 2 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 2 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中–高',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: '自动驾驶系统（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: '自动驾驶系统 → 激光雷达；自动驾驶系统 → 车载摄像头模组',
    nodes: [
      {
        key: 'chn-14-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-14-n01'
        },
        name: '自动驾驶系统',
        impact:
          '商业化进度 UP/MEDIUM；市场份额 UP/MEDIUM；技术成熟度 UP/MEDIUM；订单可见度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Momenta披露2026年上半年研发支出11.6亿元，同比增长18.6%，占营收72.6%，研发员工1102名占员工总数79.1%；Momenta在2026年上半年新增安装量产方案约32.1万套，同比增长83.7%，累计安装超100万套；交付37个量产车型，累计105个。对“自动驾驶系统”环节而言，这意味着商业化进度上升、市场份额上升、技术成熟度上升、订单可见度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '中–高',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-14-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-14-n02'
        },
        name: '乘用车整车',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“乘用车整车”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-14-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-14-n03'
        },
        name: '激光雷达',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '自动驾驶系统的商业化、订单可见度和技术成熟度同时上升，会增加对环境感知部件的配套需求。激光雷达是系统的真实组成环节，因此推测其市场需求上升。',
        timeWindow: '中期–长期（传导滞后）',
        confidence: {
          label: '中（0.69）',
          score: 0.69
        },
        hasEvidence: false
      },
      {
        key: 'chn-14-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-14-n04'
        },
        name: '车载摄像头模组',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '自动驾驶系统的商业化、订单可见度和技术成熟度同时上升，会增加多视角环境感知需求。车载摄像头模组是系统的真实组成环节，因此推测其市场需求上升。',
        timeWindow: '中期–长期（传导滞后）',
        confidence: {
          label: '中（0.69）',
          score: 0.69
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-14-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-14-n03',
        toNodeKey: 'chn-14-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-14-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-14-n04',
        toNodeKey: 'chn-14-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-14-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-14-n01',
        toNodeKey: 'chn-14-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-15',
    claimKey: 'chn-15-claim',
    displayOrder: 15,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-15'
    },
    name: 'AI药物发现服务产业链',
    conclusion:
      '算力供给出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: '算力供给（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-15-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-15-n01'
        },
        name: '算力供给',
        impact:
          '产能利用率 UP/MEDIUM；商业化进度 UP/LOW；市场供给 UP/MEDIUM；性价比 UP/MEDIUM；有效产能 UP/HIGH',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'InferenceXv3在推理过程中实现95%以上KVCache命中率；截至2026年7月底，全国智算总规模达245万PFLOPS(FP16)，其中145万PFLOPS纳入国家级监测调度平台；该节点归属 5 条链，本链语境未确定。对“算力供给”环节而言，这意味着产能利用率上升、商业化进度上升、市场供给上升、性价比上升、有效产能上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-15-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-15-n02'
        },
        name: 'AI药物发现模型平台',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“AI药物发现模型平台”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-15-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-15-n02',
        toNodeKey: 'chn-15-n01',
        relationLabel: '依赖'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-16',
    claimKey: 'chn-16-claim',
    displayOrder: 16,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-16'
    },
    name: 'MLOps平台与服务产业链',
    conclusion:
      '算力供给出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: '算力供给（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-16-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-16-n01'
        },
        name: '算力供给',
        impact:
          '产能利用率 UP/MEDIUM；商业化进度 UP/LOW；市场供给 UP/MEDIUM；性价比 UP/MEDIUM；有效产能 UP/HIGH',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'InferenceXv3在推理过程中实现95%以上KVCache命中率；截至2026年7月底，全国智算总规模达245万PFLOPS(FP16)，其中145万PFLOPS纳入国家级监测调度平台；该节点归属 5 条链，本链语境未确定。对“算力供给”环节而言，这意味着产能利用率上升、商业化进度上升、市场供给上升、性价比上升、有效产能上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-16-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-16-n02'
        },
        name: 'MLOps平台与服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“MLOps平台与服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-16-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-16-n02',
        toNodeKey: 'chn-16-n01',
        relationLabel: '依赖'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-17',
    claimKey: 'chn-17-claim',
    displayOrder: 17,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-17'
    },
    name: '新能源汽车产业链',
    conclusion:
      '纯电动汽车整车出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 5 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 5 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中–高',
      score: null
    },
    timeWindow: '中期–短期',
    pathSummary: '纯电动汽车整车（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary:
      '纯电动汽车整车 → 纯电动汽车动力电池包；纯电动汽车整车 → 纯电动汽车底盘车身系统；纯电动汽车整车 → 纯电动汽车热管理系统；纯电动汽车整车 → 新能源汽车公告认证服务；纯电动汽车整车 → 纯电动汽车电驱总成',
    nodes: [
      {
        key: 'chn-17-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-17-n01'
        },
        name: '纯电动汽车整车',
        impact: '交付周期 UP/HIGH；扩产强度 UP/HIGH；渗透率 UP/MEDIUM；订单可见度 UP/HIGH',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '小鹏GX上市12小时内获24863份订单，Ultra版交付等待时间达35周，小鹏因此增加模具和产线；小鹏为应对GX订单激增，增加关键零部件模具和产线。对“纯电动汽车整车”环节而言，这意味着交付周期上升、扩产强度上升、渗透率上升、订单可见度上升，因此本期判断为升温。',
        timeWindow: '中期–短期',
        confidence: {
          label: '中–高',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-17-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-17-n02'
        },
        name: '新能源汽车公告认证服务',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '纯电动汽车整车的订单和扩产走强，通常伴随新车型、配置和生产变更的准入需求。整车明确依赖公告认证服务，因此推测该服务需求上升，但需后续车型申报数验证。',
        timeWindow: '中期（传导假设）',
        confidence: {
          label: '低（0.48）',
          score: 0.48
        },
        hasEvidence: false
      },
      {
        key: 'chn-17-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-17-n03'
        },
        name: '纯电动汽车动力电池包',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '纯电动汽车整车的订单可见度、扩产强度和渗透率上升，会按整车产量拉动动力电池包配套。动力电池包是整车的明确组成环节，因此推测其市场需求上升。',
        timeWindow: '中期（传导滞后）',
        confidence: {
          label: '中（0.69）',
          score: 0.69
        },
        hasEvidence: false
      },
      {
        key: 'chn-17-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-17-n04'
        },
        name: '纯电动汽车底盘车身系统',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '纯电动汽车整车的订单可见度和扩产强度上升，会按整车产量增加底盘车身系统的配套量。该系统是整车的明确组成环节，因此推测其市场需求上升。',
        timeWindow: '中期（传导滞后）',
        confidence: {
          label: '中（0.69）',
          score: 0.69
        },
        hasEvidence: false
      },
      {
        key: 'chn-17-n05',
        displayOrder: 5,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-17-n05'
        },
        name: '纯电动汽车热管理系统',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '纯电动汽车整车订单和扩产走强，会增加电池、电驱和座舱热管理的配套量。热管理系统是整车的明确组成环节，因此推测其市场需求上升。',
        timeWindow: '中期（传导滞后）',
        confidence: {
          label: '中（0.69）',
          score: 0.69
        },
        hasEvidence: false
      },
      {
        key: 'chn-17-n06',
        displayOrder: 6,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-17-n06'
        },
        name: '纯电动汽车电驱总成',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '纯电动汽车整车订单和扩产走强，会按整车产量增加电驱总成的配套量。电驱总成是整车的明确组成环节，因此推测其市场需求上升。',
        timeWindow: '中期（传导滞后）',
        confidence: {
          label: '中（0.69）',
          score: 0.69
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-17-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-17-n03',
        toNodeKey: 'chn-17-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-17-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-17-n04',
        toNodeKey: 'chn-17-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-17-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-17-n05',
        toNodeKey: 'chn-17-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-17-edge-04',
        displayOrder: 4,
        fromNodeKey: 'chn-17-n01',
        toNodeKey: 'chn-17-n02',
        relationLabel: '依赖'
      },
      {
        key: 'chn-17-edge-05',
        displayOrder: 5,
        fromNodeKey: 'chn-17-n06',
        toNodeKey: 'chn-17-n01',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-18',
    claimKey: 'chn-18-claim',
    displayOrder: 18,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-18'
    },
    name: '普通钢材产业链',
    conclusion:
      '钢铁长材出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 1 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中–高',
      score: null
    },
    timeWindow: '短期–长期',
    pathSummary: '钢铁长材（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: '钢铁长材 → 普钢长材轧制工序',
    nodes: [
      {
        key: 'chn-18-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-18-n01'
        },
        name: '钢铁长材',
        impact:
          '定价权 UP/MEDIUM；成本传导能力 UP/MEDIUM；进口依赖度 UP/MEDIUM；销售价格 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国钢坯出口商将钢坯出口报价上调至465-470美元/吨FOB；欧盟调整钢铁配额及CBAM政策，增加钢铁出口至欧盟的成本，影响相关钢铁产业链节点的进口依赖度和供应格局。对“钢铁长材”环节而言，这意味着定价权上升、成本传导能力上升、进口依赖度上升、销售价格上升，因此本期判断为升温。',
        timeWindow: '短期–长期',
        confidence: {
          label: '中–高',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-18-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-18-n02'
        },
        name: '普钢长材轧制工序',
        impact: '产能利用率 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '钢铁长材的销售价格、定价权和成本传导能力上升，可能支撑国内长材生产和轧制活动。长材轧制工序是钢铁长材的明确投入环节，因此推测其产能利用率上升。',
        timeWindow: '短期–中期（传导假设）',
        confidence: {
          label: '低（0.48）',
          score: 0.48
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-18-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-18-n02',
        toNodeKey: 'chn-18-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-19',
    claimKey: 'chn-19-claim',
    displayOrder: 19,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-19'
    },
    name: '汽车一体化压铸产业链',
    conclusion:
      '新能源车整车出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: '新能源车整车（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-19-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-19-n01'
        },
        name: '新能源车整车',
        impact: '商业化进度 UP/MEDIUM；市场需求 UP/MEDIUM；性价比 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国国家市场监管总局宣布分阶段实施GB1589-2026标准，新车型需于2027年7月1日起符合，现有车型需于2028年1月1日起符合，并给予车企技术升级和过渡时间；吉利汽车发布混动SUV车型Monjaro EM-i，开启首款混动D级SUV全球推广，直接增加新能源车（混动SUV）的供给与可得性；该节点归属 5 条链，本链语境未确定。对“新能源车整车”环节而言，这意味着商业化进度上升、市场需求上升、性价比上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-19-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-19-n02'
        },
        name: '一体化压铸车身结构件',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“一体化压铸车身结构件”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-19-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-19-n02',
        toNodeKey: 'chn-19-n01',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-20',
    claimKey: 'chn-20-claim',
    displayOrder: 20,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-20'
    },
    name: '汽车热管理系统产业链',
    conclusion:
      '新能源车整车出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: '新能源车整车（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-20-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-20-n01'
        },
        name: '新能源车整车',
        impact: '商业化进度 UP/MEDIUM；市场需求 UP/MEDIUM；性价比 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国国家市场监管总局宣布分阶段实施GB1589-2026标准，新车型需于2027年7月1日起符合，现有车型需于2028年1月1日起符合，并给予车企技术升级和过渡时间；吉利汽车发布混动SUV车型Monjaro EM-i，开启首款混动D级SUV全球推广，直接增加新能源车（混动SUV）的供给与可得性；该节点归属 5 条链，本链语境未确定。对“新能源车整车”环节而言，这意味着商业化进度上升、市场需求上升、性价比上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-20-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-20-n02'
        },
        name: '汽车热管理系统',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“汽车热管理系统”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-20-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-20-n02',
        toNodeKey: 'chn-20-n01',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-21',
    claimKey: 'chn-21-claim',
    displayOrder: 21,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-21'
    },
    name: '油品石化贸易服务产业链',
    conclusion:
      '油品运输服务出现直接 Signal，当前链级聚合结果为分化，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 1 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'diverging',
      label: '分化'
    },
    confidence: {
      label: '中–高',
      score: null
    },
    timeWindow: '中期',
    pathSummary: '油品运输服务（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: '油品运输服务 → 成品油批发交付服务',
    nodes: [
      {
        key: 'chn-21-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-21-n01'
        },
        name: '油品运输服务',
        impact: '交付周期 UP/HIGH；利润率 UP/HIGH；战略资源安全 DOWN/MEDIUM；销售价格 UP/HIGH',
        result: {
          code: 'diverging',
          label: '分化'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '美伊冲突扰乱霍尔木兹海峡，导致油品运输服务的交付周期延长；美伊冲突扰乱霍尔木兹海峡，油运高景气延续，TD3C-TCE超60万美元/天。对“油品运输服务”环节而言，这意味着交付周期上升、利润率上升、战略资源安全下降、销售价格上升，因此本期判断为分化。',
        timeWindow: '中期',
        confidence: {
          label: '中–高',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-21-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-21-n02'
        },
        name: '成品油批发交付服务',
        impact: '投入成本 UP/LOW；交付周期 UP/LOW',
        result: {
          code: 'cooling',
          label: '降温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '油品运输的交付周期和销售价格上升，会把更高的物流成本和更长的到货时间传导给批发交付环节。成品油批发服务明确依赖运输服务，因此推测其投入成本和交付周期上升，经营景气受压。',
        timeWindow: '短期–中期（传导滞后）',
        confidence: {
          label: '低（0.62）',
          score: 0.62
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-21-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-21-n02',
        toNodeKey: 'chn-21-n01',
        relationLabel: '依赖'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-22',
    claimKey: 'chn-22-claim',
    displayOrder: 22,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-22'
    },
    name: '火力发电设备产业链',
    conclusion:
      '汽轮机出现直接 Signal，当前链级聚合结果为分化，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 1 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'diverging',
      label: '分化'
    },
    confidence: {
      label: '中–高',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: '汽轮机（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: '汽轮机 → 火电设备',
    nodes: [
      {
        key: 'chn-22-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-22-n01'
        },
        name: '汽轮机',
        impact:
          '产能释放周期 UP/MEDIUM；市场需求 UP/HIGH；库存水平 DOWN/MEDIUM；扩产强度 UP/MEDIUM',
        result: {
          code: 'diverging',
          label: '分化'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '美国燃气轮机需求创新高，促使制造商扩大产能；美国对燃气轮机的需求创历史新高，主要由数据中心电力需求驱动。对“汽轮机”环节而言，这意味着产能释放周期上升、市场需求上升、库存水平下降、扩产强度上升，因此本期判断为分化。',
        timeWindow: '长期–中期',
        confidence: {
          label: '中–高',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-22-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-22-n02'
        },
        name: '火电设备',
        impact: '市场需求 UP/LOW；扩产强度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '汽轮机的市场需求和扩产强度上升，表明火电和燃机设备的核心动力环节正在扩张。汽轮机是火电设备的明确组成环节，因此推测整体设备需求和扩产强度上升。',
        timeWindow: '中期–长期（传导假设）',
        confidence: {
          label: '低（0.48）',
          score: 0.48
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-22-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-22-n01',
        toNodeKey: 'chn-22-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-23',
    claimKey: 'chn-23-claim',
    displayOrder: 23,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-23'
    },
    name: '磷酸铁锂刀片电池产业链',
    conclusion:
      '新能源车整车出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: '新能源车整车（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-23-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-23-n01'
        },
        name: '新能源车整车',
        impact: '商业化进度 UP/MEDIUM；市场需求 UP/MEDIUM；性价比 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国国家市场监管总局宣布分阶段实施GB1589-2026标准，新车型需于2027年7月1日起符合，现有车型需于2028年1月1日起符合，并给予车企技术升级和过渡时间；吉利汽车发布混动SUV车型Monjaro EM-i，开启首款混动D级SUV全球推广，直接增加新能源车（混动SUV）的供给与可得性；该节点归属 5 条链，本链语境未确定。对“新能源车整车”环节而言，这意味着商业化进度上升、市场需求上升、性价比上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-23-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-23-n02'
        },
        name: '刀片电池',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“刀片电池”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-23-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-23-n02',
        toNodeKey: 'chn-23-n01',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-24',
    claimKey: 'chn-24-claim',
    displayOrder: 24,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-24'
    },
    name: '车规级功率模块产业链',
    conclusion:
      '新能源车整车出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: '新能源车整车（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-24-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-24-n01'
        },
        name: '新能源车整车',
        impact: '商业化进度 UP/MEDIUM；市场需求 UP/MEDIUM；性价比 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国国家市场监管总局宣布分阶段实施GB1589-2026标准，新车型需于2027年7月1日起符合，现有车型需于2028年1月1日起符合，并给予车企技术升级和过渡时间；吉利汽车发布混动SUV车型Monjaro EM-i，开启首款混动D级SUV全球推广，直接增加新能源车（混动SUV）的供给与可得性；该节点归属 5 条链，本链语境未确定。对“新能源车整车”环节而言，这意味着商业化进度上升、市场需求上升、性价比上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-24-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-24-n02'
        },
        name: '新能源汽车主驱逆变器',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“新能源汽车主驱逆变器”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-24-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-24-n02',
        toNodeKey: 'chn-24-n01',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-25',
    claimKey: 'chn-25-claim',
    displayOrder: 25,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-25'
    },
    name: 'AI个人电脑产业链',
    conclusion:
      'AI芯片出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: 'AI芯片（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-25-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-25-n01'
        },
        name: 'AI芯片',
        impact: '商业化进度 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Google与Marvell扩大定制芯片合作，涉及TPU等AI芯片领域；OpenAI在Hot Chips 2026披露自研AI加速器ASIC Jalapeño的基准测试结果，该芯片为通用AI负载从零设计，非GPU改造；该节点归属 7 条链，本链语境未确定。对“AI芯片”环节而言，这意味着商业化进度上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-25-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-25-n02'
        },
        name: 'AI PC',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“AI PC”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-25-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-25-n01',
        toNodeKey: 'chn-25-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-26',
    claimKey: 'chn-26-claim',
    displayOrder: 26,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-26'
    },
    name: 'AI智能眼镜产业链',
    conclusion:
      'AI芯片出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: 'AI芯片（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-26-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-26-n01'
        },
        name: 'AI芯片',
        impact: '商业化进度 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Google与Marvell扩大定制芯片合作，涉及TPU等AI芯片领域；OpenAI在Hot Chips 2026披露自研AI加速器ASIC Jalapeño的基准测试结果，该芯片为通用AI负载从零设计，非GPU改造；该节点归属 7 条链，本链语境未确定。对“AI芯片”环节而言，这意味着商业化进度上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-26-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-26-n02'
        },
        name: 'AI眼镜',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“AI眼镜”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-26-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-26-n01',
        toNodeKey: 'chn-26-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-27',
    claimKey: 'chn-27-claim',
    displayOrder: 27,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-27'
    },
    name: 'AI算力租赁服务产业链',
    conclusion:
      'AI芯片出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: 'AI芯片（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-27-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-27-n01'
        },
        name: 'AI芯片',
        impact: '商业化进度 UP/MEDIUM；技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Google与Marvell扩大定制芯片合作，涉及TPU等AI芯片领域；OpenAI在Hot Chips 2026披露自研AI加速器ASIC Jalapeño的基准测试结果，该芯片为通用AI负载从零设计，非GPU改造；该节点归属 7 条链，本链语境未确定。对“AI芯片”环节而言，这意味着商业化进度上升、技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期–中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-27-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-27-n02'
        },
        name: 'AI服务器',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“AI服务器”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-27-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-27-n01',
        toNodeKey: 'chn-27-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-28',
    claimKey: 'chn-28-claim',
    displayOrder: 28,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-28'
    },
    name: 'CAR-T细胞治疗产业链',
    conclusion:
      '医院医疗服务出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '医院医疗服务（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-28-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-28-n01'
        },
        name: '医院医疗服务',
        impact: '商业化进度 UP/MEDIUM；市场需求 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '华为与瑞金医院发布瑞智病理大模型RuiPath 2.0并落地90+医院；该节点归属 6 条链，本链语境未确定。对“医院医疗服务”环节而言，这意味着商业化进度上升、市场需求上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-28-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-28-n02'
        },
        name: 'CAR-T细胞疗法',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“CAR-T细胞疗法”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-28-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-28-n02',
        toNodeKey: 'chn-28-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-29',
    claimKey: 'chn-29-claim',
    displayOrder: 29,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-29'
    },
    name: '化学药制剂产业链',
    conclusion:
      '医院医疗服务出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '医院医疗服务（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-29-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-29-n01'
        },
        name: '医院医疗服务',
        impact: '商业化进度 UP/MEDIUM；市场需求 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '华为与瑞金医院发布瑞智病理大模型RuiPath 2.0并落地90+医院；该节点归属 6 条链，本链语境未确定。对“医院医疗服务”环节而言，这意味着商业化进度上升、市场需求上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-29-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-29-n02'
        },
        name: '医药流通服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“医药流通服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-29-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-29-n02',
        toNodeKey: 'chn-29-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-30',
    claimKey: 'chn-30-claim',
    displayOrder: 30,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-30'
    },
    name: '医用芬太尼药品产业链',
    conclusion:
      '医院医疗服务出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '医院医疗服务（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-30-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-30-n01'
        },
        name: '医院医疗服务',
        impact: '商业化进度 UP/MEDIUM；市场需求 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '华为与瑞金医院发布瑞智病理大模型RuiPath 2.0并落地90+医院；该节点归属 6 条链，本链语境未确定。对“医院医疗服务”环节而言，这意味着商业化进度上升、市场需求上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-30-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-30-n02'
        },
        name: '医药流通服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“医药流通服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-30-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-30-n02',
        toNodeKey: 'chn-30-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-31',
    claimKey: 'chn-31-claim',
    displayOrder: 31,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-31'
    },
    name: '医疗信息化系统产业链',
    conclusion:
      '医院医疗服务出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '医院医疗服务（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-31-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-31-n01'
        },
        name: '医院医疗服务',
        impact: '商业化进度 UP/MEDIUM；市场需求 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '华为与瑞金医院发布瑞智病理大模型RuiPath 2.0并落地90+医院；该节点归属 6 条链，本链语境未确定。对“医院医疗服务”环节而言，这意味着商业化进度上升、市场需求上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-31-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-31-n02'
        },
        name: '医院信息系统',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“医院信息系统”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-31-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-31-n01',
        toNodeKey: 'chn-31-n02',
        relationLabel: '依赖'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-32',
    claimKey: 'chn-32-claim',
    displayOrder: 32,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-32'
    },
    name: '医疗设备产业链',
    conclusion:
      '医院医疗服务出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '医院医疗服务（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-32-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-32-n01'
        },
        name: '医院医疗服务',
        impact: '商业化进度 UP/MEDIUM；市场需求 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '华为与瑞金医院发布瑞智病理大模型RuiPath 2.0并落地90+医院；该节点归属 6 条链，本链语境未确定。对“医院医疗服务”环节而言，这意味着商业化进度上升、市场需求上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-32-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-32-n02'
        },
        name: '多参数患者监护仪',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“多参数患者监护仪”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-32-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-32-n02',
        toNodeKey: 'chn-32-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-33',
    claimKey: 'chn-33-claim',
    displayOrder: 33,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-33'
    },
    name: '医院医疗服务产业链',
    conclusion:
      '医院医疗服务出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '医院医疗服务（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-33-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-33-n01'
        },
        name: '医院医疗服务',
        impact: '商业化进度 UP/MEDIUM；市场需求 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '华为与瑞金医院发布瑞智病理大模型RuiPath 2.0并落地90+医院；该节点归属 6 条链，本链语境未确定。对“医院医疗服务”环节而言，这意味着商业化进度上升、市场需求上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-33-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-33-n02'
        },
        name: '医护专业服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“医护专业服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-33-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-33-n03'
        },
        name: '医院药械供应服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“医院药械供应服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-33-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-33-n04'
        },
        name: '医院诊疗设施',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“医院诊疗设施”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-33-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-33-n02',
        toNodeKey: 'chn-33-n01',
        relationLabel: '投入'
      },
      {
        key: 'chn-33-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-33-n01',
        toNodeKey: 'chn-33-n04',
        relationLabel: '依赖'
      },
      {
        key: 'chn-33-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-33-n03',
        toNodeKey: 'chn-33-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-34',
    claimKey: 'chn-34-claim',
    displayOrder: 34,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-34'
    },
    name: '跨境支付服务产业链',
    conclusion:
      '跨境支付服务出现直接 Signal，节点方向为分化；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'diverging',
      label: '分化'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期–长期',
    pathSummary: '跨境支付服务（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-34-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-34-n01'
        },
        name: '跨境支付服务',
        impact: '渗透率 UP/LOW；进口依赖度 DOWN/LOW',
        result: {
          code: 'diverging',
          label: '分化'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中东客户主动询问用人民币结算以节省换汇手续费并改善账期，反映跨境支付服务采用人民币结算的意愿增强；中东客户对人民币结算的兴趣可能推动跨境支付服务减少对美元等外币的依赖，从而降低进口依赖度；该节点归属 2 条链，本链语境未确定。对“跨境支付服务”环节而言，这意味着渗透率上升、进口依赖度下降，因此本期判断为分化。',
        timeWindow: '中期–长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-34-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-34-n02'
        },
        name: '反洗钱与制裁筛查系统',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“反洗钱与制裁筛查系统”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-34-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-34-n03'
        },
        name: '外汇兑换服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“外汇兑换服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-34-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-34-n04'
        },
        name: '支付网关服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“支付网关服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-34-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-34-n01',
        toNodeKey: 'chn-34-n02',
        relationLabel: '依赖'
      },
      {
        key: 'chn-34-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-34-n04',
        toNodeKey: 'chn-34-n01',
        relationLabel: '投入'
      },
      {
        key: 'chn-34-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-34-n03',
        toNodeKey: 'chn-34-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-35',
    claimKey: 'chn-35-claim',
    displayOrder: 35,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-35'
    },
    name: '跨境电商服务产业链',
    conclusion:
      '跨境支付服务出现直接 Signal，节点方向为分化；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'diverging',
      label: '分化'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期–长期',
    pathSummary: '跨境支付服务（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-35-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-35-n01'
        },
        name: '跨境支付服务',
        impact: '渗透率 UP/LOW；进口依赖度 DOWN/LOW',
        result: {
          code: 'diverging',
          label: '分化'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中东客户主动询问用人民币结算以节省换汇手续费并改善账期，反映跨境支付服务采用人民币结算的意愿增强；中东客户对人民币结算的兴趣可能推动跨境支付服务减少对美元等外币的依赖，从而降低进口依赖度；该节点归属 2 条链，本链语境未确定。对“跨境支付服务”环节而言，这意味着渗透率上升、进口依赖度下降，因此本期判断为分化。',
        timeWindow: '中期–长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-35-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-35-n02'
        },
        name: '跨境电商服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“跨境电商服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-35-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-35-n01',
        toNodeKey: 'chn-35-n02',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-36',
    claimKey: 'chn-36-claim',
    displayOrder: 36,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-36'
    },
    name: '钢铁长材产业链',
    conclusion:
      '热轧带肋钢筋出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 1 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中–高',
      score: null
    },
    timeWindow: '短期',
    pathSummary: '热轧带肋钢筋（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: '热轧带肋钢筋 → 建筑钢筋精整加工服务',
    nodes: [
      {
        key: 'chn-36-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-36-n01'
        },
        name: '热轧带肋钢筋',
        impact: '成本传导能力 UP/HIGH；销售价格 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '土耳其螺纹钢出口报价升至588美元/吨FOB，原因为成本上涨及人民币偏强；土耳其螺纹钢出口报价本周升至588美元/吨FOB。对“热轧带肋钢筋”环节而言，这意味着成本传导能力上升、销售价格上升，因此本期判断为升温。',
        timeWindow: '短期',
        confidence: {
          label: '中–高',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-36-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-36-n02'
        },
        name: '建筑钢筋精整加工服务',
        impact: '投入成本 UP/LOW',
        result: {
          code: 'cooling',
          label: '降温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '热轧带肋钢筋的销售价格和成本传导能力上升，会提高精整加工环节的原料成本。热轧带肋钢筋是建筑钢筋精整加工的明确投入，因此推测该服务的投入成本上升，利润空间可能受压。',
        timeWindow: '短期（传导滞后）',
        confidence: {
          label: '低（0.62）',
          score: 0.62
        },
        hasEvidence: false
      },
      {
        key: 'chn-36-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-36-n03'
        },
        name: '钢筋轧制服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“钢筋轧制服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-36-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-36-n01',
        toNodeKey: 'chn-36-n02',
        relationLabel: '投入'
      },
      {
        key: 'chn-36-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-36-n03',
        toNodeKey: 'chn-36-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-37',
    claimKey: 'chn-37-claim',
    displayOrder: 37,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-37'
    },
    name: '锂电储能系统产业链',
    conclusion: '储能锂离子电芯出现直接 Signal，当前链级聚合结果为升温，向相邻节点的传播尚待验证。',
    status: '节点 Signal 明确，链内传播待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '短期',
    pathSummary: '储能锂离子电芯（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-37-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-37-n01'
        },
        name: '储能锂离子电芯',
        impact: '产业利润池 UP/UNKNOWN；利润率 UP/UNKNOWN',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '2026年上半年，中国上市电池储能公司进入中期报告季峰值，报告显示最广泛的利润复苏。对“储能锂离子电芯”环节而言，这意味着产业利润池上升、利润率上升，因此本期判断为升温。',
        timeWindow: '短期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-37-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-37-n02'
        },
        name: '锂电储能系统',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“锂电储能系统”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-37-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-37-n01',
        toNodeKey: 'chn-37-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap: '同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-38',
    claimKey: 'chn-38-claim',
    displayOrder: 38,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-38'
    },
    name: '集成电路晶圆制造产业链',
    conclusion: '半导体设备出现直接 Signal，当前链级聚合结果为分化，向相邻节点的传播尚待验证。',
    status: '节点 Signal 明确，链内传播待验证',
    result: {
      code: 'diverging',
      label: '分化'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '长期–中期',
    pathSummary: '半导体设备（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-38-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-38-n01'
        },
        name: '半导体设备',
        impact: '扩产强度 UP/MEDIUM；政策支持力度 DOWN/HIGH',
        result: {
          code: 'diverging',
          label: '分化'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '美国商务部推进CHIPS法案制造业激励项目，49个项目中24个项目完成关键里程碑，多数项目2028年前完成；美国商务部取消CHIPS法案下78亿美元研发拨款，其中大部分原计划给Natcast运营的NSTC，转向量子计算投资。对“半导体设备”环节而言，这意味着扩产强度上升、政策支持力度下降，因此本期判断为分化。',
        timeWindow: '长期–中期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-38-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-38-n02'
        },
        name: '集成电路晶圆制造服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“集成电路晶圆制造服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-38-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-38-n02',
        toNodeKey: 'chn-38-n01',
        relationLabel: '依赖'
      }
    ],
    uncertainty: {
      counterevidenceAndGap: '同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-39',
    claimKey: 'chn-39-claim',
    displayOrder: 39,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-39'
    },
    name: 'AI训练语料与数据服务产业链',
    conclusion:
      'AI语料出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: 'AI语料（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-39-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-39-n01'
        },
        name: 'AI语料',
        impact: '商业化进度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'AgentX于2026年8月24日发布开源数据集，价值300万美元，支持长上下文、多轮和子代理特性；该节点归属 2 条链，本链语境未确定。对“AI语料”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-39-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-39-n02'
        },
        name: '数据清洗标注服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“数据清洗标注服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-39-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-39-n03'
        },
        name: '模型训练服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“模型训练服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-39-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-39-n02',
        toNodeKey: 'chn-39-n01',
        relationLabel: '投入'
      },
      {
        key: 'chn-39-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-39-n01',
        toNodeKey: 'chn-39-n03',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-40',
    claimKey: 'chn-40-claim',
    displayOrder: 40,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-40'
    },
    name: '健身器材产业链',
    conclusion:
      '传感器出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: '传感器（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-40-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-40-n01'
        },
        name: '传感器',
        impact: '政策支持力度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '2026年8月29日，北京怀柔发布2026年第一批采购机遇清单，采购额约4亿元，包含超过10项采购机遇，针对高端科学仪器与传感器产业；该节点归属 7 条链，本链语境未确定。对“传感器”环节而言，这意味着政策支持力度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-40-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-40-n02'
        },
        name: '健身器材',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“健身器材”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-40-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-40-n01',
        toNodeKey: 'chn-40-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-41',
    claimKey: 'chn-41-claim',
    displayOrder: 41,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-41'
    },
    name: '创新药产业链',
    conclusion: '创新小分子药出现直接 Signal，当前链级聚合结果为升温，向相邻节点的传播尚待验证。',
    status: '节点 Signal 明确，链内传播待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '创新小分子药（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-41-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-41-n01'
        },
        name: '创新小分子药',
        impact: '渗透率 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'FDA批准Revolution Medicines的胰腺癌治疗药物Rasonque，该药在临床试验中使患者生存期几乎翻倍。对“创新小分子药”环节而言，这意味着渗透率上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-41-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-41-n02'
        },
        name: '药品上市许可',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“药品上市许可”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-41-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-41-n03'
        },
        name: '药物临床试验服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“药物临床试验服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-41-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-41-n01',
        toNodeKey: 'chn-41-n02',
        relationLabel: '依赖'
      },
      {
        key: 'chn-41-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-41-n03',
        toNodeKey: 'chn-41-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap: '同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-42',
    claimKey: 'chn-42-claim',
    displayOrder: 42,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-42'
    },
    name: '半导体先进封装产业链',
    conclusion:
      '集成电路制造出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '集成电路制造（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-42-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-42-n01'
        },
        name: '集成电路制造',
        impact: '产能利用率 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '美国商务部推进CHIPS法案制造业激励项目，多数项目计划2028年前完成；该节点归属 4 条链，本链语境未确定。对“集成电路制造”环节而言，这意味着产能利用率上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-42-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-42-n02'
        },
        name: '先进封装',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“先进封装”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-42-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-42-n01',
        toNodeKey: 'chn-42-n02',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-43',
    claimKey: 'chn-43-claim',
    displayOrder: 43,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-43'
    },
    name: '基础设施云服务产业链',
    conclusion:
      '数据中心网络设备出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 1 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '数据中心网络设备（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: '数据中心网络设备 → 云计算基础设施',
    nodes: [
      {
        key: 'chn-43-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-43-n01'
        },
        name: '数据中心网络设备',
        impact: '市场需求 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Google与Marvell扩大定制芯片合作，范围扩展至网络领域。对“数据中心网络设备”环节而言，这意味着市场需求上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-43-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-43-n02'
        },
        name: '云计算基础设施',
        impact: '资本开支 UP/LOW；扩产强度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '数据中心网络设备需求上升，通常对应云基础设施的新建、扩容或升级投入。网络设备是云计算基础设施的真实组成环节，因此推测云基础设施的资本开支和扩产强度上升。',
        timeWindow: '长期（传导假设）',
        confidence: {
          label: '低（0.48）',
          score: 0.48
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-43-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-43-n01',
        toNodeKey: 'chn-43-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-44',
    claimKey: 'chn-44-claim',
    displayOrder: 44,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-44'
    },
    name: '存储芯片产业链',
    conclusion:
      'NAND Flash存储芯片出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 1 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: 'NAND Flash存储芯片（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: 'NAND Flash存储芯片 → NAND Flash封装测试服务',
    nodes: [
      {
        key: 'chn-44-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-44-n01'
        },
        name: 'NAND Flash存储芯片',
        impact: '市场需求 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          'Google与Marvell扩大定制芯片合作，范围扩展至存储、网络等数据搬移领域。对“NAND Flash存储芯片”环节而言，这意味着市场需求上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-44-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-44-n02'
        },
        name: 'NAND Flash封装测试服务',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          'NAND Flash存储芯片需求上升，会增加芯片出货前的封装与测试工作量。封装测试服务是NAND Flash芯片的明确投入环节，因此推测其市场需求上升。',
        timeWindow: '长期（传导滞后）',
        confidence: {
          label: '低（0.54）',
          score: 0.54
        },
        hasEvidence: false
      },
      {
        key: 'chn-44-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-44-n03'
        },
        name: '固态硬盘（SSD）',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“固态硬盘（SSD）”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-44-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-44-n02',
        toNodeKey: 'chn-44-n01',
        relationLabel: '投入'
      },
      {
        key: 'chn-44-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-44-n01',
        toNodeKey: 'chn-44-n03',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-45',
    claimKey: 'chn-45-claim',
    displayOrder: 45,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-45'
    },
    name: '家用清洁电器产业链',
    conclusion:
      '传感器出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: '传感器（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-45-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-45-n01'
        },
        name: '传感器',
        impact: '政策支持力度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '2026年8月29日，北京怀柔发布2026年第一批采购机遇清单，采购额约4亿元，包含超过10项采购机遇，针对高端科学仪器与传感器产业；该节点归属 7 条链，本链语境未确定。对“传感器”环节而言，这意味着政策支持力度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-45-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-45-n02'
        },
        name: '扫地机器人',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“扫地机器人”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-45-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-45-n01',
        toNodeKey: 'chn-45-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-46',
    claimKey: 'chn-46-claim',
    displayOrder: 46,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-46'
    },
    name: '扩展现实设备及服务产业链',
    conclusion:
      '传感器出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: '传感器（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-46-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-46-n01'
        },
        name: '传感器',
        impact: '政策支持力度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '2026年8月29日，北京怀柔发布2026年第一批采购机遇清单，采购额约4亿元，包含超过10项采购机遇，针对高端科学仪器与传感器产业；该节点归属 7 条链，本链语境未确定。对“传感器”环节而言，这意味着政策支持力度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-46-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-46-n02'
        },
        name: 'XR头显整机',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“XR头显整机”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-46-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-46-n01',
        toNodeKey: 'chn-46-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-47',
    claimKey: 'chn-47-claim',
    displayOrder: 47,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-47'
    },
    name: '数字集成电路设计产业链',
    conclusion:
      '集成电路制造出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '集成电路制造（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-47-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-47-n01'
        },
        name: '集成电路制造',
        impact: '产能利用率 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '美国商务部推进CHIPS法案制造业激励项目，多数项目计划2028年前完成；该节点归属 4 条链，本链语境未确定。对“集成电路制造”环节而言，这意味着产能利用率上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-47-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-47-n02'
        },
        name: '数字芯片设计',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“数字芯片设计”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-47-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-47-n03'
        },
        name: '集成电路封测',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“集成电路封测”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-47-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-47-n01',
        toNodeKey: 'chn-47-n03',
        relationLabel: '投入'
      },
      {
        key: 'chn-47-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-47-n02',
        toNodeKey: 'chn-47-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-48',
    claimKey: 'chn-48-claim',
    displayOrder: 48,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-48'
    },
    name: '模拟集成电路设计产业链',
    conclusion:
      '集成电路制造出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '集成电路制造（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-48-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-48-n01'
        },
        name: '集成电路制造',
        impact: '产能利用率 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '美国商务部推进CHIPS法案制造业激励项目，多数项目计划2028年前完成；该节点归属 4 条链，本链语境未确定。对“集成电路制造”环节而言，这意味着产能利用率上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-48-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-48-n02'
        },
        name: '模拟芯片设计',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“模拟芯片设计”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-48-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-48-n03'
        },
        name: '集成电路封测',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“集成电路封测”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-48-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-48-n01',
        toNodeKey: 'chn-48-n03',
        relationLabel: '投入'
      },
      {
        key: 'chn-48-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-48-n02',
        toNodeKey: 'chn-48-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-49',
    claimKey: 'chn-49-claim',
    displayOrder: 49,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-49'
    },
    name: '汽车整车产业链',
    conclusion: '汽车整车出现直接 Signal，当前链级聚合结果为升温，向相邻节点的传播尚待验证。',
    status: '节点 Signal 明确，链内传播待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '汽车整车（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-49-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-49-n01'
        },
        name: '汽车整车',
        impact: '技术成熟度 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '中国国家市场监管总局宣布分阶段实施GB1589-2026标准，新车型需于2027年7月1日起符合，现有车型需于2028年1月1日起符合，并给予车企技术升级和过渡时间。对“汽车整车”环节而言，这意味着技术成熟度上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-49-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-49-n02'
        },
        name: '汽车动力总成',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“汽车动力总成”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-49-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-49-n03'
        },
        name: '汽车底盘总成',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“汽车底盘总成”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-49-n04',
        displayOrder: 4,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-49-n04'
        },
        name: '汽车整车制造服务',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“汽车整车制造服务”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-49-n05',
        displayOrder: 5,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-49-n05'
        },
        name: '汽车电子电气系统',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“汽车电子电气系统”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      },
      {
        key: 'chn-49-n06',
        displayOrder: 6,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-49-n06'
        },
        name: '汽车车身与车架总成',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“汽车车身与车架总成”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-49-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-49-n03',
        toNodeKey: 'chn-49-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-49-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-49-n04',
        toNodeKey: 'chn-49-n01',
        relationLabel: '投入'
      },
      {
        key: 'chn-49-edge-03',
        displayOrder: 3,
        fromNodeKey: 'chn-49-n02',
        toNodeKey: 'chn-49-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-49-edge-04',
        displayOrder: 4,
        fromNodeKey: 'chn-49-n06',
        toNodeKey: 'chn-49-n01',
        relationLabel: '组成'
      },
      {
        key: 'chn-49-edge-05',
        displayOrder: 5,
        fromNodeKey: 'chn-49-n05',
        toNodeKey: 'chn-49-n01',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap: '同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-50',
    claimKey: 'chn-50-claim',
    displayOrder: 50,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-50'
    },
    name: '电动垂直起降飞行器产业链',
    conclusion:
      'eVTOL推进电池系统出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 1 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 1 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: 'eVTOL推进电池系统（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: 'eVTOL推进电池系统 → eVTOL飞行器',
    nodes: [
      {
        key: 'chn-50-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-50-n01'
        },
        name: 'eVTOL推进电池系统',
        impact: '性价比 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '美国能源部授权Nomad Power Solutions在佛蒙特州推进移动电池储能示范项目，公司在新所有权下开始运营。对“eVTOL推进电池系统”环节而言，这意味着性价比上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-50-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-50-n02'
        },
        name: 'eVTOL飞行器',
        impact: '技术成熟度 UP/LOW；商业化进度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          'eVTOL推进电池系统的性价比上升，有助于缓解飞行器在重量、续航和整机成本上的约束。推进电池是飞行器的明确组成环节，因此推测整机技术成熟度和商业化进度上升。',
        timeWindow: '中期–长期（传导假设）',
        confidence: {
          label: '低（0.42）',
          score: 0.42
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-50-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-50-n01',
        toNodeKey: 'chn-50-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-51',
    claimKey: 'chn-51-claim',
    displayOrder: 51,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-51'
    },
    name: '精准医疗服务产业链',
    conclusion:
      '分子伴随诊断服务出现直接 Signal，当前链级聚合结果为升温，已形成可解释的动态传导假设，其余相邻节点仍待验证；本链新增 2 条动态传导假设。',
    status: '直接节点 Signal 明确；新增 2 条动态传导假设，其余相邻节点继续待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '分子伴随诊断服务（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary:
      '分子伴随诊断服务 → 精准用药决策服务；分子伴随诊断服务 → 基因数据分析服务',
    nodes: [
      {
        key: 'chn-51-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-51-n01'
        },
        name: '分子伴随诊断服务',
        impact: '商业化进度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '华为与瑞金医院发布瑞智病理大模型RuiPath 2.0并落地90+医院。对“分子伴随诊断服务”环节而言，这意味着商业化进度上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-51-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-51-n02'
        },
        name: '基因数据分析服务',
        impact: '市场需求 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '分子伴随诊断服务商业化加快，会增加对基因数据处理、变异解读和报告生成的需求。基因数据分析是伴随诊断的明确投入环节，因此推测其市场需求上升。',
        timeWindow: '长期（传导滞后）',
        confidence: {
          label: '低（0.45）',
          score: 0.45
        },
        hasEvidence: false
      },
      {
        key: 'chn-51-n03',
        displayOrder: 3,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-51-n03'
        },
        name: '精准用药决策服务',
        impact: '商业化进度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'reasoning_hypothesis',
          label: '推理假设'
        },
        reasoning:
          '分子伴随诊断服务商业化加快，会提高可用的患者分层和用药匹配信息供给。伴随诊断是精准用药决策的明确投入，因此推测决策服务的商业化进度上升。',
        timeWindow: '长期（传导滞后）',
        confidence: {
          label: '低（0.42）',
          score: 0.42
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-51-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-51-n01',
        toNodeKey: 'chn-51-n03',
        relationLabel: '投入'
      },
      {
        key: 'chn-51-edge-02',
        displayOrder: 2,
        fromNodeKey: 'chn-51-n02',
        toNodeKey: 'chn-51-n01',
        relationLabel: '投入'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '上述传导为经路径评分筛选的推理假设，仍需目标节点的订单、价格、产能或经营数据验证；未被推导的相邻节点继续作为 Evidence Gap。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-52',
    claimKey: 'chn-52-claim',
    displayOrder: 52,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-52'
    },
    name: '航空机电系统产业链',
    conclusion:
      '传感器出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: '传感器（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-52-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-52-n01'
        },
        name: '传感器',
        impact: '政策支持力度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '2026年8月29日，北京怀柔发布2026年第一批采购机遇清单，采购额约4亿元，包含超过10项采购机遇，针对高端科学仪器与传感器产业；该节点归属 7 条链，本链语境未确定。对“传感器”环节而言，这意味着政策支持力度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-52-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-52-n02'
        },
        name: '航空机电系统',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“航空机电系统”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-52-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-52-n01',
        toNodeKey: 'chn-52-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-53',
    claimKey: 'chn-53-claim',
    displayOrder: 53,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-53'
    },
    name: '航空电子系统产业链',
    conclusion:
      '传感器出现直接 Signal，节点方向为升温；但至少一个节点同时归属多条产业链，本链是否承接全部影响仍需链语境验证。',
    status: '节点 Signal 明确，链语境待解析',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '低–中',
      score: null
    },
    timeWindow: '中期',
    pathSummary: '传感器（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-53-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-53-n01'
        },
        name: '传感器',
        impact: '政策支持力度 UP/LOW',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '2026年8月29日，北京怀柔发布2026年第一批采购机遇清单，采购额约4亿元，包含超过10项采购机遇，针对高端科学仪器与传感器产业；该节点归属 7 条链，本链语境未确定。对“传感器”环节而言，这意味着政策支持力度上升，因此本期判断为升温。',
        timeWindow: '中期',
        confidence: {
          label: '低–中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-53-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-53-n02'
        },
        name: '航空电子系统',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“航空电子系统”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-53-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-53-n01',
        toNodeKey: 'chn-53-n02',
        relationLabel: '组成'
      }
    ],
    uncertainty: {
      counterevidenceAndGap:
        '共享节点的具体链语境尚未解析；同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  },
  {
    key: 'chn-54',
    claimKey: 'chn-54-claim',
    displayOrder: 54,
    scope: {
      type: 'industry_chain_summary',
      key: 'chn-54'
    },
    name: '风力发电产业链',
    conclusion: '风力发电出现直接 Signal，当前链级聚合结果为升温，向相邻节点的传播尚待验证。',
    status: '节点 Signal 明确，链内传播待验证',
    result: {
      code: 'warming',
      label: '升温'
    },
    confidence: {
      label: '中',
      score: null
    },
    timeWindow: '长期',
    pathSummary: '风力发电（直接 Signal 节点）→ 真实同链拓扑节点',
    acceptedHypothesisSummary: null,
    nodes: [
      {
        key: 'chn-54-n01',
        displayOrder: 1,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-54-n01'
        },
        name: '风力发电',
        impact: '市场供给 UP/MEDIUM',
        result: {
          code: 'warming',
          label: '升温'
        },
        nature: {
          code: 'direct_evidence',
          label: '直接证据'
        },
        reasoning:
          '墨西哥政府启动能源改革，开放能源领域吸引私人投资，重点扩张可再生能源部门。对“风力发电”环节而言，这意味着市场供给上升，因此本期判断为升温。',
        timeWindow: '长期',
        confidence: {
          label: '中',
          score: null
        },
        hasEvidence: true
      },
      {
        key: 'chn-54-n02',
        displayOrder: 2,
        scope: {
          type: 'industry_chain_node',
          key: 'chn-54-n02'
        },
        name: '陆上风电场基础设施',
        impact: '真实同链拓扑相邻，尚无直接 Signal',
        result: {
          code: 'pending',
          label: '待验证'
        },
        nature: {
          code: 'pending_validation',
          label: '待验证'
        },
        reasoning:
          '“陆上风电场基础设施”与本链中的直接受影响节点存在真实产业链关系，但在本次证据范围内，没有订单、价格、产能、技术进展或经营数据直接指向该环节，也没有足够依据确认影响已经传导至此，因此暂不判断升降方向。',
        timeWindow: '后续周期',
        confidence: {
          label: '低',
          score: null
        },
        hasEvidence: false
      }
    ],
    edges: [
      {
        key: 'chn-54-edge-01',
        displayOrder: 1,
        fromNodeKey: 'chn-54-n01',
        toNodeKey: 'chn-54-n02',
        relationLabel: '依赖'
      }
    ],
    uncertainty: {
      counterevidenceAndGap: '同链相邻节点缺少直接 Variable Signal 与经营观测。',
      stopCondition:
        '若后续 Event 使节点 Signal 失效、方向反转，或链语境确认不属于本链，则关闭或修正本链结论。',
      checkpoints: []
    },
    hasEvidence: true
  }
];
