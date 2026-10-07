import { expect, it } from 'vitest';
import { companyReportUrl, parseCompanyReportNames } from './route';

it('round-trips Chinese labels and keeps separators and percent sequences inside names', () => {
  const names = { stockName: '众泰汽车', companyName: '公司%20名称 &symbol=603179.SH' };
  const url = companyReportUrl('000980.SZ', names);
  const query = new URLSearchParams(url.split('?')[1]);
  expect(query.get('symbol')).toBe('000980.SZ');
  expect(parseCompanyReportNames(query.get('display'))).toEqual(names);
  expect(parseCompanyReportNames(encodeURIComponent(JSON.stringify(names)))).toEqual(names);
});
it.each([
  undefined,
  '%bad',
  '[]',
  'null',
  '{"stockName":1}',
  JSON.stringify({ stockName: 'a'.repeat(201) }),
  JSON.stringify({ stockName: 'a\nb' })
])('ignores malformed, oversized or invalid labels: %s', (value) => {
  expect(parseCompanyReportNames(value).stockName).toBeUndefined();
});
it('does not carry labels for an invalid symbol', () => {
  expect(companyReportUrl('bad', { stockName: '众泰汽车' })).toBe('/pages/company/report/index');
});
