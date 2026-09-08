import { buildReportDetailURL, type ReportDetailRoute } from './navigation';
import shareCover from '../../assets/share-cover.jpg';

// 分享 API 的代码包图片路径从根目录解析，不随首页/详情页层级改变。
const imageUrl = shareCover.startsWith('/') ? shareCover : `/${shareCover}`;

export function homeReportShare() {
  return { title: '观潮家 · 今日观潮', path: '/pages/index/index', query: '', imageUrl };
}

export function reportDetailShare(route: ReportDetailRoute | null, title?: string) {
  const path = route ? buildReportDetailURL(route) : '/pages/report/detail/index';
  return {
    title: title ? `${title} · 观潮家` : '观潮家 · 推理详情',
    path,
    imageUrl,
    query: path.split('?')[1] ?? ''
  };
}
