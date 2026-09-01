import {
  ActivityLogIcon,
  ArrowLeftIcon,
  ArrowRightIcon,
  BarChartIcon,
  CheckCircledIcon,
  ChevronDownIcon,
  ClockIcon,
  CubeIcon,
  DashboardIcon,
  ExclamationTriangleIcon,
  FileTextIcon,
  GlobeIcon,
  InfoCircledIcon,
  LayersIcon,
  Link2Icon,
  ReaderIcon,
  SewingPinIcon,
} from "@radix-ui/react-icons";
import {
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  BottomSheet,
  Carousel,
  FlowStack,
  MobileScroll,
  type FlowControls,
  type FlowScreen,
} from "./mobile";
import generatedReportData from "./report-data.generated.json";

type LayerKey = "geo" | "macro" | "industry" | "company";
type Variant = "A" | "B" | "C";
type ImpactState = "升温" | "降温" | "分化" | "待验证";
type NodeResult = ImpactState | "待判定";

type AnchorImpact = {
  id: string;
  label: string;
  state: ImpactState;
  current?: string;
  resultLabel?: string;
  why?: string;
  nature?: string;
  window?: string;
  confidence?: string;
  evidenceIds?: string[];
};

type ReasoningStep = {
  id: string;
  input: string;
  mechanism: string;
  output: string;
  type: string;
  confidence: string;
  evidenceIds: string[];
};

type LayerTransmissionPath = {
  id: string;
  targetLayer: "macro" | "industry";
  targetLabel: string;
  logic: string;
  result: string;
  nature: string;
  confidence: string;
  status: string;
};

type Layer = {
  key: LayerKey;
  label: string;
  shortLabel: string;
  eyebrow: string;
  conclusion: string;
  state: ImpactState;
  horizon: string;
  confidence: string;
  anchors: AnchorImpact[];
  evidenceIds: string[];
  steps?: ReasoningStep[];
  counterevidence?: string;
  evidenceGap?: string;
  boundary?: string;
  reversal?: string;
  downstreamSummary?: string;
  downstreamChainIds?: string[];
  transmissionPaths?: LayerTransmissionPath[];
};

type Evidence = {
  id: string;
  publishedAt: string;
  publishedAtRaw?: string;
  summary: string;
  keywords: string[];
};

type ChainNode = {
  id: string;
  title: string;
  status: string;
  tone: "observed" | "inferred" | "gap";
  result: NodeResult;
  resultLabel?: string;
  lag: string;
  confidence: string;
  impact: string;
  why: string;
  evidence: string[];
};

type IndustryChain = {
  id: string;
  title: string;
  kind: string;
  conclusion: string;
  state: ImpactState;
  status: string;
  horizon: string;
  confidence: string;
  path?: string;
  hypotheses?: string;
  gap?: string;
  evidenceIds: string[];
  graphGroups: string[][];
  graphLinks: string[];
  graphEdges?: Array<{ from: string; to: string; relation: string }>;
  stopRule: string;
  nodes: ChainNode[];
};

type EvidenceSheetState = {
  evidenceIds: string[];
};

type ReasoningReport = {
  id: string;
  title: string;
  publishedAt: string;
  updatedAt: string;
  evidenceCount: number;
  eventCount?: number;
  chainCount?: number;
  hypothesisCount?: number;
  geo: Layer;
  macro: Layer;
  industryChains: IndustryChain[];
};

const evidenceItems: Evidence[] = [
  {
    id: "EV-01",
    publishedAt: "08-31 07:10",
    summary: "重点航道风险等级上调",
    keywords: ["重点航道", "风险等级"],
  },
  {
    id: "EV-02",
    publishedAt: "08-31 07:20",
    summary: "油轮保险报价一周上升 32%",
    keywords: ["油轮保险", "报价"],
  },
  {
    id: "EV-03",
    publishedAt: "08-31 07:25",
    summary: "波斯湾—东亚 VLCC 即期运价上升 14%",
    keywords: ["波斯湾—东亚", "VLCC", "即期运价"],
  },
  {
    id: "EV-04",
    publishedAt: "08-31 07:30",
    summary: "Brent 五日上涨 5.6%，近月价差扩大",
    keywords: ["Brent", "近月价差"],
  },
  {
    id: "EV-05",
    publishedAt: "08-30 18:00",
    summary: "炼厂库存 24.6 天，高于过去 30 日库存天数中位数；中位数值未提供",
    keywords: ["炼厂库存", "库存天数"],
  },
  {
    id: "EV-06",
    publishedAt: "08-31 07:40",
    summary: "石脑油上涨 3.1%，PE 上涨 0.6%",
    keywords: ["石脑油", "PE", "现货价格"],
  },
];

const geoLayer: Layer = {
  key: "geo",
  label: "地缘政治",
  shortLabel: "政治",
  eyebrow: "通道风险",
  conclusion: "模拟风险通告、保险报价和样本航线运价共同指向短期通道压力上升，但尚无过峡流量、装船量或出口量 Evidence，不能确认原油实物供应中断。",
  state: "分化",
  horizon: "即时–3 周",
  confidence: "中置信",
  evidenceIds: ["EV-01", "EV-02", "EV-03", "EV-05"],
  anchors: [
    { id: "GEO-A01", label: "霍尔木兹海峡", state: "升温", current: "风险等级上调", window: "即时–1周", confidence: "中", evidenceIds: ["EV-01"] },
    { id: "GEO-A02", label: "波斯湾—东亚航线", state: "升温", current: "保险与运价上升", window: "1–3周", confidence: "中", evidenceIds: ["EV-02", "EV-03"] },
    { id: "GEO-A03", label: "能源出口连续性", state: "分化", current: "风险指标上行，但尚未出现可确认中断", window: "即时", confidence: "中", evidenceIds: ["EV-01", "EV-05"] },
    { id: "GEO-A04", label: "区域护航行动", state: "待验证", current: "是否缓解风险待验证", window: "1–3周", confidence: "低", evidenceIds: [] },
  ],
  steps: [
    { id: "GEO-S01", input: "航道风险等级上调", mechanism: "风险识别改变保险与航运定价", output: "航运摩擦上升", type: "Event → Inference", confidence: "中", evidenceIds: ["EV-01", "EV-02"] },
    { id: "GEO-S02", input: "保险报价与 VLCC 运价同步上升", mechanism: "波斯湾—东亚线路运输成本抬升", output: "通道压力进入贸易成本", type: "Observation → Inference", confidence: "中", evidenceIds: ["EV-02", "EV-03"] },
    { id: "GEO-S03", input: "炼厂库存仍高于近期中位数", mechanism: "库存缓冲削弱即时断供判断", output: "不发布“供应已经中断”结论", type: "Counterevidence gate", confidence: "中", evidenceIds: ["EV-05"] },
  ],
  counterevidence: "当前炼厂库存尚未显示即时实物短缺，不能把风险等级上调等同于断供。",
  evidenceGap: "缺少连续船舶通行量、港口装卸量和护航措施的可验证变化。",
  reversal: "若模拟保险报价连续三日回落且船舶通行量恢复，本层“通道压力上升”结论降级。",
};

