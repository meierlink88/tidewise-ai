export interface Company {
  id: string;
  title: string;
  stock_name: string;
  symbol: string;
  industry_label: string;
  industry_path: string;
  concepts: string[];
  is_followed: boolean;
}
export interface Page {
  items: Company[];
  total: number;
  next_cursor: string;
  has_more: boolean;
}
export interface TrackingPort {
  search(query: string, token: string, offset: number): Promise<Page>;
  list(token: string, cursor: string): Promise<Page>;
  change(token: string, id: string, add: boolean): Promise<void>;
}
export class TrackingError extends Error {
  constructor(
    message: string,
    readonly expired = false
  ) {
    super(message);
  }
}
