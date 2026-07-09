import { useEffect, useState } from 'react';
import { Button, Card, Form, Input, InputNumber, Select, Space, Switch, Typography, message } from 'antd';
import type { SchedulerConfig } from '../api/scheduler';
import { loadSchedulerConfig as defaultLoadSchedulerConfig, saveSchedulerConfig as defaultSaveSchedulerConfig } from '../api/scheduler';

interface SchedulerSettingsProps {
  token: string;
  loadConfig?: (token: string) => Promise<SchedulerConfig>;
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
  saveConfig = defaultSaveSchedulerConfig
}: SchedulerSettingsProps) {
  const [form] = Form.useForm<SchedulerConfig>();
  const [mode, setMode] = useState<SchedulerConfig['mode']>('interval');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [recentRun, setRecentRun] = useState<SchedulerConfig['recent_run']>();
  const [systemConfig, setSystemConfig] = useState<SchedulerConfig>(defaultConfig);
  const hasToken = token.trim().length > 0;

  useEffect(() => {
    let active = true;
    setLoading(true);
    loadConfig(token)
      .then((config) => {
        if (!active) {
          return;
        }
        const next = normalizeConfig(config);
        setSystemConfig(next);
        form.setFieldsValue(next);
        setMode(next.mode);
        setRecentRun(config.recent_run);
      })
      .catch(() => {
        if (active) {
          form.setFieldsValue(defaultConfig);
          setSystemConfig(defaultConfig);
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
  }, [form, loadConfig, token]);

  const handleSave = async () => {
    if (!hasToken) {
      message.warning('请先输入 Admin Token');
      return;
    }
    const values = await form.validateFields();
    const payload = normalizeConfig({
      ...systemConfig,
      ...values,
      source_filter: systemConfig.source_filter,
      timezone: defaultConfig.timezone
    });
    setSaving(true);
    try {
      const saved = await saveConfig(token, payload);
      if (saved && typeof saved === 'object') {
        setSystemConfig(normalizeConfig(saved as SchedulerConfig));
      }
      message.success('已保存');
    } catch (error) {
      message.error(`保存失败：${error instanceof Error ? error.message : '未知错误'}`);
    } finally {
      setSaving(false);
    }
  };

  return (
    <section className="scheduler-settings">
      <div className="page-title">
        <Typography.Title level={2}>调度器设置</Typography.Title>
      </div>
      <Card loading={loading}>
        <Form form={form} layout="vertical" initialValues={defaultConfig} onValuesChange={(_, values) => setMode(values.mode ?? 'interval')}>
          <div className="form-grid">
            <Form.Item label="启用调度" name="enabled" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item label="调度模式" name="mode" rules={[{ required: true }]}>
              <Select
                options={[
                  { label: '间隔运行', value: 'interval' },
                  { label: '固定时间', value: 'fixed_times' }
                ]}
              />
            </Form.Item>
            {mode === 'interval' ? (
              <Form.Item label="间隔分钟" name="interval_minutes" rules={[{ required: true, type: 'number', min: 1 }]}>
                <InputNumber min={1} className="full-width" />
              </Form.Item>
            ) : (
              <Form.Item label="固定时间" required>
                <Space direction="vertical" className="full-width">
                  {[0, 1, 2, 3, 4].map((index) => (
                    <Form.Item
                      key={index}
                      name={['fixed_times', index]}
                      rules={[{ required: true, pattern: /^\d{2}:\d{2}$/ }]}
                      noStyle
                    >
                      <Input aria-label={`固定时间 ${index + 1}`} />
                    </Form.Item>
                  ))}
                </Space>
              </Form.Item>
            )}
            <Form.Item label="并发数" name="concurrency" rules={[{ required: true, type: 'number', min: 1 }]}>
              <InputNumber min={1} className="full-width" />
            </Form.Item>
            <Form.Item label="批次大小" name="batch_size" rules={[{ required: true, type: 'number', min: 1 }]}>
              <InputNumber min={1} className="full-width" />
            </Form.Item>
            <Form.Item label="超时秒数" name="timeout_seconds" rules={[{ required: true, type: 'number', min: 1 }]}>
              <InputNumber min={1} className="full-width" />
            </Form.Item>
          </div>
          <Button type="primary" loading={saving} disabled={!hasToken} onClick={handleSave}>
            保存设置
          </Button>
        </Form>
      </Card>
      {recentRun ? (
        <Card className="run-summary">
          <Typography.Text>最近运行：{recentRun.status}</Typography.Text>
          <Typography.Text>成功 {recentRun.succeeded_sources} / 失败 {recentRun.failed_sources}</Typography.Text>
        </Card>
      ) : null}
    </section>
  );
}

function normalizeConfig(config: SchedulerConfig): SchedulerConfig {
  return {
    ...defaultConfig,
    ...config,
    fixed_times: config.fixed_times?.length ? config.fixed_times : defaultConfig.fixed_times,
    source_filter: config.source_filter ?? {}
  };
}