const macroLayer: Layer = {
  key: "macro",
  label: "宏观经济",
  shortLabel: "经济",
  eyebrow: "输入成本",
  conclusion: "模拟油价和航运费用可能先于实物短缺抬高进口能源成本，并形成 PPI 上行压力假设；炼厂库存观察仍对即时短缺构成缓冲性反证。",
  state: "分化",
  horizon: "即时–6 周",
  confidence: "中置信",
  evidenceIds: ["EV-03", "EV-04", "EV-05"],
  anchors: [
    { id: "MAC-A01", label: "Brent 价格与期限结构", state: "升温", current: "五日价格上涨、近月价差扩大；成因未拆分", window: "即时–2周", confidence: "中", evidenceIds: ["EV-04"] },
    { id: "MAC-A02", label: "VLCC 即期运价", state: "升温", current: "波斯湾—东亚航线报价上升", window: "即时–3周", confidence: "中", evidenceIds: ["EV-03"] },
    { id: "MAC-A03", label: "原油到岸成本", state: "升温", current: "存在上行压力，尚无实际到岸成本指标", window: "0–3周", confidence: "中", evidenceIds: ["EV-03", "EV-04"] },
    { id: "MAC-A04", label: "PPI 压力", state: "待验证", current: "能源与化工输入成本可能抬升", window: "2–6周", confidence: "低–中", evidenceIds: ["EV-03", "EV-04"] },
    { id: "MAC-A05", label: "沿海炼厂库存", state: "降温", current: "24.6 天，高于过去 30 日库存天数中位数；中位数值未提供", window: "0–3周", confidence: "中", evidenceIds: ["EV-05"] },
  ],
  steps: [
    { id: "MAC-S01", input: "VLCC 运价上涨 14%", mechanism: "海运费用进入进口原油成本", output: "运输成本上行", type: "Observation → Inference", confidence: "中", evidenceIds: ["EV-03"] },
    { id: "MAC-S02", input: "Brent 上涨且近月价差扩大", mechanism: "价格变化可能包含地缘、需求与仓位因素，成因尚未拆分", output: "原油采购价格存在上行压力", type: "Observation → Inference", confidence: "中", evidenceIds: ["EV-04"] },
    { id: "MAC-S03", input: "运费与采购价格同时上行", mechanism: "两类成本共同构成到岸成本", output: "进口原油到岸成本上行", type: "Synthesis", confidence: "中", evidenceIds: ["EV-03", "EV-04"] },
    { id: "MAC-S04", input: "炼厂库存 24.6 天", mechanism: "库存可能提供短期缓冲", output: "即时实物短缺判断被弱化或延后（推断）", type: "Counterevidence gate", confidence: "中", evidenceIds: ["EV-05"] },
    { id: "MAC-S05", input: "到岸成本可能上行", mechanism: "能源和化工成本进入生产资料价格", output: "PPI 存在上行压力", type: "Inference", confidence: "低–中", evidenceIds: ["EV-03", "EV-04"] },
  ],
  counterevidence: "库存缓冲意味着成本上行不必然立即转化为实物短缺或全面通胀。",
  evidenceGap: "缺少实际进口结算价、汇率变化、税费和正式 PPI 分项数据。",
  reversal: "若 Brent 价格、近月价差与运价快速回落，或汇率升值抵消成本，则下调 PPI 压力判断。",
};

const industryChains: IndustryChain[] = [
  {
    id: "CHN-01",
    title: "原油进口与炼化成本链",
    kind: "主链",
    conclusion: "保险报价和 VLCC 样本运价支持运输摩擦成本上升，但实际原油到岸成本与炼化价差仍未被观察，只能发布方向性成本压力结论。",
    state: "分化",
    status: "方向受支持、结果待验证",
    horizon: "即时–3周",
    confidence: "中",
    evidenceIds: ["EV-01", "EV-02", "EV-03", "EV-04"],
    graphGroups: [["N-01"], ["N-02", "N-03"], ["N-04"], ["N-05"]],
    graphLinks: ["风险定价 / 基准", "共同进入成本", "价差待补齐"],
    stopRule: "若保险报价、Brent 与运价同步回落，或实际进口结算价未上升，则关闭或降级该链；若炼化价差稳定或扩大，则停止“炼化利润承压”外推。",
    nodes: [
      {
        id: "N-01",
        title: "波斯湾—东亚航运风险",
        status: "综合判断",
        tone: "observed",
        result: "升温",
        lag: "短期",
        confidence: "中",
        impact: "重点航道风险等级上调，但尚无断供事实",
        why: "风险通告与保险报价同步变化，说明通道风险已经进入运输定价。",
        evidence: ["EV-01", "EV-02"],
      },
      {
        id: "N-02",
        title: "保险与 VLCC 即期运费",
        status: "已观测",
        tone: "observed",
        result: "升温",
        lag: "短期",
        confidence: "中",
        impact: "保险报价一周上升 32%，样本航线运价上升 14%",
        why: "两项运输成本代理指标同步上行，显示航运摩擦成本正在增强。",
        evidence: ["EV-02", "EV-03"],
      },
      {
        id: "N-03",
        title: "Brent 定价基准",
        status: "已观测",
        tone: "observed",
        result: "升温",
        lag: "短期",
        confidence: "中",
        impact: "Brent 五日上涨 5.6%，近月价差扩大",
        why: "原油定价基准走强，但地缘、需求和仓位的贡献尚未拆分。",
        evidence: ["EV-04"],
      },
      {
        id: "N-04",
        title: "原油到岸成本",
        status: "方向推断",
        tone: "inferred",
        result: "升温",
        lag: "短期",
        confidence: "中",
        impact: "存在上行压力，实际进口结算价尚未取得",
        why: "运费与 Brent 同时上行会共同进入进口成本，但缺少实际到岸数据验证。",
        evidence: ["EV-03", "EV-04"],
      },
      {
        id: "N-05",
        title: "炼化价差",
        status: "待验证",
        tone: "gap",
        result: "待判定",
        lag: "短期",
        confidence: "低",
        impact: "存在收窄假设，实际炼化价差尚未观测",
        why: "只有原油成本上涨快于成品价格时炼化利润才会承压，当前缺少产品端价差。",
        evidence: [],
      },
    ],
  },
  {
    id: "CHN-02",
    title: "石脑油—乙烯裂解链",
    kind: "主链",
    conclusion: "石脑油报价上涨已经观察到，但其上游成因与乙烯裂解利润变化尚未闭合，本链只能发布部分观测、条件性传导结论。",
    state: "分化",
    status: "前端报价已观测，因果端与利润端未闭合",
    horizon: "1–3周",
    confidence: "低–中",
    evidenceIds: ["EV-06"],
    graphGroups: [["N-04"], ["N-06"], ["N-07"], ["N-08"]],
    graphLinks: ["因果待验证", "缺乙烯售价", "价差待计算"],
    stopRule: "若未来两周石脑油未持续跟随原油和运费方向，或实际乙烯—石脑油价差稳定或扩大，则停止从上游成本向乙烯利润外推。",
    nodes: [
      {
        id: "N-04",
        title: "原油到岸成本",
        status: "方向推断",
        tone: "inferred",
        result: "升温",
        lag: "短期",
        confidence: "中",
        impact: "作为本链上游输入，当前存在上行压力",
        why: "运费与 Brent 同时上行会共同进入进口成本，但缺少实际到岸数据验证。",
        evidence: ["EV-03", "EV-04"],
      },
      {
        id: "N-06",
        title: "石脑油",
        status: "已观测",
        tone: "observed",
        result: "升温",
        lag: "短期",
        confidence: "中",
        impact: "报价上涨 3.1%，上涨成因尚未闭合",
        why: "现货价格已经上行，但尚不能确认完全由原油到岸成本推动。",
        evidence: ["EV-06"],
      },
      {
        id: "N-07",
        title: "乙烯产品价格",
        status: "未观测",
        tone: "gap",
        result: "待判定",
        lag: "短期",
        confidence: "低",
        impact: "尚无乙烯售价观察",
        why: "缺少乙烯价格，无法判断其是否跟随石脑油上涨并覆盖原料成本。",
        evidence: [],
      },
      {
        id: "N-08",
        title: "乙烯—石脑油裂解价差",
        status: "待验证",
        tone: "gap",
        result: "待判定",
        lag: "短期",
        confidence: "低",
        impact: "存在收窄与利润承压假设，实际价差尚未取得",
        why: "石脑油已经上涨；只有乙烯售价未同步上升时，裂解价差才会收窄。",
        evidence: [],
      },
    ],
  },
  {
    id: "CHN-03",
    title: "乙烯—PE—包装材料传价链",
    kind: "主链",
    conclusion: "当前只观察到 PE 价格小幅上涨，无法确认其由乙烯成本驱动，也无法确认成本已经传递至包装材料或下游毛利，因此本链尚未闭合。",
    state: "待验证",
    status: "单节点观测、整链未闭合",
    horizon: "2–6周",
    confidence: "低",
    evidenceIds: ["EV-06"],
    graphGroups: [["N-07"], ["N-09"], ["N-10"]],
    graphLinks: ["成因待验证", "合同与需求待补齐"],
    stopRule: "若需求走弱、社会库存上升，或 PE 提价不足以覆盖成本，则传导在 PE 节点停止。",
    nodes: [
      {
        id: "N-07",
        title: "乙烯产品价格",
        status: "未观测",
        tone: "gap",
        result: "待判定",
        lag: "短期",
        confidence: "低",
        impact: "作为 PE 原料成本输入，当前尚无价格观察",
        why: "缺少乙烯价格与裂解价差，无法确认 PE 上涨的成本成因。",
        evidence: [],
      },
      {
        id: "N-09",
        title: "PE",
        status: "已观测",
        tone: "observed",
        result: "升温",
        lag: "中期",
        confidence: "中",
        impact: "现货上涨 0.6%，但成交、库存与传价成因未验证",
        why: "PE 报价已经小幅上行，但涨幅明显弱于石脑油，显示传导可能不完整。",
        evidence: ["EV-06"],
      },
      {
        id: "N-10",
        title: "包装材料",
        status: "待验证",
        tone: "gap",
        result: "待判定",
        lag: "中期",
        confidence: "低",
        impact: "尚无采购成本、产品提价、订单和毛利观察",
        why: "缺少合同与经营数据，无法判断 PE 成本是否已经传递到包装材料。",
        evidence: [],
      },
    ],
  },
  {
    id: "BUF-01",
    title: "炼厂库存潜在缓冲分支",
    kind: "缓冲分支",
    conclusion: "沿海炼厂库存高于近期中位数，对即时短缺和成本快速贯穿全链构成反证，但单点库存不能证明库存已经实际吸收扰动或形成持续缓冲。",
    state: "降温",
    status: "反证性观察存在、缓冲机制待验证",
    horizon: "即时–3周",
    confidence: "低–中",
    evidenceIds: ["EV-05"],
    graphGroups: [["N-13"], ["N-14"], ["N-15"], ["N-04"], ["N-06"]],
    graphLinks: ["潜在缓冲", "需开工验证", "只影响时点", "可能延后"],
    stopRule: "若库存持续下降并跌破有 Evidence 支持的阈值，或运输扰动持续超过库存可覆盖周期，则该分支失效，并重新评估 CHN-01 与 CHN-02 的时滞和置信度。",
    nodes: [
      {
        id: "N-13",
        title: "沿海炼厂库存",
        status: "已观测",
        tone: "observed",
        result: "降温",
        lag: "短期",
        confidence: "中",
        impact: "库存为 24.6 天，高于过去 30 日库存天数中位数",
        why: "较高库存对“即时原料短缺”构成反证，但具体中位数和覆盖范围仍待补齐。",
        evidence: ["EV-05"],
      },
      {
        id: "N-14",
        title: "炼厂采购与补库节奏",
        status: "未观测",
        tone: "gap",
        result: "待判定",
        lag: "短期",
        confidence: "低",
        impact: "较高库存可能延后即时补库需求，但没有连续采购观察",
        why: "库存可能提供采购时间缓冲，但当前没有连续采购与补库数据验证。",
        evidence: [],
      },
      {
        id: "N-15",
        title: "炼厂原料保障与开工",
        status: "待验证",
        tone: "gap",
        result: "待判定",
        lag: "短期",
        confidence: "低",
        impact: "可能维持短期原料保障，实际开工和消耗速度未知",
        why: "库存可能暂时支持生产，但缺少开工率与日耗数据，持续性无法判断。",
        evidence: [],
      },
      {
        id: "N-04",
        title: "原油到岸成本",
        status: "可能延后",
        tone: "inferred",
        result: "分化",
        lag: "短期",
        confidence: "低",
        impact: "成本方向承压，但影响显现时点可能被库存延后",
        why: "运费和 Brent 推动成本升温，库存只可能延后影响时点，不能降低单位到岸成本。",
        evidence: ["EV-03", "EV-04", "EV-05"],
      },
      {
        id: "N-06",
        title: "石脑油",
        status: "可能延后",
        tone: "inferred",
        result: "分化",
        lag: "短期",
        confidence: "低",
        impact: "报价已经上涨，但库存缓冲可能延后新增成本进入采购价",
        why: "现货价格呈升温信号，而库存对新增采购形成潜在降温作用，两种力量尚未闭合。",
        evidence: ["EV-05", "EV-06"],
      },
    ],
  },
];

