import { Button, Image, ScrollView, Text, View } from '@tarojs/components';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  analysisKinds,
  analysisLabels,
  type AnalysisKind,
  type AnalysisGroup,
  type AnalysisSummary
} from '../../features/reports/normalized-contract';
import type { ReportHomeGroup } from '../../features/reports/contract';
import type { ReportDetailRoute, ReportEvidenceRoute } from '../../features/reports/navigation';
import { getReportPort } from '../../features/reports/port';
import globeIcon from '../../assets/icons/report-globe.svg';
import macroIcon from '../../assets/icons/report-bar-chart.svg';
import chainIcon from '../../assets/icons/report-link.svg';
import './normalized-home.scss';

const categoryIcons: Record<AnalysisKind, string> = {
  geopolitical_stories: globeIcon,
  macroeconomic_stories: macroIcon,
  concept_analyses: chainIcon
};

const directionLabels = { warming: '升温', cooling: '降温', diverging: '分化', pending: '待验证' };
const impactLabels: Record<string, string> = {
  high: '高影响',
  medium: '中影响',
  low: '低影响',
  pending: '待评估'
};
export function NormalizedHome({
  group,
  query,
  onDetail,
  onEvidence
}: {
  group: ReportHomeGroup;
  query: string;
  onDetail: (r: ReportDetailRoute) => void;
  onEvidence: (r: ReportEvidenceRoute) => void;
}) {
  const [kind, setKind] = useState<AnalysisKind>('geopolitical_stories');
  const [pages, setPages] = useState<Partial<Record<AnalysisKind, AnalysisGroup>>>({});
  const [pending, setPending] = useState<AnalysisKind | null>(null),
    [failed, setFailed] = useState<AnalysisKind | null>(null);
  const generation = useRef(0),
    inflight = useRef(false);
  const port = useMemo(() => getReportPort(), []);
  useEffect(() => {
    const version = ++generation.current;
    setPages({});
    setPending(null);
    setFailed(null);
    inflight.current = false;
    return () => {
      generation.current = version + 1;
    };
  }, [group]);
  const current = pages[kind] ?? group.analysisGroups?.find((g) => g.kind === kind);
  const items = (current?.items ?? []).filter(
    (x) =>
      !query ||
      [x.title, x.summary.conclusion, ...x.affected_anchors.map((a) => a.name)]
        .join(' ')
        .includes(query)
  );
  const load = async () => {
    if (inflight.current || !current?.next_cursor) return;
    inflight.current = true;
    const g = generation.current,
      k = kind;
    setPending(k);
    setFailed(null);
    try {
      const page = await port.getAnalyses(group.report.id, k, current.next_cursor);
      if (g !== generation.current) return;
      const keys = new Set(current.items.map((x) => x.local_key));
      setPages((p) => ({
        ...p,
        [k]: {
          kind: k,
          items: [...current.items, ...page.items.filter((x) => !keys.has(x.local_key))],
          next_cursor: page.next_cursor
        }
      }));
    } catch {
      if (g === generation.current) setFailed(k);
    } finally {
      if (g === generation.current) {
        inflight.current = false;
        setPending(null);
      }
    }
  };
  return (
    <View className='normalized-home'>
      <View className='normalized-home-navigation'>
        <ScrollView scrollX className='normalized-home-tabs-scroll'>
          <View className='normalized-home-tabs'>
            {analysisKinds.map((k) => (
              <Button
                key={k}
                className={`tidewise-button normalized-home-tab ${kind === k ? 'selected' : ''}`}
                onClick={() => setKind(k)}
              >
                <View className='normalized-home-tab-icon'>
                  <Image
                    src={categoryIcons[k]}
                    mode='scaleToFill'
                    className='normalized-home-tab-image'
                  />
                </View>
                <Text>{analysisLabels[k]}</Text>
              </Button>
            ))}
          </View>
        </ScrollView>
        <View className='normalized-home-heading'>
          <Text>今日观潮</Text>
          <Text>{items.length} 条结论</Text>
        </View>
      </View>
      <ScrollView key={kind} scrollY className='normalized-home-scroll'>
        <View className='normalized-home-list'>
          {items.map((u) => (
            <HomeCard
              key={u.local_key}
              u={u}
              publishedAt={group.report.publishedAt}
              onDetail={() =>
                onDetail({ reportId: group.report.id, targetType: kind, targetKey: u.local_key })
              }
              onEvidence={() => {
                if (u.summary.evidence_scope_token)
                  onEvidence({
                    reportId: group.report.id,
                    scopeToken: u.summary.evidence_scope_token,
                    title: u.title
                  });
              }}
            />
          ))}
          {!items.length ? (
            <View className='normalized-home-empty'>
              {query ? '暂无匹配的结论' : '本期暂无相关结论'}
            </View>
          ) : null}
          {current?.next_cursor ? (
            <Button
              className='tidewise-button normalized-home-more'
              disabled={pending !== null}
              onClick={() => void load()}
            >
              {pending === kind ? '正在加载…' : failed === kind ? '加载失败，点击重试' : '加载更多'}
            </Button>
          ) : null}
        </View>
      </ScrollView>
    </View>
  );
}
function HomeCard({
  u,
  publishedAt,
  onDetail,
  onEvidence
}: {
  u: AnalysisSummary;
  publishedAt: string;
  onDetail: () => void;
  onEvidence: () => void;
}) {
  const date = new Date(new Date(publishedAt).getTime() + 8 * 60 * 60 * 1000);
  const time = `${String(date.getUTCMonth() + 1).padStart(2, '0')}.${String(date.getUTCDate()).padStart(2, '0')} ${String(date.getUTCHours()).padStart(2, '0')}:${String(date.getUTCMinutes()).padStart(2, '0')}`;
  return (
    <View className='normalized-home-card'>
      <View className='normalized-card-meta'>
        <View className='normalized-card-story'>
          <Text>{u.title}</Text>
          <Text className={`normalized-impact ${u.summary.impact_assessment.level}`}>
            {impactLabels[u.summary.impact_assessment.level]}
          </Text>
        </View>
        <Text className='normalized-card-time'>{time} 发布</Text>
      </View>
      <Text className='normalized-card-conclusion'>{u.summary.conclusion}</Text>
      <Text className='normalized-card-logic'>{u.summary.transmission_logic}</Text>
      <View className='normalized-anchor-area'>
        <View className='normalized-anchor-count'>
          <Text>{u.affected_anchors.length}</Text>
          <Text>个受影响锚点</Text>
        </View>
        <View className='normalized-anchor-chips'>
          {u.affected_anchors.map((a) => (
            <View
              className='normalized-anchor-chip'
              key={`${a.reference.chain_local_key ?? ''}:${a.reference.local_key}`}
            >
              <Text>{a.name}</Text>
              <Text className={`normalized-anchor-direction ${a.assessment.direction}`}>
                {directionLabels[a.assessment.direction]}
              </Text>
            </View>
          ))}
        </View>
      </View>
      <View className='normalized-card-footer'>
        <Button
          className='tidewise-button normalized-card-evidence'
          ariaLabel={`查看${u.title}证据`}
          disabled={!u.summary.evidence_scope_token}
          onClick={onEvidence}
        >
          {u.summary.evidence_count} 条事件
        </Button>
        <Button className='tidewise-button normalized-card-path' onClick={onDetail}>
          查看影响路径　→
        </Button>
      </View>
    </View>
  );
}
