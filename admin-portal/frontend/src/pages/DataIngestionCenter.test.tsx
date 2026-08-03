import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as agentManagementAPI from '../api/agentManagement';
import * as dataIngestionAPI from '../api/dataIngestion';
import DataIngestionCenter from './DataIngestionCenter';

describe('DataIngestionCenter', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders the retained evidence and event tabs and loads raw documents by default', async () => {
    vi.spyOn(dataIngestionAPI, 'loadRawDocuments').mockResolvedValue({
      items: [
        {
          id: 'raw-1',
          source_ref: 'agentrun://source/bbc-business',
          source_name: '新华社',
          title: '央行公布金融数据',
          content_text: '摘要',
          collected_at: '2026-07-09T10:00:00Z',
          ingest_status: 'collected'
        }
      ],
      total: 1,
      page: 1,
      page_size: 50
    });
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 50
    });

    render(<DataIngestionCenter token='secret-token' />);

    const rawTab = await screen.findByRole('tab', { name: '原始数据' });
    expect(screen.getByRole('heading', { name: '数据采集中心' })).toBeInTheDocument();
    expect(rawTab).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '全球事件' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '采集器配置' })).toBeInTheDocument();
    expect(screen.queryByRole('tab', { name: '搜索通道' })).not.toBeInTheDocument();
    expect(screen.getAllByRole('tab')).toHaveLength(3);
    expect(screen.queryByRole('tab', { name: '调度器' })).not.toBeInTheDocument();
    expect(await screen.findByText('央行公布金融数据')).toBeInTheDocument();
    expect(screen.getByText('agentrun://source/bbc-business')).toBeInTheDocument();
    expect(dataIngestionAPI.loadRawDocuments).toHaveBeenCalledWith('secret-token', {
      page: 1,
      title: ''
    });

    await act(async () => {
      rawTab.focus();
      fireEvent.keyDown(rawTab, { key: 'ArrowRight' });
      await Promise.resolve();
    });
    await waitFor(() =>
      expect(screen.getByRole('tab', { name: '全球事件' })).toHaveAttribute('aria-selected', 'true')
    );
  });

  it('presents a safe data error and retries the current tab', async () => {
    const user = userEvent.setup();
    const loadRawDocuments = vi
      .spyOn(dataIngestionAPI, 'loadRawDocuments')
      .mockRejectedValueOnce(new Error('internal server error'))
      .mockResolvedValueOnce({
        items: [],
        total: 0,
        page: 1,
        page_size: 50
      });
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 50
    });

    render(<DataIngestionCenter token='secret-token' />);

    expect(await screen.findByRole('alert')).toHaveTextContent('数据加载失败，请稍后重试。');
    expect(screen.queryByText('internal server error')).not.toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '重试' }));

    await waitFor(() => expect(loadRawDocuments).toHaveBeenCalledTimes(2));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('applies raw title search and event filters', async () => {
    const user = userEvent.setup();
    const eventTimeFrom = '2026-07-09T00:00';
    const eventTimeTo = '2026-07-10T00:00';
    const firstSeenFrom = '2026-07-08T00:00';
    const firstSeenTo = '2026-07-11T00:00';
    vi.spyOn(dataIngestionAPI, 'loadRawDocuments').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 50
    });
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 50
    });

    render(<DataIngestionCenter token='secret-token' />);

    await screen.findByRole('tab', { name: '原始数据' });
    await user.type(screen.getByLabelText('原始数据标题搜索'), '央行');
    await user.click(screen.getByRole('button', { name: '搜索原始数据' }));

    expect(dataIngestionAPI.loadRawDocuments).toHaveBeenLastCalledWith('secret-token', {
      page: 1,
      title: '央行'
    });

    await user.click(screen.getByRole('tab', { name: '全球事件' }));
    await user.type(screen.getByLabelText('事件标题搜索'), '美联储');
    await user.click(screen.getByLabelText('事件状态'));
    await user.click(screen.getByRole('option', { name: '已确认' }));
    await user.click(screen.getByLabelText('事实状态'));
    await user.click(screen.getByRole('option', { name: '已核验' }));
    await user.type(screen.getByLabelText('事件时间开始'), eventTimeFrom);
    await user.type(screen.getByLabelText('事件时间结束'), eventTimeTo);
    await user.type(screen.getByLabelText('首次发现开始'), firstSeenFrom);
    await user.type(screen.getByLabelText('首次发现结束'), firstSeenTo);
    await user.click(screen.getByRole('button', { name: '搜索事件' }));

    expect(dataIngestionAPI.loadEvents).toHaveBeenLastCalledWith(
      'secret-token',
      expect.objectContaining({
        page: 1,
        title: '美联储',
        event_status: 'confirmed',
        fact_status: 'verified',
        event_time_from: new Date(eventTimeFrom).toISOString(),
        event_time_to: new Date(eventTimeTo).toISOString(),
        first_seen_from: new Date(firstSeenFrom).toISOString(),
        first_seen_to: new Date(firstSeenTo).toISOString()
      })
    );
  });

  it('loads collector readiness and keeps schedule configuration separate from enable state', async () => {
    const user = userEvent.setup();
    mockRawDocuments();
    const schedule = collectorSchedule();
    vi.spyOn(agentManagementAPI, 'loadAgentSchedule').mockResolvedValue(schedule);
    vi.spyOn(agentManagementAPI, 'loadModelProviders').mockResolvedValue([
      {
        provider_key: 'deepseek',
        base_url: 'https://api.deepseek.com',
        model: 'deepseek-chat',
        configured: true,
        key_configured: true,
        masked_key: '••••a9f2'
      }
    ]);
    vi.spyOn(agentManagementAPI, 'loadConnectors').mockResolvedValue(configuredConnectors());
    const saveSchedule = vi
      .spyOn(agentManagementAPI, 'saveAgentSchedule')
      .mockResolvedValue({ ...schedule, input: { prompt: '新的采集 Prompt' } });
    const setEnabled = vi
      .spyOn(agentManagementAPI, 'setAgentScheduleEnabled')
      .mockResolvedValue({ ...schedule, enabled: false });

    render(<DataIngestionCenter token='secret-token' />);
    await user.click(await screen.findByRole('tab', { name: '采集器配置' }));

    expect(await screen.findByRole('tab', { name: '定时任务' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '执行记录' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '模型配置' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '连接器配置' })).toBeInTheDocument();
    expect(await screen.findByText('模型和 7 个连接器配置完整')).toBeInTheDocument();
    expect(screen.getAllByText('已启用').length).toBeGreaterThan(0);

    const prompt = screen.getByLabelText('Collection Prompt');
    await user.clear(prompt);
    await user.type(prompt, '  新的采集 Prompt  ');
    await user.click(screen.getByRole('button', { name: '保存配置' }));

    expect(saveSchedule).toHaveBeenCalledWith('secret-token', 'collector', {
      agent_version: 'collector.v1',
      schedule_type: 'daily',
      daily_times: ['08:30', '12:30', '18:30'],
      input: { prompt: '  新的采集 Prompt  ' }
    });
    expect(setEnabled).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: '停止定时器' }));
    expect(screen.getByRole('alertdialog', { name: '停止定时器' })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '取消' }));
    expect(setEnabled).not.toHaveBeenCalled();
    await user.click(screen.getByRole('button', { name: '停止定时器' }));
    await user.click(screen.getByRole('button', { name: '确认停止' }));
    expect(setEnabled).toHaveBeenCalledWith('secret-token', 'collector', false);
  });

  it('switches to Cron, submits it, and starts a ready stopped schedule', async () => {
    const user = userEvent.setup();
    mockRawDocuments();
    const schedule = { ...collectorSchedule(), enabled: false };
    vi.spyOn(agentManagementAPI, 'loadAgentSchedule').mockResolvedValue(schedule);
    vi.spyOn(agentManagementAPI, 'loadModelProviders').mockResolvedValue([
      {
        provider_key: 'deepseek',
        base_url: 'https://api.deepseek.com',
        model: 'deepseek-chat',
        configured: true,
        key_configured: true
      }
    ]);
    vi.spyOn(agentManagementAPI, 'loadConnectors').mockResolvedValue(configuredConnectors());
    const saveSchedule = vi.spyOn(agentManagementAPI, 'saveAgentSchedule').mockResolvedValue({
      ...schedule,
      schedule_type: 'cron',
      cron_expression: '15 * * * *'
    });
    const setEnabled = vi
      .spyOn(agentManagementAPI, 'setAgentScheduleEnabled')
      .mockResolvedValue({ ...schedule, enabled: true });

    render(<DataIngestionCenter token='secret-token' />);
    await user.click(await screen.findByRole('tab', { name: '采集器配置' }));
    await user.click(screen.getByRole('tab', { name: 'Cron' }));
    await user.clear(screen.getByLabelText('Cron 表达式'));
    await user.type(screen.getByLabelText('Cron 表达式'), '15 * * * *');
    await user.click(screen.getByRole('button', { name: '保存配置' }));
    expect(saveSchedule).toHaveBeenCalledWith(
      'secret-token',
      'collector',
      expect.objectContaining({
        schedule_type: 'cron',
        cron_expression: '15 * * * *'
      })
    );

    await user.click(screen.getByRole('button', { name: '启动定时器' }));
    expect(setEnabled).toHaveBeenCalledWith('secret-token', 'collector', true);
    expect(await screen.findByText('定时器已启动')).toBeInTheDocument();
  });

  it('loads collector execution records in fixed twenty-item pages', async () => {
    const user = userEvent.setup();
    mockRawDocuments();
    mockCollectorConfiguration();
    const loadExecutions = vi.spyOn(agentManagementAPI, 'loadAgentExecutions').mockResolvedValue({
      items: [
        {
          execution_id: 'execution-1',
          agent_key: 'collector',
          agent_version: 'collector.v1',
          trigger_source: 'schedule',
          status: 'failed',
          error_summary: '上游响应不可用',
          created_at: '2026-07-24T04:30:00Z',
          triggered_at: '2026-07-24T04:30:00Z',
          started_at: '2026-07-24T04:30:01Z',
          completed_at: '2026-07-24T04:31:20Z'
        }
      ],
      page: 1,
      page_size: 20,
      total_items: 21,
      total_pages: 2
    });

    render(<DataIngestionCenter token='secret-token' />);
    await user.click(await screen.findByRole('tab', { name: '采集器配置' }));
    await user.click(await screen.findByRole('tab', { name: '执行记录' }));

    expect(await screen.findByText('execution-1')).toBeInTheDocument();
    expect(screen.getByText('上游响应不可用')).toBeInTheDocument();
    expect(loadExecutions).toHaveBeenCalledWith('secret-token', 1);
    await user.click(screen.getByRole('button', { name: '下一页' }));
    expect(loadExecutions).toHaveBeenLastCalledWith('secret-token', 2);
  });

  it('shows an execution loading state before rendering the empty state', async () => {
    const user = userEvent.setup();
    mockRawDocuments();
    mockCollectorConfiguration();
    let resolveExecutions:
      | ((value: Awaited<ReturnType<typeof agentManagementAPI.loadAgentExecutions>>) => void)
      | undefined;
    vi.spyOn(agentManagementAPI, 'loadAgentExecutions').mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveExecutions = resolve;
        })
    );

    render(<DataIngestionCenter token='secret-token' />);
    await user.click(await screen.findByRole('tab', { name: '采集器配置' }));
    await user.click(await screen.findByRole('tab', { name: '执行记录' }));
    expect(await screen.findByText('正在加载执行记录')).toBeInTheDocument();
    await act(async () => {
      resolveExecutions?.({
        items: [],
        page: 1,
        page_size: 20,
        total_items: 0,
        total_pages: 0
      });
    });
    expect(await screen.findByText('暂无执行记录')).toBeInTheDocument();
  });

  it('retries execution loading and links incomplete readiness to the affected configuration', async () => {
    const user = userEvent.setup();
    mockRawDocuments();
    vi.spyOn(agentManagementAPI, 'loadAgentSchedule').mockResolvedValue(collectorSchedule());
    vi.spyOn(agentManagementAPI, 'loadModelProviders').mockResolvedValue([
      {
        provider_key: 'deepseek',
        base_url: 'https://api.deepseek.com',
        model: 'deepseek-chat',
        configured: true,
        key_configured: true
      }
    ]);
    vi.spyOn(agentManagementAPI, 'loadConnectors').mockResolvedValue([
      {
        connector_key: 'parallel_search',
        base_url: 'https://search.example.com',
        configured: true,
        key_configured: true
      },
      {
        connector_key: 'tavily',
        base_url: 'https://api.tavily.com',
        configured: false,
        key_configured: false
      }
    ]);
    const loadExecutions = vi
      .spyOn(agentManagementAPI, 'loadAgentExecutions')
      .mockRejectedValueOnce(new Error('AgentRun 暂时不可用'))
      .mockResolvedValue({
        items: [],
        page: 1,
        page_size: 20,
        total_items: 0,
        total_pages: 0
      });

    render(<DataIngestionCenter token='secret-token' />);
    await user.click(await screen.findByRole('tab', { name: '采集器配置' }));

    expect(await screen.findByText('1 / 2 完整')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '前往连接器配置' }));
    expect(screen.getByRole('tab', { name: '连接器配置' })).toHaveAttribute(
      'aria-selected',
      'true'
    );

    await user.click(screen.getByRole('tab', { name: '执行记录' }));
    await user.click(await screen.findByRole('button', { name: '重试' }));
    await waitFor(() => expect(loadExecutions).toHaveBeenCalledTimes(2));
  });

  it('keeps a blank model key and supports explicit connector key clearing', async () => {
    const user = userEvent.setup();
    mockRawDocuments();
    mockCollectorConfiguration();
    const updateModel = vi.spyOn(agentManagementAPI, 'updateModelProvider').mockResolvedValue({
      provider_key: 'deepseek',
      base_url: 'https://api.deepseek.com',
      model: 'deepseek-chat',
      configured: true,
      key_configured: true,
      masked_key: '••••a9f2'
    });
    const updateConnector = vi.spyOn(agentManagementAPI, 'updateConnector').mockResolvedValue({
      connector_key: 'parallel_search',
      base_url: 'https://search.example.com',
      configured: false,
      key_configured: false
    });

    render(<DataIngestionCenter token='secret-token' />);
    await user.click(await screen.findByRole('tab', { name: '采集器配置' }));
    await user.click(await screen.findByRole('tab', { name: '模型配置' }));
    await user.click(await screen.findByRole('button', { name: '编辑 deepseek' }));
    await user.click(screen.getByRole('button', { name: '保存模型配置' }));
    expect(updateModel).toHaveBeenCalledWith('secret-token', 'deepseek', {
      base_url: 'https://api.deepseek.com',
      model: 'deepseek-chat'
    });
    await user.click(await screen.findByRole('button', { name: '编辑 deepseek' }));
    await user.type(screen.getByLabelText('新 API Key'), 'new-model-key');
    await user.click(screen.getByRole('button', { name: '保存模型配置' }));
    expect(updateModel).toHaveBeenLastCalledWith('secret-token', 'deepseek', {
      base_url: 'https://api.deepseek.com',
      model: 'deepseek-chat',
      api_key: 'new-model-key'
    });

    await user.click(screen.getByRole('tab', { name: '连接器配置' }));
    await user.click(await screen.findByRole('button', { name: '编辑 parallel_search' }));
    await user.click(screen.getByRole('checkbox', { name: '清除当前 Key' }));
    await user.click(screen.getByRole('button', { name: '保存连接器配置' }));
    expect(updateConnector).toHaveBeenCalledWith('secret-token', 'parallel_search', {
      base_url: 'https://search.example.com',
      api_key: ''
    });
    await user.click(await screen.findByRole('button', { name: '编辑 parallel_search' }));
    await user.type(screen.getByLabelText('新 API Key'), 'new-connector-key');
    await user.click(screen.getByRole('button', { name: '保存连接器配置' }));
    expect(updateConnector).toHaveBeenLastCalledWith('secret-token', 'parallel_search', {
      base_url: 'https://search.example.com',
      api_key: 'new-connector-key'
    });
  });

  it('keeps Data-backed tabs usable when AgentRun configuration fails locally', async () => {
    const user = userEvent.setup();
    vi.spyOn(dataIngestionAPI, 'loadRawDocuments').mockResolvedValue({
      items: [
        {
          id: 'raw-safe',
          source_name: 'BBC',
          title: 'Data 仍可用',
          content_text: '摘要',
          collected_at: '2026-07-09T10:00:00Z',
          ingest_status: 'collected'
        }
      ],
      total: 1,
      page: 1,
      page_size: 50
    });
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 50
    });
    vi.spyOn(agentManagementAPI, 'loadAgentSchedule').mockRejectedValue(
      new Error('AgentRun 暂时不可用')
    );
    vi.spyOn(agentManagementAPI, 'loadModelProviders').mockRejectedValue(
      new Error('AgentRun 暂时不可用')
    );
    vi.spyOn(agentManagementAPI, 'loadConnectors').mockRejectedValue(
      new Error('AgentRun 暂时不可用')
    );

    render(<DataIngestionCenter token='secret-token' />);
    expect(await screen.findByText('Data 仍可用')).toBeInTheDocument();
    await user.click(screen.getByRole('tab', { name: '采集器配置' }));
    expect(await screen.findByText('AgentRun 暂时不可用')).toBeInTheDocument();
    await user.click(screen.getByRole('tab', { name: '原始数据' }));
    expect(screen.getByText('Data 仍可用')).toBeInTheDocument();
  });
});