const industryLayer: Layer = {
  key: "industry",
  label: "产业链",
  shortLabel: "产业链",
  eyebrow: "石化传导",
  conclusion: "本次形成三条主产业链结论与一条库存缓冲分支；各链分别发布，不合并成单一产业层结论。",
  state: "分化",
  horizon: "短期–中期",
  confidence: "低–中置信",
  evidenceIds: ["EV-01", "EV-02", "EV-03", "EV-04", "EV-05", "EV-06"],
  anchors: [
    { id: "IND-A01", label: "原油到岸成本", state: "升温" },
    { id: "IND-A02", label: "石脑油", state: "升温" },
    { id: "IND-A03", label: "乙烯裂解价差", state: "待验证" },
    { id: "IND-A04", label: "PE", state: "升温" },
    { id: "IND-A05", label: "包装材料", state: "待验证" },
  ],
};

const companyLayer: Layer = {
  key: "company",
  label: "公司",
  shortLabel: "企业",
  eyebrow: "等待验证",
  conclusion: "本次模拟推理尚未进入公司层；只有企业成本暴露、库存和合同传价条款齐全后，才允许生成公司结论。",
  state: "待验证",
  horizon: "未进入",
  confidence: "不评级",
  evidenceIds: [],
  anchors: [
    { id: "COM-A01", label: "原料成本暴露", state: "待验证" },
    { id: "COM-A02", label: "库存天数", state: "待验证" },
    { id: "COM-A03", label: "长协定价", state: "待验证" },
    { id: "COM-A04", label: "产品提价", state: "待验证" },
  ],
};

const layers: Layer[] = [geoLayer, macroLayer, industryLayer, companyLayer];

const mockReport: ReasoningReport = {
  id: "R-0831",
  title: "海湾航运风险如何进入石化产业链",
  publishedAt: "2026.08.31 08:30",
  updatedAt: "08:30",
  evidenceCount: evidenceItems.length,
  geo: geoLayer,
  macro: macroLayer,
  industryChains,
};

const reasoningReports: ReasoningReport[] = [mockReport];
const chainNodes: ChainNode[] = industryChains[0].nodes;

