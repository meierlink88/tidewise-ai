import Taro from '@tarojs/taro';
import { Button, Text, View, ScrollView } from '@tarojs/components';
import { useMemo, useState } from 'react';
import type {
  AnalysisKind,
  AnalysisDetail,
  AnalysisChain,
  Assessment,
  EvidenceScope,
  Objections
} from '../../../features/reports/normalized-contract';
import { analysisLabels } from '../../../features/reports/normalized-contract';
import type { ReportEvidenceRoute } from '../../../features/reports/navigation';
import { getReportPort } from '../../../features/reports/port';
import { useReportResource } from '../../../features/reports/use-report-resource';
import { ReportStatePanel } from '../../../features/reports/report-components';
import { formatReportPublication } from '../../../features/reports/presentation';
import './normalized-detail.scss';

const directions = { warming: '升温', cooling: '降温', diverging: '分化', pending: '仅观察' };
const signalDirections = {
  UP: '上升',
  DOWN: '下降',
  STABLE: '稳定',
  MIXED: '分化',
  UNKNOWN: '未知'
};
export function NormalizedDetailView({
  detail,
  reportId,
  kind,
  onEvidence: openEvidence
}: {
  detail: AnalysisDetail;
  reportId: string;
  kind: AnalysisKind;
  onEvidence: (r: ReportEvidenceRoute) => void;
}) {
  const onEvidence = (route: ReportEvidenceRoute): void => {
    openEvidence({ ...route, title: `${analysisLabels[kind]} · ${detail.summary.title}` });
  };
  const tabs = useMemo(
    () => [
      ...detail.macro_impacts.map((m) => ({ ...m, type: 'macro' as const })),
      ...detail.industry_chains.map((c) => ({ ...c, type: 'chain' as const }))
    ],
    [detail]
  );
  const [selected, setSelected] = useState(tabs[0]?.local_key ?? '');
  const current = tabs.find((t) => t.local_key === selected) ?? tabs[0];
  const publication = formatReportPublication(detail.published_at);
  return (
    <View className='normalized-detail'>
      <View className='normalized-hero'>
        <View className='normalized-kicker'>
          <Text className='normalized-story'>{detail.summary.title}</Text>
          {publication && <Text className='normalized-publication'>{publication}</Text>}
        </View>
        <Text className='normalized-headline'>{detail.summary.summary.conclusion}</Text>
        <View className='normalized-hero-logic'>
          <Text>{detail.summary.summary.transmission_logic}</Text>
        </View>
      </View>
      <View className='normalized-main'>
        <ScrollView scrollX className='normalized-tabs'>
          <View
            className='normalized-tabs-row'
            style={{ width: Taro.pxTransform(Math.max(698, tabs.length * 260)) }}
          >
            {tabs.map((t) => (
              <Button
                key={t.local_key}
                className={`tidewise-button normalized-tab ${current?.local_key === t.local_key ? 'selected' : ''}`}
                onClick={() => setSelected(t.local_key)}
                ariaLabel={`${t.type === 'macro' ? '宏观经济' : '产业链'}：${t.name}`}
              >
                <Text className='normalized-tab-name'>{t.name}</Text>
                <Text className='normalized-tab-type'>
                  {t.type === 'macro' ? '宏观经济' : '产业链'}
                </Text>
              </Button>
            ))}
          </View>
        </ScrollView>
        <View
          className={
            current?.type === 'chain' ? 'normalized-chain-content' : 'normalized-tree-panel'
          }
        >
          {!current ? (
            <ReportStatePanel title='暂无因果链详情' description='' />
          ) : current.type === 'macro' ? (
            <View key={current.local_key}>
              <Conclusion a={current.assessment} reportId={reportId} onEvidence={onEvidence} />
              <Mechanism text={current.assessment.transmission_logic} />
              <AssessmentColumns
                support={current.assessment.conditions}
                objections={current.objections}
              />
            </View>
          ) : (
            <LoadedChain
              key={`${reportId}:${kind}:${detail.summary.local_key}:${current.local_key}`}
              reportId={reportId}
              kind={kind}
              unitKey={detail.summary.local_key}
              chainKey={current.local_key}
              onEvidence={onEvidence}
            />
          )}
        </View>
      </View>
    </View>
  );
}
function LoadedChain({
  reportId,
  kind,
  unitKey,
  chainKey,
  onEvidence
}: {
  reportId: string;
  kind: AnalysisKind;
  unitKey: string;
  chainKey: string;
  onEvidence: (r: ReportEvidenceRoute) => void;
}) {
  const port = useMemo(() => getReportPort(), []);
  const resource = useReportResource(`${reportId}:${kind}:${unitKey}:${chainKey}`, () =>
    port.getAnalysisChain(reportId, kind, unitKey, chainKey)
  );
  if (resource.state.status === 'error')
    return (
      <ReportStatePanel
        title='因果链加载失败'
        description=''
        actionLabel='重新加载'
        onAction={() => void resource.retry()}
      />
    );
  if (resource.state.status !== 'ready')
    return <ReportStatePanel title='正在读取因果链' description='' busy />;
  return <ChainContent c={resource.state.data} reportId={reportId} onEvidence={onEvidence} />;
}
export function EvidenceCountButton({
  scope,
  reportId,
  title,
  onEvidence
}: {
  scope: EvidenceScope;
  reportId: string;
  title: string;
  onEvidence: (r: ReportEvidenceRoute) => void;
}) {
  return (
    <Button
      className='tidewise-button normalized-evidence'
      disabled={!scope.evidence_scope_token}
      onClick={() => {
        if (scope.evidence_scope_token)
          onEvidence({ reportId, scopeToken: scope.evidence_scope_token, title });
      }}
    >
      {scope.evidence_count} 条事件 ↗
    </Button>
  );
}
function Conclusion({
  a,
  reportId,
  onEvidence
}: {
  a: Assessment;
  reportId: string;
  onEvidence: (r: ReportEvidenceRoute) => void;
}) {
  return (
    <View className='normalized-conclusion'>
      <Text className='normalized-section-label'>本链结论</Text>
      <Text className='normalized-conclusion-text'>{a.conclusion}</Text>
      {a.scope ? <Text className='normalized-scope'>{a.scope}</Text> : null}
      <EvidenceCountButton scope={a} reportId={reportId} title='本链证据' onEvidence={onEvidence} />
    </View>
  );
}
function Mechanism({ text }: { text: string }) {
  const prose = text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
    .join('；');
  return (
    <View className='normalized-mechanism'>
      <Text className='normalized-mechanism-label'>关键机制</Text>
      <Text className='normalized-mechanism-text'>{prose}</Text>
    </View>
  );
}
function AssessmentColumns({ support, objections }: { support: string[]; objections: Objections }) {
  return (
    <View className='normalized-columns'>
      <View className='normalized-support'>
        <View className='normalized-insight-heading'>
          <Text>支持</Text>
        </View>
        {support.length ? (
          support.map((s, i) => (
            <Text className='normalized-prose' key={i}>
              {s}
            </Text>
          ))
        ) : (
          <Text className='normalized-prose'>暂无支持信息</Text>
        )}
      </View>
      <View className='normalized-counter'>
        <View className='normalized-insight-heading'>
          <Text>反证</Text>
        </View>
        <Text className='normalized-prose'>{objections.summary}</Text>
      </View>
    </View>
  );
}
export function ChainContent({
  c,
  reportId,
  onEvidence
}: {
  c: AnalysisChain;
  reportId: string;
  onEvidence: (r: ReportEvidenceRoute) => void;
}) {
  const [nodeKey, setNodeKey] = useState(
    c.affected_nodes[0]?.node_local_key ?? c.graph.nodes[0]?.local_key ?? ''
  );
  const node = c.affected_nodes.find((n) => n.node_local_key === nodeKey),
    top = c.graph.nodes.find((n) => n.local_key === nodeKey);
  return (
    <View>
      <View className='normalized-tree-panel'>
        <Conclusion a={c.assessment} reportId={reportId} onEvidence={onEvidence} />
        <Mechanism text={c.reasoning_summary.logic} />
        <AssessmentColumns
          support={[c.reasoning_summary.support.text]}
          objections={c.reasoning_summary.objections}
        />
      </View>
      <View className='normalized-core-section'>
        <View className='normalized-graph-section'>
          <Text className='normalized-title'>产业链图谱</Text>
          <HorizontalGraph c={c} selected={nodeKey} onSelect={setNodeKey} />
        </View>
        {c.empty_state ? (
          <Text className='normalized-prose'>{c.empty_state.reason}</Text>
        ) : (
          <View>
            <Text className='normalized-title'>核心分析</Text>
            <View className='normalized-node-detail'>
              <View className='normalized-node-heading'>
                <Text>{top?.name}</Text>
                {node ? (
                  <Text className={`normalized-direction ${node.assessment.direction}`}>
                    {directions[node.assessment.direction]}
                  </Text>
                ) : null}
              </View>
              {node ? (
                <View className='normalized-node-impact'>
                  <Text className='normalized-node-conclusion'>{node.assessment.conclusion}</Text>
                  {node.variable_signals?.length ? (
                    <Text className='normalized-key-signals-title'>关键信号</Text>
                  ) : null}
                  {node.variable_signals?.map((signal, index) => (
                    <View
                      className='normalized-variable-signal'
                      key={`${signal.variable_id}:${signal.signal_id}:${index}`}
                    >
                      <Text className='normalized-signal-bullet'>•</Text>
                      <Text className='normalized-signal-text'>{signal.signal}</Text>
                    </View>
                  ))}
                </View>
              ) : (
                <Text className='normalized-prose'>暂无本期节点评估。</Text>
              )}
            </View>
          </View>
        )}
      </View>
    </View>
  );
}
function HorizontalGraph({
  c,
  selected,
  onSelect
}: {
  c: AnalysisChain;
  selected: string;
  onSelect: (key: string) => void;
}) {
  const nodes = c.graph.nodes;
  const width = 148,
    gap = 28,
    step = width + gap,
    pad = 10;
  const style = (values: Record<string, number>) =>
    Object.entries(values)
      .map(([k, v]) => `${k}:${Taro.pxTransform(v)}`)
      .join(';');
  return (
    <ScrollView scrollX className='normalized-graph-scroll'>
      <View
        className='normalized-graph-canvas'
        style={style({ width: Math.max(width, pad * 2 + nodes.length * step - gap) })}
      >
        {nodes.length > 1 ? (
          <View
            className='normalized-graph-baseline'
            style={style({ left: pad + width / 2, bottom: 16, width: (nodes.length - 1) * step })}
          />
        ) : null}
        {nodes.map((n, i) => {
          const hit = c.affected_nodes.find((a) => a.node_local_key === n.local_key);
          return (
            <View key={n.local_key} className='normalized-graph-item' style={style({ width })}>
              {i < nodes.length - 1 ? (
                <View className='normalized-edge' style={style({ left: width, width: gap })} />
              ) : null}
              {nodes.length > 1 ? (
                <View
                  className='normalized-graph-stem'
                  style={style({ left: width / 2, bottom: -34, height: 26 })}
                />
              ) : null}
              <Button
                className={`tidewise-button normalized-graph-node ${hit?.assessment.direction ?? 'pending'} ${hit?.judgment_origin === 'direct' ? 'direct' : ''} ${selected === n.local_key ? 'selected' : ''}`}
                onClick={() => onSelect(n.local_key)}
                ariaLabel={`查看${n.name}节点详情`}
              >
                <Text className='normalized-graph-name'>{n.name}</Text>
                {hit ? (
                  <Text className={`normalized-direction ${hit.assessment.direction}`}>
                    {directions[hit.assessment.direction]}
                  </Text>
                ) : null}
                {hit?.variable_signals?.length ? (
                  <View className='normalized-graph-variables'>
                    {hit.variable_signals.map((signal, index) => (
                      <Text
                        className='normalized-graph-variable'
                        key={`${signal.signal_id}:${signal.variable_id}:${index}`}
                      >
                        {signal.variable_name} · {signalDirections[signal.source_direction]}
                      </Text>
                    ))}
                  </View>
                ) : null}
              </Button>
            </View>
          );
        })}
      </View>
    </ScrollView>
  );
}
