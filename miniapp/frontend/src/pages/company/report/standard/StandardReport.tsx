import { useState, type ReactNode } from 'react';
import { Button, Image, ScrollView, Text, View } from '@tarojs/components';
import { PreviewSheet } from '../../../../features/design-preview/shell';
import openWindowIcon from '../../../../assets/icons/report-open-window.svg';
import type { CompanyReport } from './contract';
import './standard.scss';

function OpenInNewWindowIcon() {
  return <Image className='standard-open-icon lr-svg' src={openWindowIcon} aria-hidden />;
}

type Topic = 'decision' | 'technical' | 'fundamental' | 'news' | 'sentiment' | 'debate' | 'risk';
const titles: Record<Topic, string> = {
  decision: '综合决策',
  technical: '技术面',
  fundamental: '基本面',
  news: '新闻面',
  sentiment: '情绪面',
  debate: '多空决议',
  risk: '风险探测'
};
const dimensions = ['technical', 'fundamental', 'news', 'sentiment'] as const;
const text = (value: string | null | undefined) => value ?? '未提供';
const number = (value: number | null, digits = 2) =>
  value === null ? '未提供' : value.toFixed(digits);
const signed = (value: number | null) =>
  value === null ? '未提供' : `${value > 0 ? '+' : ''}${value.toFixed(2)}%`;
const percent = (value: number | null) => (value === null ? '未提供' : `${value}%`);
function domain(values: ReadonlyArray<number | null>) {
  const present = values.filter(
    (value): value is number => value !== null && Number.isFinite(value)
  );
  if (!present.length) return null;
  const low = Math.min(...present),
    high = Math.max(...present);
  const padding = Math.max((high - low) * 0.12, 0.1);
  return { low: low - padding, high: high + padding };
}
const position = (value: number, range: { low: number; high: number }) =>
  ((value - range.low) / (range.high - range.low)) * 100;
