import { Button, Image, ScrollView, Text, View } from '@tarojs/components';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  analysisLabels,
  type AnalysisKind,
  type AnalysisGroup,
  type AnalysisSummary
} from '../../features/reports/normalized-contract';
import type { ReportHomeGroup } from '../../features/reports/contract';
import type { ReportDetailRoute, ReportEvidenceRoute } from '../../features/reports/navigation';
import { getReportPort } from '../../features/reports/port';
import globeIcon from '../../assets/icons/report-globe-ink.svg';
import macroIcon from '../../assets/icons/report-bar-chart-ink.svg';
import chainIcon from '../../assets/icons/report-link-ink.svg';
import globeGoldIcon from '../../assets/icons/report-globe-gold.svg';
import macroGoldIcon from '../../assets/icons/report-bar-chart-gold.svg';
import chainGoldIcon from '../../assets/icons/report-link-gold.svg';
import evidenceIcon from '../../assets/icons/file-text-ink.svg';
import arrowIcon from '../../assets/icons/report-arrow-right-light-gold.svg';
import './normalized-home.scss';

const categories = ['geopolitical_stories', 'macroeconomic_stories', 'concept_analyses'] as const;
type Category = (typeof categories)[number];
const selectedCategoryIcons: Record<Category, string> = {
  geopolitical_stories: globeGoldIcon,
  macroeconomic_stories: macroGoldIcon,
  concept_analyses: chainGoldIcon
};

const categoryIcons: Record<Category, string> = {
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
  onDetail?: (r: ReportDetailRoute) => void;
  onEvidence: (r: ReportEvidenceRoute) => void;
}) {
  const [kind, setKind] = useState<Category>('geopolitical_stories');
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
  const sourceKinds: AnalysisKind[] =
    kind === 'concept_analyses' ? ['industry_chain_analyses', 'concept_analyses'] : [kind];
  const groups = sourceKinds.flatMap((sourceKind) => {
    const page = pages[sourceKind] ?? group.analysisGroups?.find((g) => g.kind === sourceKind);
    return page ? [page] : [];
  });
  const current =
    groups.find((g) => g.next_cursor && g.kind === failed) ?? groups.find((g) => g.next_cursor);
  const items = groups
    .flatMap((g) => g.items.map((u) => ({ sourceKind: g.kind, u })))
    .filter(
      ({ u }) =>
        !query ||
        [u.title, u.summary.conclusion, ...u.affected_anchors.map((a) => a.name)]
          .join(' ')
          .includes(query)
    );
  const load = async () => {
    if (inflight.current || !current?.next_cursor) return;
    inflight.current = true;
    const g = generation.current,
      k = current.kind;
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
            {categories.map((k) => (
              <Button
                key={k}
                className={`tidewise-button normalized-home-tab ${kind === k ? 'selected' : ''}`}
                onClick={() => setKind(k)}
              >
                <View className='normalized-home-tab-icon'>
                  <Image
                    src={kind === k ? selectedCategoryIcons[k] : categoryIcons[k]}
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
          <Text className='normalized-home-total'>{items.length} 条结论</Text>
        </View>
      </View>
      <ScrollView key={kind} scrollY className='normalized-home-scroll'>
        <View className='normalized-home-list'>
          {items.map(({ u, sourceKind }) => (
            <HomeCard
              key={`${sourceKind}:${u.local_key}`}
              u={u}
              publishedAt={group.report.publishedAt}
              onDetail={() =>
                onDetail?.({
                  reportId: group.report.id,
                  targetType: sourceKind,
                  targetKey: u.local_key
                })
              }
              canOpenDetail={!!onDetail}
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
              {pending === current.kind
                ? '正在加载…'
                : failed === current.kind
                  ? '加载失败，点击重试'
                  : '加载更多'}
            </Button>
          ) : null}
        </View>
      </ScrollView>
    </View>
  );
}
function HomeCard({
  canOpenDetail,
  u,
  publishedAt,
  onDetail,
  onEvidence
}: {
  canOpenDetail: boolean;
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
      <View className='normalized-card-logic'>
        {u.summary.transmission_logic
          .split(/\r?\n/)
          .filter((path) => path.trim())
          .map((path, index) => (
            <View className='normalized-card-logic-path' key={`${index}:${path}`}>
              <Text>{path}</Text>
            </View>
          ))}
      </View>
      <View className='normalized-anchor-area'>
        <View className='normalized-anchor-count'>
          <Text className='normalized-anchor-number'>{u.affected_anchors.length}</Text>
          <Text className='normalized-anchor-caption'>个受影响锚点</Text>
        </View>
        <View className='normalized-anchor-chips'>
          {u.affected_anchors.map((a) => (
            <View
              className='normalized-anchor-chip'
              key={`${a.reference.chain_local_key ?? ''}:${a.reference.local_key}`}
            >
              <Text className='normalized-anchor-name'>{a.name}</Text>
              <Text className={`normalized-anchor-direction ${a.assessment.direction}`}>
                {directionLabels[a.assessment.direction]}
              </Text>
            </View>
          ))}
        </View>
      </View>
      <View className='normalized-card-footer'>
        <Button
          className={`tidewise-button normalized-card-evidence ${!u.summary.evidence_scope_token ? 'is-disabled' : ''}`}
          ariaLabel={`查看${u.title}证据`}
          disabled={!u.summary.evidence_scope_token}
          onClick={onEvidence}
        >
          <Image src={evidenceIcon} className='normalized-evidence-icon' mode='scaleToFill' />
          <Text>{u.summary.evidence_count} 条事件</Text>
        </Button>
        <Button
          className='tidewise-button normalized-card-path'
          disabled={!canOpenDetail}
          hoverClass={canOpenDetail ? 'button-hover' : 'none'}
          onClick={canOpenDetail ? onDetail : undefined}
        >
          <Text>查看影响路径</Text>
          <View className='normalized-path-circle'>
            <Image src={arrowIcon} className='normalized-path-icon' mode='scaleToFill' />
          </View>
        </Button>
      </View>
    </View>
  );
}
