import { Button, ScrollView, Text, View } from '@tarojs/components';
import { useState } from 'react';
import type {
  EvidenceScope,
  NodeImpact,
  ReasoningBlock,
  ReasoningMetric,
  UnifiedReasoning
} from '../../../features/reports/normalized-contract';
import type { ReportEvidenceRoute } from '../../../features/reports/navigation';
import './geopolitical-detail.scss';

type EvidenceProps = {
  reportId: string;
  onEvidence: (route: ReportEvidenceRoute) => void;
};
const directions = { warming: '升温', cooling: '降温', diverging: '分化', pending: '待验证' };

export function GeopoliticalBody({
  reasonings,
  ...evidence
}: { reasonings: UnifiedReasoning[] } & EvidenceProps) {
  const [selected, setSelected] = useState(reasonings[0]?.local_key ?? '');
  const current = reasonings.find((r) => r.local_key === selected) ?? reasonings[0];
  return (
    <View className='geo-detail-main'>
      {reasonings.length > 1 && (
        <ScrollView scrollX className='geo-detail-scroll'>
          <View className='geo-detail-reasoning-tabs'>
            {reasonings.map((r) => (
              <Button
                key={r.local_key}
                className={`tidewise-button geo-detail-tab ${current?.local_key === r.local_key ? 'selected' : ''}`}
                ariaLabel={`${r.title}${current?.local_key === r.local_key ? '，已选中' : ''}`}
                onClick={() => setSelected(r.local_key)}
              >
                {r.title}
              </Button>
            ))}
          </View>
        </ScrollView>
      )}
      {current ? (
        <Reasoning key={current.local_key} reasoning={current} {...evidence} />
      ) : (
        <Text>暂无推导详情</Text>
      )}
    </View>
  );
}

function Reasoning({ reasoning, ...evidence }: { reasoning: UnifiedReasoning } & EvidenceProps) {
  return (
    <>
      <View className='geo-detail-reasoning-surface'>
        {reasoning.reasoning_blocks.map((block) => (
          <Block key={block.local_key} block={block} {...evidence} />
        ))}
        {!reasoning.reasoning_blocks.length && reasoning.assessment.conclusion && (
          <View className='geo-detail-block-head'>
            <Text className='geo-detail-title'>{reasoning.assessment.conclusion}</Text>
          </View>
        )}
        {reasoning.reasoning_summary.logic && (
          <View className='geo-detail-question'>
            <Text className='geo-detail-question-label'>关键机制问题</Text>
            <Text className='geo-detail-title'>{reasoning.reasoning_summary.logic}</Text>
          </View>
        )}
        {reasoning.empty_state && (
          <Text className='geo-detail-description'>{reasoning.empty_state.reason}</Text>
        )}
      </View>
      {!!reasoning.affected_assets.length && (
        <Assets assets={reasoning.affected_assets} {...evidence} />
      )}
    </>
  );
}

function metricValue(metric: ReasoningMetric): string {
  const value = metric.display_value || String(metric.value ?? '');
  // Keep report-formatted intervals intact and add only a missing unit.
  return metric.unit && !value.includes(metric.unit) ? `${value}${metric.unit}` : value;
}
function Block({ block, ...evidence }: { block: ReasoningBlock } & EvidenceProps) {
  return (
    <View className={`geo-detail-block ${block.relation_type}`}>
      <View className='geo-detail-block-head'>
        <Text className='geo-detail-title'>{block.title}</Text>
        {block.explanation && <Text className='geo-detail-description'>{block.explanation}</Text>}
      </View>
      {!!block.nodes.length && (
        <ScrollView scrollX className='geo-detail-scroll'>
          <View className='geo-detail-metric-row'>
            {block.nodes.map((node) => (
              <View className='geo-detail-metric-node' key={node.local_key}>
                {block.relation_type === 'comparison' && (
                  <Text className='geo-detail-metric-name'>{node.name}</Text>
                )}
                {node.metrics.map((metric, index) => (
                  <View className='geo-detail-metric' key={`${metric.name}:${index}`}>
                    {node.metrics.length > 1 && (
                      <Text className='geo-detail-metric-name'>{metric.name}</Text>
                    )}
                    <Text className='geo-detail-metric-value'>{metricValue(metric)}</Text>
                    {block.relation_type === 'causal' && node.metrics.length === 1 && (
                      <Text className='geo-detail-metric-name'>{node.name}</Text>
                    )}
                    {metric.period_label && (
                      <Text className='geo-detail-metric-note'>{metric.period_label}</Text>
                    )}
                    {metric.as_of && (
                      <Text className='geo-detail-metric-note'>截至 {metric.as_of}</Text>
                    )}
                    {metric.value_nature !== 'observed' && (
                      <Text className='geo-detail-metric-note'>
                        {metric.value_nature === 'forecast' ? '预测' : '情景假设'}
                      </Text>
                    )}
                    <Evidence scope={metric} title={metric.name} {...evidence} />
                  </View>
                ))}
                {(!node.metrics.length ||
                  (block.relation_type === 'causal' && node.metrics.length > 1)) && (
                  <Text className='geo-detail-metric-name'>{node.name}</Text>
                )}
                {node.description && (
                  <Text className='geo-detail-metric-note'>{node.description}</Text>
                )}
              </View>
            ))}
          </View>
        </ScrollView>
      )}
    </View>
  );
}

