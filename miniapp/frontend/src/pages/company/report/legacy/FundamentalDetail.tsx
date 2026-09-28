import { useState } from 'react';
// Presentation port of the approved static company report. No network or account state.
import { Button, Text, View } from '@tarojs/components';
import { ReportHtml } from './primitives';
import data from '../../../../features/design-preview/report-data/fundamental-data.json';

export default function FundamentalDetail() {
  const [tab, setTab] = useState(0);
  const [dimension, setDimension] = useState(0);
  return (
    <View className='tech-detail fundamental-detail lr-article'>
      <View className='tech-heading lr-header'>
        <Text className='lr-span'>
          <Text className='lr-strong'>6.0</Text> / 10
        </Text>
        <Text className='lr-b'>良好（下沿）</Text>
        <Text className='lr-small'>基本面质量总评</Text>
      </View>
      <View className='tech-verdict lr-div'>
        <Text className='lr-strong'>收增利减，增收不增效</Text>
        <View className='lr-p'>
          毛利改善被费用上升完全对冲，净利率与 ROE 走低；境外与座椅业务增长是亮点。
        </View>
      </View>
      <View className='tech-section lr-section'>
        <View className='lr-h3'>结论依据</View>
        <View className='tech-reasons lr-div'>
          {[
            [
              '01',
              '收入增长，盈利质量承压',
              'H1 营收同比 +7.45%，归母同比 −1.90%。毛利率提升 1.99 pct，期间费用率上升 2.02 pct，净利率仍下降。'
            ],
            [
              '02',
              '增长亮点尚未扭转整体',
              '境外与座椅业务同比增长 43.34% / 65.57%；境内营收占 73.62%，同比下降 1.4%。'
            ],
            [
              '03',
              '现金流与杠杆形成约束',
              'FCF −3.57 亿，应收周转由 91 天延长到 103 天，营业周期由 171 天延长到 188 天；资产负债率高于行业。'
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
      <View className='tech-metrics lr-div'>
        {[
          ['2026H1 营业收入', '80.15 亿', '同比 +7.45%'],
          ['2026H1 归母净利润', '4.14 亿', '同比 −1.90%'],
          ['自由现金流 FCF', '−3.57 亿', '计算值 · 2.050 − 5.623'],
          ['资产负债率', '60.51%', '行业 43.16% · 速动比率 0.89']
        ].map((r) => (
          <View key={r[0]} className='lr-div'>
            <Text className='lr-span'>{r[0]}</Text>
            <Text className='lr-strong'>{r[1]}</Text>
            <Text className='lr-small'>{r[2]}</Text>
          </View>
        ))}
      </View>
      <View className='tech-tabs lr-div' ariaLabel='基本面详情'>
        {['六维评分', '财务读数'].map((t, i) => (
          <Button
            key={t}
            onClick={() => setTab(i)}
            className={'tidewise-button ' + ('lr-button' + (tab === i ? ' is-selected' : ''))}
          >
            {t}
          </Button>
        ))}
      </View>
      <View className='lr-div'>
        {tab === 0 && (
          <>
            <View className='tech-section lr-section'>
              <View className='lr-h3'>
                六维质量画像 <Text className='lr-small'>同尺度 0—10 分</Text>
              </View>
              <View className='fund-score-chart lr-div'>
                {data.scores.map((d, i) => (
                  <Button
                    key={d.name}
                    onClick={() => setDimension(i)}
                    ariaLabel={`${d.name} ${d.score.toFixed(1)} 分依据`}
                    className={
                      'tidewise-button ' + ('lr-button' + (dimension === i ? ' is-selected' : ''))
                    }
                  >
                    <Text className='lr-span'>{d.name}</Text>
                    <Text className='lr-i'>
                      <Text
                        style={{
                          width: d.score * 10 + '%'
                        }}
                        className='lr-em'
                      />
                    </Text>
                    <Text className='lr-b'>{d.score.toFixed(1)}</Text>
                  </Button>
                ))}
              </View>
              <View className='fund-evidence lr-div'>
                <Text className='lr-b'>
                  {data.scores[dimension].name} · {data.scores[dimension].score.toFixed(1)} / 10
                </Text>
                <ReportHtml value={data.scores[dimension].body} className='lr-p' />
              </View>
            </View>
            <View className='tech-section lr-section'>
              <View className='lr-h3'>毛利改善为何没有变成利润增长</View>
              <View className='fund-offset lr-div'>
                {[
                  ['毛利率改善', '+1.99 pct', 1.99],
                  ['期间费用率上升', '+2.02 pct', 2.02]
                ].map((r) => (
                  <View key={r[0]} className='lr-div'>
                    <Text className='lr-span'>{r[0]}</Text>
                    <Text className='lr-b'>{r[1]}</Text>
                    <Text className='lr-i'>
                      <Text
                        style={{
                          width: (Number(r[2]) / 2.5) * 100 + '%'
                        }}
                        className='lr-em'
                      />
                    </Text>
                  </View>
                ))}
                <View className='lr-p'>
                  费用增幅超过毛利改善 <Text className='lr-b'>0.03 pct</Text>（两项差值）
                </View>
              </View>
              <View className='fund-inline lr-div'>
                <Text className='lr-span'>
                  净利率 <Text className='lr-b'>5.17%</Text>
                  <Text className='lr-small'>同比 −0.43 pct</Text>
                </Text>
                <Text className='lr-span'>
                  期间费用率 <Text className='lr-b'>13.26%</Text>
                  <Text className='lr-small'>上年同期 11.24%</Text>
                </Text>
              </View>
            </View>
            <View className='tech-section lr-section'>
              <View className='lr-h3'>ROE 逐级走低</View>
              <View className='fund-roe lr-div'>
                {[
                  ['2024', '17.99%'],
                  ['2025', '13.62%'],
                  ['2026H1 年化', '≈10.9%']
                ].map((r) => (
                  <View key={r[0]} className='lr-div'>
                    <Text className='lr-small'>{r[0]}</Text>
                    <Text className='lr-b'>{r[1]}</Text>
                  </View>
                ))}
              </View>
              <View className='tech-footnote lr-p'>
                2026H1 原始半年 ROE 为 5.46%；年化值沿用报告估算。
              </View>
            </View>
          </>
        )}
        {tab === 1 && (
          <>
            <View className='tech-section lr-section'>
              <View className='lr-h3'>
                增长来自哪里 <Text className='lr-small'>2026H1 · 分地区</Text>
              </View>
              <View className='fund-region-bar lr-div' ariaLabel='境内营收占73.62%，境外占26.38%'>
                <Text className='lr-span'>境内 73.62%</Text>
                <Text className='lr-span'>26.38%</Text>
              </View>
              <View className='fund-inline lr-div'>
                <Text className='lr-span'>
                  境内 <Text className='lr-b'>59.00 亿</Text>
                  <Text className='lr-small'>同比 −1.4%</Text>
                </Text>
                <Text className='lr-span'>
                  境外 <Text className='lr-b'>21.15 亿</Text>
                  <Text className='lr-small'>同比 +43.34%</Text>
                </Text>
              </View>
              <View className='fund-highlight lr-p'>
                座椅业务同比 <Text className='lr-b'>+65.57%</Text>；境外毛利率{' '}
                <Text className='lr-b'>27.6%</Text>，提升 4.8 pct。
              </View>
            </View>
            <View className='tech-section lr-section'>
              <View className='lr-h3'>
                利润兑现门槛 <Text className='lr-small'>一致预期隐含计算</Text>
              </View>
              <View className='fund-inline lr-div'>
                <Text className='lr-span'>
                  最新单季归母增速<Text className='lr-b'>+0.83%</Text>
                  <Text className='lr-small'>2026Q2 · 已实现</Text>
                </Text>
                <Text className='lr-span'>
                  H2 归母所需增速<Text className='lr-b'>+52.8%</Text>
                  <Text className='lr-small'>2026H2 · 隐含要求</Text>
                </Text>
              </View>
              <View className='tech-footnote lr-p'>
                2026E 归母 10.14 亿 − H1 4.14 亿 = H2 需 6.00 亿；2025H2 为 3.93
                亿。实际单季与预期半年属于不同期间。
              </View>
            </View>
            <View className='tech-section lr-section'>
              <View className='lr-h3'>完整财务依据</View>
              {data.finance.map((r) => (
                <View className='tech-disclosure finance-entry lr-div' key={r.name}>
                  <View className='lr-h4'>{r.name}</View>
                  <ReportHtml value={r.body} className='lr-p' />
                </View>
              ))}
            </View>
          </>
        )}
      </View>
    </View>
  );
}