const sourceGeoLayer = generatedReportData.geo as unknown as Layer;
const sourceMacroLayer = generatedReportData.macro as unknown as Layer;
const sourceIndustryChains = generatedReportData.industryChains as unknown as IndustryChain[];
const sourceEvidenceItems = generatedReportData.evidenceItems as unknown as Evidence[];
const sourceIndustryLayer: Layer = {
  key: "industry",
  label: "产业链",
  shortLabel: "产业链",
  eyebrow: "真实节点与动态传导",
  conclusion: "本层由 43 个真实 ChainNode 的 85 条直接 Signal 聚合出 54 条真实 IndustryChain，并接受 29 条动态传导假设。",
  state: "分化",
  horizon: "按链发布",
  confidence: "按链发布",
  evidenceIds: [...new Set(sourceIndustryChains.flatMap((chain) => chain.evidenceIds))],
  anchors: [],
};
const sourceCompanyLayer: Layer = {
  key: "company",
  label: "公司",
  shortLabel: "企业",
  eyebrow: "本次未发布",
  conclusion: "本报告不分析公司层。",
  state: "待验证",
  horizon: "未进入",
  confidence: "不评级",
  evidenceIds: [],
  anchors: [],
};
const sourceLayers: Layer[] = [sourceGeoLayer, sourceMacroLayer, sourceIndustryLayer, sourceCompanyLayer];
const sourceReport: ReasoningReport = {
  id: generatedReportData.id,
  title: generatedReportData.title,
  publishedAt: generatedReportData.publishedAt.replaceAll("-", "."),
  updatedAt: generatedReportData.publishedAt.slice(11),
  evidenceCount: sourceEvidenceItems.length,
  eventCount: generatedReportData.eventCount,
  chainCount: generatedReportData.chainCount,
  hypothesisCount: generatedReportData.hypothesisCount,
  geo: sourceGeoLayer,
  macro: sourceMacroLayer,
  industryChains: sourceIndustryChains,
};
const homeIndustryChains = sourceIndustryChains.filter((chain) => ["CHN-01", "CHN-02", "CHN-03", "CHN-21"].includes(chain.id));

function layerByKey(key: LayerKey) {
  return sourceLayers.find((layer) => layer.key === key) ?? sourceLayers[0];
}

function LayerGlyph({ layer }: { layer: LayerKey }) {
  if (layer === "geo") return <GlobeIcon />;
  if (layer === "macro") return <BarChartIcon />;
  if (layer === "industry") return <LayersIcon />;
  return <CubeIcon />;
}

function ReportHero() {
  return (
    <header className="report-hero">
      <div className="brand-row">
        <span className="brand-mark brand-mark-text">潮</span>
        <span className="brand-name home-brand-name">观潮</span>
        <span className="prototype-badge">仿真原型</span>
      </div>
      <div className="home-search-shell" aria-label="搜索入口展示">
        <ReaderIcon />
        <span>搜索事件、产业，或直接向问潮提问</span>
        <span className="home-search-action"><ArrowRightIcon /></span>
      </div>
    </header>
  );
}

