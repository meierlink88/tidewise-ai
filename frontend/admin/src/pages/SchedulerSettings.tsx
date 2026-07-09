import { useEffect, useState } from 'react';
import type { SchedulerConfig, SchedulerRun } from '../api/scheduler';
import {
  loadSchedulerConfig as defaultLoadSchedulerConfig,
  loadSchedulerRuns as defaultLoadSchedulerRuns,
  saveSchedulerConfig as defaultSaveSchedulerConfig
} from '../api/scheduler';
import Button from '../components/ui/Button';
import Card from '../components/ui/Card';
import Field from '../components/ui/Field';
import Input from '../components/ui/Input';
import Select from '../components/ui/Select';
import StatusBadge from '../components/ui/StatusBadge';
import Switch from '../components/ui/Switch';

interface SchedulerSettingsProps {
  token: string;
  loadConfig?: (token: string) => Promise<SchedulerConfig>;
  loadRuns?: (token: string, limit?: number) => Promise<SchedulerRun[]>;
  saveConfig?: (token: string, config: SchedulerConfig) => Promise<unknown>;
}

const defaultConfig: SchedulerConfig = {
  enabled: false,
  mode: 'interval',
  interval_minutes: 60,
  fixed_times: ['09:00', '12:00', '15:00', '18:00', '21:00'],
  concurrency: 1,
  batch_size: 10,
  timeout_seconds: 180,
  source_filter: {},
  timezone: 'Asia/Shanghai'
};

