import Taro, { useRouter } from '@tarojs/taro';
import { Button, ScrollView, Text, View } from '@tarojs/components';
import { useEffect, useMemo, useState } from 'react';
import { createResearchReasoningTreePort } from '../../../features/research-reasoning-trees/port';
import {
  ResearchReasoningTreeSession,
  type ResearchReasoningTreeDetailState,
  type ResearchReasoningTreeSessionState
} from '../../../features/research-reasoning-trees/session';
import { getHomeChromeMetrics } from '../../../platform/system-ui';
import { ReasoningThemeHero, ReasoningTreeView } from './view';
import './index.scss';

const port = createResearchReasoningTreePort();

export default function ResearchReasoningTreesPage() {
  const router = useRouter();
  const themeId = typeof router.params.theme_id === 'string' ? router.params.theme_id : '';
  const session = useMemo(() => new ResearchReasoningTreeSession(themeId, port), [themeId]);
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);
  const [state, setState] = useState<ResearchReasoningTreeSessionState>(() => session.getState());

  useEffect(() => {
    const unsubscribe = session.subscribe(setState);
    void session.start();
    return () => {
      unsubscribe();
      session.dispose();
    };
  }, [session]);

  return (
    <View className='reasoning-page'>
      <View className='reasoning-page__chrome'>
        <View style={{ height: `${chrome.statusBarHeight}px` }} />
        <View className='reasoning-page__nav' style={{ height: `${chrome.navigationBarHeight}px` }}>
          <Button
            className='tidewise-button reasoning-page__back'
            hoverClass='none'
            onClick={goBack}
          >
            <Text className='reasoning-page__back-icon'>‹</Text>
            <Text>返回</Text>
          </Button>
          <Text className='reasoning-page__nav-title'>影响路径</Text>
          <View className='reasoning-page__nav-spacer' />
        </View>
      </View>

      {state.routeStatus === 'invalid' ? (
        <PageState title='页面参数有误' description='无法识别需要查看的研究主题' />
      ) : (
        <IndexContent state={state} session={session} />
      )}
    </View>
  );
}

export function IndexContent({
  state,
  session
}: {
  state: ResearchReasoningTreeSessionState;
  session: ResearchReasoningTreeSession;
}) {
  if (state.index.status === 'idle' || state.index.status === 'loading') {
    return <PageState title='正在整理影响路径' description='正在加载研究主题与产业链锚点' />;
  }
  if (state.index.status === 'themeUnavailable') {
    return <PageState title='该研究主题暂不可用' description='请返回首页选择其他研究主题' />;
  }
  if (state.index.status === 'treesNotPublished') {
    return <PageState title='影响路径暂未生成' description='该主题尚未发布完整推理树' />;
  }
  if (state.index.status === 'error') {
    return (
      <PageState
        title='影响路径暂时不可用'
        description='服务连接失败，请稍后重试'
        actionLabel='重新加载'
        onAction={() => session.retryIndex()}
      />
    );
  }

  const { theme, reasoningTrees } = state.index.value;
  const selectedAnchorId = state.selectedAnchorId ?? reasoningTrees[0].anchorId;
  const detailState = state.detailsByAnchorId[selectedAnchorId] ?? { status: 'idle' as const };

  return (
    <View className='reasoning-page__ready'>
      <ReasoningThemeHero theme={theme} />

      <View id='reasoning-tabs-wrap' className='reasoning-tabs-wrap'>
        <View className='reasoning-tabs__label'>
          <Text>研究锚点</Text>
          <Text>{reasoningTrees.length} 条独立推理树</Text>
        </View>
        <ScrollView className='reasoning-tabs' scrollX showScrollbar={false}>
          <View className='reasoning-tabs__items'>
            {reasoningTrees.map((tree) => (
              <Button
                key={tree.anchorId}
                className={`tidewise-button reasoning-tab ${
                  selectedAnchorId === tree.anchorId ? 'reasoning-tab--active' : ''
                }`}
                hoverClass='none'
                onClick={() => {
                  if (selectedAnchorId === tree.anchorId) return;
                  session.selectAnchor(tree.anchorId);
                  scrollToReasoningTreeStart();
                }}
              >
                {tree.centerChainNode.name}
              </Button>
            ))}
          </View>
        </ScrollView>
      </View>

      <View id='reasoning-tree-content' className='reasoning-page__content'>
        <ReasoningTreeContent
          state={detailState}
          onRetry={() => session.retryAnchor(selectedAnchorId)}
        />
      </View>
    </View>
  );
}

function ReasoningTreeContent({
  state,
  onRetry
}: {
  state: ResearchReasoningTreeDetailState;
  onRetry: () => void;
}) {
  if (state.status === 'idle' || state.status === 'loading') {
    return (
      <View className='reasoning-tree-state'>
        <View className='reasoning-tree-state__pulse' />
        <Text>正在加载当前推理树</Text>
      </View>
    );
  }
  if (state.status === 'error') {
    return (
      <View className='reasoning-tree-state'>
        <Text>当前影响路径暂时不可用</Text>
        <Button
          className='tidewise-button reasoning-state-button'
          hoverClass='none'
          onClick={onRetry}
        >
          重试当前路径
        </Button>
      </View>
    );
  }

  return <ReasoningTreeView tree={state.value.reasoningTree} />;
}

function PageState({
  title,
  description,
  actionLabel,
  onAction
}: {
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <View className='reasoning-page-state'>
      <Text className='reasoning-page-state__title'>{title}</Text>
      <Text className='reasoning-page-state__description'>{description}</Text>
      {actionLabel && onAction ? (
        <Button
          className='tidewise-button reasoning-state-button'
          hoverClass='none'
          onClick={onAction}
        >
          {actionLabel}
        </Button>
      ) : null}
    </View>
  );
}

function goBack() {
  void Taro.navigateBack({
    delta: 1,
    fail: () => {
      void Taro.reLaunch({ url: '/pages/index/index' });
    }
  });
}

function scrollToReasoningTreeStart() {
  Taro.nextTick(() => {
    void Taro.pageScrollTo({
      selector: '#reasoning-tabs-wrap',
      duration: 180
    }).catch(() => undefined);
  });
}
