import { useState } from 'react';
// Presentation port of the approved static company report. No network or account state.
import { Button, Image, Text, View } from '@tarojs/components';
import priceLadderImage from '../../../../assets/report-price-ladder.svg';
import { Disclosure } from './primitives';

const resistances = [
  ['R0', '40.92', 'SMA60 · 当前检验位'],
  ['R1', '41.70', '当日盘中高点'],
  ['R2', '42.15–42.39', '三口径重合阻力带'],
  ['R3', '43.00', '阻力位'],
  ['R4', '44.13', '9 月反弹失败高点'],
  ['R5', '46.20–48.00', '密集套牢区'],
  ['R6', '53.47', '近 60 日最高收盘']
];
const supports = [
  ['S1', '40.34', 'SMA5'],
  ['S2', '40.06–40.08', '观察层 · 约 0.40×ATR'],
  ['S3', '39.64', 'SMA20 / BOLL 中轨'],
  ['S4', '39.40–39.42', '支撑带'],
  ['S5', '39.03', 'SMA10'],
  ['S6', '38.59–38.78', '38.59 为原报告风险边界'],
  ['S7', '36.63–36.89', '含 BOLL 下轨'],
  ['S8', '36.30–36.75', '含 09.16 低点'],
  ['S9', '35.05', '近 60 日最低收盘'],
  ['S10', '34.48', '全样本最低价']
];
const averages = [
  ['SMA60', 40.92],
  ['SMA5', 40.34],
  ['SMA20', 39.64],
  ['SMA50', 39.446],
  ['SMA10', 39.03]
] as const;
function Group({ title, rows }: { title: string; rows: string[][] }) {
  return (
    <View className='tech-section lr-section'>
      <View className='lr-h3'>{title}</View>
      <View className='tech-table lr-div'>
        {rows.map((r) => (
          <View key={r[0]} className='lr-div'>
            <Text className='lr-span'>{r[0]}</Text>
            <Text className='lr-strong'>{r[1]}</Text>
            <Text className='lr-small'>{r[2]}</Text>
          </View>
        ))}
      </View>
    </View>
  );
}
function PriceLadder() {
  const [active, setActive] = useState('R0');
  const nearby = [
    resistances[2],
    resistances[1],
    resistances[0],
    supports[0],
    supports[2],
    supports[5]
  ];
  const selected = [...resistances, ...supports].find((r) => r[0] === active)!;
  const y = (v: number) => 22 + ((42.6 - v) / 4.2) * 300;
  return (
    <View className='price-ladder-section lr-section'>
      <View className='lr-h3'>
        支撑与阻力地图 <Text className='lr-small'>同日价位 · 元</Text>
      </View>
      <View className='price-ladder-legend lr-div'>
        <Text className='lr-span'>● 阻力</Text>
        <Text className='lr-span'>● 支撑</Text>
        <Text className='lr-span'>━ 收盘价</Text>
      </View>
      <View className='price-ladder lr-div'>
        <Image src={priceLadderImage} className='lr-svg' mode='widthFix' />
        {nearby.map((r) => {
          const v = r[1].split('–').map(Number);
          const mid = (y(v[0]) + y(v[v.length - 1])) / 2;
          return (
            <Button
              key={r[0]}
              className={
                'tidewise-button ' +
                ('ladder-node ' +
                  (r[0][0] === 'R' ? 'resistance' : 'support') +
                  (active === r[0] ? ' active' : '') +
                  ' lr-button' +
                  (active === r[0] ? ' is-selected' : ''))
              }
              style={{
                top: `${((r[0] === 'R0' ? mid - 13 : mid) / 344) * 100}%`
              }}
              onClick={() => setActive(r[0])}
              ariaLabel={`${r[0]} ${r[1]} 价位依据`}
            >
              <Text className='lr-span'>{r[0]}</Text>
              <Text className='lr-b'>{r[1]}</Text>
            </Button>
          );
        })}
      </View>
      <View className='ladder-explanation lr-div'>
        <Text className='lr-span'>
          {selected[0][0] === 'R' ? '阻力' : '支撑'} · {selected[0]}
        </Text>
        <Text className='lr-strong'>{selected[1]}</Text>
        <View className='lr-p'>{selected[2]}</View>
      </View>
    </View>
  );
}
export default function TechnicalDetail() {
  const [tab, setTab] = useState(0);
  const [selected, setSelected] = useState(0);
  return (
    <View className='tech-detail lr-article'>
      <View className='tech-heading lr-header'>
        <Text className='lr-span'>
          <Text className='lr-strong'>5.0</Text> / 10
        </Text>
        <Text className='lr-b'>中性观察</Text>
        <Text className='lr-small'>技术面评分</Text>
      </View>
      <View className='tech-verdict lr-div'>
        <Text className='lr-strong'>中期下跌趋势内的短期反抽</Text>
        <View className='lr-p'>
          正在 SMA60 处接受检验，盘中上穿、收盘未确认，尚不构成趋势反转证据。
        </View>
      </View>
      <View className='tech-section lr-section'>
        <View className='lr-h3'>结论依据</View>
        <View className='tech-reasons lr-div'>
          {[
            [
              '01',
              '短期修复已经出现',
              '价格站上 SMA5 / 10 / 20 / 50，MACD 底部修复，近 5 日上涨 11.32%。'
            ],
            [
              '02',
              '中期趋势仍未扭转',
              'SMA60 斜率 −0.628，仍在下行；盘中最高 41.70 曾上穿，但收盘 40.91 低于 SMA60 40.92。'
            ],
            [
              '03',
              '量价尚不支持突破确认',
              'J 值 92.58 处于过热状态，成交量仅为 20 日均量的 0.75 倍，属于压力位前的缩量上冲。'
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
          ['收盘 / SMA60', '40.91 / 40.92', '仅差 0.01 元 · 突破未确认'],
          ['SMA60 斜率', '−0.628', '近 5 日仍下行'],
          ['KDJ · J 值', '92.58', '短期过热'],
          ['成交量 / 20 日均量', '0.75×', '压力位前缩量']
        ].map((r) => (
          <View key={r[0]} className='lr-div'>
            <Text className='lr-span'>{r[0]}</Text>
            <Text className='lr-strong'>{r[1]}</Text>
            <Text className='lr-small'>{r[2]}</Text>
          </View>
        ))}
      </View>
      <View className='tech-tabs lr-div' ariaLabel='技术面详情'>
        {['关键读数', '阻力与支撑'].map((t, i) => (
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
                价格与均线位置 <Text className='lr-small'>同日快照 · 元</Text>
              </View>
              <View className='tech-average-chart lr-div'>
                <View className='tech-chart-caption lr-div'>
                  <Text className='lr-span'>39.00</Text>
                  <Text className='lr-b'>收盘 40.91</Text>
                  <Text className='lr-span'>41.00</Text>
                </View>
                {averages.map(([n, v], i) => (
                  <Button
                    key={n}
                    className={
                      'tidewise-button ' +
                      ((selected === i ? 'selected' : '') +
                        ' lr-button' +
                        (selected === i ? ' is-selected' : ''))
                    }
                    onClick={() => setSelected(i)}
                    ariaLabel={`${n} ${v}`}
                  >
                    <Text className='lr-span'>{n}</Text>
                    <View className='tech-track lr-div'>
                      <Text className='tech-current lr-i' />
                      <Text
                        className='tech-dot lr-i'
                        style={{
                          left: `${((v - 39) / 2) * 100}%`
                        }}
                      />
                    </View>
                    <Text className='lr-b'>{v.toFixed(n === 'SMA50' ? 3 : 2)}</Text>
                  </Button>
                ))}
                <View className='tech-chart-note lr-div'>
                  {averages[selected][0]} <Text className='lr-strong'>{averages[selected][1]}</Text>{' '}
                  · 收盘{40.91 < averages[selected][1] ? '低于' : '高于'}该均线{' '}
                  <Text className='lr-strong'>
                    {Math.abs(40.91 - averages[selected][1])
                      .toFixed(3)
                      .replace(/0$/, '')}
                  </Text>{' '}
                  元
                </View>
              </View>
              <View className='tech-footnote lr-p'>
                非标准多头排列：SMA60 ＞ SMA5 ＞ SMA20 ＞ SMA50 ＞ SMA10。
              </View>
            </View>
            <Group
              title='动能修复，量能未跟上'
              rows={[
                [
                  'MACD',
                  'DIF +0.032 / DEA −0.263',
                  '柱 +0.589 · 09.18 零轴下金叉，09.23 DIF 上穿零轴'
                ],
                ['RSI / KDJ', 'RSI 54.46 · J 92.58', 'K 64.75 / D 50.83 · J 值过热'],
                ['BOLL', '36.89 / 39.64 / 42.39', '下轨 / 中轨 / 上轨'],
                ['波动与量能', 'ATR 2.12 · 5.18%', '量比 0.798 · VWMA20 39.906'],
                ['BIAS', '+2.94% / +4.19% / +4.16%', '6 日 / 12 日 / 24 日']
              ]}
            />
            <View className='tech-section lr-section'>
              <View className='lr-h3'>区间表现</View>
              <View className='tech-returns lr-div'>
                {[
                  ['5 日', '+11.32%'],
                  ['10 日', '+2.69%'],
                  ['20 日', '+9.04%'],
                  ['60 日', '−17.98%'],
                  ['年初至今', '−22.18%']
                ].map((r) => (
                  <View key={r[0]} className='lr-div'>
                    <Text className='lr-span'>{r[0]}</Text>
                    <Text className={(r[1][0] === '+' ? 'positive' : 'negative') + ' lr-b'}>
                      {r[1]}
                    </Text>
                  </View>
                ))}
              </View>
              <Disclosure className='tech-disclosure lr-details'>
                <View className='lr-summary'>价格极值与样本范围</View>
                <View className='lr-p'>
                  近 60 日最高收盘 <Text className='lr-b'>53.47</Text>（07.03），最低收盘{' '}
                  <Text className='lr-b'>35.05</Text>（08.20）。全区间最高价{' '}
                  <Text className='lr-b'>64.98</Text>（05.22），距高点 −37.04%；全样本最低价{' '}
                  <Text className='lr-b'>34.48</Text>（08.20）。
                </View>
              </Disclosure>
            </View>
          </>
        )}
        {tab === 1 && (
          <>
            <PriceLadder />
            <View className='tech-caution lr-div'>
              <Text className='lr-b'>观察位与风险边界分开阅读</Text>
              <View className='lr-p'>
                S2 距现价约 0.40×ATR，原报告归入噪音观察层；S3 39.64（约
                0.60×ATR）作为加速确认位，38.59（约 1.09×ATR）为剩余仓位风险边界。
              </View>
            </View>
          </>
        )}
      </View>
    </View>
  );
}
