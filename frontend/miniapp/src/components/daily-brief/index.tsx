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
  onMarketHint: () => void;
  onSentimentHint: () => void;
}

export function BriefHero({ brief, collapsed, onToggle, showSummary, showThemes, onMarketHint, onSentimentHint }: BriefHeroProps) {
  return (
    <View className='brief-hero'>
      <View className='brief-hero__date-row'>
        <Text className='brief-hero__date'><Text className='brief-hero__today'>今日</Text>{brief.displayDate}</Text>
        <View className='brief-hero__updated'><View className='brief-hero__progress' /><Text>{brief.updatedAt}</Text></View>
      </View>
      <View className='brief-focus'>
        <View className='brief-focus__kpis'>
          <View className='brief-kpi'><Text className='brief-kpi__label'>市场</Text><Text className='brief-kpi__value brief-kpi__value--neutral'>{brief.market.label}</Text><Button className='brief-kpi__hint' onClick={onMarketHint}>!</Button></View>
          <View className='brief-kpi'><Text className='brief-kpi__label'>情绪</Text><Text className='brief-kpi__value brief-kpi__value--neutral'>{brief.sentiment.label}</Text><Button className='brief-kpi__hint' onClick={onSentimentHint}>!</Button></View>
        </View>
        {!collapsed && showSummary ? (
          <View className='brief-focus__extra'>
            <Text className='brief-focus__summary'><Text className='brief-focus__lead'>今日要闻</Text>{brief.summary}</Text>
            {showThemes ? <View className='brief-focus__themes'>{brief.themes.map((theme) => <Text className='brief-chip' key={theme}>{theme}</Text>)}</View> : null}
          </View>
        ) : null}
        <View className='brief-focus__stats'>
          <Text><Text className='brief-focus__stat-number'>{brief.eventCount}</Text> 条大事件</Text>
          <View className='brief-focus__divider' />
          <Text><Text className='brief-focus__stat-number'>{brief.chainCount}</Text> 条传导链</Text>
          <View className='brief-focus__divider' />
          <Text><Text className='brief-focus__stat-number'>{brief.watchingCount}</Text> 条跟踪中</Text>
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
  onTrack: () => void;
  onKeyEvent: (event: string) => void;
}

export function ConclusionCard({ conclusion, position, total, onPrevious, onNext, onGraph, showImpacts, showEvidence, onTrack, onKeyEvent }: ConclusionCardProps) {
  return (
    <View className={`mainline-card mainline-card--${conclusion.direction}`}>
      <View className='mainline-card__top'><Text className='mainline-card__badge'>{conclusion.badge}</Text><Button className='mainline-card__track' onClick={onTrack}>＋ 跟踪</Button></View>
      <Text className='mainline-card__title'>{conclusion.title}</Text>
      <Text className='mainline-card__summary'>{conclusion.summary}</Text>
      <View className='mainline-card__key-events'>{conclusion.keyEvents.map((event) => <Button className='mainline-card__event-chip' key={event} onClick={() => onKeyEvent(event)}>{event}</Button>)}</View>
      <View className='transmission-card'>
        <View className='transmission-card__head'><Text>{conclusion.transmissionTitle}</Text><Button className='mainline-card__graph' onClick={onGraph}>看图谱 ›</Button></View>
        <View className='transmission-card__steps'>{conclusion.transmissionSteps.map((step, index) => <View className='transmission-card__step-wrap' key={step}><Text className='transmission-card__step'>{step}</Text>{index < conclusion.transmissionSteps.length - 1 ? <Text className='transmission-card__arrow'>→</Text> : null}</View>)}</View>
      </View>
      {showImpacts ? <Text className='mainline-card__subsection-title'>核心影响</Text> : null}
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
      <Text className='mainline-card__confidence'>分析置信度 · {conclusion.confidenceLabel}</Text>
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
