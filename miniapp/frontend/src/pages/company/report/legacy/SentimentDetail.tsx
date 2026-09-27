import { useState } from 'react';
// Presentation port of the approved static company report. No network or account state.
import { Button, Text, View } from '@tarojs/components';

const flows = [
  {
    date: '09.09',
    value: -13675.7
  },
  {
    date: '09.10',
    value: -1639.6
  },
  {
    date: '09.11',
    value: -4058.3
  },
  {
    date: '09.14',
    value: 4151.1
  },
  {
    date: '09.15',
    value: -1116.9
  },
  {
    date: '09.16',
    value: -2354.0
  },
  {
    date: '09.17',
    value: 7147.2
  },
  {
    date: '09.18',
    value: -2703.0
  },
  {
    date: '09.21',
    value: -4798.4
  },
  {
    date: '09.22',
    value: -6490.6
  },
  {
    date: '09.23',
    value: 1928.9
  }
];
const number = (value: number) =>
  `${value > 0 ? '+' : '−'}${Math.abs(value).toLocaleString('en-US', {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1
  })}`;
export default function SentimentDetail() {
  const [selected, setSelected] = useState(10);
  const active = flows[selected];
  return (
    <View className='tech-detail sentiment-detail lr-article'>
      <View className='tech-heading lr-header'>
        <Text className='lr-span'>
          <Text className='lr-strong'>4.0</Text> / 10
        </Text>
        <Text className='lr-b'>中性</Text>
        <Text className='lr-small'>情绪面评估</Text>
      </View>
      <View className='tech-verdict lr-div'>
        <Text className='lr-strong'>评级偏多，资金持续流出</Text>
        <View className='lr-p'>
          机构嘴上买、资金手上卖。单日流入尚未扭转区间流出，筹码趋于分散；陆股通是报告中唯一持续增持的类别。
        </View>
      </View>
      <View className='tech-section lr-section'>
        <View className='lr-h3'>结论依据</View>
        <View className='tech-reasons lr-div'>
          {[
            [
              '01',
              '当日偏多，区间仍流出',
              '委比、内外盘及当日主力净额偏多；但近 10 日主力净流出 9,933.59 万，主动主力当日也为净流出。'
            ],
            [
              '02',
              '评级与实际持仓出现反差',
              '原报告列示卖方评级 20 家次、机构零看空；主动偏股机构 39 家（减少 9 家），公募半年持股减少 34.3%。'
            ],
            [
              '03',
              '筹码趋于分散，增持并非普遍',
              '股东户数半年增加 128.9%，融资余额连续下降；陆股通持续增持是例外。综合判断为 4.0 / 10，中性。'
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
      <View className='tech-metrics sentiment-metrics lr-div'>
        <View className='lr-div'>
          <Text className='lr-span'>近 10 日主力净流出</Text>
          <Text className='lr-strong'>
            9,933.59 <Text className='lr-small'>万</Text>
          </Text>
          <Text className='lr-small'>近 5 日 −4,915.92 万</Text>
        </View>
        <View className='lr-div'>
          <Text className='lr-span'>公募持股 · 较 2025Q4</Text>
          <Text className='lr-strong'>−34.3%</Text>
          <Text className='lr-small'>2026Q2 · 9,103.00 万股</Text>
        </View>
        <View className='lr-div'>
          <Text className='lr-span'>股东户数 · 半年变化</Text>
          <Text className='lr-strong'>+128.9%</Text>
          <Text className='lr-small'>23,156 → 53,014 户</Text>
        </View>
        <View className='lr-div'>
          <Text className='lr-span'>陆股通 · 持股增加</Text>
          <Text className='lr-strong'>511.31 → 969.94</Text>
          <Text className='lr-small'>万股 · 原报告三期序列</Text>
        </View>
      </View>
      <View className='tech-section lr-section'>
        <View className='lr-h3'>
          主力资金逐日净额 <Text className='lr-small'>万元 · NeoData</Text>
        </View>
        <View className='sentiment-chart lr-div'>
          <View className='sentiment-chart-key lr-div'>
            <Text className='positive lr-span'>红 · 净流入</Text>
            <Text className='negative lr-span'>绿 · 净流出</Text>
            <Text className='lr-small'>点选日期看读数</Text>
          </View>
          <View className='sentiment-bars lr-div'>
            {flows.map((flow, i) => (
              <Button
                key={flow.date}
                ariaLabel={`${flow.date} ${flow.value > 0 ? '净流入' : '净流出'} ${Math.abs(flow.value)} 万元`}
                onClick={() => setSelected(i)}
                className={
                  'tidewise-button ' + ('lr-button' + (selected === i ? ' is-selected' : ''))
                }
              >
                <Text className='sentiment-bar-area lr-span'>
                  <Text
                    className={(flow.value > 0 ? 'inflow' : 'outflow') + ' lr-i'}
                    style={{
                      height: `${(Math.abs(flow.value) / 13675.7) * 50}%`
                    }}
                  />
                </Text>
                <Text className='lr-small'>{flow.date.slice(3)}</Text>
              </Button>
            ))}
          </View>
          <View className='sentiment-chart-axis lr-div'>
            <Text className='lr-span'>09.09</Text>
            <Text className='lr-span'>2026 年 9 月</Text>
            <Text className='lr-span'>09.23</Text>
          </View>
          <View className='sentiment-selected lr-div'>
            <Text className='lr-span'>
              {active.date} · {active.value > 0 ? '净流入' : '净流出'}
            </Text>
            <Text className={(active.value > 0 ? 'positive' : 'negative') + ' lr-b'}>
              {number(active.value)} <Text className='lr-small'>万元</Text>
            </Text>
          </View>
        </View>
        <View className='tech-footnote lr-p'>
          原报告逐日序列含 11 个交易日；区间汇总的“近 10 日”是独立口径。本页保留原数，不以这 11
          条直接替代汇总。
        </View>
      </View>
      <View className='tech-section lr-section'>
        <View className='lr-h3'>关键读数</View>
        <View className='sentiment-readings lr-div'>
          <View ariaLabel='情绪面关键读数' className='lr-table'>
            <View className='lr-thead'>
              <View className='lr-tr'>
                <View className='lr-th'>项目</View>
                <View className='lr-th'>值</View>
              </View>
            </View>
            <View className='lr-tbody'>
              <View className='lr-tr'>
                <View className='lr-th'>
                  主力资金<Text className='lr-small'>当日 09.23</Text>
                </View>
                <View className='lr-td'>
                  <View className='lr-p'>
                    NeoData <Text className='positive lr-b'>+1,928.88 万</Text> / 同花顺{' '}
                    <Text className='positive lr-b'>+1,796.43 万</Text>（差 +7.4%，
                    <Text className='lr-b'>方向一致</Text>）。
                  </View>
                  <View className='lr-p'>
                    但主动主力净流入 <Text className='lr-b'>44,962,149 − 48,769,919 = −381 万</Text>{' '}
                    → 报告解释为<Text className='lr-b'>净流入来自被动挂单</Text>。
                  </View>
                </View>
              </View>
              <View className='lr-tr'>
                <View className='lr-th'>
                  主力资金<Text className='lr-small'>区间</Text>
                </View>
                <View className='lr-td'>
                  <View className='lr-p'>
                    近 5 日 <Text className='lr-b'>−4,915.92 万</Text>
                    <View className='lr-br' />近 10 日 <Text className='lr-b'>−9,933.59 万</Text>
                    <View className='lr-br' />近 20 日 <Text className='lr-b'>≈ −1.10 亿</Text>
                    （0.38% × 292.2 亿）。
                  </View>
                  <View className='lr-p'>原报告注明：逐日加总与 NeoData 字段逐分吻合。</View>
                </View>
              </View>
              <View className='lr-tr'>
                <View className='lr-th'>散户镜像</View>
                <View className='lr-td'>
                  散户与主力<Text className='lr-b'>完全镜像</Text> → 近 10 日散户净买入{' '}
                  <Text className='lr-b'>≈ +0.99 亿</Text>。
                </View>
              </View>
              <View className='lr-tr'>
                <View className='lr-th'>机构持仓</View>
                <View className='lr-td'>
                  <View className='lr-p'>
                    公募 <Text className='lr-b'>138,514,038 股</Text>（25Q4，+3.41pp，426 家）
                    <View className='lr-br' />→ <Text className='lr-b'>59,443,212 股</Text>
                    （26Q1，−0.88pp，50 家）
                    <View className='lr-br' />→ <Text className='lr-b'>91,030,002 股</Text>
                    （26Q2，−1.74pp，350 家）。
                  </View>
                  <View className='lr-p'>
                    主动偏股 <Text className='lr-b'>39 家（−9）</Text>；
                    <Text className='lr-b'>7.69%（−3.96pp）</Text>。
                  </View>
                </View>
              </View>
              <View className='lr-tr'>
                <View className='lr-th'>
                  北向 /<View className='lr-br' />
                  陆股通
                </View>
                <View className='lr-td'>
                  <Text className='lr-b'>5,113,118 → 6,735,109 → 9,699,442 股</Text>
                  （唯一持续增持类别）。
                </View>
              </View>
              <View className='lr-tr'>
                <View className='lr-th'>股东户数</View>
                <View className='lr-td'>
                  <View className='lr-p'>
                    <Text className='lr-b'>23,156</Text>（25Q4）→{' '}
                    <Text className='lr-b'>29,530</Text>（26.02）→{' '}
                    <Text className='lr-b'>35,330</Text>（26Q1）→{' '}
                    <Text className='lr-b'>53,014</Text>（26Q2）。
                  </View>
                  <View className='lr-p'>
                    环比 <Text className='lr-b'>+50.05%</Text>，半年{' '}
                    <Text className='lr-b'>+128.9%</Text>。
                  </View>
                </View>
              </View>
              <View className='lr-tr'>
                <View className='lr-th'>融资融券</View>
                <View className='lr-td'>
                  <View className='lr-p'>
                    融资余额 <Text className='lr-b'>22.77 亿</Text>（09.09）→{' '}
                    <Text className='lr-b'>22.16 亿</Text>（09.22），
                    <Text className='lr-b'>−2.7%，3 连降</Text>；融资余额 / 流通市值{' '}
                    <Text className='lr-b'>≈ 7.58%</Text>（偏高）。
                  </View>
                  <View className='lr-p'>
                    融券余量 232,032 → <Text className='lr-b'>156,372 股</Text>（约 631 万元，占市值
                    0.02%）。
                  </View>
                </View>
              </View>
              <View className='lr-tr'>
                <View className='lr-th'>盘口</View>
                <View className='lr-td'>
                  <View className='lr-p'>
                    09.23 <Text className='lr-b'>委比 +74.08%</Text>，外盘 78,370 ＞ 内盘 71,037。
                  </View>
                  <View className='lr-p'>
                    Wind 分钟线：
                    <Text className='lr-b'>09:40–09:55 拉升至 41.70 后单边回落至 40.08</Text>。
                  </View>
                </View>
              </View>
              <View className='lr-tr'>
                <View className='lr-th'>其他</View>
                <View className='lr-td'>
                  龙虎榜：近期<Text className='lr-b'>未上榜</Text>；大宗交易：
                  <Text className='lr-b'>无记录</Text>；2027.08.19 解禁 838.40 万股（1.14%）。
                </View>
              </View>
            </View>
          </View>
        </View>
      </View>
    </View>
  );
}
