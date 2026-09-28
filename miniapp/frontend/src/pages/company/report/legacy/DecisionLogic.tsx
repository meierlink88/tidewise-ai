import { useState } from 'react';
// Presentation port of the approved static company report. No network or account state.
import { Button, Text, View } from '@tarojs/components';
import { ReportHtml, Disclosure } from './primitives';
import { ChevronDownIcon, DotFilledIcon, ChevronRightIcon } from './icons';
import logic from '../../../../features/design-preview/report-data/decision-logic.json';

const metrics = [
  'H1 归母 −1.90% · 费用率 +2.02pct',
  'H2 隐含增速 +52.8% · 最新单季 +0.83%',
  '10日主力 −9,933.59万 · 公募持股 −34.3%',
  '近5日 +11.32% · 量比 0.798',
  '中报：尚未开展业务 · 未在审厂名单',
  '休市8天 + 三季报窗口 · 融资占比 ≈7.58%'
];
function SectionTitle({ n, title }: { n: string; title: string }) {
  return (
    <View className='logic-section-title lr-h2'>
      <Text className='lr-span'>{n}</Text>
      {title}
    </View>
  );
}
const priceGroups = [
  {
    name: '执行与风控',
    indices: [1, 0, 2, 3, 4]
  },
  {
    name: '目标区间',
    indices: [11, 10, 9, 8, 5, 6, 7]
  },
  {
    name: '失效条件',
    indices: [14, 13, 12]
  }
];
function plain(html: string) {
  return html.replace(/<[^>]+>/g, '');
}
export function PriceOverview({ execution = false }: { execution?: boolean }) {
  const [group, setGroup] = useState(0);
  const [selected, setSelected] = useState(0);
  const levels = logic.levels.map((level, i) => ({
    ...level,
    description:
      level.description +
      (execution
        ? ((
            {
              0: '；2026-09-30 收盘前完成至目标仓。',
              1: '；反弹档若发生，为可选优化，非主路径。',
              3: '；时间窗：至 09-30。',
              4: '；时间窗：至 10-23；硬止损条件为收盘跌破 38.59。',
              5: '；时间窗：至 12-23。',
              8: '；时间窗：至 09-30。',
              9: '；时间窗：至 10-23。',
              10: '；时间窗：至 10-23。',
              11: '；长尾情形，时间窗：至 12-23。'
            } as Record<number, string>
          )[i] ?? '')
        : '')
  }));
  const active = levels[selected];
  return (
    <View className='logic-panel lr-section'>
      <SectionTitle n='01' title='关键价位速览' />
      <View className='price-current-banner lr-div'>
        <Text className='lr-span'>
          现价基准
          <Text className='lr-strong'>
            40.91 <Text className='lr-small'>元</Text>
          </Text>
        </Text>
        <View className='lr-div'>
          <Text className='lr-b'>现价即减</Text>
          <Text className='lr-small'>不设价格门槛</Text>
        </View>
      </View>
      <View className='price-group-tabs lr-div' ariaLabel='价位分类'>
        {priceGroups.map((g, i) => (
          <Button
            key={g.name}
            onClick={() => {
              setGroup(i);
              setSelected(i === 0 ? 0 : g.indices[0]);
            }}
            className={'tidewise-button ' + ('lr-button' + (group === i ? ' is-selected' : ''))}
          >
            {g.name}
          </Button>
        ))}
      </View>
      <View className='price-map-caption lr-div'>
        <Text className='lr-span'>类型</Text>
        <Text className='lr-span'>价位 / 元</Text>
      </View>
      <View className='price-navigation lr-div'>
        {priceGroups[group].indices.map((i) => {
          const l = levels[i];
          const tone =
            i === 0
              ? 'now'
              : [1, 8, 9, 10, 12, 13, 14].includes(i)
                ? 'rise'
                : [3, 4, 5, 6].includes(i)
                  ? 'fall'
                  : 'neutral';
          return (
            <Button
              key={i}
              className={
                'tidewise-button ' +
                ('price-node ' +
                  tone +
                  (selected === i ? ' selected' : '') +
                  ' lr-button' +
                  (selected === i ? ' is-selected' : ''))
              }
              onClick={() => setSelected(i)}
            >
              <DotFilledIcon />
              <Text className='lr-span'>{plain(l.name)}</Text>
              <Text className='lr-strong'>
                {plain(l.price).replace('现价 ', '').replace(' 附近', '')}
              </Text>
              <ChevronRightIcon />
            </Button>
          );
        })}
      </View>
      <View className='selected-price-note lr-div'>
        <Text className='lr-span'>价位说明</Text>
        <ReportHtml value={active.name} className='lr-h3' />
        <ReportHtml value={active.description} className='lr-p' />
      </View>
    </View>
  );
}
export default function DecisionLogic() {
  return (
    <View className='decision-logic lr-div'>
      <View className='logic-panel lr-section'>
        <SectionTitle n='01' title='核心决策因素' />
        <View className='factor-list lr-div'>
          {logic.factors.map((f, i) => (
            <Disclosure className='factor lr-details' key={f.n}>
              <View className='lr-summary'>
                <Text className='factor-number lr-span'>{f.n}</Text>
                <View className='lr-div'>
                  <ReportHtml value={f.title} className='lr-h3' />
                  <View className='lr-p'>{metrics[i]}</View>
                  <Text className='factor-type lr-span'>{f.tag.replace(/[（）]/g, '')}</Text>
                </View>
                <ChevronDownIcon />
              </View>
              <ReportHtml className='factor-body lr-div' value={f.body} />
            </Disclosure>
          ))}
        </View>
      </View>
      <View className='logic-panel lr-section'>
        <SectionTitle n='02' title='反证理由' />
        {logic.rejections.map((r) => (
          <View className='rejection lr-div' key={r.title}>
            <View className='lr-h3'>{r.title}</View>
            <ReportHtml value={r.body} className='lr-div' />
          </View>
        ))}
      </View>
    </View>
  );
}
