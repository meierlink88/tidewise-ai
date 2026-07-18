import { mockMarkets } from '@/data/mock-markets';
import type { MarketAnchor } from '@/models/market';
import { request } from './request';

export function getMarketAnchors(): Promise<MarketAnchor[]> {
  return request({
    mock: () => mockMarkets
  });
}
