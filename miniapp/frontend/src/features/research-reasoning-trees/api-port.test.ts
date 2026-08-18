import { describe, expect, it, vi } from 'vitest';
import listFixture from '../../mocks/research-reasoning-trees/list.json';
import detailFixture from '../../mocks/research-reasoning-trees/detail.json';
import { createResearchReasoningTreeApiPort } from './api-port';

const themeId = 'c26337f2-a79f-5089-84f4-63d57bc32230';
const treeId = 'd7e19e24-5d6c-568e-9b1c-93e6d7956d5b';
const success = (result: unknown) => ({ request_id: 'miniapp-reasoning-test', result });

describe('research reasoning tree BFF adapter', () => {
  it('maps the V1 shared list and detail fixtures through the public Port', async () => {
    const list = listFixture.result;
    const detail = detailFixture.result;
    const request = vi
      .fn()
      .mockResolvedValueOnce({ statusCode: 200, data: success(list) })
      .mockResolvedValueOnce({ statusCode: 200, data: success(detail) });
    const port = createResearchReasoningTreeApiPort({
      baseUrl: 'https://miniapp.example.test/',
      request
    });

    const index = await port.list(themeId);
    const tree = await port.get(themeId, treeId);

    expect(request).toHaveBeenNthCalledWith(2, {
      url: `https://miniapp.example.test/api/miniapp/v1/research/themes/${themeId}/reasoning-trees/${treeId}`,
      method: 'GET',
      dataType: 'json'
    });
    expect(index).toMatchObject({
      theme: { id: themeId, impactStrength: 'medium', transmissionStage: 'validation' },
      reasoningTrees: [
        { reasoningTreeId: treeId, displayName: '高速光模块产业链' },
        { reasoningTreeId: 'f9f7fd7e-06cf-5f53-b749-66c75785d3dc', title: 'DSP 芯片' }
      ]
    });
    expect(tree).toMatchObject({
      themeId,
      impactNodeIds: expect.arrayContaining(['33333333-3333-4333-8333-333333333333']),
      reasoningTree: {
        reasoningTreeId: treeId,
        supportSummary: '端口计划上调，产业链传导关系已确认。',
        counterSummary: '采购尚未发生，替代技术路线可能降低可插拔模块与 DSP 用量。',
        conclusionBoundarySummary: '当前仍处于采购与排产验证阶段。',
        invalidationConditions: ['采购计划取消', '替代技术路线成为主流'],
        checkpoints: [
          { type: 'event', summary: '采购数量' },
          { type: 'metric', summary: '单端口模块用量' },
          { type: 'metric', summary: '光模块排产' }
        ],
        eventCount: 2,
        nodes: [
          {
            displayName: '交换机',
            impactStrength: 'medium',
            incomingTransmissionTitle: null,
            incomingTransmissionMechanism: null,
            incomingConditionSummary: null,
            primarySignal: {
              signalRole: 'primary',
              displaySummary: '端口计划 +80%'
            }
          },
          {
            displayName: '高速光模块',
            incomingTransmissionTitle: '端口配置传导',
            incomingTransmissionMechanism: '新增交换机端口增加可插拔光模块配置需求。',
            incomingConditionSummary: '采购发生且可插拔路线延续',
            primarySignal: {
              signalRole: 'primary',
              displaySummary: '模块需求 ↑'
            }
          },
          {
            displayName: 'DSP 芯片',
            incomingTransmissionTitle: '排产向核心器件传导',
            incomingTransmissionMechanism: '高速光模块排产若增加，将提高 DSP 芯片备料需求。',
            incomingConditionSummary: '模块排产增加，且当前 DSP 技术方案延续',
            signals: expect.any(Array)
          }
        ]
      }
    });
  });

  it.each([
    [404, 'RESEARCH_THEME_NOT_FOUND', 'themeUnavailable'],
    [404, 'RESEARCH_REASONING_TREES_NOT_FOUND', 'treesNotPublished'],
    [404, 'RESEARCH_REASONING_TREE_NOT_FOUND', 'treeUnavailable'],
    [502, 'RESEARCH_DATA_UNAVAILABLE', 'serviceUnavailable']
  ] as const)('maps HTTP %s %s to %s', async (statusCode, code, kind) => {
    const request = vi.fn().mockResolvedValue({
      statusCode,
      data: { error: { code, message: 'hidden' } }
    });
    const port = createResearchReasoningTreeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    });

    await expect(port.list(themeId)).rejects.toMatchObject({ kind });
  });

  it('rejects a detail payload whose identity does not match the route', async () => {
    const detail = detailFixture.result;
    const request = vi.fn().mockResolvedValue({ statusCode: 200, data: success(detail) });
    const port = createResearchReasoningTreeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    });

    await expect(port.get(themeId, 'f9f7fd7e-06cf-5f53-b749-66c75785d3dc')).rejects.toMatchObject({
      kind: 'serviceUnavailable'
    });
  });

  it('rejects the retired Anchor V1 identity field', async () => {
    const detail = detailFixture.result as {
      reasoning_tree: Record<string, unknown>;
    };
    const legacyDetail = {
      ...detail,
      reasoning_tree: {
        ...detail.reasoning_tree,
        anchor_id: detail.reasoning_tree.reasoning_tree_id
      }
    };
    const request = vi.fn().mockResolvedValue({ statusCode: 200, data: success(legacyDetail) });
    const port = createResearchReasoningTreeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    });

    await expect(port.get(themeId, treeId)).rejects.toMatchObject({
      kind: 'serviceUnavailable'
    });
  });
});
