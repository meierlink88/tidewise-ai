interface PaginationProps {
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
}

export default function Pagination({ page, pageSize, total, onPageChange }: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  return (
    <div className="ui-pagination">
      <span>共 {total} 条</span>
      <button disabled={page <= 1} onClick={() => onPageChange(page - 1)} type="button">上一页</button>
      <span>{page} / {totalPages}</span>
      <button disabled={page >= totalPages} onClick={() => onPageChange(page + 1)} type="button">下一页</button>
    </div>
  );
}
