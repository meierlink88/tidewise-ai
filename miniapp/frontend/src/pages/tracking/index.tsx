import { useRef, useState } from 'react';
import { Button, Input, RootPortal, ScrollView, Text, View } from '@tarojs/components';
import { useDidHide } from '@tarojs/taro';
import { useTracking } from '../../features/tracking/use-tracking';
import type { Company } from '../../features/tracking/contract';
import { openLogin, readSession } from '../../platform/identity';
import { confirmUnfollow } from '../../platform/tracking';
import './index.scss';

export default function TrackingPage() {
  const tracking = useTracking();
  const [detail, setDetail] = useState<Company | null>(null);
  const confirmPending = useRef(false);
  const visibility = useRef(0);
  const [confirmation, setConfirmation] = useState('');
  const [modalError, setModalError] = useState('');
  useDidHide(() => {
    visibility.current++;
    setDetail(null);
    setConfirmation('');
  });
  const searching = tracking.query.trim().length > 0;
  const items = searching ? tracking.results : tracking.items;
  const status = searching ? tracking.searchStatus : tracking.status;
  async function remove(item: Company) {
    if (confirmPending.current || tracking.pending) return;
    confirmPending.current = true;
    const visible = visibility.current;
    const owner = readSession()?.session_token;
    setConfirmation(item.id);
    setModalError('');
    try {
      if (await confirmUnfollow(item.title, item.symbol)) {
        if (visible === visibility.current && owner && readSession()?.session_token === owner)
          await tracking.change(item, false);
      }
    } catch {
      setModalError('未能打开确认窗口，请重试');
    } finally {
      confirmPending.current = false;
      setConfirmation('');
    }
  }
  function add(item: Company) {
    if (tracking.guest) {
      void openLogin();
      return;
    }
    void tracking.change(item, true);
  }
  return (
    <View className='tracking-page'>
      <View className='tracking-header'>
        <View className='tracking-caption'>
          <Text>{tracking.guest ? '关注公司，持续跟踪' : `已跟踪 ${tracking.total} 家公司`}</Text>
        </View>
        <View className='tracking-search-line'>
          <View className='tracking-search'>
            <Text className='tracking-search-icon'>⌕</Text>
            <Input
              value={tracking.query}
              maxlength={64}
              placeholder='股票代码 / 股票名称拼音首字母'
              confirmType='search'
              onInput={(event) => tracking.setQuery(event.detail.value)}
              onConfirm={() => void tracking.retry()}
            />
            {tracking.query && (
              <Button
                className='tracking-clear'
                ariaLabel='清空搜索'
                onClick={() => tracking.setQuery('')}
              >
                ×
              </Button>
            )}
          </View>
          {tracking.query && (
            <Button className='tracking-cancel' onClick={() => tracking.setQuery('')}>
              取消
            </Button>
          )}
        </View>
      </View>
      <View className='tracking-content'>
        <View className='tracking-list-label'>
          <Text>{searching ? '搜索结果' : '我跟踪的公司'}</Text>
          <Text>{searching ? '按公司了解，按代码查找' : '公司 / 行业 / 主题'}</Text>
        </View>
        {(tracking.error || modalError) && (
          <View className='tracking-error'>
            <Text>{modalError || tracking.error}</Text>
            <Button
              onClick={() => {
                setModalError('');
                void tracking.retry();
              }}
            >
              重试
            </Button>
          </View>
        )}
        {items.map((item) => (
          <View key={item.id} className='tracking-company'>
            <Button className='tracking-title' onClick={() => setDetail(item)}>
              {item.title}
            </Button>
            {(item.industry_label || item.concepts.length > 0) && (
              <View className='tracking-tags'>
                {item.industry_label && (
                  <Text className='tracking-industry'>{item.industry_label}</Text>
                )}
                {item.concepts.slice(0, 2).map((tag, index) => (
                  <Text key={tag + index} className='tracking-concept'>
                    {tag}
                  </Text>
                ))}
                {item.concepts.length > 2 && (
                  <Button
                    className='tracking-more-tags'
                    ariaLabel='查看全部主题概念'
                    onClick={() => setDetail(item)}
                  >
                    +{item.concepts.length - 2}
                  </Button>
                )}
              </View>
            )}
            <View className='tracking-meta'>
              <Text>
                {item.stock_name} · {item.symbol}
              </Text>
              {searching ? (
                <Button
                  className={item.is_followed ? 'tracking-followed' : 'tracking-add'}
                  disabled={item.is_followed || !!tracking.pending}
                  onClick={() => add(item)}
                >
                  {tracking.pending === item.id ? '添加中' : item.is_followed ? '已跟踪' : '+ 跟踪'}
                </Button>
              ) : (
                <Button
                  className='tracking-remove'
                  disabled={!!tracking.pending || !!confirmation}
                  onClick={() => void remove(item)}
                >
                  {tracking.pending === item.id ? '处理中' : '取消跟踪'}
                </Button>
              )}
            </View>
          </View>
        ))}
        {status === 'loading' && <View className='tracking-message'>加载中…</View>}
        {items.length === 0 && status !== 'loading' && status !== 'error' && (
          <View className='tracking-empty'>
            <Text className='tracking-empty-title'>
              {searching
                ? '没有找到匹配的公司'
                : tracking.guest
                  ? '登录后保存你的跟踪'
                  : '从一家公司开始'}
            </Text>
            <Text>
              {searching
                ? '试试完整股票代码或名称首字母，例如 000001、PAYH'
                : '通过上方搜索，找到你想持续了解的公司'}
            </Text>
            {!searching && tracking.guest && (
              <Button className='tracking-login' onClick={() => void openLogin()}>
                登录 / 注册
              </Button>
            )}
          </View>
        )}
        {tracking.hasMore && status !== 'loading' && (
          <Button
            className='tracking-load'
            disabled={!!tracking.pending}
            onClick={() => void tracking.loadMore()}
          >
            加载更多
          </Button>
        )}
        {items.length > 0 && !tracking.hasMore && status === 'ready' && (
          <View className='tracking-message'>
            {searching ? '已展示全部匹配结果' : '以上是你跟踪的公司'}
          </View>
        )}
      </View>
      <RootPortal>
        <View className='tracking-overlay-scope'>
          {detail && (
            <View className='tracking-overlay' onClick={() => setDetail(null)} catchMove>
              <View className='tracking-sheet' onClick={(event) => event.stopPropagation()}>
                <View className='tracking-sheet-heading'>
                  <Text>公司资料</Text>
                  <Button ariaLabel='关闭公司资料' onClick={() => setDetail(null)}>
                    ×
                  </Button>
                </View>
                <ScrollView scrollY className='tracking-sheet-scroll'>
                  <View className='tracking-sheet-body'>
                    <Text className='tracking-detail-title'>{detail.title}</Text>
                    <Text className='tracking-detail-reference'>
                      {detail.stock_name} · {detail.symbol}
                    </Text>
                    <View className='tracking-detail-label'>所属行业</View>
                    <Text>{detail.industry_path || '暂无行业资料'}</Text>
                    <View className='tracking-detail-label'>
                      主题概念 · {detail.concepts.length}
                    </View>
                    <View className='tracking-all-tags'>
                      {detail.concepts.map((tag, index) => (
                        <Text className='tracking-concept' key={tag + index}>
                          {tag}
                        </Text>
                      ))}
                    </View>
                    {detail.concepts.length === 0 && <Text>暂无主题概念</Text>}
                    <View className='tracking-detail-note'>
                      主题按资料来源顺序展示，不代表权重或排名。
                    </View>
                  </View>
                </ScrollView>
              </View>
            </View>
          )}
        </View>
      </RootPortal>
    </View>
  );
}
