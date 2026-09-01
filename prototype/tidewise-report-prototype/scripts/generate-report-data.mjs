import { readFile, writeFile } from "node:fs/promises";
import path from "node:path";

const sourcePath = process.argv[2];
if (!sourcePath) {
  throw new Error("Usage: node scripts/generate-report-data.mjs <report.md>");
}

const outputPath = path.resolve("src/report-data.generated.json");
const markdown = await readFile(path.resolve(sourcePath), "utf8");

function sectionBetween(source, start, end) {
  const startIndex = source.indexOf(start);
  if (startIndex < 0) throw new Error(`Missing section: ${start}`);
  const endIndex = end ? source.indexOf(end, startIndex + start.length) : source.length;
  return source.slice(startIndex, endIndex < 0 ? source.length : endIndex);
}

function clean(value = "") {
  return value
    .replace(/`/g, "")
    .replace(/\*\*/g, "")
    .replace(/<br\s*\/?\s*>/gi, "；")
    .trim();
}

function cells(line) {
  return line.trim().replace(/^\|/, "").replace(/\|$/, "").split("|").map(clean);
}

function tableAfter(section, heading) {
  const headingIndex = section.indexOf(heading);
  if (headingIndex < 0) return [];
  const lines = section.slice(headingIndex + heading.length).split("\n");
  const tableStart = lines.findIndex((line) => line.trim().startsWith("|"));
  if (tableStart < 0) return [];
  const tableLines = [];
  for (const line of lines.slice(tableStart)) {
    if (!line.trim().startsWith("|")) break;
    tableLines.push(line);
  }
  if (tableLines.length < 3) return [];
  const headers = cells(tableLines[0]);
  return tableLines.slice(2).map((line) => {
    const values = cells(line);
    return Object.fromEntries(headers.map((header, index) => [header, values[index] ?? ""]));
  });
}

function eventIds(value = "") {
  return [...new Set(value.match(/EVT[a-zA-Z0-9]+/g) ?? [])];
}

function bullet(section, label) {
  return clean(section.match(new RegExp(`^- \\*\\*${label}\\*\\*：(.+)$`, "m"))?.[1] ?? "");
}

function normalizeResult(value) {
  if (value.includes("分化")) return "分化";
  if (value.includes("/") || value.includes("局部稳定")) return "分化";
  if (value.includes("降温")) return "降温";
  if (value.includes("升温")) return "升温";
  return "待验证";
}

function formatPublishedAt(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const parts = new Intl.DateTimeFormat("zh-CN", {
    timeZone: "Asia/Shanghai",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).formatToParts(date);
  const values = Object.fromEntries(parts.map((part) => [part.type, part.value]));
  return `${values.month}-${values.day} ${values.hour}:${values.minute}`;
}

function parseFrontmatter(source) {
  const body = source.match(/^---\n([\s\S]*?)\n---/)?.[1] ?? "";
  const fields = {};
  for (const line of body.split("\n")) {
    const match = line.match(/^([a-z_]+):\s*(.+)$/);
    if (match) fields[match[1]] = clean(match[2]);
  }
  return fields;
}

function parseLayerTransmission(section) {
  const macroRows = tableAfter(section, "#### 传到宏观经济");
  const industryRows = tableAfter(section, "#### 传到产业链");
  const candidateRows = tableAfter(section, "### 2.4 向下传导");
  const summary = clean(section.match(/### [12]\.4 向下传导\s*\n+([^\n|#]+)/)?.[1] ?? "");
  const paths = [
    ...macroRows.map((row) => ({
      id: row["Path ID"],
      targetLayer: "macro",
      targetLabel: row["目标锚点"],
      logic: row["传导逻辑"],
      result: row["目标结果"],
      nature: row["关系性质"],
      confidence: row["置信度"],
      status: row["状态"],
    })),
    ...industryRows.map((row) => ({
      id: row["Path ID"],
      targetLayer: "industry",
      targetLabel: row["目标产业链 / 节点"],
      logic: row["传导逻辑"],
      result: row["目标结果"],
      nature: row["关系性质"],
      confidence: row["置信度"],
      status: row["状态"],
    })),
    ...candidateRows.map((row, index) => ({
      id: `MAC-X-CANDIDATE-${String(index + 1).padStart(2, "0")}`,
      targetLayer: "industry",
      targetLabel: row["候选机制"],
      logic: row["当前推导逻辑"],
      result: row["状态"],
      nature: "待补证",
      confidence: "",
      status: row["为什么没有发布下层结论"],
    })),
  ];
  return {
    downstreamSummary: summary,
    downstreamChainIds: [...new Set(industryRows.flatMap((row) => row["目标产业链 / 节点"].match(/CHN-\d+/g) ?? []))],
    transmissionPaths: paths,
  };
}

function parseLayer(section, config) {
  const conclusion = clean(section.match(/\*\*\[CONCLUSION [^\]]+\]\s*([\s\S]*?)\*\*/)?.[1] ?? "");
  const anchors = tableAfter(section, "### 1.2 受影响锚点").length
    ? tableAfter(section, "### 1.2 受影响锚点")
    : tableAfter(section, "### 2.2 受影响锚点");
  const steps = tableAfter(section, "### 1.3 推导逻辑").length
    ? tableAfter(section, "### 1.3 推导逻辑")
    : tableAfter(section, "### 2.3 推导逻辑");
  const guardrailLines = Object.fromEntries(
    [...section.matchAll(/^- \*\*(反证|Evidence Gap|边界|反转条件)\*\*：(.+)$/gm)].map((match) => [match[1], clean(match[2])]),
  );
  return {
    key: config.key,
    label: config.label,
    shortLabel: config.shortLabel,
    eyebrow: config.eyebrow,
    conclusion,
    state: config.state,
    horizon: config.horizon,
    confidence: config.confidence,
    evidenceIds: [...new Set(anchors.flatMap((row) => eventIds(row["直接证据（Event ID）"])))],
    anchors: anchors.map((row, index) => ({
      id: `${config.key.toUpperCase()}-A${String(index + 1).padStart(2, "0")}`,
      label: row["锚点"],
      state: normalizeResult(row["结果"]),
      resultLabel: row["结果"],
      current: row["当前状态"],
      why: row["传导逻辑（为什么到这里）"],
      nature: row["结论性质"],
      window: row["时间窗口"],
      confidence: row["置信度"],
      evidenceIds: eventIds(row["直接证据（Event ID）"]),
    })),
    steps: steps.map((row) => ({
      id: row.Step,
      input: row["输入"],
      mechanism: row["传导机制"],
      output: row["输出"],
      type: row["类型"],
      confidence: row["置信度"],
      evidenceIds: eventIds(row.Evidence),
    })),
    counterevidence: guardrailLines["反证"] ?? "",
    evidenceGap: guardrailLines["Evidence Gap"] ?? "",
    boundary: guardrailLines["边界"] ?? "",
    reversal: guardrailLines["反转条件"] ?? "",
    ...parseLayerTransmission(section),
  };
}

function parseGraph(section, nodes) {
  const mermaid = section.match(/```mermaid\n([\s\S]*?)```/)?.[1] ?? "";
  const edgeMatches = [...mermaid.matchAll(/^\s*([A-Za-z0-9_]+)\[(.+?)\]\s*-->\|(.+?)\|\s*([A-Za-z0-9_]+)\[(.+?)\]\s*$/gm)];
  const labelToId = new Map(nodes.map((node) => [node.title, node.id]));
  const graphEdges = edgeMatches
    .map((match) => ({
      from: labelToId.get(clean(match[2])),
      to: labelToId.get(clean(match[5])),
      relation: clean(match[3]),
    }))
    .filter((edge) => edge.from && edge.to);
  const transmissionGroups = [
    nodes.filter((node) => node.tone === "observed").map((node) => node.id),
    nodes.filter((node) => node.tone === "inferred").map((node) => node.id),
    nodes.filter((node) => node.tone === "gap").map((node) => node.id),
  ].filter((group) => group.length > 0);
  if (transmissionGroups.length > 1) {
    return {
      graphGroups: transmissionGroups,
      graphLinks: transmissionGroups.slice(1).map((group) => {
        const firstNode = nodes.find((node) => node.id === group[0]);
        return firstNode?.tone === "inferred" ? "动态传导" : "同链待验证";
      }),
      graphEdges,
    };
  }
  const edges = graphEdges;
  if (!edges.length) return { graphGroups: nodes.map((node) => [node.id]), graphLinks: nodes.slice(1).map(() => "同链"), graphEdges };

  const ids = new Set(edges.flatMap((edge) => [edge.from, edge.to]));
  nodes.forEach((node) => ids.add(node.id));
  const indegree = new Map([...ids].map((id) => [id, 0]));
  edges.forEach((edge) => indegree.set(edge.to, (indegree.get(edge.to) ?? 0) + 1));
  let frontier = [...ids].filter((id) => indegree.get(id) === 0);
  const groups = [];
  const visited = new Set();
  while (frontier.length) {
    groups.push(frontier);
    const next = [];
    frontier.forEach((id) => {
      visited.add(id);
      edges.filter((edge) => edge.from === id).forEach((edge) => {
        indegree.set(edge.to, (indegree.get(edge.to) ?? 1) - 1);
        if (indegree.get(edge.to) === 0) next.push(edge.to);
      });
    });
    frontier = [...new Set(next)].filter((id) => !visited.has(id));
  }
  const remaining = [...ids].filter((id) => !visited.has(id));
  if (remaining.length) groups.push(remaining);
  const links = groups.slice(0, -1).map((group, index) => {
    const next = new Set(groups[index + 1]);
    return edges.find((edge) => group.includes(edge.from) && next.has(edge.to))?.relation ?? "传导";
  });
  return { graphGroups: groups, graphLinks: links, graphEdges };
}

function parseIndustryChains(source) {
  const matches = [...source.matchAll(/^### 3\.(\d+)\s+(CHN-\d+)\s+(.+)$/gm)];
  return matches.map((match, index) => {
    const section = source.slice(match.index, matches[index + 1]?.index ?? source.indexOf("## 附录 A.", match.index));
    const rows = tableAfter(section, "#### 受影响节点");
    const nodes = rows.map((row, nodeIndex) => {
      const nature = row["结论性质"];
      return {
        id: `${match[2]}-N${String(nodeIndex + 1).padStart(2, "0")}`,
        title: row["受影响节点"],
        status: nature === "—" ? "待验证" : nature,
        tone: nature === "直接证据" ? "observed" : nature === "推理假设" ? "inferred" : "gap",
        result: normalizeResult(row["结果"]),
        resultLabel: row["结果"],
        lag: row["时间窗口"],
        confidence: row["置信度"],
        impact: row["本次影响"],
        why: row["传导逻辑（为什么到这里）"],
        evidence: eventIds(row["直接证据（Event ID）"]),
      };
    });
    const graph = parseGraph(section, nodes);
    const conclusion = clean(section.match(/\*\*一句话结论（[^）]+）\*\*：(.+)/)?.[1] ?? "");
    const directEvidence = eventIds(bullet(section, "直接 Evidence"));
    return {
      id: match[2],
      title: clean(match[3]),
      kind: "产业链",
      conclusion,
      state: normalizeResult(bullet(section, "链结果")),
      status: bullet(section, "链状态"),
      horizon: bullet(section, "时间窗口 / 置信度").split("/")[0]?.trim() ?? "",
      confidence: bullet(section, "时间窗口 / 置信度").split("/").slice(1).join("/").trim(),
      path: bullet(section, "路径"),
      hypotheses: bullet(section, "已接受的动态传导假设"),
      evidenceIds: directEvidence.length ? directEvidence : [...new Set(nodes.flatMap((node) => node.evidence))],
      stopRule: bullet(section, "停止条件"),
      gap: bullet(section, "反证与 Gap"),
      ...graph,
      nodes,
    };
  });
}

function parseEvidence(source) {
  return tableAfter(sectionBetween(source, "## 附录 A. Evidence 清单", "## 附录 B."), "## 附录 A. Evidence 清单")
    .map((row) => ({
      id: row["Evidence ID"],
      publishedAt: formatPublishedAt(row["发布时间"]),
      publishedAtRaw: row["发布时间"],
      summary: row["摘要"],
      keywords: row.Keywords === "—" ? [] : row.Keywords.split(/[、,，]/).map(clean).filter(Boolean),
    }));
}

const frontmatter = parseFrontmatter(markdown);
const geoSection = sectionBetween(markdown, "## 1. 地缘政治面", "## 2. 宏观经济面");
const macroSection = sectionBetween(markdown, "## 2. 宏观经济面", "## 3. 产业链面");
const geo = parseLayer(geoSection, {
  key: "geo",
  label: "地缘政治",
  shortLabel: "政治",
  eyebrow: "安全对抗与通道可用性",
  state: "分化",
  horizon: "即时–中期",
  confidence: "中–高",
});
const macro = parseLayer(macroSection, {
  key: "macro",
  label: "宏观经济",
  shortLabel: "经济",
  eyebrow: "增长预期与政策利率",
  state: "分化",
  horizon: "中期",
  confidence: "中",
});
const industryChains = parseIndustryChains(markdown);
const evidenceItems = parseEvidence(markdown);

const report = {
  id: frontmatter.report_id,
  title: frontmatter.title,
  publishedAt: frontmatter.generated_at.replace("T", " ").replace("+08:00", "").slice(0, 16),
  generatedAt: frontmatter.generated_at,
  eventCount: Number(frontmatter.event_count),
  chainCount: Number(frontmatter.industry_chain_count),
  hypothesisCount: Number(frontmatter.transmission_hypothesis_count),
  geo,
  macro,
  industryChains,
  evidenceItems,
};

await writeFile(outputPath, `${JSON.stringify(report, null, 2)}\n`, "utf8");
console.log(`Generated ${outputPath}: ${industryChains.length} chains, ${evidenceItems.length} evidence items.`);
