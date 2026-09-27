import { useState } from 'react';
// Presentation port of the approved static company report. No network or account state.
import { Button, ScrollView, Text, View } from '@tarojs/components';
import { ReportHtml } from './primitives';
import { PreviewSheet } from '../../../../features/design-preview/shell';
import debateData from '../../../../features/design-preview/report-data/debate-detail-data.json';
import detailData from '../../../../features/design-preview/report-data/remaining-details.json';
import SentimentDetail from './SentimentDetail';
import NewsDetail from './NewsDetail';
import FundamentalDetail from './FundamentalDetail';
import DecisionLogic, { PriceOverview } from './DecisionLogic';
import TechnicalDetail from './TechnicalDetail';
import { OpenInNewWindowIcon } from './icons';
import report from '../../../../features/design-preview/report-data/report.json';

type Topic = Exclude<keyof typeof report, 'scores'>;
const topicNames: Record<Topic, string> = {
  decision: '决策逻辑',
  plan: '执行计划',
  risk: '风险探测',
  debate: '多空决议',
  fundamental: '基本面评估',
  news: '新闻面评估',
  sentiment: '情绪面评估',
  technical: '技术面评估',
  levels: '关键价位'
};
const dims = [
  {
    id: 'fundamental' as Topic,
    name: '基本面',
    score: '6.0',
    verdict: '良好·下沿',
    summary: '增收不增效',
    value: 'H1 归母 4.14亿 · 同比 −1.90%'
  },
  {
    id: 'news' as Topic,
    name: '新闻面',
    score: '6.5',
    verdict: '偏多',
    summary: '行业催化密集，公司兑现待证',
    value: '09.09 最新可检索公告'
  },
  {
    id: 'sentiment' as Topic,
    name: '情绪面',
    score: '4.0',
    verdict: '中性',
    summary: '评级偏多，资金持续流出',
    value: '近10日主力净流出 9,933.59万'
  },
  {
    id: 'technical' as Topic,
    name: '技术面',
    score: '5.0',
    verdict: '中性观察',
    summary: '贴近SMA60，突破未确认',
    value: 'SMA60 40.92 / 收盘 40.91'
  }
];
function DetailBody({ id }: { id: Topic }) {
  if (id === 'plan') return <ExecutionDetail />;
  if (id === 'risk') return <RiskDetail />;
  if (id === 'debate') return <DebateDetail />;
  if (id === 'sentiment') return <SentimentDetail />;
  if (id === 'news') return <NewsDetail />;
  if (id === 'fundamental') return <FundamentalDetail />;
  if (id === 'decision') return <DecisionLogic />;
  if (id === 'technical') return <TechnicalDetail />;
  return (
    <View className='detail-body lr-article'>
      <View className='detail-meta lr-p'>新泉股份 · 2026.09.23</View>
      <ReportHtml className='original lr-div' value={report[id]} />
    </View>
  );
}
function Overview({ open }: { open: (id: Topic) => void }) {
  return (
    <View className='overview lr-main'>
      <View className='decision-hero lr-section'>
        <View className='hero-identity lr-header'>
          <View className='lr-h1'>
            新泉股份 <Text className='lr-small'>603179.SH</Text>
          </View>
          <Text className='identity-detail lr-span'>
            <Text className='lr-span'>2026.09.23</Text>
            <Button
              ariaLabel='打开决策逻辑'
              onClick={() => open('decision')}
              className='tidewise-button lr-button'
            >
              <OpenInNewWindowIcon />
            </Button>
          </Text>
        </View>
        <Button
          className='tidewise-button hero-verdict lr-button'
          onClick={() => open('decision')}
          ariaLabel='查看决策逻辑'
        >
          <View className='hero-conclusion lr-p'>
            已发生的事实，站在空头一侧——H1 归母{' '}
            <Text className='lr-strong'>4.14 亿（同比 −1.90%）</Text>
          </View>
          <View className='verdict-grid lr-div'>
            <View className='verdict-cell sell-cell lr-div'>
              <Text className='lr-span'>方向</Text>
              <Text className='lr-b'>防御倾向</Text>
              <Text className='lr-small'>风险优先</Text>
            </View>
            <View className='verdict-cell lr-div'>
              <Text className='lr-span'>信心水平</Text>
              <Text className='lr-b'>中高</Text>
              <Text className='lr-small'>主理人判断</Text>
            </View>
            <View className='verdict-cell risk-cell lr-div'>
              <Text className='lr-span'>风险等级</Text>
              <Text className='lr-b'>中高</Text>
              <Text className='lr-small'>
                <Text className='lr-strong'>7.0</Text> / 10
              </Text>
            </View>
          </View>
          <View className='allocation-strip lr-div'>
            <Text className='lr-span'>建议仓位</Text>
            <Text className='lr-b'>
              ≤1/6 <Text className='lr-small'>上沿</Text>
            </Text>
            <Text className='lr-span'>中枢 ≈1/8 · 未持仓 0</Text>
            <OpenInNewWindowIcon />
          </View>
        </Button>
        <View className='hero-dimension-cards lr-div'>
          {[dims[3], dims[0], dims[1], dims[2]].map((d) => (
            <Button
              key={d.id}
              onClick={() => open(d.id)}
              ariaLabel={d.name + '详情与逻辑'}
              className='tidewise-button lr-button'
            >
              <View className='lr-div'>
                <Text className='lr-span'>{d.name}</Text>
                <OpenInNewWindowIcon />
              </View>
              <View
                className='dimension-verdict lr-div'
                ariaLabel={`${d.verdict}，评分 ${d.score}/10`}
              >
                <Text className='lr-span'>{d.verdict}</Text>
                <Text className='lr-small'>{d.score}</Text>
              </View>
              <Text className='lr-b'>
                {d.id === 'news'
                  ? '行业催化，公司待兑现'
                  : d.id === 'sentiment'
                    ? '评级偏多，资金流出'
                    : d.summary}
              </Text>
              <View className='lr-p'>
                {d.id === 'fundamental'
                  ? '归母 4.14亿 · −1.90%'
                  : d.id === 'news'
                    ? '09.10—09.23 无新增公告'
                    : d.id === 'sentiment'
                      ? '10日净流出 9,933.59万'
                      : 'SMA60 40.92 · 突破未确认'}
              </View>
            </Button>
          ))}
        </View>
      </View>
      <View className='paired debate-row lr-div'>
        <Button
          className='tidewise-button summary-card debate-summary lr-button'
          onClick={() => open('debate')}
          ariaLabel='多空决议详情'
        >
          <View className='summary-title lr-div'>
            <Text className='lr-span'>多空决议</Text>
            <Text className='debate-verdict lr-span'>
              防御倾向
              <OpenInNewWindowIcon />
            </Text>
          </View>
          <View className='debate-conclusion lr-div'>
            费用抵消毛利改善，资金持续流出；多头利润兑现仍需验证。
          </View>
          <View className='debate-comparison lr-div'>
            <View className='debate-labels lr-div'>
              <Text className='lr-span'>
                多头 <Text className='lr-b'>37 / 70</Text>
              </Text>
              <Text className='lr-span'>
                空头 <Text className='lr-b'>49 / 70</Text>
              </Text>
            </View>
            <View
              className='debate-split lr-div'
              ariaLabel='多头 37/70，空头 49/70；红绿段按双方得分 37:49 显示'
            >
              <Text className='bull-segment lr-span' />
              <Text className='bear-segment lr-span' />
            </View>
          </View>
        </Button>
      </View>
      <Button
        className='tidewise-button summary-card execution-summary lr-button'
        onClick={() => open('plan')}
        ariaLabel='执行计划详情'
      >
        <View className='summary-title lr-div'>
          <Text className='lr-span'>执行计划</Text>
          <Text className='execution-deadline lr-span'>
            09.30 收盘前强制
            <OpenInNewWindowIcon />
          </Text>
        </View>
        <View className='execution-target lr-div'>
          <Text className='lr-span'>无条件完成至目标仓</Text>
          <View className='lr-div'>
            <Text className='lr-strong'>
              ≤1/6 <Text className='lr-small'>（上沿）</Text>
            </Text>
            <Text className='lr-b'>中枢 ≈1/8</Text>
          </View>
        </View>
        <View className='execution-steps lr-div'>
          <View className='lr-div'>
            <Text className='lr-span'>现价即减</Text>
            <Text className='lr-b'>40.91 附近</Text>
            <Text className='lr-small'>不等反弹 · 先减至 ≤1/3</Text>
          </View>
          <View className='lr-div'>
            <Text className='lr-span'>反弹再减（若发生）</Text>
            <Text className='lr-b'>41.70–42.39</Text>
            <Text className='lr-small'>继续向 ≤1/6 收敛</Text>
          </View>
          <View className='stop-action lr-div'>
            <Text className='lr-span'>剩余仓位止损</Text>
            <Text className='lr-b'>38.59</Text>
            <Text className='lr-small'>收盘跌破即止损 · 1.09×ATR</Text>
          </View>
        </View>
      </Button>
      <View className='paired lr-div'>
        <Button
          className='tidewise-button summary-card risk-summary lr-button'
          onClick={() => open('risk')}
          ariaLabel='风险探测详情'
        >
          <View className='summary-title lr-div'>
            <Text className='lr-span'>风险探测</Text>
            <Text className='risk-badge lr-span'>
              7.0 / 10
              <OpenInNewWindowIcon />
            </Text>
          </View>
          <View className='risk-conclusion lr-div'>
            在无对冲工具、且有明确二元事件的窗口前，时间比价格重要，仓位是唯一的风控杠杆。
          </View>
          <View className='risk-decision-grid lr-div'>
            <View className='risk-level-cell lr-div'>
              <Text className='lr-span'>风险水平</Text>
              <Text className='lr-b'>
                7.0 <Text className='lr-small'>/ 10</Text>
              </Text>
              <View className='lr-p'>中高 · 以现价持有未来 3 个月</View>
            </View>
            <View className='lr-div'>
              <Text className='lr-span'>授权口径</Text>
              <Text className='lr-b'>资本保全优先</Text>
              <View className='lr-p'>保护组合免受不可对冲尾部损失</View>
            </View>
            <View className='lr-div'>
              <Text className='lr-span'>仓位</Text>
              <Text className='lr-b'>
                ≤1/6 <Text className='lr-small'>上沿</Text>
              </Text>
              <View className='lr-p'>中枢 ≈1/8 · 空仓者 0 · 不裸空</View>
            </View>
            <View className='risk-time-cell lr-div'>
              <Text className='lr-span'>时机</Text>
              <Text className='lr-b'>现价即减 + 09.30 强制</Text>
              <View className='lr-p'>否决“等反弹”作为主路径</View>
            </View>
          </View>
        </Button>
      </View>
    </View>
  );
}
export default function InvestmentReport() {
  const [topic, setTopic] = useState<Topic | null>(null);
  return (
    <View className='report-app-shell compact-shell'>
      <ScrollView scrollY className='overview-scroll'>
        <Overview open={setTopic} />
      </ScrollView>
      {topic && (
        <PreviewSheet title={topicNames[topic]} close={() => setTopic(null)}>
          <View className='legacy-report-detail'>
            <DetailBody key={topic} id={topic} />
          </View>
        </PreviewSheet>
      )}
    </View>
  );
}
function ReportText({ value }: { value: string }) {
  return <ReportHtml value={value} className='lr-span' />;
}
function DetailTabs({
  names,
  value,
  onChange,
  label
}: {
  names: string[];
  value: number;
  onChange: (n: number) => void;
  label: string;
}) {
  return (
    <View className='tech-tabs lr-div' ariaLabel={label}>
      {names.map((name, i) => (
        <Button
          key={name}
          onClick={() => onChange(i)}
          className={'tidewise-button ' + ('lr-button' + (value === i ? ' is-selected' : ''))}
        >
          {name}
        </Button>
      ))}
    </View>
  );
}
function ExecutionEvents({ kind }: { kind: 'catalyst' | 'risk' }) {
  const entries =
    kind === 'catalyst'
      ? detailData.events.rows.map((r) => ({
          date: r[0],
          title: r[1],
          label: '方向',
          body: r[2]
        }))
      : detailData.eventRisks.map((r) => ({
          date: r.date,
          title: r.title,
          label: '机制',
          body: r.body
        }));
  return (
    <View className='execution-events lr-div'>
      {entries.map((r, i) => (
        <View key={i} className='lr-section'>
          <View className='lr-header'>
            <Text className='lr-time'>
              <ReportText value={r.date} />
            </Text>
            <Text className='lr-span'>{String(i + 1).padStart(2, '0')}</Text>
          </View>
          <View className='lr-h4'>
            <ReportText value={r.title} />
          </View>
          <View className='execution-event-detail lr-div'>
            <Text className='lr-span'>{r.label}</Text>
            <View className='lr-p'>
              <ReportText value={r.body} />
            </View>
          </View>
        </View>
      ))}
    </View>
  );
}
function ExecutionDetail() {
  const [tab, setTab] = useState(0);
  return (
    <View className='tech-detail execution-detail lr-article'>
      <View className='tech-heading lr-header'>
        <Text className='lr-b'>执行计划</Text>
        <Text className='lr-small'>2026.09.23</Text>
      </View>
      <View className='execution-primary lr-section'>
        <View className='lr-h3'>
          执行节奏 <Text className='lr-small'>三笔</Text>
        </View>
        <View className='execution-rhythm lr-ol'>
          {detailData.steps.rows.map((r, i) => (
            <View key={r[0]} className='lr-li'>
              <Text className='rhythm-marker lr-span'>{i + 1}</Text>
              <View className='rhythm-content lr-div'>
                <View className='lr-h4'>
                  {r[0]} · <ReportText value={r[1]} />
                </View>
                <View className='rhythm-trigger lr-p'>
                  <ReportText value={r[2]} />
                </View>
                <View className='rhythm-action lr-p'>
                  <ReportText value={r[3]} />
                  <Text className='lr-span'> → </Text>
                  <Text className='lr-strong'>
                    <ReportText value={r[4]} />
                  </Text>
                </View>
              </View>
            </View>
          ))}
        </View>
        <View className='execution-inline-note lr-p'>
          第二笔为「若发生」的<Text className='lr-b'>可选优化，不是主路径</Text>。
        </View>
      </View>
      <DetailTabs
        names={['价格要素', '催化剂', '风险事件']}
        value={tab}
        onChange={setTab}
        label='执行计划详情'
      />
      <View className='lr-div'>
        {tab === 0 && (
          <View className='decision-logic lr-div'>
            <PriceOverview execution />
          </View>
        )}
        {tab === 1 && (
          <View className='tech-section lr-section'>
            <View className='lr-h3'>关注催化剂</View>
            <ExecutionEvents kind='catalyst' />
          </View>
        )}
        {tab === 2 && (
          <View className='tech-section lr-section'>
            <View className='lr-h3'>关注风险事件</View>
            <ExecutionEvents kind='risk' />
          </View>
        )}
      </View>
    </View>
  );
}
// Condensed from report §5.1–5.3 and §6.5; the risk inventory remains verbatim §7.
const riskImpactLogic = [
  {
    title: '事件跳空，使价格触发来不及执行',
    fact: '休市期间无法交易；复牌与三季报次日可能跳空，直接穿透 40.06 / 39.64 等价格触发线。',
    effect: '触发价格不等于可以成交的价格，单靠到价后的动作无法覆盖跳空损失。',
    result: '时间提前：现价即减第一档，并在 09-30 收盘前完成至目标仓。'
  },
  {
    title: '杠杆放大，使尾部损失成为主要约束',
    fact: '融资余额 / 流通市值约 7.58%；下跳可能触发平仓，抛压进一步强化。',
    effect: '在无对冲工具的条件下，剩余仓位决定组合承受的跳空敞口。',
    result: '仓位收紧：≤1/6 为上沿，默认中枢约 1/8；不裸空。'
  },
  {
    title: '日常波动，使过近的触发线失去区分度',
    fact: '40.06 距现价仅 0.40×ATR；两档触发区间宽度仅 0.77×ATR，单日振幅可达 4.01%。',
    effect: '触发线落在噪音区，既容易被日常波动反复穿越，也挡不住跳空。',
    result: '触发调整：剩余仓位硬止损改为收盘跌破 38.59（1.09×ATR）。'
  },
  {
    title: '赔率不稳健，不能单独支撑较大仓位',
    fact: '路径加权减仓赔率约 1.45–1.5 : 1，未达 2 : 1；EV 约 −0.8%，接近零且符号不稳健。',
    effect: '赔率只能用于相对比较；边际负 EV 也不足以单独支持强制清仓。',
    result: '有度分档：以降低不可对冲的尾部敞口为目的，不一次性清仓。'
  }
];
function RiskDetail() {
  const [group, setGroup] = useState(0);
  return (
    <View className='tech-detail risk-detail lr-article'>
      <View className='tech-heading lr-header'>
        <Text className='lr-span'>
          <Text className='lr-strong'>7.0</Text> / 10
        </Text>
        <Text className='lr-b'>中高风险</Text>
        <Text className='lr-small'>未来 3 个月</Text>
      </View>
      <View className='tech-verdict lr-div'>
        <Text className='lr-strong'>资本保全优先</Text>
        <View className='lr-p'>
          <ReportText value={detailData.risk.find((r) => r.label === '裁决一句话')?.body ?? ''} />
        </View>
      </View>
      <View className='tech-section lr-section'>
        <View className='lr-h3'>结论依据</View>
        <View className='tech-reasons lr-div'>
          {riskImpactLogic.map((r, i) => (
            <View key={r.title} className='lr-div'>
              <Text className='lr-span'>{String(i + 1).padStart(2, '0')}</Text>
              <View className='lr-section'>
                <Text className='lr-b'>{r.title}</Text>
                <View className='lr-p'>
                  {r.fact} {r.effect}
                </View>
                <View className='lr-p'>{r.result}</View>
              </View>
            </View>
          ))}
        </View>
      </View>
      <View className='tech-section risk-inventory lr-section'>
        <View className='lr-h3'>风险清单</View>
        <DetailTabs
          names={['市场', '公司', '宏观']}
          value={group}
          onChange={setGroup}
          label='风险清单分类'
        />
        <View className='lr-div'>
          <View className='tech-reasons risk-inventory-items lr-div'>
            {detailData.riskGroups[group].items.map((x, i) => {
              const match = x.match(/^<b>(.*?)<\/b>[：:]?\s*([\s\S]*)$/);
              return (
                <View key={x} className='lr-div'>
                  <Text className='lr-span'>{String(i + 1).padStart(2, '0')}</Text>
                  <View className='lr-section'>
                    <Text className='lr-b'>
                      <ReportText value={match?.[1] ?? x} />
                    </Text>
                    {match && (
                      <View className='lr-p'>
                        <ReportText value={match[2]} />
                      </View>
                    )}
                  </View>
                </View>
              );
            })}
          </View>
        </View>
      </View>
    </View>
  );
}
function DebateDetail() {
  const [tab, setTab] = useState(0);
  return (
    <View className='tech-detail debate-detail lr-article'>
      <View className='tech-heading lr-header'>
        <Text className='lr-b'>防御倾向</Text>
        <Text className='lr-small'>多空决议 · 2026.09.23</Text>
      </View>
      <View className='tech-verdict lr-div'>
        <Text className='lr-strong'>盈利与资金压力更明确，改善线索仍待兑现</Text>
        <View className='lr-p'>
          费用增长抵消毛利改善，区间资金持续流出；单季盈利、海外与座椅业务虽有改善，尚不足以支持整体盈利趋势已经扭转。
        </View>
      </View>
      <View className='tech-section lr-section'>
        <View className='lr-h3'>结论依据</View>
        <View className='tech-reasons lr-div'>
          {[
            [
              '01',
              '已发生的压力',
              'H1 归母同比 −1.90%，费用率 +2.02pct；近 10 日主力净流出 9,933.6 万元。'
            ],
            [
              '02',
              '改善与兑现之间仍有距离',
              '最新单季归母同比 +0.83%，一致预期隐含 H2 需 +52.8%；局部改善不能直接外推为持续增长。'
            ],
            [
              '03',
              '两项关键推论缺乏支撑',
              '历史低 PB 分位不等于绝对便宜；积极评级也不等于持仓与深入研究形成共识。'
            ]
          ].map((r) => (
            <View key={r[0]} className='lr-div'>
              <Text className='lr-span'>{r[0]}</Text>
              <View className='lr-section'>
                <Text className='lr-b'>{r[1]}</Text>
                <View className='lr-p'>{r[2]}</View>
              </View>
            </View>
          ))}
        </View>
      </View>
      <DetailTabs
        names={['空头论点 · 7', '多头论点 · 7']}
        value={tab}
        onChange={setTab}
        label='多空论点'
      />
      <View className='lr-div'>
        {tab === 0 ? (
          <View className='debate-arguments lr-div'>
            {debateData.bear.map((r, i) => (
              <View key={r[0]} className='debate-argument lr-section'>
                <View className='lr-header'>
                  <Text className='argument-number lr-span'>{String(i + 1).padStart(2, '0')}</Text>
                  <View className='lr-h3'>{r[0]}</View>
                </View>
                <Text className='argument-source lr-small'>论据类型 · {r[1]}</Text>
                <View className='lr-dl'>
                  <View className='lr-div'>
                    <View className='lr-dt'>论据</View>
                    <View className='lr-dd'>{r[2]}</View>
                  </View>
                  <View className='argument-reason lr-div'>
                    <View className='lr-dt'>成立理由</View>
                    <View className='lr-dd'>{r[3]}</View>
                  </View>
                </View>
              </View>
            ))}
          </View>
        ) : (
          <>
            <View className='debate-result-summary lr-p'>5 项部分成立 · 2 项不成立</View>
            <View className='debate-arguments lr-div'>
              {debateData.bull.map((r, i) => (
                <View key={r[0]} className='debate-argument lr-section'>
                  <View className='lr-header'>
                    <Text className='argument-number lr-span'>
                      {String(i + 1).padStart(2, '0')}
                    </Text>
                    <View className='lr-h3'>{r[0]}</View>
                    <Text
                      className={
                        'argument-status ' + (r[1] === '不成立' ? 'unsupported' : '') + ' lr-span'
                      }
                    >
                      {r[1]}
                    </Text>
                  </View>
                  <Text className='argument-source lr-small'>论据类型 · {r[2]}</Text>
                  <View className='lr-dl'>
                    <View className='lr-div'>
                      <View className='lr-dt'>论据</View>
                      <View className='lr-dd'>{r[3]}</View>
                    </View>
                    <View className='lr-div'>
                      <View className='lr-dt'>成立部分</View>
                      <View className='lr-dd'>{r[4]}</View>
                    </View>
                    <View className='argument-reason lr-div'>
                      <View className='lr-dt'>反证与理由</View>
                      <View className='lr-dd'>{r[5]}</View>
                    </View>
                  </View>
                </View>
              ))}
            </View>
          </>
        )}
      </View>
    </View>
  );
}