function mockRawDocuments() {
  vi.spyOn(dataIngestionAPI, 'loadRawDocuments').mockResolvedValue({
    items: [],
    total: 0,
    page: 1,
    page_size: 50
  });
  vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({
    items: [],
    total: 0,
    page: 1,
    page_size: 50
  });
}

function mockCollectorConfiguration() {
  vi.spyOn(agentManagementAPI, 'loadAgentSchedule').mockResolvedValue(collectorSchedule());
  vi.spyOn(agentManagementAPI, 'loadModelProviders').mockResolvedValue([
    {
      provider_key: 'deepseek',
      base_url: 'https://api.deepseek.com',
      model: 'deepseek-chat',
      configured: true,
      key_configured: true,
      masked_key: '••••a9f2'
    }
  ]);
  vi.spyOn(agentManagementAPI, 'loadConnectors').mockResolvedValue(configuredConnectors());
}

function collectorSchedule() {
  return {
    schedule_id: 'schedule-1',
    agent_key: 'collector',
    agent_version: 'collector.v1',
    schedule_type: 'daily' as const,
    daily_times: ['08:30', '12:30', '18:30'],
    input: { prompt: '采集全球政经事实' },
    enabled: true,
    last_triggered_at: '2026-07-24T04:30:00Z',
    next_run_at: '2026-07-24T10:30:00Z',
    created_at: '2026-07-20T01:00:00Z',
    updated_at: '2026-07-24T04:30:00Z'
  };
}

function configuredConnectors() {
  return [
    'parallel_search',
    'tavily',
    'bocha',
    'cls_telegraph',
    'eastmoney_fastnews',
    'eastmoney_stock_news',
    'stcn_quicknews'
  ].map((connector_key) => ({
    connector_key,
    base_url:
      connector_key === 'parallel_search'
        ? 'https://search.example.com'
        : `https://${connector_key}.example.com`,
    configured: true,
    key_configured: true,
    masked_key: '••••cafe'
  }));
}
