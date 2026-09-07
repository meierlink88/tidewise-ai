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
import './normalized-detail.scss';

const directions = { warming: '升温', cooling: '降温', diverging: '分化', pending: '仅观察' };
const confidences = { low: '低', medium: '中', high: '高' };
export function NormalizedDetailView({
  detail,
  reportId,
  kind,
  onEvidence
}: {
  detail: AnalysisDetail;
  reportId: string;
  kind: AnalysisKind;
  onEvidence: (r: ReportEvidenceRoute) => void;
}) {
  const tabs = useMemo(
    () => [
      ...detail.macro_impacts.map((m) => ({ ...m, type: 'macro' as const })),
      ...detail.industry_chains.map((c) => ({ ...c, type: 'chain' as const }))
    ],
    [detail]
  );
  const [selected, setSelected] = useState(tabs[0]?.local_key ?? '');
  const current = tabs.find((t) => t.local_key === selected) ?? tabs[0];
  const level = { high: '高影响', medium: '中影响', low: '低影响', pending: '待评估' }[
    detail.summary.summary.impact_assessment.level
  ];
  return (
    <View className='normalized-detail'>
      <View className='normalized-hero'>
        <View className='normalized-kicker'>
          <Text className='normalized-kicker-badge'>{level}</Text>
          <Text>{analysisLabels[kind]}</Text>
        </View>
        <Text className='normalized-story'>{detail.summary.title}</Text>
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
                <Text className='normalized-tab-type'>
                  {t.type === 'macro' ? '宏观经济' : '产业链'}
                </Text>
                <Text>{t.name}</Text>
              </Button>
            ))}
          </View>
        </ScrollView>
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
            <FollowUp paragraphs={current.assessment.follow_up} />
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
function Signals({ a }: { a: Assessment }) {
  return (
    <View className='normalized-signals'>
      <Text className={`normalized-direction ${a.direction}`}>{directions[a.direction]}</Text>
      {a.confidence ? (
        <Text className='normalized-signal-chip'>置信度 {confidences[a.confidence]}</Text>
      ) : null}
      {a.forecast_window.kind !== 'not_applicable' ? (
        <Text className='normalized-signal-chip'>{a.forecast_window.description}</Text>
      ) : null}
      <Text className='normalized-signal-chip'>
        {a.conclusion_basis === 'observation_only' ? '仅观察' : '推理'}
      </Text>
    </View>
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
      <Signals a={a} />
      {a.scope ? <Text className='normalized-scope'>{a.scope}</Text> : null}
      <EvidenceCountButton scope={a} reportId={reportId} title='本链证据' onEvidence={onEvidence} />
    </View>
  );
}
function Mechanism({ text }: { text: string }) {
  return (
    <View className='normalized-mechanism'>
      <Text className='normalized-title'>关键机制</Text>
      <Text className='normalized-prose'>{text}</Text>
    </View>
  );
}
function AssessmentColumns({ support, objections }: { support: string[]; objections: Objections }) {
  return (
    <View className='normalized-columns'>
      <View className='normalized-support'>
        <Text className='normalized-title'>支持</Text>
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
        <Text className='normalized-title'>反证</Text>
        <Text className='normalized-prose'>{objections.summary}</Text>
      </View>
    </View>
  );
}
function FollowUp({ paragraphs }: { paragraphs: string[] }) {
  return (
    <View className='normalized-followup'>
      <Text className='normalized-title'>后续验证</Text>
      {paragraphs.map((p, i) => (
        <Text className='normalized-prose' key={i}>
          {p}
        </Text>
      ))}
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
      <Conclusion a={c.assessment} reportId={reportId} onEvidence={onEvidence} />
      <Mechanism text={c.reasoning_summary.logic} />
      <AssessmentColumns
        support={[c.reasoning_summary.support.text]}
        objections={c.reasoning_summary.objections}
      />
      <View className='normalized-graph-section'>
        <Text className='normalized-title'>产业链图谱</Text>
        <HorizontalGraph c={c} selected={nodeKey} onSelect={setNodeKey} />
        {c.empty_state ? (
          <Text className='normalized-prose'>{c.empty_state.reason}</Text>
        ) : (
          <View>
            <Text className='normalized-title'>节点详情</Text>
            <View className='normalized-node-detail'>
              <View className='normalized-node-heading'>
                <Text>{top?.name}</Text>
                {node ? (
                  <Text className='normalized-node-basis'>
                    {node.assessment.conclusion_basis === 'observation_only' ? '仅观察' : '推理'}
                  </Text>
                ) : null}
              </View>
              {node ? (
                <View>
                  <View className='normalized-node-impact'>
                    <Text className='normalized-prose'>{node.assessment.conclusion}</Text>
                    <Text className='normalized-prose'>{node.assessment.transmission_logic}</Text>
                  </View>
                  <View className='normalized-node-body'>
                    <AssessmentColumns
                      support={node.assessment.conditions}
                      objections={node.objections}
                    />
                    <FollowUp paragraphs={node.assessment.follow_up} />
                  </View>
                </View>
              ) : (
                <Text className='normalized-prose'>暂无本期节点评估。</Text>
              )}
            </View>
          </View>
        )}
      </View>
      <FollowUp paragraphs={c.empty_state?.follow_up ?? c.assessment.follow_up} />
    </View>
  );
}
function NodeMetadata({ assessment: a }: { assessment: Assessment }) {
  return (
    <View className='normalized-node-metadata'>
      <View className='normalized-node-badges'>
        <Text className={`normalized-direction ${a.direction}`}>{directions[a.direction]}</Text>
        {a.confidence ? (
          <Text className='normalized-node-confidence'>置信度 {confidences[a.confidence]}</Text>
        ) : null}
      </View>
      {a.forecast_window.kind !== 'not_applicable' ? (
        <Text className='normalized-node-period'>{a.forecast_window.description}</Text>
      ) : null}
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
  const nodes = c.graph.nodes,
    edges = c.graph.edges,
    indexes = new Map(nodes.map((n, i) => [n.local_key, i]));
  const width = 300,
    gap = 52,
    step = width + gap,
    pad = 20;
  const longs = edges.filter(
      (e) =>
        Math.abs(
          (indexes.get(e.from_node_local_key) ?? 0) - (indexes.get(e.to_node_local_key) ?? 0)
        ) > 1
    ),
    top = 50 + longs.length * 34;
  const style = (values: Record<string, number>) =>
    Object.entries(values)
      .map(([k, v]) => `${k}:${Taro.pxTransform(v)}`)
      .join(';');
  return (
    <ScrollView scrollX className='normalized-graph-scroll'>
      <View
        className='normalized-graph-canvas'
        style={style({ width: pad * 2 + nodes.length * step - gap, height: top + 370 })}
      >
        {edges.map((e, i) => {
          const from = indexes.get(e.from_node_local_key)!,
            to = indexes.get(e.to_node_local_key)!;
          const adjacent = Math.abs(from - to) === 1;
          const sx = pad + from * step + (adjacent ? (from < to ? width : 0) : width / 2),
            tx = pad + to * step + (adjacent ? (from < to ? 0 : width) : width / 2);
          const y = adjacent ? top + 90 : 18 + longs.indexOf(e) * 34;
          return (
            <View key={`${e.from_node_local_key}:${e.to_node_local_key}:${i}`}>
              <View
                className='normalized-edge'
                style={style({ left: Math.min(sx, tx), top: y, width: Math.abs(sx - tx) })}
              >
                <Text className='normalized-edge-label'>{e.relation_label}</Text>
              </View>
              {!adjacent ? (
                <>
                  <View
                    className='normalized-edge-vertical'
                    style={style({ left: sx, top: y, height: top - y })}
                  />
                  <View
                    className='normalized-edge-vertical'
                    style={style({ left: tx, top: y, height: top - y })}
                  />
                </>
              ) : null}
              <View
                className={`normalized-arrow ${adjacent ? (from < to ? 'right' : 'left') : 'down'}`}
                style={style({ left: tx - 6, top: adjacent ? y - 6 : top - 10 })}
              />
            </View>
          );
        })}
        {nodes.map((n, i) => {
          const hit = c.affected_nodes.find((a) => a.node_local_key === n.local_key);
          return (
            <Button
              key={n.local_key}
              className={`tidewise-button normalized-graph-node ${selected === n.local_key ? 'selected' : ''}`}
              style={style({ left: pad + i * step, top, width })}
              onClick={() => onSelect(n.local_key)}
              ariaLabel={`查看${n.name}节点详情`}
            >
              <View className='normalized-graph-heading'>
                <Text className='normalized-graph-name'>{n.name}</Text>
                {hit ? (
                  <Text className='normalized-node-method'>
                    {hit.assessment.conclusion_basis === 'observation_only' ? '仅观察' : '推理'}
                  </Text>
                ) : null}
              </View>
              <View className='normalized-graph-signal-space' />
              {hit ? (
                <NodeMetadata assessment={hit.assessment} />
              ) : (
                <Text className='normalized-unassessed'>暂无本期评估</Text>
              )}
            </Button>
          );
        })}
      </View>
    </ScrollView>
  );
}
