import { useState } from 'react';
// Presentation port of the approved static company report. No network or account state.
import { Button, Text, View } from '@tarojs/components';
import { Disclosure } from './primitives';

const events = [
  {
    date: '09.17',
    kind: '行业',
    title: '约 5,000 台机器人订单',
    body: '报告记录特斯拉下发约 5,000 台订单；属于行业催化，不能归为新泉公司订单。'
  },
  {
    date: '09.16—17',
    kind: '行业',
    title: '机器人团队落地宁波审厂',
    body: '报告列示覆盖拓普、三花、均胜；新泉未出现在该审厂名单。'
  },
  {
    date: '09.10—23',
    kind: '公告',
    title: '报告检索窗口内无新增公告',
    body: '原报告称官方与第三方双通道均未见新增，最新可检索公告为 09.09。此处沿用报告当时的检索结论。'
  },
  {
    date: '09.09',
    kind: '公告',
    title: '董事会会议与担保事项',
    body: '第五届董事会第二十三次会议：为美国集团提供不超过 3 亿担保，并变更注册资本。'
  },
  {
    date: '09.08',
    kind: '评级',
    title: '跟踪观点维持积极',
    body: '原报告列示机构跟踪评级“强推”，目标价 57.29 元；为机构观点。'
  },
  {
    date: '09.01',
    kind: '评级',
    title: '德邦证券首次覆盖',
    body: '原评级为“买入”，未给目标价；此处为来源原评级，不是本页建议。'
  },
  {
    date: '08.27',
    kind: '评级',
    title: '方正证券积极评级',
    body: '原评级为“强烈推荐”。'
  },
  {
    date: '08.21',
    kind: '公告',
    title: '股权激励首次授予公告',
    body: '740 人 / 2,096 万股，授予价 25.91 元，募集 5.43 亿补流。授予日为 07.02，登记完成日为 08.19。'
  },
  {
    date: '08.07',
    kind: '评级',
    title: '维持推荐评级',
    body: '沿用报告列示的评级记录。'
  },
  {
    date: '08.04',
    kind: '公告',
    title: '半年报发布',
    body: '2026 中报明文披露机器人业务“尚未开展业务”。'
  }
];
const robot = [
  ['2025.06', '投资墨的智能', '1,600 万 · 16%'],
  ['2025.12.09', '机器人子公司成立', '常州新泉智能机器人 · 注册资本 1 亿'],
  ['2026.01.26', '与凯迪战略合作', '合作事项已发生'],
  ['2026.05', '增资中科摩通', '2,000 万 · 2.0636%'],
  ['2026.07', '转让新泉机器人股权', '向凯迪转让 30% · 3,000 万']
];
export default function NewsDetail() {
  const [tab, setTab] = useState(0);
  const [filter, setFilter] = useState('全部');
  return (
    <View className='tech-detail news-detail lr-article'>
      <View className='tech-heading lr-header'>
        <Text className='lr-span'>
          <Text className='lr-strong'>6.5</Text> / 10
        </Text>
        <Text className='lr-b'>偏多（正面）</Text>
        <Text className='lr-small'>新闻面评估</Text>
      </View>
      <View className='tech-verdict lr-div'>
        <Text className='lr-strong'>行业催化密集，公司兑现待证</Text>
        <View className='lr-p'>
          机构评级偏积极，公告面未见负面；行业热度尚不能直接证明公司的业务兑现。
        </View>
      </View>
      <View className='tech-section lr-section'>
        <View className='lr-h3'>结论依据</View>
        <View className='tech-reasons lr-div'>
          {[
            [
              '01',
              '积极评级不等于深入覆盖',
              '近 6 月评级 20 家次，调研仅 1 次；原报告据此判断评级密度与调研强度不匹配。'
            ],
            [
              '02',
              '公告空窗本身是边际信息',
              '09.10—09.23 未见新增公告，尚无公司机器人订单或定点公告支持业务兑现。'
            ],
            [
              '03',
              '行业催化与公司证据分开',
              '审厂与约 5,000 台订单属于行业层面。报告认为反弹以行业因素为主，新泉超额约 2–4.6 pct。'
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
      <View className='tech-tabs lr-div' ariaLabel='新闻面详情'>
        {['事件时间轴', '业务进展'].map((t, i) => (
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
                消息来自哪里 <Text className='lr-small'>2026 年 · 按日期倒序</Text>
              </View>
              <View className='news-filters lr-div' ariaLabel='事件类型'>
                {['全部', '公告', '评级', '行业'].map((f) => (
                  <Button
                    key={f}
                    onClick={() => setFilter(f)}
                    className={
                      'tidewise-button ' + ('lr-button' + (filter === f ? ' is-selected' : ''))
                    }
                  >
                    {f}
                  </Button>
                ))}
              </View>
              <View className='news-timeline lr-div'>
                {events
                  .filter((e) => filter === '全部' || e.kind === filter)
                  .map((e) => (
                    <View className='news-event lr-div' key={e.date + e.kind}>
                      <View className='news-event-heading lr-div'>
                        <Text className='lr-time'>{e.date}</Text>
                        <Text className='lr-span'>
                          <Text
                            className={
                              'news-tag lr-em ' +
                              (e.kind === '行业'
                                ? 'kind-industry'
                                : e.kind === '评级'
                                  ? 'kind-rating'
                                  : 'kind-announcement')
                            }
                          >
                            {e.kind}
                          </Text>
                          <Text className='lr-b'>{e.title}</Text>
                        </Text>
                      </View>
                      <View className='lr-p'>{e.body}</View>
                    </View>
                  ))}
              </View>
            </View>
            <View className='tech-section lr-section'>
              <View className='lr-h3'>
                股权激励考核 <Text className='lr-small'>营业收入 · 亿元</Text>
              </View>
              <View className='news-targets lr-div'>
                <View className='lr-div'>
                  <Text className='lr-span'>年度</Text>
                  <Text className='lr-span'>目标值</Text>
                  <Text className='lr-span'>触发值</Text>
                </View>
                {[
                  ['2026', '200', '180'],
                  ['2027', '240', '216'],
                  ['2028', '300', '270']
                ].map((r) => (
                  <View key={r[0]} className='lr-div'>
                    {r.map((c) => (
                      <Text key={c} className='lr-b'>
                        {c}
                      </Text>
                    ))}
                  </View>
                ))}
              </View>
            </View>
            <View className='tech-footnote lr-p'>
              “近 1 月”标签沿用原报告；所列记录包含 08.04、08.07，时间覆盖实际更宽。
            </View>
          </>
        )}
        {tab === 1 && (
          <>
            <View className='news-boundary lr-div'>
              <Text className='lr-b'>关键边界：布局不等于收入兑现</Text>
              <View className='lr-p'>
                报告列示新泉未出现在 Optimus 审厂名单，2026 中报披露“尚未开展业务”。
              </View>
            </View>
            <View className='tech-section lr-section'>
              <View className='lr-h3'>机器人业务已发生事项</View>
              <View className='news-business lr-div'>
                {robot.map((r) => (
                  <View key={r[0]} className='lr-div'>
                    <Text className='lr-time'>{r[0]}</Text>
                    <Text className='lr-b'>{r[1]}</Text>
                    <View className='lr-p'>{r[2]}</View>
                  </View>
                ))}
              </View>
            </View>
            <View className='tech-section lr-section'>
              <View className='lr-h3'>尚未获公告证实</View>
              <View className='tech-caution lr-div'>
                <Text className='lr-b'>“批量交付 / 月出货 &gt;500 套”</Text>
                <View className='lr-p'>
                  2026.09.04 微信公众号信息，原报告标为弱源，不纳入事实层。
                </View>
              </View>
              <View className='tech-source lr-div'>
                <Text className='lr-b'>“推进谐波减速器量产”</Text>
                <View className='lr-p'>来自方正 2026.01.17 转述，仍属于预期性描述。</View>
              </View>
            </View>
            <View className='tech-section lr-section'>
              <View className='lr-h3'>其他公司事项</View>
              <Disclosure className='tech-disclosure lr-details'>
                <View className='lr-summary'>H 股申请 · 2026.01.27 递交</View>
                <View className='lr-p'>截至本报告时间，近 8 个月无聆讯进展更新。</View>
              </Disclosure>
              <Disclosure className='tech-disclosure lr-details'>
                <View className='lr-summary'>德国拜仁 · 增资 1 亿美元</View>
                <View className='lr-p'>
                  报告列示预计 2026.09 投产，新增 25 万套；投产时间为预期，并非已投产确认。
                </View>
              </Disclosure>
            </View>
          </>
        )}
      </View>
    </View>
  );
}
