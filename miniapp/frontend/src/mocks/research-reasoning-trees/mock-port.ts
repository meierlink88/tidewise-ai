import listFixture from './list.json';
import detailFixture from './detail.json';
import { ResearchReasoningTreeError } from '../../features/research-reasoning-trees/contract';
import type { ResearchReasoningTreePort } from '../../features/research-reasoning-trees/contract';
import {
  parseResearchReasoningTreeDetail,
  parseResearchReasoningTreeIndex
} from '../../features/research-reasoning-trees/wire-contract';

export function createMockResearchReasoningTreePort(): ResearchReasoningTreePort {
  const index = parseResearchReasoningTreeIndex(listFixture.result);
  const themeId = index.theme.id;
  const detail = parseResearchReasoningTreeDetail(
    detailFixture.result,
    themeId,
    detailFixture.result.reasoning_tree.reasoning_tree_id
  );
  const secondSummary = index.reasoningTrees[1];
  const secondDetail = {
    ...detail,
    reasoningTree: {
      ...detail.reasoningTree,
      reasoningTreeId: secondSummary.reasoningTreeId,
      industryChainEntityId: secondSummary.industryChainEntityId,
      industryChainName: secondSummary.industryChainName,
      title: secondSummary.title,
      displayOrder: secondSummary.displayOrder,
      publishedAt: secondSummary.publishedAt
    }
  };
  const details = new Map([
    [detail.reasoningTree.reasoningTreeId, detail],
    [secondDetail.reasoningTree.reasoningTreeId, secondDetail]
  ]);

  return {
    async list(requestedThemeId) {
      if (requestedThemeId !== themeId) throw new ResearchReasoningTreeError('themeUnavailable');
      return index;
    },
    async get(requestedThemeId, reasoningTreeId) {
      if (requestedThemeId !== themeId) throw new ResearchReasoningTreeError('themeUnavailable');
      const value = details.get(reasoningTreeId);
      if (!value) {
        throw new ResearchReasoningTreeError('treeUnavailable');
      }
      return value;
    }
  };
}