export default function SchedulerSettings({
  token,
  loadConfig = defaultLoadSchedulerConfig,
  loadRuns = defaultLoadSchedulerRuns,
  saveConfig = defaultSaveSchedulerConfig
}: SchedulerSettingsProps) {
  const [config, setConfig] = useState<SchedulerConfig>(defaultConfig);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [recentRun, setRecentRun] = useState<SchedulerConfig['recent_run']>();
  const [recentRuns, setRecentRuns] = useState<SchedulerRun[]>([]);
  const [notice, setNotice] = useState<{ tone: 'success' | 'danger'; text: string }>();
  const hasToken = token.trim().length > 0;

  useEffect(() => {
    let active = true;
    setLoading(true);
    loadConfig(token)
      .then((loadedConfig) => {
        if (!active) {
          return;
        }
        const next = normalizeConfig(loadedConfig);
        setConfig(next);
        setRecentRun(loadedConfig.recent_run);
        if (hasToken) {
          void loadRuns(token, 50)
            .then((runs) => {
              if (active) {
                setRecentRuns(runs);
              }
            })
            .catch(() => {
              if (active) {
                setRecentRuns([]);
              }
            });
        } else {
          setRecentRuns([]);
        }
      })
      .catch(() => {
        if (active) {
          setConfig(defaultConfig);
          setRecentRuns([]);
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
  }, [hasToken, loadConfig, loadRuns, token]);

  const handleSave = async () => {
    if (!hasToken) {
      setNotice({ tone: 'danger', text: '请先登录' });
      return;
    }
    const validationError = validateConfig(config);
    if (validationError) {
      setNotice({ tone: 'danger', text: validationError });
      return;
    }
    const payload = normalizeConfig({
      ...config,
      timezone: defaultConfig.timezone
    });
    setSaving(true);
    setNotice(undefined);
    try {
      const saved = await saveConfig(token, payload);
      if (saved && typeof saved === 'object') {
        setConfig(normalizeConfig(saved as SchedulerConfig));
      }
      setNotice({ tone: 'success', text: '已保存' });
    } catch (error) {
      setNotice({ tone: 'danger', text: `保存失败：${error instanceof Error ? error.message : '未知错误'}` });
    } finally {
      setSaving(false);
    }
  };

  const summary = executionSummary(recentRuns);

  return (
    <section className="scheduler-settings">
      <div className="page-title">
        <span className="eyebrow">Ingestion Scheduler</span>
        <h1>调度器设置</h1>
      </div>
      <div className="scheduler-grid">
        <Card className="scheduler-form-card">
          {loading ? (
            <div className="loading-state">正在加载调度器设置</div>
          ) : (
            <div className="form-grid scheduler-form-grid">
              <Switch
                checked={config.enabled}
                label="启用调度"
                onChange={(enabled) => setConfig((current) => ({ ...current, enabled }))}
              />
              <Field label="调度模式">
                <Select
                  aria-label="调度模式"
                  onChange={(event) => setConfig((current) => ({ ...current, mode: event.target.value as SchedulerConfig['mode'] }))}
                  value={config.mode}
                >
                  <option value="interval">间隔运行</option>
                  <option value="fixed_times">固定时间</option>
                </Select>
              </Field>
              {config.mode === 'interval' ? (
                <Field label="间隔分钟">
                  <Input
                    min={1}
                    onChange={(event) => setConfig((current) => ({ ...current, interval_minutes: numberValue(event.target.value) }))}
                    type="number"
                    value={config.interval_minutes}
                  />
                </Field>
              ) : (
                <div className="fixed-time-grid">
                  {[0, 1, 2, 3, 4].map((index) => (
                    <Field key={index} label={`固定时间 ${index + 1}`}>
                      <Input
                        aria-label={`固定时间 ${index + 1}`}
                        onChange={(event) => setConfig((current) => ({
                          ...current,
                          fixed_times: updateFixedTime(current.fixed_times, index, event.target.value)
                        }))}
                        placeholder="09:00"
                        value={config.fixed_times[index] ?? ''}
                      />
                    </Field>
                  ))}
                </div>
              )}
              <Field label="并发数">
                <Input
                  min={1}
                  onChange={(event) => setConfig((current) => ({ ...current, concurrency: numberValue(event.target.value) }))}
                  type="number"
                  value={config.concurrency}
                />
              </Field>
              <Field label="批次大小">
                <Input
                  min={1}
                  onChange={(event) => setConfig((current) => ({ ...current, batch_size: numberValue(event.target.value) }))}
                  type="number"
                  value={config.batch_size}
                />
              </Field>
              <Field label="超时秒数">
                <Input
                  min={1}
                  onChange={(event) => setConfig((current) => ({ ...current, timeout_seconds: numberValue(event.target.value) }))}
                  type="number"
                  value={config.timeout_seconds}
                />
              </Field>
            </div>
          )}
          <div className="form-actions">
            <Button disabled={!hasToken || saving || loading} onClick={handleSave}>
              {saving ? '保存中' : '保存设置'}
            </Button>
            {notice ? <div className={`ui-alert ${notice.tone}`}>{notice.text}</div> : null}
          </div>
        </Card>
        <Card className="scheduler-records" aria-label="调度器执行记录">
          <div className="card-heading-row">
            <div>
              <span className="eyebrow">Latest 50</span>
              <h2>执行记录</h2>
            </div>
            <StatusBadge tone={summary.failed > 0 ? 'danger' : 'success'}>{summary.total ? `${summary.total} 轮` : '暂无'}</StatusBadge>
          </div>
          {recentRun ? (
            <div className="run-summary">
              <div>
                <span className="summary-label">最近一轮：{recentRun.status}</span>
                <StatusBadge tone={statusTone(recentRun.status)}>{recentRun.status}</StatusBadge>
              </div>
              <div>
                <span className="summary-label">执行轮次 {summary.total}</span>
                <strong>{summary.total}</strong>
              </div>
              <div>
                <span className="summary-label">结果统计</span>
                <strong>成功 {summary.succeeded} / 失败 {summary.failed}</strong>
              </div>
            </div>
          ) : null}
          <div className="run-list">
            {recentRuns.length ? recentRuns.map((run) => (
              <div className="run-list-item" key={run.id}>
                <div>
                  <strong>{run.id}</strong>
                  <span>{formatDateTime(run.started_at)}</span>
                </div>
                <StatusBadge tone={statusTone(run.status)}>{run.status}</StatusBadge>
                <span>source {run.succeeded_sources}/{run.total_sources}</span>
              </div>
            )) : <div className="ui-table-empty">暂无执行记录</div>}
          </div>
        </Card>
      </div>
    </section>
  );
}

function executionSummary(runs: SchedulerRun[]) {
  const completedRuns = runs.filter((run) => run.status !== 'running');
  return {
    total: completedRuns.length,
    succeeded: completedRuns.filter((run) => run.status === 'succeeded').length,
    failed: completedRuns.filter((run) => run.status === 'failed' || run.status === 'partial' || run.status === 'skipped').length
  };
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString('zh-CN', {
    hour12: false,
    timeZone: 'Asia/Shanghai'
  });
}

function numberValue(value: string): number {
  const next = Number(value);
  return Number.isFinite(next) ? next : 0;
}

function statusTone(status: string): 'success' | 'danger' | 'neutral' {
  if (status === 'succeeded') {
    return 'success';
  }
  if (status === 'failed' || status === 'partial' || status === 'skipped') {
    return 'danger';
  }
  return 'neutral';
}

function updateFixedTime(values: string[], index: number, value: string): string[] {
  const next = normalizeFixedTimes(values);
  next[index] = value;
  return next;
}

function normalizeFixedTimes(values: string[]): string[] {
  const next = values.length ? [...values] : [...defaultConfig.fixed_times];
  while (next.length < 5) {
    next.push(defaultConfig.fixed_times[next.length] ?? '09:00');
  }
  return next.slice(0, 5);
}

function validateConfig(config: SchedulerConfig): string | undefined {
  if (config.mode === 'interval' && config.interval_minutes < 1) {
    return '间隔分钟必须大于 0';
  }
  if (config.mode === 'fixed_times' && !normalizeFixedTimes(config.fixed_times).every((value) => /^\d{2}:\d{2}$/.test(value))) {
    return '固定时间必须使用 HH:mm 格式';
  }
  if (config.concurrency < 1 || config.batch_size < 1 || config.timeout_seconds < 1) {
    return '并发数、批次大小和超时秒数必须大于 0';
  }
  return undefined;
}

function normalizeConfig(config: SchedulerConfig): SchedulerConfig {
  return {
    ...defaultConfig,
    ...config,
    fixed_times: normalizeFixedTimes(config.fixed_times ?? []),
    source_filter: config.source_filter ?? {}
  };
}
