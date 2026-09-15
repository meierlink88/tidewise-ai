import { Button, ScrollView, Text, View } from '@tarojs/components';
import { useState } from 'react';
import type {
  AnalysisDetail,
  Assessment,
  EvidenceScope,
  Objections,
  UnifiedReasoning,
  ReasoningMetric,
  VariableSignal
} from '../../../features/reports/normalized-contract';
import type { ReportEvidenceRoute } from '../../../features/reports/navigation';
import { formatReportPublication } from '../../../features/reports/presentation';
import './unified-detail.scss';

const labels = { warming: '升温', cooling: '降温', diverging: '分化', pending: '仅观察' };
type EvidenceAction = (scope: EvidenceScope, title: string) => void;
export function UnifiedDetailView({
  detail,
  reportId,
  onEvidence
}: {
  detail: AnalysisDetail;
  reportId: string;
  onEvidence: (route: ReportEvidenceRoute) => void;
}) {
  const items = detail.reasonings ?? [];
  const [selected, setSelected] = useState(items[0]?.local_key ?? '');
  const current = items.find((r) => r.local_key === selected) ?? items[0];
  const summary = detail.summary;
  const evidence: EvidenceAction = (scope, title) => {
    if (scope.evidence_scope_token)
      onEvidence({
        reportId,
        scopeToken: scope.evidence_scope_token,
        title: `${summary.title} · ${title}`
      });
  };
  return (
    <View className='normalized-detail unified-detail'>
      <View className='normalized-hero'>
        <View className='normalized-kicker'>
          <Text className='normalized-story'>{summary.title}</Text>
          {detail.published_at && (
            <Text className='normalized-publication'>
              {formatReportPublication(detail.published_at)}
            </Text>
          )}
        </View>
        <Text className='normalized-headline'>{summary.summary.conclusion}</Text>
        <View className='normalized-hero-logic'>
          <Text>{summary.summary.transmission_logic}</Text>
        </View>
        {summary.summary.judgment && (
          <Text className='unified-judgment'>判断边界：{summary.summary.judgment}</Text>
        )}
      </View>
      {items.length > 1 && (
        <ScrollView scrollX className='unified-scroll'>
          <View className='unified-tabs'>
            {items.map((r) => (
              <Button
                key={r.local_key}
                className={`tidewise-button unified-tab ${current?.local_key === r.local_key ? 'selected' : ''}`}
                onClick={() => setSelected(r.local_key)}
                ariaLabel={`查看${r.title}推导`}
              >
                {r.title}
              </Button>
            ))}
          </View>
        </ScrollView>
      )}
      {current ? (
        <Reasoning key={current.local_key} reasoning={current} onEvidence={evidence} />
      ) : (
        <Text>暂无推导内容</Text>
      )}
    </View>
  );
}
function EvidenceButton({
  scope,
  title,
  onEvidence
}: {
  scope: EvidenceScope;
  title: string;
  onEvidence: EvidenceAction;
}) {
  return scope.evidence_count > 0 ? (
    <Button
      className='tidewise-button unified-evidence'
      ariaLabel={`查看${title}的${scope.evidence_count}条证据`}
      onClick={() => onEvidence(scope, title)}
    >
      {scope.evidence_count} 条证据 ↗
    </Button>
  ) : null;
}
function Signals({ signals }: { signals?: VariableSignal[] }) {
  return signals?.length ? (
    <View className='unified-signals'>
      <Text className='unified-label'>关键信号</Text>
      {signals.map((s) => (
        <Text key={`${s.signal_id}:${s.variable_id}`}>{s.signal}</Text>
      ))}
    </View>
  ) : null;
}
function ObjectionContent({
  value,
  onEvidence
}: {
  value: Objections;
  onEvidence: EvidenceAction;
}) {
  return (
    <View className='unified-objections unified-proof-item'>
      <Text className='unified-label'>反证与限制</Text>
      <Text>{value.summary}</Text>
      {[...value.counterevidence, ...value.buffers].map((c, i) => (
        <View key={`${i}:${c.text}`}>
          <Text>{c.text}</Text>
          <EvidenceButton scope={c} title='反证与缓冲' onEvidence={onEvidence} />
        </View>
      ))}
      {value.scope_limits.map((s) => (
        <Text key={s}>{s}</Text>
      ))}
    </View>
  );
}
function metricValue(m: ReasoningMetric) {
  if (m.display_value) return m.display_value;
  if (m.value === undefined) return '';
  return `${m.measure_type === 'change_rate' && m.value > 0 ? '+' : ''}${m.value}${m.unit}`;
}
function delta(a: Assessment) {
  const n = a.weight_delta_pp;
  return n === undefined
    ? labels[a.direction]
    : n === 0
      ? '—'
      : `${n > 0 ? '↑' : '↓'}${Math.abs(n)}%`;
}
function deltaClass(a: Assessment) {
  const n = a.weight_delta_pp;
  return n === undefined ? a.direction : n === 0 ? 'neutral' : n > 0 ? 'warming' : 'cooling';
}
function Reasoning({
  reasoning: r,
  onEvidence
}: {
  reasoning: UnifiedReasoning;
  onEvidence: EvidenceAction;
}) {
  const [selected, setSelected] = useState(r.affected_assets[0]?.local_key ?? '');
  const asset = r.affected_assets.find((a) => a.local_key === selected) ?? r.affected_assets[0];
  return (
    <View className='unified-body'>
      <View className='unified-reasoning-surface'>
        {r.reasoning_blocks.length === 0 && (
          <View className='unified-conclusion'>
            <Text className='unified-label'>本链结论</Text>
            <Text className='unified-heading'>{r.assessment.conclusion}</Text>
            {r.assessment.scope && <Text>{r.assessment.scope}</Text>}
            <EvidenceButton scope={r.assessment} title={r.title} onEvidence={onEvidence} />
          </View>
        )}
        {r.reasoning_blocks.map((b) => (
          <View key={b.local_key} className='unified-block'>
            <View className='unified-conclusion'>
              <Text className='unified-heading'>{b.title}</Text>
              <Text>{b.explanation}</Text>
            </View>
            <View className='unified-connector' />
            <ScrollView scrollX className='unified-scroll'>
              <View className={`unified-metrics ${b.relation_type}`}>
                {b.nodes.map((n) => (
                  <View key={n.local_key} className='unified-metric'>
                    <Text className='unified-metric-name'>{n.name}</Text>
                    {n.metrics.map((m, i) => (
                      <View key={`${i}:${m.name}`}>
                        <Text className='unified-metric-value'>{metricValue(m)}</Text>
                        {n.metrics.length > 1 && <Text>{m.name}</Text>}
                        {m.value_nature !== 'observed' && (
                          <Text>{m.value_nature === 'forecast' ? '预测' : '假设'}</Text>
                        )}
                        <Text className='unified-period'>
                          {[m.period_label, m.as_of].filter(Boolean).join(' · ')}
                        </Text>
                      </View>
                    ))}
                    {n.description && <Text className='unified-period'>{n.description}</Text>}
                  </View>
                ))}
              </View>
            </ScrollView>
            {b.links?.map((e) => (
              <Text
                className='unified-link-description'
                key={`${e.from_node_local_key}:${e.to_node_local_key}`}
              >
                {b.nodes.find((n) => n.local_key === e.from_node_local_key)?.name} →{' '}
                {e.label ? `${e.label} → ` : ''}
                {b.nodes.find((n) => n.local_key === e.to_node_local_key)?.name}
              </Text>
            ))}
            <View className='unified-connector' />
          </View>
        ))}
        <View className='unified-mechanism'>
          <Text className='unified-label'>关键机制</Text>
          <Text className='unified-heading'>{r.reasoning_summary.logic}</Text>
        </View>
        {(r.reasoning_summary.support || r.reasoning_summary.objections.summary) && (
          <View className='unified-proof'>
            {r.reasoning_summary.support && (
              <View className='unified-proof-item'>
                <Text className='unified-label'>支持</Text>
                <Text>{r.reasoning_summary.support.text}</Text>
                <EvidenceButton
                  scope={r.reasoning_summary.support}
                  title='支持'
                  onEvidence={onEvidence}
                />
              </View>
            )}
            <ObjectionContent value={r.reasoning_summary.objections} onEvidence={onEvidence} />
          </View>
        )}
        {r.empty_state && (
          <View className='unified-time'>
            <Text className='unified-label'>观察事项</Text>
            <Text>{r.empty_state.reason}</Text>
            {r.empty_state.follow_up.map((item) => (
              <Text key={item}>{item}</Text>
            ))}
          </View>
        )}
        <Signals signals={r.variable_signals} />
      </View>
      {r.affected_assets.length > 0 && (
        <View className='unified-assets'>
          {r.graph && r.graph.nodes.length > 0 && (
            <View className='unified-graph'>
              <Text className='unified-heading'>产业链图谱</Text>
              <ScrollView scrollX className='unified-scroll'>
                <View className='unified-tabs'>
                  {r.graph.nodes.map((n) => {
                    const a = r.affected_assets.find((x) => x.node_local_key === n.local_key);
                    return (
                      <Button
                        key={n.local_key}
                        disabled={!a}
                        className={`tidewise-button unified-tab ${asset?.node_local_key === n.local_key ? 'selected' : ''}`}
                        onClick={() => a && setSelected(a.local_key)}
                      >
                        {n.name}
                      </Button>
                    );
                  })}
                </View>
              </ScrollView>
            </View>
          )}
          {r.affected_assets.length > 3 && (
            <Text className='unified-scroll-hint'>向右滑动查看更多资产 →</Text>
          )}
          <ScrollView scrollX className='unified-scroll'>
            <View className='unified-asset-track'>
              {r.affected_assets.map((a) => (
                <Button
                  key={a.local_key}
                  className={`tidewise-button unified-asset ${asset?.local_key === a.local_key ? 'selected' : ''}`}
                  onClick={() => setSelected(a.local_key)}
                  ariaLabel={`查看${a.name}详情`}
                >
                  <Text className='unified-asset-name'>{a.name}</Text>
                  <Text className={`unified-delta ${deltaClass(a.assessment)}`}>
                    {delta(a.assessment)}
                  </Text>
                  {a.assessment.adjustment_purpose && (
                    <Text className='unified-asset-purpose'>{a.assessment.adjustment_purpose}</Text>
                  )}
                </Button>
              ))}
            </View>
          </ScrollView>
          {asset && (
            <View className='unified-asset-detail' key={asset.local_key}>
              <View className='unified-asset-header'>
                <Text className='unified-heading'>{asset.name}</Text>
                <Text className={`unified-delta ${deltaClass(asset.assessment)}`}>
                  {delta(asset.assessment)}
                </Text>
              </View>
              {asset.assessment.adjustment_purpose && (
                <Text className='unified-purpose'>{asset.assessment.adjustment_purpose}</Text>
              )}
              <Text>{asset.assessment.conclusion}</Text>
              {asset.assessment.transmission_logic !== asset.assessment.conclusion && (
                <Text>{asset.assessment.transmission_logic}</Text>
              )}
              {asset.assessment.forecast_window.description &&
                asset.assessment.forecast_window.kind !== 'not_applicable' && (
                  <View className='unified-time'>
                    <Text className='unified-label'>传导时间</Text>
                    <Text>{asset.assessment.forecast_window.description}</Text>
                  </View>
                )}
              {asset.assessment.conditions.length > 0 && (
                <View className='unified-conditions'>
                  <Text className='unified-label'>判断条件</Text>
                  {asset.assessment.conditions.map((t) => (
                    <Text key={t}>{t}</Text>
                  ))}
                </View>
              )}
              <Signals signals={asset.variable_signals} />
              <ObjectionContent value={asset.objections} onEvidence={onEvidence} />
              <EvidenceButton scope={asset.assessment} title={asset.name} onEvidence={onEvidence} />
            </View>
          )}
        </View>
      )}
    </View>
  );
}
