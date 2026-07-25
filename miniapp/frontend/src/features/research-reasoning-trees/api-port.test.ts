import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { describe, expect, it, vi } from 'vitest';
import { createResearchReasoningTreeApiPort } from './api-port';

const themeId = '11111111-1111-4111-8111-111111111111';
const anchorId = '534d83be-774b-51d9-ad00-cdee4ba91799';
const success = (result: unknown) => ({ request_id: 'miniapp-reasoning-test', result });

describe('research reasoning tree BFF adapter', () => {
  it('maps the shared list and detail fixtures through the public Port', async () => {
    const list = await fixtureResult('01-reasoning-tree-list-result.json');
    const detail = await fixtureResult('02-reasoning-tree-with-contradiction-result.json');
    const request = vi
      .fn()
      .mockResolvedValueOnce({ statusCode: 200, data: success(list) })
      .mockResolvedValueOnce({ statusCode: 200, data: success(detail) });
    const port = createResearchReasoningTreeApiPort({
      baseUrl: 'https://miniapp.example.test/',
      request
    });

    const index = await port.list(themeId);
    const tree = await port.get(themeId, anchorId);

    expect(request).toHaveBeenNthCalledWith(1, {
      url: `https://miniapp.example.test/api/miniapp/v1/research/themes/${themeId}/reasoning-trees`,
      method: 'GET',
      dataType: 'json'
    });
    expect(request).toHaveBeenNthCalledWith(2, {
      url: `https://miniapp.example.test/api/miniapp/v1/research/themes/${themeId}/reasoning-trees/${anchorId}`,
      method: 'GET',
      dataType: 'json'
    });
    expect(index).toMatchObject({
      theme: { id: themeId, impactLevel: 'high', transmissionStage: 'diffusion' },
      reasoningTrees: [
        { anchorId: '5c18fc57-6bd8-5612-9a24-01a4e928b761', centerChainNode: { name: '先进封装' } },
        { anchorId, centerChainNode: { name: '光模块' } }
      ]
    });
    expect(tree).toMatchObject({
      themeId,
      reasoningTree: {
        anchorId,
        supportSummary: '云厂商资本开支上调支持高速光互联需求扩张，但兑现仍取决于订单和交付改善',
        counterSummary: '不同规格和客户的交付节奏仍有分化，反驳需求已经全面兑现的判断',
        eventCount: 2,
        events: [
          { evidenceRole: 'driver', eventTime: '2026-07-20T01:00:00Z' },
          { evidenceRole: 'contradicting' }
        ],
        pathNodes: [
          { name: 'AI芯片', incomingTransmissionMechanism: null },
          { name: '光模块', changeDirection: 'mixed' }
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
    const request = vi
      .fn()
      .mockResolvedValue({ statusCode, data: { error: { code, message: 'hidden' } } });
    const port = createResearchReasoningTreeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    });

    await expect(port.list(themeId)).rejects.toMatchObject({ kind });
  });

  it('fails closed when the BFF returns an invalid success payload', async () => {
    const request = vi
      .fn()
      .mockResolvedValue({ statusCode: 200, data: success({ reasoning_trees: [] }) });
    const port = createResearchReasoningTreeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    });

    await expect(port.list(themeId)).rejects.toMatchObject({
      kind: 'serviceUnavailable'
    });
  });

  it('rejects a detail payload whose event_time is not UTC RFC3339', async () => {
    const detail = structuredClone(
      await fixtureResult('02-reasoning-tree-with-contradiction-result.json')
    ) as {
      reasoning_tree: { events: Array<{ event_time: string | null }> };
    };
    detail.reasoning_tree.events[0].event_time = '2026/07/20 09:00:00';
    const request = vi.fn().mockResolvedValue({ statusCode: 200, data: success(detail) });
    const port = createResearchReasoningTreeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    });

    await expect(port.get(themeId, anchorId)).rejects.toMatchObject({
      kind: 'serviceUnavailable'
    });
  });

  it('accepts an explicit null counter summary without inventing a counter claim', async () => {
    const detail = await fixtureResult(
      '03-reasoning-tree-without-contradiction-unquantified-result.json'
    );
    const fixtureAnchorId = (detail as { reasoning_tree: { anchor_id: string } }).reasoning_tree
      .anchor_id;
    const request = vi.fn().mockResolvedValue({ statusCode: 200, data: success(detail) });
    const port = createResearchReasoningTreeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    });

    await expect(port.get(themeId, fixtureAnchorId)).resolves.toMatchObject({
      reasoningTree: {
        supportSummary: expect.any(String),
        counterSummary: null
      }
    });
  });

  it('fails closed when the anchor support summary is missing', async () => {
    const detail = structuredClone(
      await fixtureResult('02-reasoning-tree-with-contradiction-result.json')
    ) as { reasoning_tree: Record<string, unknown> };
    delete detail.reasoning_tree.support_summary;
    const request = vi.fn().mockResolvedValue({ statusCode: 200, data: success(detail) });
    const port = createResearchReasoningTreeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    });

    await expect(port.get(themeId, anchorId)).rejects.toMatchObject({
      kind: 'serviceUnavailable'
    });
  });

  it('rejects an ordered path whose downstream node has no transmission mechanism', async () => {
    const detail = structuredClone(
      await fixtureResult('02-reasoning-tree-with-contradiction-result.json')
    ) as {
      reasoning_tree: {
        path_nodes: Array<{ incoming_transmission_mechanism: string | null }>;
      };
    };
    detail.reasoning_tree.path_nodes[1].incoming_transmission_mechanism = null;
    const request = vi.fn().mockResolvedValue({ statusCode: 200, data: success(detail) });
    const port = createResearchReasoningTreeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    });

    await expect(port.get(themeId, anchorId)).rejects.toMatchObject({
      kind: 'serviceUnavailable'
    });
  });
});

async function fixtureResult(name: string): Promise<unknown> {
  const path = resolve(import.meta.dirname, '../../../../../testdata/reasoning-tree-v1', name);
  const fixture = JSON.parse(await readFile(path, 'utf8')) as { result: unknown };
  return fixture.result;
}
