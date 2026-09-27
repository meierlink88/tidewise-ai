import { useState } from 'react';
import { Button, Text, View } from '@tarojs/components';
import { chains, companies } from '../../features/design-preview/fixtures';
import { aiEvents } from '../../features/design-preview/events';
import { PreviewPage, PreviewSheet, PreviewTabs } from '../../features/design-preview/shell';
import { PreviewCompanyRow } from '../../features/design-preview/company-row';

export default function IndustryPage() {
  const [active, setActive] = useState(0);
  const [node, setNode] = useState<string[] | null>(null);
  const [events, setEvents] = useState(false);
  const chain = chains[active];
  return (
    <PreviewPage
      title='产业透析'
      subtitle='沿着产业链，理解价值如何传递'
      filters={
        <PreviewTabs names={chains.map((c) => c.name)} selected={active} onChange={setActive} />
      }
    >
      <Text className='preview-muted'>产业链全景</Text>
      <Text className='preview-heading'>{chain.name}</Text>
      <Text>{chain.caption}</Text>
      {active === 0 && (
        <Button className='tidewise-button preview-row' onClick={() => setEvents(true)}>
          <Text>政经事件</Text>
          <Text>›</Text>
        </Button>
      )}
      {chain.nodes.map((n, i) => (
        <Button className='tidewise-button preview-row' key={n[0]} onClick={() => setNode(n)}>
          <Text className='preview-order'>0{i + 1}</Text>
          <View>
            <Text className='preview-muted'>{n[0]}</Text>
            <Text className='preview-row-name'>{n[1]}</Text>
            <Text className='preview-muted'>{n[2]}</Text>
          </View>
          <Text className='preview-arrow'>›</Text>
        </Button>
      ))}
      <View className='preview-card'>
        <Text className='preview-heading'>从环节进入，聚焦关键约束</Text>
        <Text>点击上中下游环节，查看关联公司与该环节的观察方向。</Text>
      </View>
      <Text className='preview-note'>产业结构与说明为设计模拟，非已发布研究</Text>
      {node && (
        <PreviewSheet title={node[1]} close={() => setNode(null)}>
          <Text className='preview-muted'>
            设计模拟 · {chain.name} · {node[0]}
          </Text>
          <Text className='preview-heading'>{node[1]}</Text>
          <Text>{node[2]}</Text>
          <Text className='preview-heading'>环节观察</Text>
          {[
            ['供给约束', '观察产能、关键技术与交付周期的变化。此处用于展示环节分析的阅读结构。'],
            ['需求传导', '关注下游需求变化是否传导到订单与产能利用率。此处尚未接入真实研究。'],
            ['验证线索', '以公司公告、财务披露和行业数据作为后续观察入口。']
          ].map(([label, body]) => (
            <View className='preview-card' key={label}>
              <Text className='preview-heading'>{label}</Text>
              <Text>{body}</Text>
            </View>
          ))}
          <Text className='preview-heading'>关联公司样例</Text>
          {companies
            .filter((c) =>
              (active === 1 ? ['xq', 'byd', 'catl'] : active === 2 ? ['xq', 'midea'] : []).includes(
                c.id
              )
            )
            .map((c) => (
              <PreviewCompanyRow company={c} key={c.id} />
            ))}
          {active === 0 && <Text>当前保留公司样例未覆盖该产业，暂不补造公司关联。</Text>}
        </PreviewSheet>
      )}
      {events && (
        <PreviewSheet title='产业链 · 人工智能' close={() => setEvents(false)}>
          <Text className='preview-muted'>设计模拟 · 事件时间线</Text>
          {aiEvents.map((e, i) => (
            <View key={e.action}>
              <Text className='preview-muted'>{e.time}</Text>
              <View className='preview-event'>
                <View className='preview-event-tags'>
                  <Text>{e.actor}</Text>
                  <Text className='action'>{e.action}</Text>
                  <Text>{e.subject}</Text>
                  {e.metrics.map((m) => (
                    <Text key={m.label}>
                      {m.label} · {m.value} · {m.unit}
                    </Text>
                  ))}
                </View>
                <Text className='preview-event-summary'>{e.summary || '原截图正文未完整提供'}</Text>
                <View className='preview-event-tags'>
                  {e.keywords.map((k) => (
                    <Text key={k}>{k}</Text>
                  ))}
                </View>
              </View>
            </View>
          ))}
        </PreviewSheet>
      )}
    </PreviewPage>
  );
}
