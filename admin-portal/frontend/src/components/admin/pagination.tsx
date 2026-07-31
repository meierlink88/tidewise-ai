interface PaginationProps {
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
}

export function Pagination({ page, pageSize, total, onPageChange }: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  return (
    <div className='flex flex-wrap items-center justify-end gap-2 pt-4 text-sm text-muted-foreground'>
      <span className='mr-2'>共 {total} 条</span>
      <Button
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
        size='sm'
        variant='outline'
      >
        <ChevronLeft className='size-4' />
        上一页
      </Button>
      <span className='px-2 font-medium text-foreground'>
        {page} / {totalPages}
      </span>
      <Button
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
        size='sm'
        variant='outline'
      >
        下一页
        <ChevronRight className='size-4' />
      </Button>
    </div>
  );
}
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '../ui/Button';