function LayerTabs({ active, onChange }: { active: LayerKey; onChange: (layer: LayerKey) => void }) {
  return (
    <div className="layer-tabs-wrap">
      <div className="layer-tabs" role="tablist" aria-label="报告层级">
        {layers.map((layer) => (
          <button
            key={layer.key}
            id={`layer-tab-${layer.key}`}
            type="button"
            role="tab"
            aria-controls={`layer-panel-${layer.key}`}
            aria-selected={active === layer.key}
            tabIndex={active === layer.key ? 0 : -1}
            className={active === layer.key ? "is-active" : ""}
            onClick={() => onChange(layer.key)}
          >
            <LayerGlyph layer={layer.key} /><span>{layer.shortLabel}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

function ImpactPill({ state, compact = false }: { state: ImpactState; compact?: boolean }) {
  return <span className={compact ? "impact-pill is-compact" : "impact-pill"} data-state={state}>{state}</span>;
}

function nodeImpactState(node: ChainNode): ImpactState {
  return node.result === "待判定" ? "待验证" : node.result;
}

function NodeNatureIcon({ tone }: { tone: ChainNode["tone"] }) {
  if (tone === "observed") return <FileTextIcon />;
  if (tone === "inferred") return <Link2Icon />;
  return <InfoCircledIcon />;
}

function NodeSignalTags({ node, compact = false }: { node: ChainNode; compact?: boolean }) {
  return (
    <div className={`node-signal-tags ${compact ? "is-compact" : ""}`}>
      <ImpactPill state={nodeImpactState(node)} compact />
      <span className="node-signal-tag is-confidence"><CheckCircledIcon />{node.confidence}</span>
      <span className="node-signal-tag is-window"><ClockIcon />{node.lag}</span>
      <span className={`node-signal-tag is-nature tone-${node.tone}`}><NodeNatureIcon tone={node.tone} />{node.status}</span>
    </div>
  );
}

function impactStateFromText(value: string): ImpactState {
  if (value.includes("升温")) return "升温";
  if (value.includes("降温")) return "降温";
  if (value.includes("分化")) return "分化";
  return "待验证";
}

function AnchorChips({ anchors, compact = false }: { anchors: AnchorImpact[]; compact?: boolean }) {
  return (
    <div className={compact ? "anchor-chips is-compact" : "anchor-chips"}>
      {anchors.map((anchor) => (
        <span key={anchor.id} className="anchor-impact-chip">
          <b>{anchor.label}</b>
          <ImpactPill state={anchor.state} compact />
        </span>
      ))}
    </div>
  );
}

function EvidenceEntry({ onClick }: { onClick: () => void }) {
  return (
    <button className="evidence-icon-button" type="button" onClick={onClick} aria-label="查看证据">
      <FileTextIcon />
    </button>
  );
}

function DirectEvidenceList({ evidenceIds }: { evidenceIds: string[] }) {
  const items = evidenceIds
    .map((id) => sourceEvidenceItems.find((item) => item.id === id))
    .filter((item): item is Evidence => Boolean(item))
    .sort((left, right) => right.publishedAt.localeCompare(left.publishedAt));
  return (
    <div className="direct-evidence-list">
      {items.map((item) => (
        <div className="direct-evidence-timeline-item" key={item.id}>
          <article className="direct-evidence-item">
            <time dateTime={item.publishedAtRaw ?? item.publishedAt}>
              <ClockIcon />
              <span>{item.publishedAt}</span>
            </time>
            <p>{item.summary}</p>
            {item.keywords.length ? (
              <div className="direct-evidence-keywords" role="list" aria-label="关键词">
                {item.keywords.map((keyword) => <span role="listitem" key={keyword}>{keyword}</span>)}
              </div>
            ) : null}
          </article>
        </div>
      ))}
    </div>
  );
}

function PrimaryAction({ label, onClick }: { label: string; onClick: () => void }) {
  return <button className="primary-action" type="button" onClick={onClick}><span>{label}</span><ArrowRightIcon /></button>;
}

function EmptyCompany() {
  return (
    <section className="empty-company" aria-labelledby="company-empty-title">
      <div className="empty-icon"><CubeIcon /></div>
      <span className="section-kicker">公司层 · 模拟占位</span>
      <h2 id="company-empty-title">本次推理未进入公司层</h2>
      <p>为避免从行业标签直接映射公司，需先补齐企业经营变量，再生成公司影响结论。</p>
      <div className="readiness-list">
        {["原料采购与库存天数", "产品价差与提价记录", "长协定价和运费条款", "订单与产能利用率"].map((item) => (
          <div key={item}><InfoCircledIcon /><span>{item}</span><small>待补齐</small></div>
        ))}
      </div>
      <div className="empty-boundary"><ExclamationTriangleIcon /> 当前页面不生成模拟公司或股票结论</div>
    </section>
  );
}

function RelatedChainTags({ chains, onEvidence }: { chains: IndustryChain[]; onEvidence: (state: EvidenceSheetState) => void }) {
  return (
    <div className="related-chain-tags" aria-label="相关产业链与状态">
      {chains.map((chain) => (
        <div className="related-chain-tag" key={chain.id}>
          <span><b>{chain.title}</b></span>
          <ImpactPill state={chain.state} compact />
          <button type="button" className="evidence-icon-button" aria-label={`查看${chain.title}的证据`} onClick={() => onEvidence({ evidenceIds: chain.evidenceIds })}>
            <FileTextIcon />
          </button>
        </div>
      ))}
    </div>
  );
}

function TransmissionBridge({ label }: { label: string }) {
  return (
    <div className="transmission-bridge" aria-label={label}>
      <span><ChevronDownIcon /></span><strong>{label}</strong>
    </div>
  );
}

function StageEvidence({ onClick }: { onClick: () => void }) {
  return <div className="stage-evidence"><EvidenceEntry onClick={onClick} /></div>;
}

function GeoReportCard({ report, onOpen, onEvidence }: { report: ReasoningReport; onOpen: () => void; onEvidence: (state: EvidenceSheetState) => void }) {
  return (
    <article className="layer-report-card hierarchy-report-card geo-report-card">
      <div className="card-conclusion"><p>{report.geo.conclusion}</p></div>
      <div className="transmission-stack">
        <section className="transmission-stage is-origin">
          <AnchorChips anchors={report.geo.anchors} compact />
          <StageEvidence onClick={() => onEvidence({ evidenceIds: report.geo.evidenceIds })} />
        </section>
        <TransmissionBridge label="推导至宏观经济" />
        <section className="transmission-stage">
          <p className="stage-conclusion">{report.macro.conclusion}</p>
          <AnchorChips anchors={report.macro.anchors} compact />
          <StageEvidence onClick={() => onEvidence({ evidenceIds: report.macro.evidenceIds })} />
        </section>
        <TransmissionBridge label="传导至产业链" />
        <section className="transmission-stage">
          <RelatedChainTags chains={report.industryChains} onEvidence={onEvidence} />
        </section>
      </div>
      <div className="card-action-row hierarchy-action-row"><PrimaryAction label="推导逻辑" onClick={onOpen} /></div>
    </article>
  );
}

function MacroReportCard({ report, onOpen, onEvidence }: { report: ReasoningReport; onOpen: () => void; onEvidence: (state: EvidenceSheetState) => void }) {
  return (
    <article className="layer-report-card hierarchy-report-card macro-report-card">
      <div className="card-conclusion"><p>{report.macro.conclusion}</p></div>
      <div className="transmission-stack">
        <section className="transmission-stage is-origin">
          <AnchorChips anchors={report.macro.anchors} compact />
          <StageEvidence onClick={() => onEvidence({ evidenceIds: report.macro.evidenceIds })} />
        </section>
        <TransmissionBridge label="传导至产业链" />
        <section className="transmission-stage">
          <RelatedChainTags chains={report.industryChains} onEvidence={onEvidence} />
        </section>
      </div>
      <div className="card-action-row hierarchy-action-row"><PrimaryAction label="推导逻辑" onClick={onOpen} /></div>
    </article>
  );
}

function IndustryReportCard({ chain, onOpen, onEvidence }: { chain: IndustryChain; onOpen: () => void; onEvidence: (state: EvidenceSheetState) => void }) {
  return (
    <article className="industry-report-card">
      <div className="card-conclusion industry-chain-conclusion"><p>{chain.conclusion}</p></div>
      <div className="industry-chain-anchor">
        <div><small>{chain.kind}</small><strong>{chain.title}</strong></div>
        <ImpactPill state={chain.state} compact />
      </div>
      <TransmissionBridge label="影响到具体节点" />
      <ol className="industry-node-list">
        {chain.nodes.map((node, index) => (
          <li key={`${chain.id}-${node.id}`}>
            <span className="industry-node-order">{String(index + 1).padStart(2, "0")}</span>
            <strong>{node.title}</strong>
            <ImpactPill state={node.result === "待判定" ? "待验证" : node.result} compact />
          </li>
        ))}
      </ol>
      <StageEvidence onClick={() => onEvidence({ evidenceIds: chain.evidenceIds })} />
      <div className="industry-card-footer"><PrimaryAction label="推导逻辑" onClick={onOpen} /></div>
    </article>
  );
}

function VariantA({ layer, onOpen, onEvidence }: { layer: Layer; onOpen: (layer: LayerKey, chainId?: string) => void; onEvidence: (state: EvidenceSheetState) => void }) {
  return (
    <div className="variant-content variant-a">
      {layer.key === "geo" ? reasoningReports.map((report) => <GeoReportCard key={report.id} report={report} onOpen={() => onOpen("geo")} onEvidence={onEvidence} />) : null}
      {layer.key === "macro" ? reasoningReports.map((report) => <MacroReportCard key={report.id} report={report} onOpen={() => onOpen("macro")} onEvidence={onEvidence} />) : null}
      {layer.key === "industry" ? mockReport.industryChains.map((chain) => <IndustryReportCard key={chain.id} chain={chain} onOpen={() => onOpen("industry", chain.id)} onEvidence={onEvidence} />) : null}
    </div>
  );
}

function VariantB({ activeLayer, onSelect, onOpen }: { activeLayer: Layer; onSelect: (layer: LayerKey) => void; onOpen: () => void }) {
  return (
    <div className="variant-content variant-b">
      <div className="section-heading"><div><span>方案 B · 报告纵览</span><h2>一次推理的完整结构</h2></div><small>模拟编号 R-0831</small></div>
      <article className="report-overview-card">
        <div className="overview-title"><span className="overview-icon"><ReaderIcon /></span><div><small>模拟主题</small><h3>海湾航运风险如何进入石化链</h3></div></div>
        <div className="overview-facts"><span>仿真事件 7</span><span>仿真 Evidence 6</span><span>检查点 3</span></div>
        <div className="impact-ladder" aria-label="报告层级纵览">
          {layers.map((item, index) => (
            <button key={item.key} type="button" className={activeLayer.key === item.key ? "is-active" : ""} onClick={() => onSelect(item.key)}>
              <span className="ladder-index">0{index + 1}</span>
              <span className="ladder-icon"><LayerGlyph layer={item.key} /></span>
              <span className="ladder-copy"><strong>{item.label}</strong><small>{item.key === "company" ? "本次未进入" : item.eyebrow}</small></span>
              {item.key === "company" ? <InfoCircledIcon /> : <CheckCircledIcon />}
            </button>
          ))}
        </div>
      </article>
      {activeLayer.key === "company" ? <EmptyCompany /> : (
        <article className="overview-focus-card">
          <div className="focus-heading"><span><LayerGlyph layer={activeLayer.key} /> 当前焦点</span><small>{activeLayer.confidence}</small></div>
          <h3>一句话结论 · 仿真</h3><p>{activeLayer.conclusion}</p>
          <AnchorChips anchors={activeLayer.anchors.slice(0, 3)} compact />
          <PrimaryAction label="进入本层推导" onClick={onOpen} />
        </article>
      )}
    </div>
  );
}

function VariantC({ layer, onOpen }: { layer: Layer; onOpen: () => void }) {
  const [selectedAnchorId, setSelectedAnchorId] = useState(layer.anchors[0].id);
  const anchorIndex = Math.max(0, layer.anchors.findIndex((anchor) => anchor.id === selectedAnchorId));
  const selectedAnchor = layer.anchors[anchorIndex];
  return (
    <div className="variant-content variant-c" key={layer.key}>
      <div className="section-heading"><div><span>方案 C · 锚点优先</span><h2>先看受影响的关键对象</h2></div><small>{layer.label} · 仿真</small></div>
      <Carousel className="anchor-carousel" contentClassName="anchor-carousel-track" ariaLabel="模拟影响锚点">
        {layer.anchors.map((anchor, index) => (
          <button key={anchor.id} type="button" aria-pressed={selectedAnchorId === anchor.id} className={selectedAnchorId === anchor.id ? "anchor-tile is-active" : "anchor-tile"} onClick={() => setSelectedAnchorId(anchor.id)}>
            <span><SewingPinIcon /> 锚点 0{index + 1}</span><strong>{anchor.label}</strong><ImpactPill state={anchor.state} compact />
          </button>
        ))}
      </Carousel>
      <article className="anchor-focus-card">
        <div className="anchor-focus-top">
          <span className="anchor-focus-icon"><DashboardIcon /></span>
          <div><small>当前锚点 · 仿真</small><h3>{selectedAnchor.label}</h3></div>
          <span className="confidence-tag">{anchorIndex < 2 ? "中置信" : "待验证"}</span>
        </div>
        <div className="anchor-signal-grid"><div><small>影响结果</small><strong>{selectedAnchor.state}</strong></div><div><small>影响窗口</small><strong>{layer.horizon}</strong></div><div><small>仿真 Evidence</small><strong>{Math.max(1, 4 - anchorIndex)}</strong></div></div>
        <p>{anchorIndex === 0 ? layer.conclusion : `该锚点位于“${layer.eyebrow}”路径中，当前仅展示与它直接相关的模拟影响，不自动延伸到公司结论。`}</p>
        <button type="button" className="text-action" onClick={onOpen}>在推理树中定位 <ArrowRightIcon /></button>
      </article>
      <article className="related-report-strip"><div><FileTextIcon /><span><small>来自模拟报告</small><strong>海湾航运风险如何进入石化链</strong></span></div></article>
    </div>
  );
}

function ReportCardActions({ onEvidence, onOpen }: { onEvidence: () => void; onOpen: () => void }) {
  return (
    <div className="report-card-actions">
      <button type="button" className="evidence-text-action" onClick={onEvidence}><FileTextIcon /><span>依据</span></button>
      <PrimaryAction label="看传导" onClick={onOpen} />
    </div>
  );
}

function HomeImpactSignals({ state, confidence, window }: { state: ImpactState; confidence: string; window: string }) {
  return (
    <div className="home-impact-signals" aria-label={`结果${state}，置信度${confidence}，时间窗口${window}`}>
      <span className="home-impact-signal is-result" data-state={state}><ActivityLogIcon /><b>{state}</b></span>
      <span className="home-impact-signal is-confidence"><CheckCircledIcon /><span>置信</span><b>{confidence}</b></span>
      <span className="home-impact-signal is-window"><ClockIcon /><b>{window}</b></span>
    </div>
  );
}

function HomeAnchorGrid({ anchors }: { anchors: AnchorImpact[] }) {
  return (
    <div className="home-anchor-grid" aria-label="受影响锚点">
      {anchors.map((anchor) => (
        <div className="home-anchor-item" key={anchor.id}>
          <strong>{anchor.label}</strong>
          <HomeImpactSignals state={anchor.state} confidence={anchor.confidence ?? "—"} window={anchor.window ?? "—"} />
        </div>
      ))}
    </div>
  );
}

function LayerHomeCard({ layer, onEvidence, onOpen }: { layer: Layer; onEvidence: () => void; onOpen: () => void }) {
  return (
    <article className={`report-home-card report-home-card-${layer.key}`}>
      <p className="home-card-conclusion">{layer.conclusion}</p>
      <HomeAnchorGrid anchors={layer.anchors} />
      <ReportCardActions onEvidence={onEvidence} onOpen={onOpen} />
    </article>
  );
}

function IndustryHomeCard({ chain, onEvidence, onOpen }: { chain: IndustryChain; onEvidence: () => void; onOpen: () => void }) {
  return (
    <article className="report-home-card report-home-card-industry">
      <p className="home-card-conclusion">{chain.conclusion}</p>
      <div className="industry-home-identity">
        <span>{chain.kind}</span>
        <strong>{chain.title}</strong>
      </div>
      <div className="home-anchor-grid home-node-grid" aria-label={`${chain.title}受影响节点`}>
        {chain.nodes.map((node) => (
          <div className="home-anchor-item" key={`${chain.id}-${node.id}`}>
            <strong>{node.title}</strong>
            <HomeImpactSignals state={node.result === "待判定" ? "待验证" : node.result} confidence={node.confidence} window={node.lag} />
          </div>
        ))}
      </div>
      <ReportCardActions onEvidence={onEvidence} onOpen={onOpen} />
    </article>
  );
}

function ReportColumnHeading({ layer, subtitle }: { layer: LayerKey; subtitle: string }) {
  const item = layerByKey(layer);
  return (
    <header className="report-column-heading">
      <span className="report-column-icon"><LayerGlyph layer={layer} /></span>
      <div><h2>{layer === "company" ? item.shortLabel : item.label}</h2><p>{subtitle}</p></div>
    </header>
  );
}

function ReportBundle({ onOpen, onEvidence }: { onOpen: (layer: LayerKey, chainId?: string) => void; onEvidence: (state: EvidenceSheetState) => void }) {
  return (
    <section className="report-bundle" aria-label="本期观潮报告">
      <header className="report-bundle-header">
        <div className="report-publish-row"><ClockIcon /><span>发布时间</span><time dateTime={generatedReportData.generatedAt}>{sourceReport.publishedAt}</time></div>
      </header>

      <div className="report-column-flow">
        <section className="report-column" aria-labelledby="home-geo-heading">
          <ReportColumnHeading layer="geo" subtitle="安全对抗与通道可用性" />
          <div id="home-geo-heading"><LayerHomeCard layer={sourceReport.geo} onEvidence={() => onEvidence({ evidenceIds: sourceReport.geo.evidenceIds })} onOpen={() => onOpen("geo")} /></div>
        </section>

        <section className="report-column" aria-labelledby="home-macro-heading">
          <ReportColumnHeading layer="macro" subtitle="增长预期与政策利率" />
          <div id="home-macro-heading"><LayerHomeCard layer={sourceReport.macro} onEvidence={() => onEvidence({ evidenceIds: sourceReport.macro.evidenceIds })} onOpen={() => onOpen("macro")} /></div>
        </section>

        <section className="report-column" aria-labelledby="home-industry-heading">
          <ReportColumnHeading layer="industry" subtitle="54 条真实产业链 · 首页展示 4 条" />
          <div id="home-industry-heading" className="industry-home-list">
            {homeIndustryChains.map((chain) => (
              <IndustryHomeCard key={chain.id} chain={chain} onEvidence={() => onEvidence({ evidenceIds: chain.evidenceIds })} onOpen={() => onOpen("industry", chain.id)} />
            ))}
          </div>
        </section>

        <section className="report-column company-report-column" aria-labelledby="home-company-heading">
          <ReportColumnHeading layer="company" subtitle="本次报告未发布企业层结论" />
          <article id="home-company-heading" className="company-report-boundary">
            <InfoCircledIcon />
            <div><strong>本次推理尚未进入企业层</strong><p>当前报告发布范围止于产业链，不生成企业影响结论。</p></div>
          </article>
        </section>
      </div>
    </section>
  );
}

function FeedScreen({ flow, variant }: { flow: FlowControls; variant: Variant }) {
  const [evidenceSheet, setEvidenceSheet] = useState<EvidenceSheetState | null>(null);
  const openDetail = (layerKey: LayerKey, chainId?: string) => {
    if (layerKey !== "company") flow.push(createDetailScreen(layerKey, chainId));
  };
  return (
    <>
      <MobileScroll className="app-screen">
        <main className="screen-content feed-content">
          <ReportHero />
          <section className="today-reasoning-head" aria-labelledby="today-reasoning-title">
            <div><h1 id="today-reasoning-title">今日观潮</h1></div>
          </section>
          <ReportBundle onOpen={openDetail} onEvidence={setEvidenceSheet} />
          <div className="prototype-disclaimer">页面内容来自本期已完成报告；直接证据、推理假设与待验证项分开呈现。</div>
        </main>
      </MobileScroll>
      <BottomSheet
        open={Boolean(evidenceSheet)}
        onOpenChange={(open) => { if (!open) setEvidenceSheet(null); }}
        title="证据列表"
        snap={0.76}
      >
        {evidenceSheet ? <DirectEvidenceList evidenceIds={evidenceSheet.evidenceIds} /> : null}
      </BottomSheet>
    </>
  );
}

function DetailHeader({ flow, layer }: { flow: FlowControls; layer: Layer }) {
  return (
    <div className="tw-detail-header">
      <button type="button" onClick={flow.pop} aria-label="返回上一页"><ArrowLeftIcon /></button>
      <div><strong>传导详情</strong><small>从{layer.shortLabel}层开始</small></div>
      <span className="detail-header-mark"><ActivityLogIcon /></span>
    </div>
  );
}

function DetailAnchorGrid({ layer, onEvidence }: { layer: Layer; onEvidence: (state: EvidenceSheetState) => void }) {
  return (
    <div className="detail-anchor-grid" aria-label={`${layer.label}受影响锚点`}>
      {layer.anchors.map((anchor) => (
        <article className="detail-anchor-card" data-state={anchor.state} key={anchor.id}>
          <div className="detail-anchor-title">
            <strong>{anchor.label}</strong>
            {anchor.nature ? <span className="anchor-nature-tag" data-nature={anchor.nature}>{anchor.nature}</span> : null}
          </div>
          <HomeImpactSignals state={anchor.state} confidence={anchor.confidence ?? "—"} window={anchor.window ?? "—"} />
          <p>{anchor.current}</p>
          <div className="detail-anchor-why"><span>为什么</span><p>{anchor.why}</p></div>
          {anchor.evidenceIds?.length ? (
            <button type="button" className="anchor-evidence-link" onClick={() => onEvidence({ evidenceIds: anchor.evidenceIds ?? [] })}>
              <FileTextIcon /><span>依据</span>
            </button>
          ) : null}
        </article>
      ))}
    </div>
  );
}

function LayerTransmissionBlock({ layer }: { layer: Layer }) {
  const items = layer.transmissionPaths ?? [];
  if (!items.length) return null;
  return (
    <section className="layer-transmission-block" aria-label={`${layer.label}向下传导`}>
      <div className="layer-subheading"><strong>向下传导</strong></div>
      <div className="layer-transmission-list">
        {items.map((item) => {
          const chainId = item.targetLabel.match(/CHN-\d+/)?.[0];
          const chain = chainId ? sourceIndustryChains.find((candidate) => candidate.id === chainId) : undefined;
          const result = chain?.state ?? (item.result.includes("待补证") ? "待补证" : item.result);
          const state = impactStateFromText(result);
          return (
            <article key={item.id} data-state={state}>
              <div className="transmission-target"><Link2Icon /><span>传到{item.targetLayer === "macro" ? "宏观经济" : "产业链"}</span><ImpactPill state={state} compact /></div>
              <strong>{chain?.title ?? item.targetLabel}</strong>
              <span className="transmission-nature">{item.nature}</span>
              <p>{item.logic}</p>
              {item.status ? <div className="transmission-boundary"><span>边界</span><p>{item.status}</p></div> : null}
            </article>
          );
        })}
      </div>
    </section>
  );
}

function DownstreamChainList({ onOpenChain }: { onOpenChain: (chainId: string) => void }) {
  const chains = sourceIndustryChains;
  return (
    <section className="detail-section downstream-chain-section" aria-labelledby="downstream-chain-title">
      <div className="detail-section-heading">
        <div><span>INDUSTRY CHAINS</span><h2 id="downstream-chain-title">产业链</h2></div>
        <small>{chains.length} 条</small>
      </div>
      <div className="downstream-chain-list">
        {chains.map((chain) => (
          <button type="button" data-state={chain.state} key={chain.id} onClick={() => onOpenChain(chain.id)} aria-label={`查看${chain.title}推导详情`}>
            <strong>{chain.title}</strong>
            <ImpactPill state={chain.state} compact />
            <ArrowRightIcon />
          </button>
        ))}
      </div>
    </section>
  );
}

function LayerReasoningBlock({ layer, onEvidence }: { layer: Layer; onEvidence: (state: EvidenceSheetState) => void }) {
  return (
    <section className={`detail-section layer-reasoning-block layer-${layer.key}`} aria-labelledby={`layer-reasoning-${layer.key}`}>
      <div className="layer-stage-heading">
        <span className="layer-stage-order"><LayerGlyph layer={layer.key} /></span>
        <div><small>{layer.label.toUpperCase()}</small><h2 id={`layer-reasoning-${layer.key}`}>{layer.label}层推导</h2></div>
        <EvidenceEntry onClick={() => onEvidence({ evidenceIds: layer.evidenceIds })} />
      </div>
      <div className="layer-conclusion-card">
        <span>一句话结论</span>
        <p>{layer.conclusion}</p>
      </div>
      <div className="layer-subheading"><strong>影响锚点</strong></div>
      <DetailAnchorGrid layer={layer} onEvidence={onEvidence} />
      {layer.reversal ? (
        <aside className="layer-reversal-card">
          <span className="layer-reversal-icon"><ActivityLogIcon /></span>
          <div><strong>反转条件</strong><p>{layer.reversal}</p></div>
        </aside>
      ) : null}
      <LayerTransmissionBlock layer={layer} />
    </section>
  );
}

function ChainReasoningTree({ chain, selectedId, onSelect, onEvidence }: { chain: IndustryChain; selectedId: string; onSelect: (nodeId: string) => void; onEvidence: (state: EvidenceSheetState) => void }) {
  const selected = chain.nodes.find((node) => node.id === selectedId) ?? chain.nodes[0];
  const graphEdges = chain.graphEdges ?? [];
  const indegree = new Map(chain.nodes.map((node) => [node.id, 0]));
  graphEdges.forEach((edge) => indegree.set(edge.to, (indegree.get(edge.to) ?? 0) + 1));
  const orderedIds: string[] = [];
  let frontier = chain.nodes.filter((node) => indegree.get(node.id) === 0).map((node) => node.id);
  while (frontier.length) {
    const next: string[] = [];
    frontier.forEach((nodeId) => {
      if (orderedIds.includes(nodeId)) return;
      orderedIds.push(nodeId);
      graphEdges.filter((edge) => edge.from === nodeId).forEach((edge) => {
        indegree.set(edge.to, (indegree.get(edge.to) ?? 1) - 1);
        if (indegree.get(edge.to) === 0) next.push(edge.to);
      });
    });
    frontier = [...new Set(next)];
  }
  chain.nodes.forEach((node) => { if (!orderedIds.includes(node.id)) orderedIds.push(node.id); });
  const graphNodes = orderedIds.map((nodeId) => chain.nodes.find((node) => node.id === nodeId)).filter((node): node is ChainNode => Boolean(node));
  const nodeWidth = 176;
  const nodeGap = 28;
  const canvasWidth = graphNodes.length * nodeWidth + Math.max(0, graphNodes.length - 1) * nodeGap;
  const nodeIndex = new Map(graphNodes.map((node, index) => [node.id, index]));
  const longEdges = graphEdges.filter((edge) => {
    const sourceIndex = nodeIndex.get(edge.from);
    const targetIndex = nodeIndex.get(edge.to);
    return sourceIndex !== undefined && targetIndex !== undefined && Math.abs(sourceIndex - targetIndex) > 1;
  });
  const edgeLaneHeight = longEdges.length ? 32 + longEdges.length * 13 : 10;
  return (
    <section className="detail-section chain-reasoning-section" aria-labelledby="chain-reasoning-heading">
      <div className="chain-summary-top">
        <h2 id="chain-reasoning-heading">{chain.title}</h2>
        <div className="chain-summary-actions"><EvidenceEntry onClick={() => onEvidence({ evidenceIds: chain.evidenceIds })} /></div>
      </div>
      <p className="chain-conclusion">{chain.conclusion}</p>
      <div className="chain-level-signals" aria-label="产业链结论标签">
        <span className="is-result" data-state={chain.state}><ActivityLogIcon /><small>链结果</small><strong>{chain.state}</strong></span>
        <span><ClockIcon /><small>时间窗口</small><strong>{chain.horizon}</strong></span>
        <span><CheckCircledIcon /><small>置信度</small><strong>{chain.confidence}</strong></span>
      </div>
      <div className="layer-subheading chain-graph-heading"><strong>产业链图</strong><span>横向滑动 · 点击节点查看详情</span></div>
      <Carousel className="chain-map-carousel" contentClassName="chain-map-track" ariaLabel={`${chain.title}产业链图`}>
        <div className="chain-map-canvas" style={{ width: canvasWidth, paddingTop: edgeLaneHeight }}>
          {graphEdges.length ? (
            <svg className="chain-map-lines" width={canvasWidth} height={edgeLaneHeight + 58} style={{ width: canvasWidth, height: edgeLaneHeight + 58 }} viewBox={`0 0 ${canvasWidth} ${edgeLaneHeight + 58}`} aria-hidden="true">
              <defs>
                <marker id={`chain-arrow-${chain.id}`} markerWidth="7" markerHeight="7" refX="6" refY="3.5" orient="auto">
                  <path d="M0,0 L7,3.5 L0,7 Z" />
                </marker>
              </defs>
              {graphEdges.map((edge, edgeIndex) => {
                const sourceIndex = nodeIndex.get(edge.from);
                const targetIndex = nodeIndex.get(edge.to);
                if (sourceIndex === undefined || targetIndex === undefined) return null;
                const isAdjacent = Math.abs(sourceIndex - targetIndex) === 1;
                const movesRight = sourceIndex < targetIndex;
                const sourceCenterX = sourceIndex * (nodeWidth + nodeGap) + nodeWidth / 2;
                const targetCenterX = targetIndex * (nodeWidth + nodeGap) + nodeWidth / 2;
                const sourceX = sourceIndex * (nodeWidth + nodeGap) + (movesRight ? nodeWidth : 0);
                const targetX = targetIndex * (nodeWidth + nodeGap) + (movesRight ? 0 : nodeWidth);
                const longEdgeIndex = longEdges.findIndex((candidate) => candidate === edge);
                const sourcePeers = longEdges.filter((candidate) => candidate.from === edge.from);
                const targetPeers = longEdges.filter((candidate) => candidate.to === edge.to);
                const sourceOffset = (sourcePeers.indexOf(edge) - (sourcePeers.length - 1) / 2) * 12;
                const targetOffset = (targetPeers.indexOf(edge) - (targetPeers.length - 1) / 2) * 12;
                const routedSourceX = sourceCenterX + sourceOffset;
                const routedTargetX = targetCenterX + targetOffset;
                const laneY = 12 + Math.max(0, longEdgeIndex) * 13;
                const nodeTopY = edgeLaneHeight + 2;
                const inlineY = edgeLaneHeight + 28;
                const path = isAdjacent
                  ? `M ${sourceX} ${inlineY} L ${targetX} ${inlineY}`
                  : `M ${routedSourceX} ${nodeTopY} V ${laneY} H ${routedTargetX} V ${nodeTopY}`;
                const labelX = isAdjacent ? (sourceX + targetX) / 2 : (routedSourceX + routedTargetX) / 2;
                const labelY = isAdjacent ? inlineY - 5 : laneY - 3;
                return (
                  <g key={`${edge.from}-${edge.to}-${edgeIndex}`}>
                    <path d={path} markerEnd={`url(#chain-arrow-${chain.id})`} />
                    <text x={labelX} y={labelY}>{edge.relation}</text>
                  </g>
                );
              })}
            </svg>
          ) : null}
          <div className="chain-map-nodes" style={{ gridTemplateColumns: `repeat(${graphNodes.length}, ${nodeWidth}px)`, gap: nodeGap }}>
            {graphNodes.map((node) => (
              <button type="button" aria-pressed={selected.id === node.id} className={`chain-map-node ${selected.id === node.id ? "is-active" : ""}`} key={node.id} onClick={() => onSelect(node.id)}>
                <strong>{node.title}</strong>
                <NodeSignalTags node={node} compact />
              </button>
            ))}
          </div>
        </div>
      </Carousel>
      <article className="node-detail-panel">
        <div className="node-detail-title"><h3>{selected.title}</h3></div>
        <NodeSignalTags node={selected} />
        <div className="node-fact-row"><strong>本次影响</strong><p>{selected.impact}</p></div>
        <div className="node-transmission-copy"><span>传导逻辑</span><p>{selected.why}</p></div>
        <div className="node-evidence-entry"><span>{selected.evidence.length ? (selected.tone === "observed" ? "该节点有直接依据" : "查看起点依据") : "当前节点无直接依据"}</span>{selected.evidence.length ? <EvidenceEntry onClick={() => onEvidence({ evidenceIds: selected.evidence })} /> : <span className="evidence-gap-mark">待补证</span>}</div>
      </article>
      {chain.gap ? <div className="chain-gap-note"><InfoCircledIcon /><div><strong>反证与缺口</strong><p>{chain.gap}</p></div></div> : null}
      <div className="chain-stop-rule"><ExclamationTriangleIcon /><div><strong>停止条件</strong><p>{chain.stopRule}</p></div></div>
    </section>
  );
}

function DetailScreen({ flow, entryLayer, entryChainId }: { flow: FlowControls; entryLayer: LayerKey; entryChainId?: string }) {
  const activeChain = sourceIndustryChains.find((chain) => chain.id === entryChainId) ?? sourceIndustryChains[0];
  const [selectedNodeId, setSelectedNodeId] = useState(activeChain.nodes[0].id);
  const [evidenceSheet, setEvidenceSheet] = useState<EvidenceSheetState | null>(null);
  const layerSequence = entryLayer === "geo" ? [sourceGeoLayer, sourceMacroLayer] : entryLayer === "macro" ? [sourceMacroLayer] : [];
  return (
    <>
      <MobileScroll className="app-screen detail-screen">
        <main className="screen-content detail-content">
          {layerSequence.map((sequenceLayer) => (
            <div key={sequenceLayer.key}>
              <LayerReasoningBlock layer={sequenceLayer} onEvidence={setEvidenceSheet} />
              <div className="stage-transition"><span><ChevronDownIcon /></span><strong>{sequenceLayer.key === "geo" ? "继续看宏观经济" : "查看相关产业链"}</strong></div>
            </div>
          ))}
          {entryLayer === "geo" || entryLayer === "macro" ? (
            <DownstreamChainList onOpenChain={(chainId) => flow.push(createDetailScreen("industry", chainId))} />
          ) : null}
          {entryLayer === "industry" ? <ChainReasoningTree chain={activeChain} selectedId={selectedNodeId} onSelect={setSelectedNodeId} onEvidence={setEvidenceSheet} /> : null}
          <div className="prototype-disclaimer">该详情页仅呈现报告已发布的结构化输入、机制、输出、直接依据与待验证边界。</div>
        </main>
      </MobileScroll>
      <BottomSheet open={Boolean(evidenceSheet)} onOpenChange={(open) => { if (!open) setEvidenceSheet(null); }} title="证据列表" snap={0.76}>
        {evidenceSheet ? <DirectEvidenceList evidenceIds={evidenceSheet.evidenceIds} /> : null}
      </BottomSheet>
    </>
  );
}

function createDetailScreen(layerKey: LayerKey, chainId?: string): FlowScreen {
  const layer = layerByKey(layerKey);
  return { id: `report-detail-${layerKey}${chainId ? `-${chainId}` : ""}`, headerHeight: 54, header: (flow) => <DetailHeader flow={flow} layer={layer} />, render: (flow) => <DetailScreen flow={flow} entryLayer={layerKey} entryChainId={chainId} /> };
}

function getInitialVariant(): Variant {
  const value = new URLSearchParams(window.location.search).get("variant");
  return value === "B" || value === "C" ? value : "A";
}

export default function Prototype() {
  const variant = useMemo<Variant>(getInitialVariant, []);
  useEffect(() => {
    document.title = "Tidewise 投研报告原型";
  }, []);
  const initialScreen = useMemo<FlowScreen>(() => ({ id: "report-feed", render: (flow) => <FeedScreen flow={flow} variant={variant} /> }), [variant]);
  return <div className="tw-prototype-shell"><FlowStack initial={initialScreen} /></div>;
}
