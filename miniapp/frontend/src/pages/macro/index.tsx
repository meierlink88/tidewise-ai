import { useState } from 'react';
import { Button, Text, View } from '@tarojs/components';
import { useResearchAccess } from '../../features/identity/use-research-access';
import { ResearchAccessState } from '../../features/identity/research-access-state';
import { indices } from '../../features/design-preview/fixtures';
import { PreviewPage, PreviewSheet, PreviewTabs } from '../../features/design-preview/shell';

export default function MacroPage() {
  const access = useResearchAccess('macro');
  const [market, setMarket] = useState(0);
  const [item, setItem] = useState<(typeof indices)[number] | null>(null);
  const markets = ['全部', 'A股', '港股', '美股'];
  if (!access.allowed) return <ResearchAccessState error={access.error} retry={access.retry} />;
  return (
    <PreviewPage
      title='宏观指数'
      subtitle='从市场温度，观察宏观变化'
      filters={<PreviewTabs names={markets} selected={market} onChange={setMarket} />}
    >
      <View className='preview-between'>
        <Text className='preview-heading'>市场概览</Text>
        <Text className='preview-muted'>模拟数据</Text>
      </View>
      {indices
        .filter((x) => market === 0 || x.market === markets[market])
        .map((x) => (
          <Button className='tidewise-button preview-row' key={x.code} onClick={() => setItem(x)}>
            <View>
              <Text className='preview-row-name'>{x.name}</Text>
              <Text className='preview-muted'>{x.code}</Text>
            </View>
            <View className='preview-row-value'>
              <Text className='preview-row-name'>{x.value}</Text>
              <Text className={x.change.startsWith('+') ? 'preview-rise' : 'preview-fall'}>
                {x.change}
              </Text>
            </View>
            <Text className='preview-arrow'>›</Text>
          </Button>
        ))}
      <View className='preview-card'>
        <Text className='preview-heading'>如何阅读指数</Text>
        <Text className='preview-heading'>先看市场，再看估值</Text>
        <Text>
          价格变化帮助识别方向，估值读数提供观察位置。进入指数可切换区间，查看数据展示方式。
        </Text>
      </View>
      <Text className='preview-note'>本板块为功能与视觉模拟，不代表实时行情</Text>
      {item && (
        <PreviewSheet title='指数详情' close={() => setItem(null)}>
          <IndexDetail item={item} />
        </PreviewSheet>
      )}
    </PreviewPage>
  );
}
function IndexDetail({ item }: { item: (typeof indices)[number] }) {
  const [period, setPeriod] = useState(0);
  const labels = [
    ['周一', '周二', '周三', '周四', '周五'],
    ['第1周', '第2周', '第3周', '第4周'],
    ['1月', '2月', '3月', '4月', '5月', '6月']
  ];
  const samples = [
    [3818, 3829, 3807, 3822, 3842],
    [3750, 3792, 3807, 3842],
    [3610, 3675, 3630, 3740, 3792, 3842]
  ];
  const values = samples[period].map((v) => (v * Number(item.value.replace(/,/g, ''))) / 3842);
  return (
    <View>
      <Text className='preview-muted'>设计模拟</Text>
      <Text className='preview-heading'>{item.name}</Text>
      <Text className='preview-muted'>{item.code}</Text>
      <View className='preview-card'>
        <Text className='preview-price'>{item.value}</Text>
        <Text className={item.change.startsWith('+') ? 'preview-rise' : 'preview-fall'}>
          {item.change}
        </Text>
        <Text className='preview-note'>模拟行情 · 不代表实时点位</Text>
      </View>
      <Text className='preview-heading'>区间观察</Text>
      <PreviewTabs names={['近一周', '近一月', '近半年']} selected={period} onChange={setPeriod} />
      <View className='preview-chart' ariaLabel='模拟指数区间走势；完整读数见下方列表'>
        {values.map((v, i) => (
          <View
            key={labels[period][i]}
            className='preview-chart-bar'
            style={{
              height: `${20 + (v / Math.max(...values)) * 75}%`
            }}
          />
        ))}
      </View>
      {values.map((v, i) => (
        <View className='preview-row' key={labels[period][i]}>
          <Text>{labels[period][i]}</Text>
          <Text>{v.toFixed(2)}</Text>
        </View>
      ))}
      <Text className='preview-heading'>估值快照</Text>
      <View className='preview-stats'>
        {[
          ['市盈率 PE', item.pe + '×'],
          ['市净率 PB', item.pb + '×'],
          ['股息率', item.yield]
        ].map(([label, value]) => (
          <View className='preview-stat' key={label}>
            <Text className='preview-muted'>{label}</Text>
            <Text>{value}</Text>
          </View>
        ))}
      </View>
      <Text className='preview-note'>所有点位与估值均为设计样例</Text>
    </View>
  );
}
