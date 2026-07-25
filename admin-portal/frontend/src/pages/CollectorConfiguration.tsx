import { useEffect, useMemo, useState } from 'react';
import {
  AdminAgentRunAPIError,
  loadAgentExecutions,
  loadAgentSchedule,
  loadConnectors,
  loadModelProviders,
  saveAgentSchedule,
  setAgentScheduleEnabled,
  updateConnector,
  updateModelProvider,
  type AgentExecution,
  type AgentExecutionPage,
  type AgentSchedule,
  type ConnectorConfiguration,
  type ModelProviderConfiguration,
  type ScheduleType
} from '../api/agentManagement';
import Button from '../components/ui/Button';
import DataTable, { type DataTableColumn } from '../components/ui/DataTable';
import Field from '../components/ui/Field';
import Input from '../components/ui/Input';
import Pagination from '../components/ui/Pagination';
import StatusBadge from '../components/ui/StatusBadge';
import Tabs, { TabPanel } from '../components/ui/Tabs';

type CollectorSection = 'schedule' | 'executions' | 'models' | 'connectors';
type EditTarget =
  | { kind: 'model'; value: ModelProviderConfiguration }
  | { kind: 'connector'; value: ConnectorConfiguration };

const agentKey = 'collector';
const agentVersion = 'collector.v1';
const sectionItems: { id: CollectorSection; label: string }[] = [
  { id: 'schedule', label: '定时任务' },
  { id: 'executions', label: '执行记录' },
  { id: 'models', label: '模型配置' },
  { id: 'connectors', label: '连接器配置' }
];

