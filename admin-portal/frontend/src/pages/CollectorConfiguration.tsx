import { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import {
  AdminAgentRunAPIError,
  loadAgentSchedule,
  loadConnectors,
  loadModelProviders,
  saveAgentSchedule,
  setAgentScheduleEnabled,
  updateConnector,
  updateModelProvider,
  type AgentSchedule,
  type ConnectorConfiguration,
  type ModelProviderConfiguration,
  type ScheduleType
} from '../api/agentManagement';
import { OverflowTooltip } from '../components/admin/overflow-tooltip';
import StatusAlert from '../components/admin/status-alert';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '../components/ui/AlertDialog';
import { Button } from '../components/ui/Button';
import { Checkbox } from '../components/ui/Checkbox';
import { Field } from '../components/ui/Field';
import { Input } from '../components/ui/Input';
import { StatusBadge } from '../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/Tabs';
import { Textarea } from '../components/ui/Textarea';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '../components/ui/table';
import { Sheet, SheetContent, SheetDescription, SheetTitle } from '../components/ui/sheet';

type CollectorSection = 'schedule' | 'models' | 'connectors';
type EditTarget =
  | { kind: 'model'; value: ModelProviderConfiguration }
  | { kind: 'connector'; value: ConnectorConfiguration };

const agentKey = 'collector';
const agentVersion = 'collector.v1';
const sectionItems: { id: CollectorSection; label: string }[] = [
  { id: 'schedule', label: '定时配置' },
  { id: 'models', label: '模型配置' },
  { id: 'connectors', label: '连接器配置' }
];

const sectionTabsListClassName =
  'h-11 w-full justify-start rounded-none border-y bg-transparent px-4 py-0';
const sectionTabClassName =
  'relative h-full flex-none rounded-none border-0 px-3.5 py-0 text-xs font-medium shadow-none after:absolute after:inset-x-0 after:-bottom-px after:h-0.5 after:bg-transparent data-[state=active]:border-0 data-[state=active]:bg-transparent data-[state=active]:text-primary data-[state=active]:shadow-none data-[state=active]:after:bg-primary';

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
  const [loading, setLoading] = useState(true);
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

  const configuredModels = models.filter((model) => model.configured).length;
  const configuredConnectors = connectors.filter((connector) => connector.configured).length;
  const readinessComplete =
    models.length > 0 &&
    configuredModels === models.length &&
    connectors.length > 0 &&
    configuredConnectors === connectors.length;

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
    setReloadVersion((value) => value + 1);
  };

  if (loading) {
    return (
      <div className='grid min-h-56 place-items-center rounded-lg border border-dashed text-sm text-muted-foreground'>
        正在加载采集器配置
      </div>
    );
  }

  return (
    <Tabs
      className='grid min-h-full content-start'
      onValueChange={(value) => isCollectorSection(value) && setSection(value)}
      value={section}
    >
      <div className='flex items-center justify-between gap-4 px-5 py-4 max-sm:items-start'>
        <div>
          <div className='flex items-center gap-2.5'>
            <h2 className='m-0 text-lg font-semibold tracking-[-0.015em]'>综合采集 Agent</h2>
            <StatusBadge tone={schedule?.enabled ? 'success' : 'neutral'}>
              {schedule?.enabled ? '已启用' : '已停止'}
            </StatusBadge>
          </div>
          <p className='mb-0 mt-1 font-mono text-xs text-muted-foreground'>
            <span className='font-mono'>collector</span>
            <span aria-hidden='true'> · </span>
            <span className='font-mono'>collector.v1</span>
          </p>
        </div>
        {schedule?.enabled ? (
          <Button disabled={saving} onClick={() => setStopConfirmation(true)} variant='destructive'>
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
      </div>

      <div className='overflow-x-auto'>
        <TabsList aria-label='采集器配置板块' className={sectionTabsListClassName}>
          {sectionItems.map((item) => (
            <TabsTrigger className={sectionTabClassName} key={item.id} value={item.id}>
              {item.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </div>

      {error ? (
        <div className='px-4 pt-4'>
          <StatusAlert actionLabel='重试' onAction={retryCurrentSection} tone='destructive'>
            {error}
          </StatusAlert>
        </div>
      ) : null}
      {notice ? (
        <div className='px-4 pt-4'>
          <StatusAlert role='status' tone='success'>
            {notice}
          </StatusAlert>
        </div>
      ) : null}

      <TabsContent className='p-4' value='schedule'>
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
      </TabsContent>

      <TabsContent className='p-4' value='models'>
        <ConfigurationTable
          kind='model'
          models={models}
          onEdit={(value) => setEditTarget({ kind: 'model', value })}
        />
      </TabsContent>

      <TabsContent className='p-4' value='connectors'>
        <ConfigurationTable
          connectors={connectors}
          kind='connector'
          onEdit={(value) => setEditTarget({ kind: 'connector', value })}
        />
      </TabsContent>

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
    </Tabs>
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
    <section aria-label='定时任务配置' className='grid gap-3.5'>
      {!props.readinessComplete ? (
        <div className='flex items-center justify-between gap-4 rounded-lg border border-destructive-border bg-destructive-subtle px-4 py-3 text-destructive-foreground max-sm:flex-col max-sm:items-stretch'>
          <div className='grid gap-1'>
            <strong className='text-sm'>配置尚未完整，暂不能启动定时器</strong>
            <span className='text-xs text-muted-foreground'>最终校验由 AgentRun 执行</span>
          </div>
          <div className='flex flex-wrap gap-2'>
            {props.modelsTotal === 0 || props.configuredModels < props.modelsTotal ? (
              <Button onClick={() => props.onSectionChange('models')} size='sm' variant='outline'>
                前往模型配置
              </Button>
            ) : null}
            {props.connectorsTotal === 0 || props.configuredConnectors < props.connectorsTotal ? (
              <Button
                onClick={() => props.onSectionChange('connectors')}
                size='sm'
                variant='outline'
              >
                前往连接器配置
              </Button>
            ) : null}
          </div>
        </div>
      ) : null}

      <div className='grid gap-3.5 xl:grid-cols-[minmax(0,1.4fr)_minmax(17.5rem,1fr)]'>
        <section className='overflow-hidden rounded-lg border bg-card p-4'>
          <header className='mb-4 flex items-start justify-between gap-4'>
            <div>
              <h3 className='m-0 text-sm font-semibold'>执行计划</h3>
              <p className='mb-0 mt-1 text-xs leading-5 text-muted-foreground'>
                保存后影响下一次触发，不改变当前启停状态。
              </p>
            </div>
            <StatusBadge>{props.schedule ? '已保存' : '未保存'}</StatusBadge>
          </header>

          <div className='grid gap-4 [&>div]:gap-1.5 [&_input]:text-xs [&_label]:text-xs [&_label]:text-muted-foreground [&_textarea]:text-xs'>
            <fieldset className='m-0 border-0 p-0'>
              <legend className='mb-1.5 text-xs font-medium text-muted-foreground'>执行策略</legend>
              <Tabs
                onValueChange={(value) =>
                  props.onScheduleTypeChange(value === 'cron' ? 'cron' : 'daily')
                }
                value={props.scheduleType}
              >
                <TabsList aria-label='执行策略' className='h-8'>
                  <TabsTrigger className='text-xs' value='daily'>
                    每日定时
                  </TabsTrigger>
                  <TabsTrigger className='text-xs' value='cron'>
                    Cron
                  </TabsTrigger>
                </TabsList>
              </Tabs>
            </fieldset>

            {props.scheduleType === 'daily' ? (
              <div className='grid gap-2'>
                <div className='flex justify-between gap-3'>
                  <strong className='text-xs font-medium text-muted-foreground'>
                    每日执行时间
                  </strong>
                  <span className='text-xs text-muted-foreground'>使用 AgentRun 服务器时间</span>
                </div>
                <div className='flex flex-wrap items-center gap-2'>
                  {props.dailyTimes.map((time) => (
                    <span
                      className='inline-flex min-h-8 items-center gap-1.5 rounded-full bg-secondary py-0 pl-3 pr-1.5 font-mono text-xs'
                      key={time}
                    >
                      {time}
                      <Button
                        aria-label={`移除 ${time}`}
                        className='size-6 rounded-full text-muted-foreground'
                        onClick={() => props.onDailyTimeRemove(time)}
                        size='icon'
                        variant='ghost'
                      >
                        <X className='size-3' />
                      </Button>
                    </span>
                  ))}
                  <Input
                    aria-label='新增执行时间'
                    className='max-w-32'
                    onChange={(event) => props.onNewDailyTimeChange(event.target.value)}
                    type='time'
                    value={props.newDailyTime}
                  />
                  <Button onClick={props.onAddDailyTime} variant='secondary'>
                    添加时间
                  </Button>
                </div>
              </div>
            ) : (
              <Field hint='标准五段式，最小粒度为分钟' label='Cron 表达式'>
                <Input
                  aria-label='Cron 表达式'
                  className='font-mono'
                  onChange={(event) => props.onCronExpressionChange(event.target.value)}
                  value={props.cronExpression}
                />
              </Field>
            )}

            <Field hint='由 collector.v1 校验' label='Collection Prompt'>
              <Textarea
                aria-label='Collection Prompt'
                className='min-h-24 resize-y text-sm leading-5'
                onChange={(event) => props.onPromptChange(event.target.value)}
                value={props.prompt}
              />
            </Field>
          </div>
          <footer className='mt-4 flex items-center justify-between gap-4 border-t pt-3.5 text-xs text-muted-foreground max-sm:flex-col max-sm:items-stretch'>
            <span>
              {props.schedule
                ? `上次保存：${formatDateTime(props.schedule.updated_at)}`
                : '尚未保存'}
            </span>
            <Button disabled={props.saving} onClick={props.onSave}>
              保存配置
            </Button>
          </footer>
        </section>

        <aside className='overflow-hidden rounded-lg border bg-card p-4'>
          <header className='mb-2 flex items-start justify-between gap-4'>
            <div>
              <h3 className='m-0 text-sm font-semibold'>配置就绪</h3>
              <p className='mb-0 mt-1 text-xs leading-5 text-muted-foreground'>
                配置与运行时间由 AgentRun 返回。
              </p>
            </div>
          </header>
          <dl className='m-0 grid'>
            <DetailRow label='当前状态'>
              <StatusBadge tone={props.schedule?.enabled ? 'success' : 'neutral'}>
                {props.schedule?.enabled ? '已启用' : '已停止'}
              </StatusBadge>
            </DetailRow>
            <DetailRow label='调度策略'>
              {scheduleSummary(props.scheduleType, props.dailyTimes, props.cronExpression)}
            </DetailRow>
            <DetailRow label='下次计划'>
              {props.schedule?.next_run_at ? formatDateTime(props.schedule.next_run_at) : '-'}
            </DetailRow>
            <DetailRow label='上次触发'>
              {props.schedule?.last_triggered_at
                ? formatDateTime(props.schedule.last_triggered_at)
                : '-'}
            </DetailRow>
            <DetailRow label='模型配置'>
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
            <DetailRow label='连接器配置'>
              <StatusBadge
                tone={
                  props.connectorsTotal > 0 && props.configuredConnectors === props.connectorsTotal
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
    </section>
  );
}

function DetailRow({ children, label }: { children: React.ReactNode; label: string }) {
  return (
    <div className='grid grid-cols-[minmax(5.625rem,0.8fr)_minmax(0,1.2fr)] items-center gap-4 border-b py-2.5 text-xs last:border-b-0'>
      <dt className='text-muted-foreground'>{label}</dt>
      <dd className='m-0 text-right font-medium leading-5'>{children}</dd>
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
    <section aria-label={isModel ? '模型配置' : '连接器配置'} className='grid gap-3.5'>
      <div className='flex items-start justify-between gap-4'>
        <div>
          <h3 className='m-0 text-sm font-semibold'>{isModel ? '模型配置' : '连接器配置'}</h3>
          <p className='mb-0 mt-1 text-xs leading-5 text-muted-foreground'>
            {isModel
              ? '只维护 AgentRun 代码已注册的模型供应商。'
              : '连接器能力由代码注册，管理页只维护当前连接信息。'}
          </p>
        </div>
      </div>
      <div className='overflow-hidden rounded-lg border bg-card'>
        <Table className='table-fixed text-xs [&_td]:py-2.5 [&_th]:h-9'>
          <TableHeader>
            <TableRow className='hover:bg-transparent'>
              <TableHead className={isModel ? 'w-[18%]' : 'w-[20%]'}>
                {isModel ? 'Provider' : 'Connector'}
              </TableHead>
              <TableHead className={isModel ? 'w-[23%]' : 'w-[34%]'}>Base URL</TableHead>
              {isModel ? <TableHead className='w-[15%]'>模型</TableHead> : null}
              <TableHead className='w-[12%]'>Key</TableHead>
              <TableHead className='w-[10%]'>状态</TableHead>
              <TableHead className={isModel ? 'w-[15%]' : 'w-[16%]'}>更新时间</TableHead>
              <TableHead aria-label='操作' className={isModel ? 'w-[7%]' : 'w-[8%]'} />
            </TableRow>
          </TableHeader>
          <TableBody>
            {isModel
              ? props.models.map((item) => (
                  <TableRow key={item.provider_key}>
                    <TableCell className='max-w-0'>
                      <strong className='block truncate'>{providerName(item.provider_key)}</strong>
                      <small className='mt-1 block truncate font-mono text-muted-foreground'>
                        {item.provider_key}
                      </small>
                    </TableCell>
                    <TableCell className='max-w-0'>
                      <OverflowTooltip className='font-mono' value={item.base_url || '-'} />
                    </TableCell>
                    <TableCell className='max-w-0'>
                      <OverflowTooltip className='font-mono' value={item.model || '-'} />
                    </TableCell>
                    <TableCell className='max-w-0'>
                      <OverflowTooltip className='font-mono' value={item.masked_key || '未配置'} />
                    </TableCell>
                    <TableCell>
                      <StatusBadge tone={item.configured ? 'success' : 'danger'}>
                        {item.configured ? '已配置' : '未配置'}
                      </StatusBadge>
                    </TableCell>
                    <TableCell className='max-w-0'>
                      <OverflowTooltip
                        value={item.updated_at ? formatDateTime(item.updated_at) : '-'}
                      />
                    </TableCell>
                    <TableCell className='text-right'>
                      <Button
                        aria-label={`编辑 ${item.provider_key}`}
                        className='text-xs'
                        onClick={() => props.onEdit(item)}
                        size='sm'
                        variant='secondary'
                      >
                        编辑
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              : props.connectors.map((item) => (
                  <TableRow key={item.connector_key}>
                    <TableCell className='max-w-0'>
                      <strong className='block truncate'>
                        {connectorName(item.connector_key)}
                      </strong>
                      <small className='mt-1 block truncate font-mono text-muted-foreground'>
                        {item.connector_key}
                      </small>
                    </TableCell>
                    <TableCell className='max-w-0'>
                      <OverflowTooltip className='font-mono' value={item.base_url || '-'} />
                    </TableCell>
                    <TableCell className='max-w-0'>
                      <OverflowTooltip className='font-mono' value={item.masked_key || '未配置'} />
                    </TableCell>
                    <TableCell>
                      <StatusBadge tone={item.configured ? 'success' : 'danger'}>
                        {item.configured ? '已配置' : '未配置'}
                      </StatusBadge>
                    </TableCell>
                    <TableCell className='max-w-0'>
                      <OverflowTooltip
                        value={item.updated_at ? formatDateTime(item.updated_at) : '-'}
                      />
                    </TableCell>
                    <TableCell className='text-right'>
                      <Button
                        aria-label={`编辑 ${item.connector_key}`}
                        className='text-xs'
                        onClick={() => props.onEdit(item)}
                        size='sm'
                        variant='secondary'
                      >
                        编辑
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
          </TableBody>
        </Table>
      </div>
    </section>
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
  const [model, setModel] = useState(props.target.kind === 'model' ? props.target.value.model : '');
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
    <Sheet open onOpenChange={(open) => !open && props.onClose()}>
      <SheetContent
        aria-label={isModel ? `编辑模型 ${key}` : `编辑连接器 ${key}`}
        className='grid grid-rows-[auto_minmax(0,1fr)_auto] gap-0'
        closeLabel='关闭'
        side='right'
      >
        <header className='border-b pb-5 pr-10'>
          <div>
            <SheetTitle>{isModel ? '编辑模型配置' : '编辑连接器配置'}</SheetTitle>
            <SheetDescription className='sr-only'>
              {isModel ? '更新已注册模型供应商配置' : '更新已注册连接器配置'}
            </SheetDescription>
            <p className='mt-1.5 font-mono text-sm text-muted-foreground'>{key}</p>
          </div>
        </header>
        <div className='grid min-h-0 content-start gap-5 overflow-y-auto py-6'>
          <Field label='Base URL'>
            <Input
              aria-label='Base URL'
              onChange={(event) => setBaseURL(event.target.value)}
              value={baseURL}
            />
          </Field>
          {isModel ? (
            <Field label='模型'>
              <Input
                aria-label='模型'
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
            label='新 API Key'
          >
            <Input
              aria-label='新 API Key'
              disabled={clearKey}
              onChange={(event) => setAPIKey(event.target.value)}
              type='password'
              value={apiKey}
            />
          </Field>
          {!isModel ? (
            <label className='flex items-center gap-2 text-sm text-destructive-foreground'>
              <Checkbox
                aria-label='清除当前 Key'
                checked={clearKey}
                onCheckedChange={(checked) => setClearKey(checked === true)}
              />
              清除当前 Key
            </label>
          ) : null}
        </div>
        <footer className='flex justify-end gap-2 border-t pt-5'>
          <Button onClick={props.onClose} variant='outline'>
            取消
          </Button>
          <Button disabled={saving} onClick={() => void save()}>
            {isModel ? '保存模型配置' : '保存连接器配置'}
          </Button>
        </footer>
      </SheetContent>
    </Sheet>
  );
}

function ConfirmationDialog(props: { busy: boolean; onCancel: () => void; onConfirm: () => void }) {
  return (
    <AlertDialog open onOpenChange={(open) => !open && props.onCancel()}>
      <AlertDialogContent aria-label='停止定时器'>
        <AlertDialogHeader>
          <AlertDialogTitle>停止定时器</AlertDialogTitle>
          <AlertDialogDescription>
            停止后只会阻止未来触发，不会取消已经开始的采集执行。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel asChild>
            <Button variant='outline'>取消</Button>
          </AlertDialogCancel>
          <AlertDialogAction asChild>
            <Button disabled={props.busy} onClick={props.onConfirm} variant='destructive'>
              确认停止
            </Button>
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function promptFromSchedule(schedule: AgentSchedule): string {
  const value = schedule.input.prompt;
  return typeof value === 'string' ? value : '';
}

function scheduleSummary(type: ScheduleType, dailyTimes: string[], cronExpression: string): string {
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

function isCollectorSection(value: string): value is CollectorSection {
  return sectionItems.some((item) => item.id === value);
}
