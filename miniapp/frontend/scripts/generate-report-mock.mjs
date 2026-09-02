import { readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(scriptDirectory, '../../..');
const sourcePath = path.join(
  repositoryRoot,
  'prototype/tidewise-report-prototype/src/report-data.generated.json'
);
const outputPath = path.join(
  scriptDirectory,
  '../src/mocks/reports/report-industry-chains.generated.ts'
);

const source = JSON.parse(readFileSync(sourcePath, 'utf8'));

const resultByLabel = {
  升温: { code: 'warming', label: '升温' },
  降温: { code: 'cooling', label: '降温' },
  分化: { code: 'diverging', label: '分化' },
  待验证: { code: 'pending', label: '待验证' }
};

const natureByLabel = {
  直接证据: { code: 'direct_evidence', label: '直接证据' },
  推理假设: { code: 'reasoning_hypothesis', label: '推理假设' },
  待验证: { code: 'pending_validation', label: '待验证' }
};

const toKey = (value) => value.toLowerCase();

const confidence = (label) => {
  const scoreMatch = label.match(/（(0(?:\.\d+)?|1(?:\.0+)?)）/);
  return { label, score: scoreMatch ? Number(scoreMatch[1]) : null };
};

const chains = source.industryChains.map((chain, chainIndex) => {
  const key = toKey(chain.id);
  return {
    key,
    claimKey: `${key}-claim`,
    displayOrder: chainIndex + 1,
    scope: { type: 'industry_chain', key },
    name: chain.title,
    conclusion: chain.conclusion,
    status: chain.status,
    result: resultByLabel[chain.state],
    confidence: confidence(chain.confidence),
    timeWindow: chain.horizon,
    pathSummary: chain.path || null,
    acceptedHypothesisSummary: chain.hypotheses || null,
    nodes: chain.nodes.map((node, nodeIndex) => {
      const nodeKey = toKey(node.id);
      return {
        key: nodeKey,
        displayOrder: nodeIndex + 1,
        scope: { type: 'industry_chain_node', key: nodeKey },
        name: node.title,
        impact: node.impact,
        result: resultByLabel[node.result],
        nature: natureByLabel[node.status],
        reasoning: node.why,
        timeWindow: node.lag,
        confidence: confidence(node.confidence),
        hasEvidence: node.status === '直接证据' && node.evidence.length > 0
      };
    }),
    edges: chain.graphEdges.map((edge, edgeIndex) => ({
      key: `${key}-edge-${String(edgeIndex + 1).padStart(2, '0')}`,
      displayOrder: edgeIndex + 1,
      fromNodeKey: toKey(edge.from),
      toNodeKey: toKey(edge.to),
      relationLabel: edge.relation
    })),
    uncertainty: {
      counterevidenceAndGap: chain.gap || null,
      stopCondition: chain.stopRule || null,
      checkpoints: []
    },
    hasEvidence: chain.evidenceIds.length > 0
  };
});

const output =
  `// Generated from the approved report prototype data. Do not edit manually.\n` +
  `// Run: npm run generate:report-mock\n` +
  `import type { ReportIndustryChainDetailContent } from '../../features/reports/contract';\n\n` +
  `export const generatedIndustryChainDetails: ReportIndustryChainDetailContent[] = ${JSON.stringify(chains, null, 2)};\n`;

writeFileSync(outputPath, output);
console.log(`generated ${chains.length} industry chains: ${outputPath}`);
