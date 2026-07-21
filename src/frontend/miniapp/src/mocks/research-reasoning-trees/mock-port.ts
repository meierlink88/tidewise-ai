import listFixture from '../../../../../testdata/reasoning-tree-v1/01-reasoning-tree-list-result.json';
import contradictionFixture from '../../../../../testdata/reasoning-tree-v1/02-reasoning-tree-with-contradiction-result.json';
import unquantifiedFixture from '../../../../../testdata/reasoning-tree-v1/03-reasoning-tree-without-contradiction-unquantified-result.json';
import { ResearchReasoningTreeError } from '../../features/research-reasoning-trees/contract';
import {
  parseResearchReasoningTreeDetail,
  parseResearchReasoningTreeIndex
} from '../../features/research-reasoning-trees/wire-contract';
import type {
  ResearchReasoningTreeDetail,
  ResearchReasoningTreeIndex,
  ResearchReasoningTreePort
} from '../../features/research-reasoning-trees/contract';
import type { HomeResearchThemeItem } from '../../features/research-themes/contract';
import { mockResearchThemeFeed } from '../research-themes/mock-port';

export function createMockResearchReasoningTreePort(): ResearchReasoningTreePort {
  const index = parseResearchReasoningTreeIndex(listFixture.result);
  const expectedThemeId = index.theme.id;
  const details = new Map<string, ResearchReasoningTreeDetail>([
    [
      contradictionFixture.result.reasoning_tree.anchor_id,
      parseResearchReasoningTreeDetail(
        contradictionFixture.result,
        expectedThemeId,
        contradictionFixture.result.reasoning_tree.anchor_id
      )
    ],
    [
      unquantifiedFixture.result.reasoning_tree.anchor_id,
      parseResearchReasoningTreeDetail(
        unquantifiedFixture.result,
        expectedThemeId,
        unquantifiedFixture.result.reasoning_tree.anchor_id
      )
    ]
  ]);
  const themes = new Map(mockResearchThemeFeed.items.map((theme) => [theme.id, theme]));
  const detailTemplates = index.reasoningTrees.map((tree) => details.get(tree.anchorId)!);

  return {
    async list(themeId) {
      const theme = themes.get(themeId);
      if (!theme) throw new ResearchReasoningTreeError('themeUnavailable');
      return themeId === expectedThemeId ? index : createThemeIndex(theme);
    },
    async get(themeId, anchorId) {
      const theme = themes.get(themeId);
      if (!theme) throw new ResearchReasoningTreeError('themeUnavailable');
      const detail =
        themeId === expectedThemeId
          ? details.get(anchorId)
          : createThemeDetail(theme, anchorId, detailTemplates);
      if (!detail) throw new ResearchReasoningTreeError('treeUnavailable');
      return detail;
    }
  };
}

function createThemeIndex(theme: HomeResearchThemeItem): ResearchReasoningTreeIndex {
  return {
    theme: {
      id: theme.id,
      name: theme.name,
      oneLineConclusion: theme.oneLineConclusion,
      impactLevel: theme.impactLevel,
      transmissionPath: theme.transmissionPath,
      tradingDirection: theme.tradingDirection,
      transmissionStage: theme.transmissionStage,
      nextCheckpoint: theme.nextCheckpoint,
      marketConfirmationSummary: theme.marketConfirmationSummary,
      publishedAt: theme.publishedAt,
      affectedChainNodes: theme.affectedChainNodes,
      relatedIndices: theme.relatedIndices,
      supportingEventCount: theme.supportingEventCount,
      contradictingEventCount: theme.contradictingEventCount
    },
    reasoningTrees: theme.affectedChainNodes.map((node) => ({
      anchorId: node.id,
      centerChainNode: { id: node.id, name: node.name }
    }))
  };
}

function createThemeDetail(
  theme: HomeResearchThemeItem,
  anchorId: string,
  detailTemplates: ResearchReasoningTreeDetail[]
): ResearchReasoningTreeDetail | undefined {
  const centerIndex = theme.affectedChainNodes.findIndex((node) => node.id === anchorId);
  if (centerIndex < 0) return undefined;
  const center = theme.affectedChainNodes[centerIndex];
  const template = detailTemplates[centerIndex % detailTemplates.length].reasoningTree;

  return {
    themeId: theme.id,
    reasoningTree: {
      ...template,
      anchorId,
      centerChainNode: { id: center.id, name: center.name },
      oneLineConclusion: theme.oneLineConclusion,
      factSummary: theme.marketConfirmationSummary,
      netDirectionSummary: center.impactSummary,
      tradingDirection: theme.tradingDirection,
      nextCheckpoint: theme.nextCheckpoint,
      events: template.events.map((event, eventIndex) => ({
        ...event,
        title: `${theme.name}研究证据 ${eventIndex + 1}`,
        summary: theme.marketConfirmationSummary,
        evidenceSummary: center.impactSummary
      })),
      pathNodes: theme.affectedChainNodes.map((node, nodeIndex) => ({
        chainNodeId: node.id,
        name: node.name,
        changeDirection:
          node.relationRole === 'constraint'
            ? 'decrease'
            : node.relationRole === 'exposure'
              ? 'uncertain'
              : 'increase',
        changeSummary: node.impactSummary,
        impactSummary: node.impactSummary,
        incomingTransmissionMechanism: nodeIndex === 0 ? null : theme.transmissionPath
      }))
    }
  };
}
