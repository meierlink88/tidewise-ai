import type { Direction } from '../../contracts/daily-brief-v1';

const confidenceLabels = { high: '高可信', medium: '中可信', low: '待验证' } as const;

const directionMeta = {
  up: { label: '向上', symbol: '↑', className: 'up' },
  down: { label: '承压', symbol: '↓', className: 'down' },
  neutral: { label: '中性', symbol: '—', className: 'neutral' },
  divergent: { label: '分化', symbol: '↕', className: 'divergent' }
} as const;

const resourceCopy = {
  loading: { title: '正在汇集今日信号' },
  empty: { title: '今日暂无可展示简报', description: '稍后再来，新的市场线索正在整理中' },
  error: { title: '今日观潮加载失败', action: '重新加载' }
} as const;

export function getDirectionMeta(direction: Direction) {
  return directionMeta[direction];
}

export function getResourceStateCopy(status: keyof typeof resourceCopy) {
  return resourceCopy[status];
}

export function getConfidenceLabel(confidence: keyof typeof confidenceLabels) {
  return confidenceLabels[confidence];
}
