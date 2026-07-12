import { Button, Text, View } from '@tarojs/components';
import type { DailyBriefHomeView, HomeConclusionView } from '../../models/daily-brief-view';
import type { ResourceState } from '../../models/resource-state';
import { getConfidenceLabel, getDirectionMeta, getResourceStateCopy } from './ui-meta';
import './index.scss';

interface BriefHeroProps {
  brief: DailyBriefHomeView;
  collapsed: boolean;
  onToggle: () => void;
  showSummary: boolean;
  showThemes: boolean;
}

export function BriefHero({ brief, collapsed, onToggle, showSummary, showThemes }: BriefHeroProps) {
  return (
    <View className='brief-hero'>
      <View className='brief-hero__date-row'>
        <Text className='brief-hero__date'><Text className='brief-hero__today'>今日</Text>{brief.displayDate}</Text>
        <View className='brief-hero__updated'><View className='brief-hero__progress' /><Text>{brief.updatedAt}</Text></View>
      </View>
      <View className='brief-focus'>
        <View className='brief-focus__kpis'>
          <View className='brief-kpi'><Text className='brief-kpi__label'>市场</Text><Text className='brief-kpi__value brief-kpi__value--neutral'>{brief.market.label}</Text></View>
          <View className='brief-kpi'><Text className='brief-kpi__label'>情绪</Text><Text className='brief-kpi__value brief-kpi__value--neutral'>{brief.sentiment.label}</Text></View>
        </View>
        {!collapsed && showSummary ? (
          <View className='brief-focus__extra'>
            <Text className='brief-focus__summary'><Text className='brief-focus__lead'>今日要闻</Text>{brief.summary}</Text>
            {showThemes ? <View className='brief-focus__themes'>{brief.themes.map((theme) => <Text className='brief-chip' key={theme}>{theme}</Text>)}</View> : null}
          </View>
        ) : null}
        <View className='brief-focus__stats'>
          <Text><Text className='brief-focus__stat-number'>{brief.conclusions.length}</Text> 条主线</Text>
          <View className='brief-focus__divider' />
          <Text><Text className='brief-focus__stat-number'>{brief.conclusions.reduce((count, item) => count + item.impacts.length, 0)}</Text> 个影响</Text>
          <View className='brief-focus__divider' />
          <Text><Text className='brief-focus__stat-number'>{brief.conclusions.reduce((count, item) => count + item.evidence.length, 0)}</Text> 条证据</Text>
        </View>
      </View>
      <View className='brief-wave'><View className='brief-wave__curve' /><Button className='brief-wave__toggle' onClick={onToggle}>{collapsed ? '⌄' : '⌃'}</Button></View>
    </View>
  );
}

interface ConclusionCardProps {
  conclusion: HomeConclusionView;
  position: number;
  total: number;
  onPrevious: () => void;
  onNext: () => void;
  onGraph: () => void;
  showImpacts: boolean;
  showEvidence: boolean;
}

export function ConclusionCard({ conclusion, position, total, onPrevious, onNext, onGraph, showImpacts, showEvidence }: ConclusionCardProps) {
  return (
    <View className={`mainline-card mainline-card--${conclusion.direction}`}>
      <View className='mainline-card__top'><Text className='mainline-card__badge'>{conclusion.badge}</Text><Text className='mainline-card__confidence'>{conclusion.confidenceLabel}</Text></View>
      <Text className='mainline-card__title'>{conclusion.title}</Text>
      <Text className='mainline-card__summary'>{conclusion.summary}</Text>
      <View className='mainline-card__section-head'><Text>{showImpacts ? '核心影响' : '主线工具'}</Text><Button className='mainline-card__graph' onClick={onGraph}>看图谱 ›</Button></View>
      {showImpacts ? <View className='impact-list'>
        {conclusion.impacts.map((impact) => {
          const meta = getDirectionMeta(impact.direction);
          return (
            <View className='impact-card' key={impact.id}>
              <View className='impact-card__head'><Text className='impact-card__name'>{impact.entityName}</Text><Text className={`impact-card__direction impact-card__direction--${meta.className}`}>{meta.symbol} {meta.label}</Text></View>
              <Text className='impact-card__rationale'>{impact.rationale}</Text>
              <Text className='impact-card__uncertainty'>不确定性 · {impact.uncertainty}</Text>
            </View>
          );
        })}
      </View> : null}
      {showEvidence ? <Text className='mainline-card__evidence-title'>证据摘要</Text> : null}
      {showEvidence ? <View className='evidence-list'>
        {conclusion.evidence.length ? conclusion.evidence.map((evidence) => (
          <View className='evidence-item' key={evidence.id}>
            <View className='evidence-item__meta'><Text>{evidence.source}</Text><Text>{evidence.publishedAt} · {getConfidenceLabel(evidence.confidence)}</Text></View>
            <Text className='evidence-item__title'>{evidence.title}</Text>
            <Text className='evidence-item__summary'>{evidence.summary}</Text>
          </View>
        )) : <Text className='evidence-list__empty'>暂无关联证据</Text>}
      </View> : null}
      <View className='mainline-card__pager'>
        <Button className='pager-button' disabled={position === 0} onClick={onPrevious}>‹</Button>
        <Text>{position + 1} / {total}</Text>
        <Button className='pager-button' disabled={position === total - 1} onClick={onNext}>›</Button>
      </View>
    </View>
  );
}

interface ResourceStatePanelProps<T> {
  state: ResourceState<T>;
  onRetry: () => void;
}

export function ResourceStatePanel<T>({ state, onRetry }: ResourceStatePanelProps<T>) {
  if (state.status === 'ready') return null;
  const status = state.status === 'idle' ? 'loading' : state.status;
  const copy = getResourceStateCopy(status);
  return (
    <View className={`resource-panel resource-panel--${status}`}>
      {status === 'loading' ? <View className='resource-panel__spinner' /> : null}
      <Text className='resource-panel__title'>{copy.title}</Text>
      {'description' in copy ? <Text className='resource-panel__description'>{copy.description}</Text> : null}
      {'action' in copy ? <Button className='resource-panel__action' onClick={onRetry}>{copy.action}</Button> : null}
    </View>
  );
}

export function SafetyNote({ text }: { text: string }) {
  return <Text className='daily-brief-safety'>{text}</Text>;
}
