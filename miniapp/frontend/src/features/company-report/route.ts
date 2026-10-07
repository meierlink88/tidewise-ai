const reportPath = '/pages/company/report/index';

export type CompanyReportNames = { readonly stockName?: string; readonly companyName?: string };

// Report identity is a complete security symbol, never a display name or fixture ID.
export function parseCompanyReportSymbol(value: unknown): string | undefined {
  return typeof value === 'string' && /^\d{6}\.(SH|SZ|BJ)$/.test(value) ? value : undefined;
}
function displayName(value: unknown): string | undefined {
  return typeof value === 'string' &&
    value.trim().length > 0 &&
    value.length <= 200 &&
    ![...value].some((character) => character.charCodeAt(0) < 32 || character.charCodeAt(0) === 127)
    ? value.trim()
    : undefined;
}

// Native and H5 may expose encoded or already decoded query values. Parse JSON first
// so a percent sequence inside a legitimate name is never decoded twice.
export function parseCompanyReportNames(value: unknown): CompanyReportNames {
  if (typeof value !== 'string' || value.length > 4096) return {};
  let parsed: unknown;
  try {
    parsed = JSON.parse(value);
  } catch {
    try {
      parsed = JSON.parse(decodeURIComponent(value));
    } catch {
      return {};
    }
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {};
  const names = parsed as Record<string, unknown>;
  return { stockName: displayName(names.stockName), companyName: displayName(names.companyName) };
}

export function companyReportUrl(symbol?: string, names?: CompanyReportNames): string {
  const parsed = parseCompanyReportSymbol(symbol);
  if (!parsed) return reportPath;
  const display = {
    stockName: displayName(names?.stockName),
    companyName: displayName(names?.companyName)
  };
  const query =
    display.stockName || display.companyName
      ? `&display=${encodeURIComponent(JSON.stringify(display))}`
      : '';
  return `${reportPath}?symbol=${encodeURIComponent(parsed)}${query}`;
}
