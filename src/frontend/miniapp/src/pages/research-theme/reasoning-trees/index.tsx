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

function IndexContent({
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
    <View className='reasoning-page__content'>
      <View className='reasoning-theme'>
        <Text className='reasoning-theme__eyebrow'>{theme.name}</Text>
        <Text className='reasoning-theme__title'>{theme.oneLineConclusion}</Text>
      </View>

      <ScrollView className='reasoning-tabs' scrollX enhanced showScrollbar={false}>
        <View className='reasoning-tabs__items'>
          {reasoningTrees.map((tree) => (
            <Button
              key={tree.anchorId}
              className={`tidewise-button reasoning-tab ${
                selectedAnchorId === tree.anchorId ? 'reasoning-tab--active' : ''
              }`}
              hoverClass='none'
              onClick={() => session.selectAnchor(tree.anchorId)}
            >
              {tree.centerChainNode.name}
            </Button>
          ))}
        </View>
      </ScrollView>

      <ReasoningTreeSkeleton
        state={detailState}
        onRetry={() => session.retryAnchor(selectedAnchorId)}
      />
    </View>
  );
}

function ReasoningTreeSkeleton({
  state,
  onRetry
}: {
  state: ResearchReasoningTreeDetailState;
  onRetry: () => void;
}) {
  if (state.status === 'idle' || state.status === 'loading') {
    return <View className='reasoning-tree-state'>正在加载当前推理树</View>;
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

  const tree = state.value.reasoningTree;
  return (
    <View className='reasoning-tree-skeleton'>
      <Text className='reasoning-tree-skeleton__label'>{tree.centerChainNode.name} · 研究锚点</Text>
      <Text className='reasoning-tree-skeleton__title'>{tree.oneLineConclusion}</Text>
      <View className='reasoning-tree-skeleton__block'>
        <Text className='reasoning-tree-skeleton__block-title'>事实摘要</Text>
        <Text>{tree.factSummary}</Text>
      </View>
      <View className='reasoning-tree-skeleton__metrics'>
        <Text>{tree.eventCount} 条事件证据</Text>
        <Text>{tree.pathNodes.length} 个传导节点</Text>
      </View>
      <Text className='reasoning-tree-skeleton__handoff'>完整证据与传导视觉将在下一阶段呈现</Text>
    </View>
  );
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
