import { buildReportDetailURL, type ReportDetailRoute } from './navigation';

export function homeReportShare() {
  return { title: '观潮家 · 今日观潮', path: '/pages/index/index', query: '' };
}

export function reportDetailShare(route: ReportDetailRoute | null, title?: string) {
  const path = route ? buildReportDetailURL(route) : '/pages/report/detail/index';
  return {
    title: title ? `${title} · 观潮家` : '观潮家 · 推理详情',
    path,
    query: path.split('?')[1] ?? ''
  };
}