function allocation(asset: NodeImpact): string {
  const delta = asset.assessment.weight_delta_pp;
  if (delta === undefined) return directions[asset.assessment.direction];
  if (delta === 0) return '—';
  return `${delta > 0 ? '↑' : '↓'}${Math.abs(delta)}%`;
}
function allocationTone(asset: NodeImpact): string {
  const delta = asset.assessment.weight_delta_pp;
  return delta === undefined
    ? asset.assessment.direction
    : delta > 0
      ? 'warming'
      : delta < 0
        ? 'cooling'
        : 'pending';
}
function allocationLabel(asset: NodeImpact): string {
  const delta = asset.assessment.weight_delta_pp;
  return delta === undefined
    ? directions[asset.assessment.direction]
    : delta === 0
      ? '配置不变'
      : `配置${delta > 0 ? '增加' : '减少'}${Math.abs(delta)}个百分点`;
}
function Assets({ assets, ...evidence }: { assets: NodeImpact[] } & EvidenceProps) {
  const [selected, setSelected] = useState(assets[0]?.local_key ?? '');
  const current = assets.find((a) => a.local_key === selected) ?? assets[0];
  if (!current) return null;
  const a = current.assessment;
  return (
    <View className='geo-detail-assets-surface'>
      {assets.length > 3 && <Text className='geo-detail-scroll-hint'>向右滑动查看更多资产 →</Text>}
      <ScrollView scrollX className='geo-detail-scroll'>
        <View className='geo-detail-asset-row'>
          {assets.map((asset) => (
            <Button
              key={asset.local_key}
              className={`tidewise-button geo-detail-asset-tile ${allocationTone(asset)} ${asset.local_key === current.local_key ? 'selected' : ''}`}
              ariaLabel={`${asset.name}，${allocationLabel(asset)}${asset.local_key === current.local_key ? '，已选中' : ''}`}
              onClick={() => setSelected(asset.local_key)}
            >
              <Text className='geo-detail-title'>{asset.name}</Text>
              <Text className={`geo-detail-allocation ${allocationTone(asset)}`}>
                {allocation(asset)}
              </Text>
              {asset.assessment.adjustment_purpose && (
                <Text className='geo-detail-metric-note'>
                  {asset.assessment.adjustment_purpose}
                </Text>
              )}
            </Button>
          ))}
        </View>
      </ScrollView>
      <View className='geo-detail-asset-analysis'>
        <View className='geo-detail-asset-heading'>
          <Text className='geo-detail-title'>{current.name}</Text>
          <Text className={`geo-detail-allocation ${allocationTone(current)}`}>
            {allocation(current)}
          </Text>
        </View>
        <Text className='geo-detail-title'>{a.conclusion}</Text>
        {a.transmission_logic && (
          <Text className='geo-detail-description'>{a.transmission_logic}</Text>
        )}
        {a.forecast_window.description.trim() && (
          <View className='geo-detail-timing'>
            <Text className='geo-detail-timing-label'>传导时间</Text>
            <Text className='geo-detail-timing-text'>{a.forecast_window.description}</Text>
          </View>
        )}
        <Evidence scope={a} title={current.name} {...evidence} />
      </View>
    </View>
  );
}

function Evidence({
  scope,
  title,
  reportId,
  onEvidence
}: { scope: EvidenceScope; title: string } & EvidenceProps) {
  const token = scope.evidence_scope_token;
  if (!token || !scope.evidence_count) return null;
  return (
    <Button
      className='tidewise-button geo-detail-evidence'
      ariaLabel={`查看${title}证据`}
      onClick={() => onEvidence({ reportId, scopeToken: token, title })}
    >
      {scope.evidence_count} 条证据 ↗
    </Button>
  );
}
