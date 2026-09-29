import { useEffect, useRef, useState } from 'react';
import { Button, Input, ScrollView, Text, View } from '@tarojs/components';
import { hideOverlayNavigation, restoreOverlayNavigation } from '../../platform/overlay-navigation';
import { OverlayRoot } from '../../platform/overlay-root';
import { loadFilterOptions } from './api';
import { emptyFilters, type CompanyFilters as Selection, type FilterOptions } from './contract';
import './company-filters.scss';

const dimensions = [
  { key: 'industry_ids', label: '行业' },
  { key: 'concept_ids', label: '概念' },
  { key: 'industry_chain_ids', label: '产业链' }
] as const;
type Dimension = (typeof dimensions)[number]['key'];
export function CompanyFilters({
  value,
  onApply,
  load = loadFilterOptions
}: {
  value: Selection;
  onApply: (value: Selection) => void;
  load?: () => Promise<FilterOptions>;
}) {
  const [active, setActive] = useState<Dimension | null>(null);
  const [draft, setDraft] = useState<Selection>(emptyFilters);
  const [options, setOptions] = useState<FilterOptions | null>(null);
  const [query, setQuery] = useState('');
  const [visible, setVisible] = useState(40);
  const [status, setStatus] = useState<'idle' | 'loading' | 'ready' | 'error'>('idle');
  const [message, setMessage] = useState('');
  const sequence = useRef(0);
  useEffect(
    () => () => {
      ++sequence.current;
    },
    []
  );
  useEffect(() => {
    if (!active) return;
    let closed = false;
    void hideOverlayNavigation().catch(() => {
      if (!closed) {
        setActive(null);
        setMessage('暂时无法打开筛选，请重试');
      }
    });
    return () => {
      closed = true;
      void restoreOverlayNavigation();
    };
  }, [active]);
  async function fetchOptions() {
    const seq = ++sequence.current;
    setStatus('loading');
    setMessage('');
    try {
      const data = await load();
      if (seq === sequence.current) {
        setOptions(data);
        setStatus('ready');
      }
    } catch {
      if (seq === sequence.current) {
        setStatus('error');
        setMessage('筛选选项暂不可用，请重试');
      }
    }
  }
  function open(key: Dimension) {
    setDraft({
      industry_ids: [...value.industry_ids],
      concept_ids: [...value.concept_ids],
      industry_chain_ids: [...value.industry_chain_ids]
    });
    setActive(key);
    setQuery('');
    setVisible(40);
    setMessage('');
    if (!options) void fetchOptions();
  }
  function toggle(id: string) {
    if (!active) return;
    const current = draft[active];
    if (!current.includes(id) && current.length >= 20) {
      setMessage('每类最多选择20项');
      return;
    }
    setMessage('');
    setDraft({
      ...draft,
      [active]: current.includes(id) ? current.filter((x) => x !== id) : [...current, id]
    });
  }
  const items =
    !options || !active
      ? []
      : active === 'industry_ids'
        ? options.industries.flatMap((x) => [
            { ...x, parent: '' },
            ...x.children.map((y) => ({ ...y, parent: x.name }))
          ])
        : (active === 'concept_ids' ? options.concepts : options.industry_chains).map((x) => ({
            ...x,
            parent: ''
          }));
  const matches = items.filter((x) =>
    (x.name + ' ' + x.parent).toLowerCase().includes(query.trim().toLowerCase())
  );
  const count = Object.values(value).reduce((n, ids) => n + ids.length, 0);
  return (
    <>
      <View className='company-filter-bar'>
        {dimensions.map((d) => (
          <Button
            key={d.key}
            className={
              'tidewise-button company-filter-trigger' + (value[d.key].length ? ' is-selected' : '')
            }
            ariaLabel={d.label + '筛选，已选' + value[d.key].length + '项'}
            onClick={() => open(d.key)}
          >
            <Text>
              {d.label}
              {value[d.key].length ? ` · ${value[d.key].length}` : ''}
            </Text>
            <Text>⌄</Text>
          </Button>
        ))}
        {count > 0 && (
          <Button
            className='tidewise-button company-filter-clear'
            onClick={() => onApply(emptyFilters())}
          >
            清空
          </Button>
        )}
      </View>
      {!active && message && <Text className='company-filter-message'>{message}</Text>}
      {active && (
        <OverlayRoot>
          <View className='company-filter-overlay'>
            <View className='company-filter-backdrop' onClick={() => setActive(null)} />
            <View className='company-filter-panel'>
              <View className='company-filter-heading'>
                <Text>{dimensions.find((d) => d.key === active)?.label}筛选</Text>
                <Button
                  className='tidewise-button company-filter-cancel'
                  onClick={() => setActive(null)}
                >
                  取消
                </Button>
              </View>
              <Text className='company-filter-help'>同类满足任一项，不同类别同时满足</Text>
              <View className='company-filter-search'>
                <Input
                  value={query}
                  maxlength={64}
                  placeholder='搜索筛选项'
                  onInput={(e) => {
                    setQuery(e.detail.value);
                    setVisible(40);
                  }}
                />
                {query && (
                  <Button
                    className='tidewise-button'
                    ariaLabel='清空筛选项搜索'
                    onClick={() => {
                      setQuery('');
                      setVisible(40);
                    }}
                  >
                    ×
                  </Button>
                )}
              </View>
              {message && <Text className='company-filter-message'>{message}</Text>}
              <ScrollView
                scrollY
                className='company-filter-options'
                onScrollToLower={() => setVisible((n) => n + 40)}
              >
                {status === 'loading' && (
                  <Text className='company-filter-help'>正在加载筛选项…</Text>
                )}
                {status === 'error' && (
                  <Button
                    className='tidewise-button company-filter-retry'
                    onClick={() => void fetchOptions()}
                  >
                    重试加载
                  </Button>
                )}
                {status === 'ready' && matches.length === 0 && (
                  <Text className='company-filter-help'>没有找到筛选项</Text>
                )}
                {status === 'ready' &&
                  matches.slice(0, visible).map((item) => (
                    <Button
                      key={item.id}
                      className={
                        'tidewise-button company-filter-option' +
                        (draft[active].includes(item.id) ? ' is-selected' : '')
                      }
                      ariaLabel={`${item.parent ? item.parent + '，' : ''}${item.name}${draft[active].includes(item.id) ? '，已选中' : ''}`}
                      onClick={() => toggle(item.id)}
                    >
                      <View className='company-filter-option-label'>
                        <Text>{item.name}</Text>
                        {item.parent && (
                          <Text className='company-filter-parent'>{item.parent}</Text>
                        )}
                      </View>
                      <Text className='company-filter-check'>
                        {draft[active].includes(item.id) ? '✓' : '＋'}
                      </Text>
                    </Button>
                  ))}
                {status === 'ready' && matches.length > visible && (
                  <Button
                    className='tidewise-button company-filter-retry'
                    onClick={() => setVisible((n) => n + 40)}
                  >
                    显示更多（{visible}/{matches.length}）
                  </Button>
                )}
              </ScrollView>
              <View className='company-filter-footer'>
                <Button
                  className='tidewise-button company-filter-reset'
                  onClick={() => {
                    setDraft({ ...draft, [active]: [] });
                    setMessage('');
                  }}
                >
                  重置本类
                </Button>
                <Button
                  disabled={status !== 'ready'}
                  className='tidewise-button company-filter-apply'
                  onClick={() => {
                    onApply(draft);
                    setActive(null);
                  }}
                >
                  应用筛选（{draft[active].length}）
                </Button>
              </View>
            </View>
          </View>
        </OverlayRoot>
      )}
    </>
  );
}
