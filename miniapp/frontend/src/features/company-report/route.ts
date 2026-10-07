const reportPath = '/pages/company/report/index';

// Report identity is a complete security symbol, never a display name or fixture ID.
export function parseCompanyReportSymbol(value: unknown): string | undefined {
  return typeof value === 'string' && /^\d{6}\.(SH|SZ|BJ)$/.test(value) ? value : undefined;
}

export function companyReportUrl(symbol?: string): string {
  const parsed = parseCompanyReportSymbol(symbol);
  return parsed ? `${reportPath}?symbol=${encodeURIComponent(parsed)}` : reportPath;
}
