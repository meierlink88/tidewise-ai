// Frozen design examples from approved Tidetell 2.0. Never used as API fallback.
export type Company = {
  fullName: string;
  extraTagCount?: number;
  id: string;
  name: string;
  symbol: string;
  initials: string;
  industry: string;
  tags: string[];
  report: boolean;
};
export const companies: Company[] = [
  {
    id: 'xq',
    name: '新泉股份',
    fullName: '江苏新泉汽车饰件股份有限公司',
    symbol: '603179.SH',
    initials: 'XQGF',
    industry: '汽车与汽车零部件',
    tags: ['汽车配件精选', '新能源汽车'],
    report: true
  },
  {
    id: 'xw',
    fullName: '合肥芯碁微电子装备股份有限公司',
    name: '芯碁微装',
    symbol: '688630.SH',
    initials: 'XQWZ',
    industry: '技术硬件与设备',
    tags: ['新质生产力综合', 'TMT'],
    extraTagCount: 5,
    report: false
  },
  {
    id: 'xbst',
    fullName: '南通星球石墨股份有限公司',
    name: '星球石墨',
    symbol: '688633.SH',
    initials: 'XQSM',
    industry: '资本货物',
    tags: ['新质生产力综合'],
    report: false
  },
  {
    id: 'byd',
    name: '比亚迪',
    fullName: '比亚迪股份有限公司',
    symbol: '002594.SZ',
    initials: 'BYD',
    industry: '乘用车',
    tags: ['新能源汽车', '动力电池'],
    report: false
  },
  {
    id: 'catl',
    name: '宁德时代',
    fullName: '宁德时代新能源科技股份有限公司',
    symbol: '300750.SZ',
    initials: 'NDSD',
    industry: '电池',
    tags: ['动力电池', '储能'],
    report: false
  },
  {
    id: 'pa',
    name: '平安银行',
    fullName: '平安银行股份有限公司',
    symbol: '000001.SZ',
    initials: 'PAYH',
    industry: '银行',
    tags: ['股份制银行'],
    report: false
  },
  {
    id: 'midea',
    name: '美的集团',
    fullName: '美的集团股份有限公司',
    symbol: '000333.SZ',
    initials: 'MDJT',
    industry: '家用电器',
    tags: ['智能家居', '工业自动化'],
    report: false
  },
  {
    id: 'zj',
    name: '紫金矿业',
    fullName: '紫金矿业集团股份有限公司',
    symbol: '601899.SH',
    initials: 'ZJKY',
    industry: '有色金属',
    tags: ['黄金', '铜'],
    report: false
  }
];

export const indices = [
  {
    name: '沪深300',
    code: '000300.SH',
    market: 'A股',
    value: '3,842.16',
    change: '+0.86%',
    pe: '12.8',
    pb: '1.34',
    yield: '3.12%'
  },
  {
    name: '上证指数',
    code: '000001.SH',
    market: 'A股',
    value: '3,268.72',
    change: '+0.42%',
    pe: '13.6',
    pb: '1.28',
    yield: '2.84%'
  },
  {
    name: '恒生指数',
    code: 'HSI',
    market: '港股',
    value: '19,426.30',
    change: '−0.31%',
    pe: '9.4',
    pb: '0.98',
    yield: '4.10%'
  },
  {
    name: '标普500',
    code: 'SPX',
    market: '美股',
    value: '5,678.25',
    change: '+0.57%',
    pe: '24.6',
    pb: '4.72',
    yield: '1.32%'
  }
];
export const chains = [
  {
    name: '人工智能',
    caption: '从算力基础设施到应用落地',
    nodes: [
      ['上游', '芯片与设备', '算力芯片、半导体设备、先进材料'],
      ['中游', '算力基础设施', '服务器、光模块、数据中心'],
      ['下游', '模型与应用', '基础模型、企业应用、智能终端']
    ]
  },
  {
    name: '新能源汽车',
    caption: '从核心材料到整车与服务',
    nodes: [
      ['上游', '材料与资源', '锂资源、正负极材料、结构材料'],
      ['中游', '核心零部件', '动力电池、电驱、汽车内饰'],
      ['下游', '整车与服务', '乘用车、充换电、汽车服务']
    ]
  },
  {
    name: '人形机器人',
    caption: '从核心部件到系统集成',
    nodes: [
      ['上游', '核心零部件', '减速器、伺服电机、传感器'],
      ['中游', '本体制造', '运动控制、结构件、系统集成'],
      ['下游', '应用场景', '工业制造、物流、服务场景']
    ]
  }
];