function Section({
  title,
  note,
  children
}: {
  title: string;
  note?: string | null;
  children: ReactNode;
}) {
  return (
    <View className='tech-section lr-section'>
      <View className='lr-h3'>
        {title}
        {note && <Text className='lr-small'>{note}</Text>}
      </View>
      {children}
    </View>
  );
}
function Paragraph({ children }: { children: ReactNode }) {
  return <View className='lr-p'>{children}</View>;
}
function Tabs({
  names,
  value,
  change
}: {
  names: readonly string[];
  value: number;
  change: (value: number) => void;
}) {
  return (
    <View className='tech-tabs lr-div'>
      {names.map((name, index) => (
        <Button
          key={name}
          className={`tidewise-button lr-button${index === value ? ' is-selected' : ''}`}
          ariaLabel={`${name}${index === value ? '，已选中' : ''}`}
          onClick={() => change(index)}
        >
          {name}
        </Button>
      ))}
    </View>
  );
}
function Assessment({
  score,
  rating,
  label,
  conclusion,
  note
}: {
  score: number | null;
  rating: string | null;
  label: string | null;
  conclusion: string | null;
  note: string | null;
}) {
  return (
    <>
      <View className='tech-heading lr-header'>
        <Text className='lr-span'>
          <Text className='lr-strong'>{score === null ? '未评分' : number(score, 1)}</Text>
          {score !== null && ' / 10'}
        </Text>
        <Text className='lr-b'>{text(rating)}</Text>
        <Text className='lr-small'>{text(label)}</Text>
      </View>
      <View className='tech-verdict lr-div'>
        <Text className='lr-strong'>{text(conclusion)}</Text>
        <Paragraph>{text(note)}</Paragraph>
      </View>
    </>
  );
}
function Reasons({
  rows
}: {
  rows: ReadonlyArray<{ reason_title: string | null; reason_content: string | null }>;
}) {
  return (
    <Section title='结论依据'>
      <View className='tech-reasons lr-div'>
        {rows.map((row, index) => (
          <View key={row.reason_title ?? `missing-${index}`} className='lr-div'>
            <Text className='lr-span'>{String(index + 1).padStart(2, '0')}</Text>
            <View className='lr-section'>
              <Text className='lr-b'>{text(row.reason_title)}</Text>
              <Paragraph>{text(row.reason_content)}</Paragraph>
            </View>
          </View>
        ))}
      </View>
    </Section>
  );
}
function Metrics({
  rows
}: {
  rows: ReadonlyArray<{
    metric_name: string | null;
    metric_value: string | null;
    metric_note: string | null;
  }>;
}) {
  return (
    <View className='tech-metrics lr-div standard-metrics'>
      {rows.map((row, index) => (
        <View key={row.metric_name ?? `missing-${index}`} className='lr-div'>
          <Text className='lr-span'>{text(row.metric_name)}</Text>
          <Text className='lr-strong'>{text(row.metric_value)}</Text>
          <Text className='lr-small'>{text(row.metric_note)}</Text>
        </View>
      ))}
    </View>
  );
}
function Entry({ title, children }: { title: string | null; children: ReactNode }) {
  return (
    <View className='standard-entry'>
      <Text className='lr-b'>{text(title)}</Text>
      {children}
    </View>
  );
}
function ScorePair({ value }: { value: { score: number | null; max_score: number | null } }) {
  return (
    <Text>
      {value.score === null ? '未评分' : `${value.score} / ${value.max_score ?? '未提供'}`}
    </Text>
  );
}
function Overview({ report, open }: { report: CompanyReport; open: (topic: Topic) => void }) {
  const decision = report.decision.summary,
    debate = report.debate.summary,
    risk = report.risk.summary;
  const riskScore = risk.risk_assessment;
  const bull = debate.bull_score,
    bear = debate.bear_score;
  const comparable =
    bull.score !== null &&
    bear.score !== null &&
    bull.max_score !== null &&
    bull.max_score === bear.max_score &&
    bull.score + bear.score > 0;
  return (
    <View className='overview lr-main'>
      <View className='decision-hero lr-section'>
        <View className='hero-identity lr-header'>
          <View className='lr-h1'>
            {text(report.report_info.company_name)}{' '}
            <Text className='lr-small'>{text(report.report_info.stock_code)}</Text>
          </View>
          <Text className='identity-detail lr-span'>
            {text(report.report_info.report_date)}
            <Button
              className='tidewise-button lr-button'
              ariaLabel='打开综合决策'
              onClick={() => open('decision')}
            >
              <OpenInNewWindowIcon />
            </Button>
          </Text>
        </View>
        <Button
          className='tidewise-button hero-verdict lr-button'
          onClick={() => open('decision')}
          ariaLabel='查看综合决策'
        >
          <View className='hero-conclusion lr-p'>
            {text(decision.core_conclusion.core_conclusion)}
          </View>
          <View className='verdict-grid lr-div'>
            <View className='verdict-cell sell-cell lr-div'>
              <Text className='lr-span'>方向</Text>
              <Text className='lr-b'>{text(decision.direction.direction)}</Text>
              <Text className='lr-small'>{text(decision.direction.direction_note)}</Text>
            </View>
            <View className='verdict-cell lr-div'>
              <Text className='lr-span'>信心水平</Text>
              <Text className='lr-b'>{text(decision.confidence.confidence)}</Text>
              <Text className='lr-small'>{text(decision.confidence.confidence_note)}</Text>
            </View>
            <View className='verdict-cell risk-cell lr-div'>
              <Text className='lr-span'>风险等级</Text>
              <Text className='lr-b'>{text(decision.risk_level.risk_level)}</Text>
              <Text className='lr-small'>
                <ScorePair value={decision.risk_level.risk_score_scale} />
              </Text>
            </View>
          </View>
          <View className='allocation-strip lr-div'>
            <Text className='lr-span'>建议仓位</Text>
            <Text className='lr-b'>
              {text(decision.position.position_cap)} <Text className='lr-small'>上沿</Text>
            </Text>
            <Text className='lr-span'>中枢 {text(decision.position.position_center)}</Text>
            <OpenInNewWindowIcon />
          </View>
        </Button>
        <View className='hero-dimension-cards lr-div'>
          {dimensions.map((key) => {
            const item = report.four_dimensions.summary[key];
            return (
              <Button
                key={key}
                className='tidewise-button lr-button'
                onClick={() => open(key)}
                ariaLabel={`${titles[key]}详情与逻辑`}
              >
                <View className='lr-div'>
                  <Text className='lr-span'>{titles[key]}</Text>
                  <OpenInNewWindowIcon />
                </View>
                <View className='dimension-verdict lr-div'>
                  <Text className='lr-span'>{text(item.rating)}</Text>
                  <Text className='lr-small'>
                    {item.score === null ? '未评分' : number(item.score, 1)}
                  </Text>
                </View>
                <Text className='lr-b'>{text(item.core_judgment)}</Text>
                <Paragraph>{text(item.key_evidence_summary)}</Paragraph>
              </Button>
            );
          })}
        </View>
      </View>
      <Button
        className='tidewise-button summary-card debate-summary lr-button'
        onClick={() => open('debate')}
        ariaLabel='多空决议详情'
      >
        <View className='summary-title lr-div'>
          <Text className='lr-span'>多空决议</Text>
          <Text className='debate-verdict lr-span'>
            {text(debate.final_direction)} <OpenInNewWindowIcon />
          </Text>
        </View>
        <View className='debate-conclusion lr-div'>{text(debate.decision_summary)}</View>
        <View className='debate-comparison lr-div'>
          <View className='debate-labels lr-div'>
            <Text className='lr-span'>
              多头 <ScorePair value={bull} />
            </Text>
            <Text className='lr-span'>
              空头 <ScorePair value={bear} />
            </Text>
          </View>
          {comparable ? (
            <View className='debate-split lr-div'>
              <Text className='bull-segment lr-span' style={{ flex: bull.score ?? 0 }} />
              <Text className='bear-segment lr-span' style={{ flex: bear.score ?? 0 }} />
            </View>
          ) : (
            <Paragraph>暂无可比较评分</Paragraph>
          )}
        </View>
      </Button>
      <Button
        className='tidewise-button summary-card risk-summary lr-button'
        onClick={() => open('risk')}
        ariaLabel='风险探测详情'
      >
        <View className='summary-title lr-div'>
          <Text className='lr-span'>风险探测</Text>
          <Text className='risk-badge lr-span'>
            <ScorePair value={{ score: riskScore.score, max_score: riskScore.max_score }} />{' '}
            <OpenInNewWindowIcon />
          </Text>
        </View>
        <View className='risk-conclusion lr-div'>{text(risk.core_conclusion)}</View>
        <View className='risk-decision-grid lr-div'>
          <View className='risk-level-cell lr-div'>
            <Text className='lr-span'>风险水平</Text>
            <Text className='lr-b'>
              {riskScore.score === null ? '未评分' : number(riskScore.score, 1)}{' '}
              <Text className='lr-small'>/ {riskScore.max_score ?? '未提供'}</Text>
            </Text>
            <Paragraph>{text(risk.assessment_basis)}</Paragraph>
          </View>
          <View className='lr-div'>
            <Text className='lr-span'>授权口径</Text>
            <Text className='lr-b'>{text(risk.mandate)}</Text>
            <Paragraph>{text(risk.mandate_note)}</Paragraph>
          </View>
          <View className='lr-div'>
            <Text className='lr-span'>仓位</Text>
            <Paragraph>{text(risk.position_requirements)}</Paragraph>
          </View>
          <View className='risk-time-cell lr-div'>
            <Text className='lr-span'>时机</Text>
            <Text className='lr-b'>{text(risk.action_timing)}</Text>
            <Paragraph>{text(risk.timing_note)}</Paragraph>
          </View>
        </View>
      </Button>
    </View>
  );
}
function DecisionDetail({ value }: { value: CompanyReport['decision']['detail'] }) {
  return (
    <View className='decision-logic'>
      <View className='logic-panel'>
        <Section title='核心决策因素'>
          {value.decision_factors.map((factor, index) => (
            <Entry key={factor.factor_title ?? `missing-${index}`} title={factor.factor_title}>
              <Paragraph>{text(factor.key_data_summary)}</Paragraph>
              <Text className='standard-tag'>{text(factor.evidence_type)}</Text>
            </Entry>
          ))}
        </Section>
      </View>
      <View className='logic-panel'>
        <Section title='反证理由'>
          {value.rejections.map((reason, index) => (
            <Entry key={reason.reason_title ?? `missing-${index}`} title={reason.reason_title}>
              <Paragraph>{text(reason.reason_content)}</Paragraph>
            </Entry>
          ))}
        </Section>
      </View>
    </View>
  );
}
function TechnicalDetail({
  value
}: {
  value: CompanyReport['four_dimensions']['detail']['technical'];
}) {
  const [tab, setTab] = useState(0),
    [average, setAverage] = useState(0),
    [level, setLevel] = useState(0);
  const assessment = value.assessment,
    readings = value.key_readings,
    averages = readings.price_and_moving_averages;
  const map = value.support_and_resistance.price_map,
    active = map.levels[level];
  const averageRange = domain([
    averages.close_price,
    ...averages.moving_averages.map((row) => row.average_value)
  ]);
  const priceRange = domain([
    map.close_price,
    ...map.levels.flatMap((row) => [row.price_value.min, row.price_value.max])
  ]);
  return (
    <>
      <Assessment
        score={assessment.score}
        rating={assessment.rating}
        label={assessment.score_note}
        conclusion={assessment.core_conclusion}
        note={assessment.judgment_note}
      />
      <Reasons rows={value.reasons} />
      <Metrics rows={value.key_metrics} />
      <Tabs names={['关键读数', '阻力与支撑']} value={tab} change={setTab} />
      {tab === 0 ? (
        <>
          <Section title='价格与均线位置' note={averages.reading_basis}>
            <View className='standard-average-caption'>
              <Text>{averageRange ? number(averageRange.low) : '—'}</Text>
              <Text>收盘 {number(averages.close_price)}</Text>
              <Text>{averageRange ? number(averageRange.high) : '—'}</Text>
            </View>
            {averages.moving_averages.map((row, index) => (
              <Button
                key={row.average_name ?? `missing-${index}`}
                className={`tidewise-button standard-average${average === index ? ' selected' : ''}`}
                onClick={() => setAverage(index)}
                ariaLabel={`${text(row.average_name)} ${number(row.average_value, 3)}`}
              >
                <Text>{text(row.average_name)}</Text>
                <View className='standard-track'>
                  {averageRange && averages.close_price !== null && (
                    <Text
                      className='standard-current'
                      style={{ left: `${position(averages.close_price, averageRange)}%` }}
                    />
                  )}
                  {averageRange && row.average_value !== null && (
                    <Text
                      className='standard-dot'
                      style={{ left: `${position(row.average_value, averageRange)}%` }}
                    />
                  )}
                </View>
                <Text>
                  {number(
                    row.average_value,
                    row.average_value !== null && (row.average_value * 100) % 1 !== 0 ? 3 : 2
                  )}
                </Text>
              </Button>
            ))}
            <Paragraph>{text(averages.moving_averages[average]?.comparison_note)}</Paragraph>
          </Section>
          <Section title={text(readings.momentum_volatility_volume.analysis_title)}>
            {readings.momentum_volatility_volume.metric_groups.map((row, index) => (
              <Entry key={row.group_name ?? `missing-${index}`} title={row.group_name}>
                <Paragraph>{text(row.readings)}</Paragraph>
                <Text className='lr-small'>{text(row.metric_note)}</Text>
              </Entry>
            ))}
          </Section>
          <Section title='区间表现'>
            <View className='standard-return-grid'>
              {readings.period_returns.map((row, index) => (
                <View key={row.period ?? `missing-${index}`}>
                  <Text className='lr-small'>{text(row.period)}</Text>
                  <Text
                    className={`lr-b ${row.return_pct !== null && row.return_pct > 0 ? 'positive' : row.return_pct !== null && row.return_pct < 0 ? 'negative' : ''}`}
                  >
                    {signed(row.return_pct)}
                  </Text>
                </View>
              ))}
            </View>
          </Section>
        </>
      ) : (
        <>
          <Section title={text(map.map_title)} note={map.price_basis}>
            <View className='price-ladder-legend lr-div'>
              <Text className='standard-legend-resistance'>● 阻力</Text>
              <Text className='standard-legend-support'>● 支撑</Text>
              <Text>━ 收盘价</Text>
            </View>
            {priceRange ? (
              <View className='standard-price-map'>
                {[0, 25, 50, 75, 100].map((tick) => (
                  <View
                    key={tick}
                    className='standard-gridline'
                    style={{ top: `${10 + tick * 0.8}%` }}
                  >
                    <Text>
                      {number(priceRange.high - (tick / 100) * (priceRange.high - priceRange.low))}
                    </Text>
                  </View>
                ))}
                {map.close_price !== null && (
                  <View
                    className='standard-close-line'
                    style={{ top: `${10 + (100 - position(map.close_price, priceRange)) * 0.8}%` }}
                  >
                    <Text>
                      收盘 {number(map.close_price)}
                      {active?.price_value.min !== null && active?.price_value.min !== undefined
                        ? ` · 距 ${text(active.level_code)} ${number(Math.abs(map.close_price - active.price_value.min))}`
                        : ''}
                    </Text>
                  </View>
                )}
                {map.levels.map((row, index) => {
                  const bounds = row.price_value;
                  if (bounds.min === null || bounds.max === null) return null;
                  const top = 100 - position(bounds.max, priceRange),
                    bottom = 100 - position(bounds.min, priceRange);
                  return (
                    <Button
                      key={row.level_code ?? `missing-${index}`}
                      ariaLabel={`${text(row.level_code)} ${row.level_type} 价位依据`}
                      className={`tidewise-button standard-price-node ${row.level_type === '阻力' ? 'resistance' : 'support'}${level === index ? ' selected' : ''}`}
                      style={{ top: `${10 + ((top + bottom) / 2) * 0.8}%` }}
                      onClick={() => setLevel(index)}
                    >
                      <Text
                        className={`standard-price-band${bounds.min === bounds.max ? ' point' : ''}`}
                        style={{ height: `${Math.max((bottom - top) * 4.48, 6)}rpx` }}
                      />
                      <View className='standard-price-label'>
                        <Text className='lr-small'>{text(row.level_code)}</Text>
                        <Text className='lr-b'>
                          {number(bounds.min)}
                          {bounds.min !== bounds.max && `–${number(bounds.max)}`}
                        </Text>
                      </View>
                    </Button>
                  );
                })}
              </View>
            ) : (
              <Paragraph>暂无有效价位</Paragraph>
            )}
            {active && (
              <View className='standard-price-evidence'>
                <View>
                  <Text>
                    {text(active.level_type)} · {text(active.level_code)}
                  </Text>
                  <Text className='lr-b'>
                    {number(active.price_value.min)}
                    {active.price_value.max !== active.price_value.min &&
                      `–${number(active.price_value.max)}`}
                  </Text>
                </View>
                <Paragraph>{text(active.price_basis_note)}</Paragraph>
              </View>
            )}
          </Section>
          <View className='standard-price-boundary'>
            <Section title={text(value.support_and_resistance.risk_boundary_note.note_title)}>
              <Paragraph>
                {text(value.support_and_resistance.risk_boundary_note.note_content)}
              </Paragraph>
            </Section>
          </View>
        </>
      )}
    </>
  );
}
function FundamentalDetail({
  value
}: {
  value: CompanyReport['four_dimensions']['detail']['fundamental'];
}) {
  const [tab, setTab] = useState(0),
    [selected, setSelected] = useState(0);
  const a = value.assessment,
    six = value.six_dimension_scores,
    finance = value.financial_readings,
    margins = six.margin_expense_comparison;
  const marginMax = Math.max(
    1,
    Math.abs(margins.gross_margin_change_pp ?? 0),
    Math.abs(margins.expense_ratio_change_pp ?? 0)
  );
  const hasNegativeMargin =
    (margins.gross_margin_change_pp ?? 0) < 0 || (margins.expense_ratio_change_pp ?? 0) < 0;
  const chosen = six.quality_profile[selected];
  return (
    <>
      <Assessment
        score={a.score}
        rating={a.rating}
        label={a.score_note}
        conclusion={a.core_conclusion}
        note={a.conclusion_note}
      />
      <Reasons rows={value.reasons} />
      <Metrics rows={value.key_metrics} />
      <Tabs names={['六维评分', '财务读数']} value={tab} change={setTab} />
      {tab === 0 ? (
        <>
          <Section title='六维质量画像' note='同尺度 0—10 分'>
            <View className='standard-quality'>
              {six.quality_profile.map((row, index) => (
                <Button
                  key={row.dimension_name}
                  className={`tidewise-button standard-quality-row${selected === index ? ' selected' : ''}`}
                  onClick={() => setSelected(index)}
                  ariaLabel={`${row.dimension_name} ${row.score === null ? '未评分' : number(row.score, 1)}`}
                >
                  <Text>{row.dimension_name}</Text>
                  <View className='standard-track'>
                    {row.score !== null && (
                      <Text className='standard-fill' style={{ width: `${row.score * 10}%` }} />
                    )}
                  </View>
                  <Text>{row.score === null ? '未评分' : number(row.score, 1)}</Text>
                </Button>
              ))}
            </View>
            {chosen && (
              <Paragraph>
                {chosen.dimension_name} ·{' '}
                {chosen.score === null ? '未评分' : number(chosen.score, 1)}
              </Paragraph>
            )}
          </Section>
          <Section title={text(margins.analysis_title)}>
            <View className='fund-offset'>
              {[
                { name: '毛利率变化', value: margins.gross_margin_change_pp },
                { name: '期间费用率变化', value: margins.expense_ratio_change_pp }
              ].map((row) => (
                <View key={row.name} className='standard-offset'>
                  <Text>{row.name}</Text>
                  <Text>
                    {row.value === null
                      ? '未提供'
                      : `${row.value > 0 ? '+' : ''}${row.value} 个百分点`}
                  </Text>
                  <View
                    className={`standard-track${hasNegativeMargin ? ' standard-signed-track' : ''}`}
                  >
                    <Text
                      className={`standard-fill${row.value !== null && row.value < 0 ? ' standard-fill-negative' : ''}`}
                      style={{
                        width: `${(Math.abs(row.value ?? 0) / marginMax) * (hasNegativeMargin ? 50 : 100)}%`,
                        left: hasNegativeMargin
                          ? `${row.value !== null && row.value < 0 ? 50 - (Math.abs(row.value) / marginMax) * 50 : 50}%`
                          : '0%'
                      }}
                    />
                  </View>
                </View>
              ))}
              <Paragraph>{text(margins.difference_note)}</Paragraph>
            </View>
            <View className='fund-inline'>
              <View className='lr-span'>
                <Text>净利率</Text>
                <Text className='lr-b'>{percent(margins.net_margin_pct)}</Text>
                <Text className='lr-small'>
                  同比 {margins.net_margin_change_pp ?? '未提供'} 个百分点
                </Text>
              </View>
              <View className='lr-span'>
                <Text>期间费用率</Text>
                <Text className='lr-b'>{percent(margins.expense_ratio_pct)}</Text>
                <Text className='lr-small'>
                  上年同期 {percent(margins.prior_expense_ratio_pct)}
                </Text>
              </View>
            </View>
          </Section>
          <Section title={text(six.roe_trend.trend_title)}>
            <View className='fund-roe'>
              {six.roe_trend.history.map((row, index) => (
                <View className='lr-div' key={row.period ?? `missing-${index}`}>
                  <Text className='lr-small'>{text(row.period)}</Text>
                  <Text className='lr-b'>{percent(row.roe_pct)}</Text>
                </View>
              ))}
            </View>
            <Paragraph>{text(six.roe_trend.basis_note)}</Paragraph>
          </Section>
        </>
      ) : (
        <>
          <Section
            title={text(finance.growth_sources.analysis_title)}
            note={finance.growth_sources.period}
          >
            <View className='standard-region-bar'>
              {finance.growth_sources.regional_revenue.map((row, index) => (
                <Text
                  key={row.region_name ?? `missing-${index}`}
                  style={{ width: `${row.revenue_share_pct ?? 0}%` }}
                >
                  {text(row.region_name)} {percent(row.revenue_share_pct)}
                </Text>
              ))}
            </View>
            <View className='fund-inline'>
              {finance.growth_sources.regional_revenue.map((row, index) => (
                <View className='lr-span' key={row.region_name ?? `missing-${index}`}>
                  <Text>{text(row.region_name)}</Text>
                  <Text className='lr-b'>{number(row.revenue_100m_cny)} 亿</Text>
                  <Text className='lr-small'>同比 {signed(row.revenue_yoy_pct)}</Text>
                </View>
              ))}
            </View>
            <View className='fund-highlight'>
              {text(finance.growth_sources.business_highlights)}
            </View>
          </Section>
          <Section
            title={text(finance.profit_realization_threshold.analysis_title)}
            note={finance.profit_realization_threshold.calculation_basis}
          >
            <View className='fund-inline'>
              <View className='lr-span'>
                <Text>{text(finance.profit_realization_threshold.actual_metric_name)}</Text>
                <Text className='lr-b'>
                  {text(finance.profit_realization_threshold.actual_metric_value)}
                </Text>
                <Text className='lr-small'>
                  {text(finance.profit_realization_threshold.actual_period_note)}
                </Text>
              </View>
              <View className='lr-span'>
                <Text>{text(finance.profit_realization_threshold.required_metric_name)}</Text>
                <Text className='lr-b'>
                  {text(finance.profit_realization_threshold.required_metric_value)}
                </Text>
                <Text className='lr-small'>
                  {text(finance.profit_realization_threshold.required_period_note)}
                </Text>
              </View>
            </View>
            <Paragraph>{text(finance.profit_realization_threshold.calculation_note)}</Paragraph>
          </Section>
          <View className='standard-finance-evidence'>
            <Section title='完整财务依据'>
              {finance.financial_evidence.map((row, index) => (
                <Entry key={row.category_name ?? `missing-${index}`} title={row.category_name}>
                  <Paragraph>{text(row.financial_content)}</Paragraph>
                </Entry>
              ))}
            </Section>
          </View>
        </>
      )}
    </>
  );
}
function NewsDetail({ value }: { value: CompanyReport['four_dimensions']['detail']['news'] }) {
  const [tab, setTab] = useState(0),
    [filter, setFilter] = useState(0);
  const a = value.assessment,
    messages = value.event_timeline.messages,
    filters = ['全部', '公告', '评级', '行业'];
  const events = messages.events.filter(
    (event) => filter === 0 || event.event_type === filters[filter]
  );
  return (
    <>
      <Assessment
        score={a.score}
        rating={a.rating}
        label={a.score_note}
        conclusion={a.core_conclusion}
        note={a.conclusion_note}
      />
      <Reasons rows={value.reasons} />
      <Tabs names={['事件时间轴', '业务进展']} value={tab} change={setTab} />
      {tab === 0 ? (
        <Section title='消息来自哪里' note={messages.time_range_note}>
          <View className='news-filters'>
            {filters.map((name, index) => (
              <Button
                className={`tidewise-button lr-button${filter === index ? ' is-selected' : ''}`}
                key={name}
                onClick={() => setFilter(index)}
                ariaLabel={`${name}${filter === index ? '，已选中' : ''}`}
              >
                {name}
              </Button>
            ))}
          </View>
          <View className='news-timeline lr-div'>
            {events.map((event, index) => (
              <View
                key={`${event.event_date}-${event.event_title ?? index}`}
                className='news-event lr-div'
              >
                <View className='news-event-heading lr-div'>
                  <Text className='lr-time'>{text(event.event_date)}</Text>
                  <Text className='lr-span'>
                    <Text
                      className={`news-tag lr-em ${event.event_type === '行业' ? 'kind-industry' : event.event_type === '评级' ? 'kind-rating' : 'kind-announcement'}`}
                    >
                      {text(event.event_type)}
                    </Text>
                    <Text className='lr-b'>{text(event.event_title)}</Text>
                  </Text>
                </View>
                <Paragraph>{text(event.event_content)}</Paragraph>
              </View>
            ))}
          </View>
          {events.length === 0 && <Paragraph>该类别暂无事件</Paragraph>}
        </Section>
      ) : (
        <>
          <View className='news-boundary lr-div'>
            <Text className='lr-b'>
              {text(value.business_progress.business_boundary.boundary_conclusion)}
            </Text>
            <Paragraph>{text(value.business_progress.business_boundary.boundary_note)}</Paragraph>
          </View>
          <Section title='机器人业务已发生事项'>
            <View className='news-business'>
              {value.business_progress.robotics_completed_actions.map((row, index) => (
                <View className='lr-div' key={`${row.action_date}-${row.action_name ?? index}`}>
                  <Text className='lr-time'>{text(row.action_date)}</Text>
                  <Text className='lr-b'>{text(row.action_name)}</Text>
                  <Paragraph>{text(row.action_content)}</Paragraph>
                </View>
              ))}
            </View>
          </Section>
        </>
      )}
    </>
  );
}
function SentimentDetail({
  value
}: {
  value: CompanyReport['four_dimensions']['detail']['sentiment'];
}) {
  const a = value.assessment,
    flows = value.daily_main_fund_flow;
  const [selected, setSelected] = useState(Math.max(0, flows.daily_values.length - 1));
  const active = flows.daily_values[selected];
  const max = Math.max(1, ...flows.daily_values.map((row) => Math.abs(row.net_flow ?? 0)));
  return (
    <>
      <Assessment
        score={a.score}
        rating={a.rating}
        label={a.score_note}
        conclusion={a.core_conclusion}
        note={a.conclusion_note}
      />
      <Reasons rows={value.reasons} />
      <View className='standard-sentiment-metrics'>
        <Metrics rows={value.key_metrics} />
      </View>
      <Section title='主力资金逐日净额' note={flows.unit}>
        <View className='standard-flow-container'>
          <View className='standard-flow-legend'>
            <Text className='positive'>红 · 净流入</Text>
            <Text className='negative'>绿 · 净流出</Text>
          </View>
          <View className='standard-flow-chart'>
            {flows.daily_values.map((row, index) => (
              <Button
                key={row.trade_date ?? `missing-${index}`}
                className={`tidewise-button standard-flow-column${selected === index ? ' selected' : ''}`}
                onClick={() => setSelected(index)}
                ariaLabel={`${text(row.trade_date)} ${row.net_flow === null ? '未提供' : row.net_flow > 0 ? '净流入' : row.net_flow < 0 ? '净流出' : '零值'} ${number(row.net_flow, 1)} ${flows.unit}`}
              >
                <View className='standard-flow-area'>
                  <Text
                    className={`standard-flow-bar${row.net_flow !== null && row.net_flow > 0 ? ' inflow' : ' outflow'}`}
                    style={{
                      height: `${(Math.abs(row.net_flow ?? 0) / max) * 45}%`,
                      top:
                        row.net_flow !== null && row.net_flow > 0
                          ? `${50 - (row.net_flow / max) * 45}%`
                          : '50%'
                    }}
                  />
                </View>
                <Text className='lr-small'>{row.trade_date?.slice(-2) ?? '—'}</Text>
              </Button>
            ))}
          </View>
          <View className='standard-flow-axis'>
            <Text>{text(flows.daily_values[0]?.trade_date)}</Text>
            <Text>{text(flows.period)}</Text>
            <Text>{text(flows.daily_values[flows.daily_values.length - 1]?.trade_date)}</Text>
          </View>
          {active && (
            <View className='sentiment-selected lr-div'>
              <Text>
                {text(active.trade_date)} ·{' '}
                {active.net_flow === null
                  ? '未提供'
                  : active.net_flow > 0
                    ? '净流入'
                    : active.net_flow < 0
                      ? '净流出'
                      : '零值'}
              </Text>
              <Text
                className={`lr-b ${active.net_flow !== null && active.net_flow > 0 ? 'positive' : active.net_flow !== null && active.net_flow < 0 ? 'negative' : ''}`}
              >
                {active.net_flow !== null && active.net_flow > 0 ? '+' : ''}
                {number(active.net_flow, 1)} {flows.unit}
              </Text>
            </View>
          )}
          {!active && <Paragraph>暂无资金读数</Paragraph>}
        </View>
      </Section>
      <Section title='关键读数'>
        <View className='standard-readings-table'>
          <View className='standard-reading-head'>
            <Text>项目</Text>
            <Text>值</Text>
          </View>
          {value.key_readings.map((row, index) => (
            <View
              className='standard-reading-row'
              key={`${row.item_name}-${row.reading_basis ?? index}`}
            >
              <View>
                <Text>{text(row.item_name)}</Text>
                {row.reading_basis && <Text className='lr-small'>{row.reading_basis}</Text>}
              </View>
              <View>
                <Paragraph>{text(row.reading_content)}</Paragraph>
              </View>
            </View>
          ))}
        </View>
      </Section>
    </>
  );
}
function DebateDetail({ value }: { value: CompanyReport['debate']['detail'] }) {
  const [tab, setTab] = useState(0),
    a = value.assessment,
    rows = tab === 0 ? value.bear_arguments : value.bull_arguments;
  return (
    <>
      <View className='tech-heading lr-header'>
        <Text className='lr-b'>{text(a.final_direction)}</Text>
        <Text className='lr-small'>{text(a.decision_date)}</Text>
      </View>
      <View className='tech-verdict lr-div'>
        <Text className='lr-strong'>{text(a.core_conclusion)}</Text>
        <Paragraph>{text(a.conclusion_note)}</Paragraph>
      </View>
      <Reasons rows={value.reasons} />
      <Tabs
        names={[
          `空头论点 · ${value.bear_arguments.length}`,
          `多头论点 · ${value.bull_arguments.length}`
        ]}
        value={tab}
        change={setTab}
      />
      <View className='debate-arguments lr-div'>
        {rows.map((row, index) => (
          <View
            key={row.argument_title ?? `missing-${index}`}
            className='debate-argument lr-section'
          >
            <View className='lr-header'>
              <Text className='argument-number lr-span'>{String(index + 1).padStart(2, '0')}</Text>
              <View className='lr-h3'>{text(row.argument_title)}</View>
            </View>
            <Text className='argument-source lr-small'>
              论据类型 · {text(row.evidence_category)}
            </Text>
            <Entry title='论据'>
              <Paragraph>{text(row.evidence)}</Paragraph>
            </Entry>
            <Entry title='成立理由'>
              <Paragraph>{text(row.supporting_reason)}</Paragraph>
            </Entry>
          </View>
        ))}
      </View>
    </>
  );
}
function RiskDetail({ value }: { value: CompanyReport['risk']['detail'] }) {
  const [group, setGroup] = useState(0),
    a = value.assessment;
  const rows = [
    value.risk_inventory.market_risks,
    value.risk_inventory.company_risks,
    value.risk_inventory.macro_risks
  ][group];
  return (
    <>
      <Assessment
        score={a.risk_score}
        rating={a.risk_level}
        label={a.assessment_horizon}
        conclusion={a.core_conclusion}
        note={a.conclusion_note}
      />
      <Section title='结论依据'>
        <View className='tech-reasons'>
          {value.reasons.map((row, index) => (
            <View className='lr-div' key={row.reason_title ?? `missing-${index}`}>
              <Text className='lr-span'>{String(index + 1).padStart(2, '0')}</Text>
              <View className='lr-section'>
                <Text className='lr-b'>{text(row.reason_title)}</Text>
                <Paragraph>{text(row.risk_fact)}</Paragraph>
                <Paragraph>{text(row.impact_mechanism)}</Paragraph>
                <View className='standard-risk-response'>{text(row.response_requirement)}</View>
              </View>
            </View>
          ))}
        </View>
      </Section>
      <Section title='风险清单'>
        <Tabs names={['市场', '公司', '宏观']} value={group} change={setGroup} />
        {rows.map((row, index) => (
          <Entry key={row.risk_title ?? `missing-${index}`} title={row.risk_title}>
            <Paragraph>{text(row.risk_content)}</Paragraph>
          </Entry>
        ))}
      </Section>
    </>
  );
}
function Detail({ topic, report }: { topic: Topic; report: CompanyReport }) {
  switch (topic) {
    case 'decision':
      return <DecisionDetail value={report.decision.detail} />;
    case 'technical':
      return <TechnicalDetail value={report.four_dimensions.detail.technical} />;
    case 'fundamental':
      return <FundamentalDetail value={report.four_dimensions.detail.fundamental} />;
    case 'news':
      return <NewsDetail value={report.four_dimensions.detail.news} />;
    case 'sentiment':
      return <SentimentDetail value={report.four_dimensions.detail.sentiment} />;
    case 'debate':
      return <DebateDetail value={report.debate.detail} />;
    case 'risk':
      return <RiskDetail value={report.risk.detail} />;
  }
}
export default function StandardReport({ report }: { report: CompanyReport }) {
  const [topic, setTopic] = useState<Topic | null>(null);
  return (
    <View className='standard-report report-app-shell compact-shell'>
      <ScrollView scrollY className='overview-scroll'>
        <Overview report={report} open={setTopic} />
      </ScrollView>
      {topic && (
        <PreviewSheet
          title={titles[topic]}
          subtitle={`${text(report.report_info.company_name)} · ${text(report.report_info.report_date)} · 样例报告`}
          close={() => setTopic(null)}
        >
          <View
            className={`standard-report legacy-report-detail tech-detail ${topic}-detail lr-article`}
          >
            <Detail key={topic} topic={topic} report={report} />
          </View>
        </PreviewSheet>
      )}
    </View>
  );
}