export default function CollectorConfiguration({ token }: { token: string }) {
  const [section, setSection] = useState<CollectorSection>('schedule');
  const [schedule, setSchedule] = useState<AgentSchedule | null>(null);
  const [models, setModels] = useState<ModelProviderConfiguration[]>([]);
  const [connectors, setConnectors] = useState<ConnectorConfiguration[]>([]);
  const [scheduleType, setScheduleType] = useState<ScheduleType>('daily');
  const [dailyTimes, setDailyTimes] = useState<string[]>(['08:30']);
  const [newDailyTime, setNewDailyTime] = useState('08:30');
  const [cronExpression, setCronExpression] = useState('0 * * * *');
  const [prompt, setPrompt] = useState('');
  const [executions, setExecutions] = useState<AgentExecutionPage>({
    items: [],
    page: 1,
    page_size: 20,
    total_items: 0,
    total_pages: 0
  });
  const [executionPage, setExecutionPage] = useState(1);
  const [executionReloadVersion, setExecutionReloadVersion] = useState(0);
  const [loading, setLoading] = useState(true);
  const [executionLoading, setExecutionLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [stopConfirmation, setStopConfirmation] = useState(false);
  const [editTarget, setEditTarget] = useState<EditTarget | null>(null);
  const [reloadVersion, setReloadVersion] = useState(0);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError('');
    Promise.all([
      loadAgentSchedule(token, agentKey).catch((loadError) => {
        if (loadError instanceof AdminAgentRunAPIError && loadError.status === 404) {
          return null;
        }
        throw loadError;
      }),
      loadModelProviders(token),
      loadConnectors(token)
    ])
      .then(([loadedSchedule, loadedModels, loadedConnectors]) => {
        if (!active) {
          return;
        }
        setSchedule(loadedSchedule);
        setModels(loadedModels);
        setConnectors(loadedConnectors);
        if (loadedSchedule) {
          setScheduleType(loadedSchedule.schedule_type);
          setDailyTimes(loadedSchedule.daily_times ?? []);
          setCronExpression(loadedSchedule.cron_expression ?? '0 * * * *');
          setPrompt(promptFromSchedule(loadedSchedule));
        }
      })
      .catch((loadError) => {
        if (active) {
          setError(errorText(loadError));
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, [reloadVersion, token]);

  useEffect(() => {
    if (section !== 'executions') {
      return;
    }
    let active = true;
    setExecutionLoading(true);
    setError('');
    loadAgentExecutions(token, executionPage)
      .then((page) => {
        if (active) {
          setExecutions(page);
        }
      })
      .catch((loadError) => {
        if (active) {
          setError(errorText(loadError));
        }
      })
      .finally(() => {
        if (active) {
          setExecutionLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, [executionPage, executionReloadVersion, section, token]);

  const configuredModels = models.filter((model) => model.configured).length;
  const configuredConnectors = connectors.filter((connector) => connector.configured).length;
  const readinessComplete =
    models.length > 0 &&
    configuredModels === models.length &&
    connectors.length > 0 &&
    configuredConnectors === connectors.length;

  const executionColumns = useMemo<DataTableColumn<AgentExecution>[]>(
    () => [
      {
        key: 'triggered',
        header: '执行时间',
        render: (item) => formatDateTime(item.triggered_at)
      },
      {
        key: 'trigger',
        header: '触发方式',
        render: (item) => triggerSourceLabel(item.trigger_source)
      },
      {
        key: 'status',
        header: '状态',
        render: (item) => (
          <StatusBadge tone={executionStatusTone(item.status)}>
            {executionStatusLabel(item.status)}
          </StatusBadge>
        )
      },
      {
        key: 'duration',
        header: '耗时',
        render: (item) => executionDuration(item)
      },
      {
        key: 'reason',
        header: '停止或失败原因',
        render: (item) => item.stop_reason || item.error_summary || '-'
      },
      {
        key: 'id',
        header: '执行 ID',
        render: (item) => <span className="collector-mono">{item.execution_id}</span>
      }
    ],
    []
  );

  const saveConfiguration = async () => {
    const normalizedPrompt = prompt.trim();
    if (!normalizedPrompt) {
      setError('Collection Prompt 不能为空');
      return;
    }
    if (scheduleType === 'daily' && dailyTimes.length === 0) {
      setError('每日定时至少需要一个执行时间');
      return;
    }
    if (scheduleType === 'cron' && !cronExpression.trim()) {
      setError('Cron 表达式不能为空');
      return;
    }
    setSaving(true);
    setError('');
    setNotice('');
    try {
      const nextSchedule = await saveAgentSchedule(token, agentKey, {
        agent_version: agentVersion,
        schedule_type: scheduleType,
        ...(scheduleType === 'daily'
          ? { daily_times: [...dailyTimes].sort() }
          : { cron_expression: cronExpression.trim() }),
        input: { prompt }
      });
      setSchedule(nextSchedule);
      setNotice('定时配置已保存');
    } catch (saveError) {
      setError(errorText(saveError));
    } finally {
      setSaving(false);
    }
  };

  const changeEnabled = async (enabled: boolean) => {
    setSaving(true);
    setError('');
    setNotice('');
    try {
      const nextSchedule = await setAgentScheduleEnabled(token, agentKey, enabled);
      setSchedule(nextSchedule);
      setNotice(enabled ? '定时器已启动' : '定时器已停止');
      setStopConfirmation(false);
    } catch (changeError) {
      setError(errorText(changeError));
    } finally {
      setSaving(false);
    }
  };

  const addDailyTime = () => {
    if (!newDailyTime || dailyTimes.includes(newDailyTime)) {
      return;
    }
    setDailyTimes((current) => [...current, newDailyTime].sort());
  };

  const retryCurrentSection = () => {
    if (section === 'executions') {
      setExecutionReloadVersion((value) => value + 1);
      return;
    }
    setReloadVersion((value) => value + 1);
  };

  if (loading) {
    return <div className="collector-panel-state">正在加载采集器配置</div>;
  }

  return (
    <TabPanel label="采集器配置">
      <section className="collector-management">
        <header className="collector-agent-header">
          <div>
            <div className="collector-eyebrow">AGENTRUN CONTROL PLANE</div>
            <div className="collector-agent-title-row">
              <h2>综合采集 Agent</h2>
              <StatusBadge tone={schedule?.enabled ? 'success' : 'neutral'}>
                {schedule?.enabled ? '已启用' : '已停止'}
              </StatusBadge>
            </div>
            <p>
              <span className="collector-mono">collector</span>
              <span aria-hidden="true"> · </span>
              <span className="collector-mono">collector.v1</span>
            </p>
          </div>
          {schedule?.enabled ? (
            <Button
              disabled={saving}
              onClick={() => setStopConfirmation(true)}
              variant="danger"
            >
              停止定时器
            </Button>
          ) : (
            <Button
              disabled={saving || !readinessComplete || !schedule}
              onClick={() => void changeEnabled(true)}
            >
              启动定时器
            </Button>
          )}
        </header>

        <div className="collector-section-tabs">
          <Tabs
            active={section}
            ariaLabel="采集器配置板块"
            items={sectionItems}
            onChange={setSection}
          />
        </div>

        {error ? (
          <div className="ui-alert danger collector-local-alert">
            <span>{error}</span>
            <button onClick={retryCurrentSection} type="button">
              重试
            </button>
          </div>
        ) : null}
        {notice ? <div className="ui-alert success">{notice}</div> : null}

        {section === 'schedule' ? (
          <SchedulePanel
            configuredConnectors={configuredConnectors}
            configuredModels={configuredModels}
            connectorsTotal={connectors.length}
            cronExpression={cronExpression}
            dailyTimes={dailyTimes}
            newDailyTime={newDailyTime}
            prompt={prompt}
            readinessComplete={readinessComplete}
            modelsTotal={models.length}
            saving={saving}
            schedule={schedule}
            scheduleType={scheduleType}
            onAddDailyTime={addDailyTime}
            onCronExpressionChange={setCronExpression}
            onDailyTimeRemove={(time) =>
              setDailyTimes((current) => current.filter((value) => value !== time))
            }
            onNewDailyTimeChange={setNewDailyTime}
            onPromptChange={setPrompt}
            onSave={() => void saveConfiguration()}
            onScheduleTypeChange={setScheduleType}
            onSectionChange={setSection}
          />
        ) : null}

        {section === 'executions' ? (
          <TabPanel label="采集执行记录">
            <div className="collector-panel-heading">
              <div>
                <h3>采集执行记录</h3>
                <p>只展示 Collector Execution 的安全审计摘要。</p>
              </div>
              <Button
                onClick={() => setExecutionReloadVersion((value) => value + 1)}
                variant="secondary"
              >
                刷新
              </Button>
            </div>
            <div className="collector-surface">
              <DataTable
                columns={executionColumns}
                emptyText={executionLoading ? '正在加载执行记录' : '暂无执行记录'}
                getRowKey={(item) => item.execution_id}
                items={executions.items}
              />
              <Pagination
                page={executions.page}
                pageSize={executions.page_size}
                total={executions.total_items}
                onPageChange={setExecutionPage}
              />
            </div>
          </TabPanel>
        ) : null}

        {section === 'models' ? (
          <ConfigurationTable
            kind="model"
            models={models}
            onEdit={(value) => setEditTarget({ kind: 'model', value })}
          />
        ) : null}

        {section === 'connectors' ? (
          <ConfigurationTable
            connectors={connectors}
            kind="connector"
            onEdit={(value) => setEditTarget({ kind: 'connector', value })}
          />
        ) : null}
      </section>

      {stopConfirmation ? (
        <ConfirmationDialog
          busy={saving}
          onCancel={() => setStopConfirmation(false)}
          onConfirm={() => void changeEnabled(false)}
        />
      ) : null}

      {editTarget ? (
        <ConfigurationDrawer
          target={editTarget}
          token={token}
          onClose={() => setEditTarget(null)}
          onConnectorSaved={(configuration) => {
            setConnectors((current) =>
              current.map((item) =>
                item.connector_key === configuration.connector_key ? configuration : item
              )
            );
            setEditTarget(null);
            setNotice('连接器配置已保存');
          }}
          onError={setError}
          onModelSaved={(configuration) => {
            setModels((current) =>
              current.map((item) =>
                item.provider_key === configuration.provider_key ? configuration : item
              )
            );
            setEditTarget(null);
            setNotice('模型配置已保存');
          }}
        />
      ) : null}
    </TabPanel>
  );
}

interface SchedulePanelProps {
  configuredConnectors: number;
  configuredModels: number;
  connectorsTotal: number;
  cronExpression: string;
  dailyTimes: string[];
  newDailyTime: string;
  prompt: string;
  readinessComplete: boolean;
  modelsTotal: number;
  saving: boolean;
  schedule: AgentSchedule | null;
  scheduleType: ScheduleType;
  onAddDailyTime: () => void;
  onCronExpressionChange: (value: string) => void;
  onDailyTimeRemove: (time: string) => void;
  onNewDailyTimeChange: (value: string) => void;
  onPromptChange: (value: string) => void;
  onSave: () => void;
  onScheduleTypeChange: (value: ScheduleType) => void;
  onSectionChange: (value: CollectorSection) => void;
}

function SchedulePanel(props: SchedulePanelProps) {
  return (
    <TabPanel label="定时任务配置">
      <div
        className={`collector-readiness ${props.readinessComplete ? 'ready' : 'incomplete'}`}
      >
        <div className="collector-readiness-copy">
          <strong>
            {props.readinessComplete
              ? `模型和 ${props.connectorsTotal} 个连接器配置完整`
              : '配置尚未完整，暂不能启动定时器'}
          </strong>
          <span>最终校验由 AgentRun 执行</span>
        </div>
        {!props.readinessComplete ? (
          <div className="collector-readiness-actions">
            {props.modelsTotal === 0 || props.configuredModels < props.modelsTotal ? (
              <button onClick={() => props.onSectionChange('models')} type="button">
                前往模型配置
              </button>
            ) : null}
            {props.connectorsTotal === 0 ||
            props.configuredConnectors < props.connectorsTotal ? (
              <button onClick={() => props.onSectionChange('connectors')} type="button">
                前往连接器配置
              </button>
            ) : null}
          </div>
        ) : null}
      </div>

      <div className="collector-schedule-grid">
        <section className="collector-surface">
          <header className="collector-surface-head">
            <div>
              <h3>定时配置</h3>
              <p>保存后影响下一次触发，不改变当前启停状态。</p>
            </div>
            <StatusBadge>{props.schedule ? '已保存' : '未保存'}</StatusBadge>
          </header>

          <div className="collector-form-stack">
            <fieldset className="collector-fieldset">
              <legend>执行策略</legend>
              <div className="collector-segmented">
                <button
                  aria-pressed={props.scheduleType === 'daily'}
                  className={props.scheduleType === 'daily' ? 'active' : ''}
                  onClick={() => props.onScheduleTypeChange('daily')}
                  type="button"
                >
                  每日定时
                </button>
                <button
                  aria-pressed={props.scheduleType === 'cron'}
                  className={props.scheduleType === 'cron' ? 'active' : ''}
                  onClick={() => props.onScheduleTypeChange('cron')}
                  type="button"
                >
                  Cron
                </button>
              </div>
            </fieldset>

            {props.scheduleType === 'daily' ? (
              <div className="collector-form-field">
                <div className="collector-field-label">
                  <strong>每日执行时间</strong>
                  <span>使用 AgentRun 服务器时间</span>
                </div>
                <div className="collector-time-row">
                  {props.dailyTimes.map((time) => (
                    <span className="collector-time-chip" key={time}>
                      {time}
                      <button
                        aria-label={`移除 ${time}`}
                        onClick={() => props.onDailyTimeRemove(time)}
                        type="button"
                      >
                        ×
                      </button>
                    </span>
                  ))}
                  <Input
                    aria-label="新增执行时间"
                    className="collector-time-input"
                    onChange={(event) => props.onNewDailyTimeChange(event.target.value)}
                    type="time"
                    value={props.newDailyTime}
                  />
                  <Button onClick={props.onAddDailyTime} variant="secondary">
                    添加时间
                  </Button>
                </div>
              </div>
            ) : (
              <Field hint="标准五段式，最小粒度为分钟" label="Cron 表达式">
                <Input
                  aria-label="Cron 表达式"
                  className="collector-mono"
                  onChange={(event) => props.onCronExpressionChange(event.target.value)}
                  value={props.cronExpression}
                />
              </Field>
            )}

            <Field hint="由 collector.v1 校验" label="Collection Prompt">
              <textarea
                aria-label="Collection Prompt"
                className="collector-textarea"
                onChange={(event) => props.onPromptChange(event.target.value)}
                value={props.prompt}
              />
            </Field>
          </div>
          <footer className="collector-form-actions">
            <span>
              {props.schedule ? `上次保存：${formatDateTime(props.schedule.updated_at)}` : '尚未保存'}
            </span>
            <Button disabled={props.saving} onClick={props.onSave}>
              保存配置
            </Button>
          </footer>
        </section>

        <aside className="collector-surface">
          <header className="collector-surface-head">
            <div>
              <h3>运行信息</h3>
              <p>时间由 AgentRun 返回。</p>
            </div>
          </header>
          <dl className="collector-details">
            <DetailRow label="当前状态">
              <StatusBadge tone={props.schedule?.enabled ? 'success' : 'neutral'}>
                {props.schedule?.enabled ? '已启用' : '已停止'}
              </StatusBadge>
            </DetailRow>
            <DetailRow label="调度策略">
              {scheduleSummary(
                props.scheduleType,
                props.dailyTimes,
                props.cronExpression
              )}
            </DetailRow>
            <DetailRow label="下次计划">
              {props.schedule?.next_run_at ? formatDateTime(props.schedule.next_run_at) : '-'}
            </DetailRow>
            <DetailRow label="上次触发">
              {props.schedule?.last_triggered_at
                ? formatDateTime(props.schedule.last_triggered_at)
                : '-'}
            </DetailRow>
            <DetailRow label="模型配置">
              <StatusBadge
                tone={
                  props.modelsTotal > 0 && props.configuredModels === props.modelsTotal
                    ? 'success'
                    : 'danger'
                }
              >
                {`${props.configuredModels} / ${props.modelsTotal} 完整`}
              </StatusBadge>
            </DetailRow>
            <DetailRow label="连接器配置">
              <StatusBadge
                tone={
                  props.connectorsTotal > 0 &&
                  props.configuredConnectors === props.connectorsTotal
                    ? 'success'
                    : 'danger'
                }
              >
                {`${props.configuredConnectors} / ${props.connectorsTotal} 完整`}
              </StatusBadge>
            </DetailRow>
          </dl>
        </aside>
      </div>
    </TabPanel>
  );
}

function DetailRow({ children, label }: { children: React.ReactNode; label: string }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{children}</dd>
    </div>
  );
}

type ConfigurationTableProps =
  | {
      kind: 'model';
      models: ModelProviderConfiguration[];
      onEdit: (value: ModelProviderConfiguration) => void;
    }
  | {
      kind: 'connector';
      connectors: ConnectorConfiguration[];
      onEdit: (value: ConnectorConfiguration) => void;
    };

function ConfigurationTable(props: ConfigurationTableProps) {
  const isModel = props.kind === 'model';
  return (
    <TabPanel label={isModel ? '模型配置' : '连接器配置'}>
      <div className="collector-panel-heading">
        <div>
          <h3>{isModel ? '模型配置' : '连接器配置'}</h3>
          <p>
            {isModel
              ? '只维护 AgentRun 代码已注册的模型供应商。'
              : '连接器能力由代码注册，管理页只维护当前连接信息。'}
          </p>
        </div>
      </div>
      <div className="collector-surface collector-configuration-table">
        <table className="ui-data-table">
          <thead>
            <tr>
              <th>{isModel ? 'Provider' : 'Connector'}</th>
              <th>Base URL</th>
              {isModel ? <th>模型</th> : null}
              <th>Key</th>
              <th>状态</th>
              <th>更新时间</th>
              <th aria-label="操作" />
            </tr>
          </thead>
          <tbody>
            {isModel
              ? props.models.map((item) => (
                  <tr key={item.provider_key}>
                    <td>
                      <strong>{providerName(item.provider_key)}</strong>
                      <small className="collector-table-subtitle collector-mono">
                        {item.provider_key}
                      </small>
                    </td>
                    <td className="collector-mono">{item.base_url || '-'}</td>
                    <td className="collector-mono">{item.model || '-'}</td>
                    <td className="collector-mono">{item.masked_key || '未配置'}</td>
                    <td>
                      <StatusBadge tone={item.configured ? 'success' : 'danger'}>
                        {item.configured ? '已配置' : '未配置'}
                      </StatusBadge>
                    </td>
                    <td>{item.updated_at ? formatDateTime(item.updated_at) : '-'}</td>
                    <td>
                      <Button
                        aria-label={`编辑 ${item.provider_key}`}
                        onClick={() => props.onEdit(item)}
                        variant="secondary"
                      >
                        编辑
                      </Button>
                    </td>
                  </tr>
                ))
              : props.connectors.map((item) => (
                  <tr key={item.connector_key}>
                    <td>
                      <strong>{connectorName(item.connector_key)}</strong>
                      <small className="collector-table-subtitle collector-mono">
                        {item.connector_key}
                      </small>
                    </td>
                    <td className="collector-mono">{item.base_url || '-'}</td>
                    <td className="collector-mono">{item.masked_key || '未配置'}</td>
                    <td>
                      <StatusBadge tone={item.configured ? 'success' : 'danger'}>
                        {item.configured ? '已配置' : '未配置'}
                      </StatusBadge>
                    </td>
                    <td>{item.updated_at ? formatDateTime(item.updated_at) : '-'}</td>
                    <td>
                      <Button
                        aria-label={`编辑 ${item.connector_key}`}
                        onClick={() => props.onEdit(item)}
                        variant="secondary"
                      >
                        编辑
                      </Button>
                    </td>
                  </tr>
                ))}
          </tbody>
        </table>
      </div>
    </TabPanel>
  );
}

interface ConfigurationDrawerProps {
  target: EditTarget;
  token: string;
  onClose: () => void;
  onConnectorSaved: (value: ConnectorConfiguration) => void;
  onError: (message: string) => void;
  onModelSaved: (value: ModelProviderConfiguration) => void;
}

function ConfigurationDrawer(props: ConfigurationDrawerProps) {
  const isModel = props.target.kind === 'model';
  const [baseURL, setBaseURL] = useState(props.target.value.base_url);
  const [model, setModel] = useState(
    props.target.kind === 'model' ? props.target.value.model : ''
  );
  const [apiKey, setAPIKey] = useState('');
  const [clearKey, setClearKey] = useState(false);
  const [saving, setSaving] = useState(false);
  const key = isModel
    ? (props.target.value as ModelProviderConfiguration).provider_key
    : (props.target.value as ConnectorConfiguration).connector_key;

  const save = async () => {
    setSaving(true);
    props.onError('');
    try {
      if (props.target.kind === 'model') {
        const configuration = await updateModelProvider(props.token, key, {
          base_url: baseURL.trim(),
          model: model.trim(),
          ...(apiKey.trim() ? { api_key: apiKey.trim() } : {})
        });
        props.onModelSaved(configuration);
      } else {
        const configuration = await updateConnector(props.token, key, {
          base_url: baseURL.trim(),
          ...(clearKey ? { api_key: '' } : apiKey.trim() ? { api_key: apiKey.trim() } : {})
        });
        props.onConnectorSaved(configuration);
      }
    } catch (saveError) {
      props.onError(errorText(saveError));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="collector-overlay">
      <button
        aria-label="关闭配置编辑"
        className="collector-overlay-dismiss"
        onClick={props.onClose}
        type="button"
      />
      <aside
        aria-label={isModel ? `编辑模型 ${key}` : `编辑连接器 ${key}`}
        className="collector-drawer"
        role="dialog"
      >
        <header>
          <div>
            <h3>{isModel ? '编辑模型配置' : '编辑连接器配置'}</h3>
            <p className="collector-mono">{key}</p>
          </div>
          <button aria-label="关闭" onClick={props.onClose} type="button">
            ×
          </button>
        </header>
        <div className="collector-drawer-fields">
          <Field label="Base URL">
            <Input
              aria-label="Base URL"
              onChange={(event) => setBaseURL(event.target.value)}
              value={baseURL}
            />
          </Field>
          {isModel ? (
            <Field label="模型">
              <Input
                aria-label="模型"
                onChange={(event) => setModel(event.target.value)}
                value={model}
              />
            </Field>
          ) : null}
          <Field
            hint={
              isModel
                ? '留空保持当前 Key，不提供清除操作。'
                : '留空保持当前 Key；勾选下方选项可明确清除。'
            }
            label="新 API Key"
          >
            <Input
              aria-label="新 API Key"
              disabled={clearKey}
              onChange={(event) => setAPIKey(event.target.value)}
              type="password"
              value={apiKey}
            />
          </Field>
          {!isModel ? (
            <label className="collector-clear-key">
              <input
                aria-label="清除当前 Key"
                checked={clearKey}
                onChange={(event) => setClearKey(event.target.checked)}
                type="checkbox"
              />
              清除当前 Key
            </label>
          ) : null}
        </div>
        <footer>
          <Button onClick={props.onClose} variant="secondary">
            取消
          </Button>
          <Button disabled={saving} onClick={() => void save()}>
            {isModel ? '保存模型配置' : '保存连接器配置'}
          </Button>
        </footer>
      </aside>
    </div>
  );
}

function ConfirmationDialog(props: {
  busy: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  return (
    <div className="collector-overlay collector-confirmation-overlay">
      <section aria-label="停止定时器" className="collector-confirmation" role="dialog">
        <h3>停止定时器</h3>
        <p>停止后只会阻止未来触发，不会取消已经开始的采集执行。</p>
        <div>
          <Button onClick={props.onCancel} variant="secondary">
            取消
          </Button>
          <Button disabled={props.busy} onClick={props.onConfirm} variant="danger">
            确认停止
          </Button>
        </div>
      </section>
    </div>
  );
}

function promptFromSchedule(schedule: AgentSchedule): string {
  const value = schedule.input.prompt;
  return typeof value === 'string' ? value : '';
}

function scheduleSummary(
  type: ScheduleType,
  dailyTimes: string[],
  cronExpression: string
): string {
  if (type === 'daily') {
    return dailyTimes.length > 0 ? `每日 ${dailyTimes.join('、')}` : '每日定时未配置';
  }
  return `Cron ${cronExpression || '-'}`;
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString('zh-CN', {
    hour12: false,
    timeZone: 'Asia/Shanghai'
  });
}

function executionDuration(execution: AgentExecution): string {
  if (!execution.started_at || !execution.completed_at) {
    return '-';
  }
  const duration = Math.max(
    0,
    new Date(execution.completed_at).getTime() - new Date(execution.started_at).getTime()
  );
  const seconds = Math.floor(duration / 1000);
  const minutes = Math.floor(seconds / 60);
  return `${String(minutes).padStart(2, '0')}:${String(seconds % 60).padStart(2, '0')}`;
}

function triggerSourceLabel(value: string): string {
  if (value === 'schedule') {
    return '定时';
  }
  if (value === 'api') {
    return 'API';
  }
  return value || '-';
}

function executionStatusLabel(value: string): string {
  const labels: Record<string, string> = {
    queued: '排队中',
    running: '执行中',
    succeeded: '已完成',
    failed: '失败',
    skipped: '已跳过',
    cancelled: '已取消'
  };
  return labels[value] ?? value;
}

function executionStatusTone(value: string): 'success' | 'danger' | 'neutral' {
  if (value === 'succeeded') {
    return 'success';
  }
  if (value === 'failed' || value === 'cancelled') {
    return 'danger';
  }
  return 'neutral';
}

function providerName(key: string): string {
  return key === 'deepseek' ? 'DeepSeek' : key;
}

function connectorName(key: string): string {
  const names: Record<string, string> = {
    parallel_search: 'Parallel Search',
    tavily: 'Tavily',
    bocha: '博查',
    cls_telegraph: '财联社电报',
    eastmoney_fastnews: '东方财富快讯',
    eastmoney_stock_news: '东方财富个股资讯',
    stcn_quicknews: '证券时报快讯'
  };
  return names[key] ?? key;
}

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : 'AgentRun 暂时不可用';
}
