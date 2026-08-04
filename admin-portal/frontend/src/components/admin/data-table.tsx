import type { ReactNode } from 'react';
import { cn } from '../../lib/utils';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../ui/table';

export interface DataTableColumn<T> {
  cellClassName?: string;
  headerClassName?: string;
  key: string;
  header: string;
  render: (item: T) => ReactNode;
}

interface DataTableProps<T> {
  className?: string;
  columns: DataTableColumn<T>[];
  emptyText: string;
  getRowKey: (item: T) => string;
  items: T[];
  scrollAreaLabel: string;
  tableClassName?: string;
}

export function DataTable<T>({
  className,
  columns,
  emptyText,
  getRowKey,
  items,
  scrollAreaLabel,
  tableClassName
}: DataTableProps<T>) {
  if (items.length === 0) {
    return (
      <div
        className={cn(
          'grid h-full min-h-32 place-items-center rounded-md border border-dashed px-4 text-sm text-muted-foreground',
          className
        )}
      >
        {emptyText}
      </div>
    );
  }

  return (
    <div className={cn('min-h-0 overflow-hidden rounded-md border', className)}>
      <Table
        className={tableClassName}
        containerClassName='h-full min-h-0 overflow-auto'
        scrollAreaLabel={scrollAreaLabel}
      >
        <TableHeader className='sticky top-0 z-20 bg-muted [&_th]:bg-muted'>
          <TableRow className='hover:bg-transparent'>
            {columns.map((column) => (
              <TableHead className={column.headerClassName} key={column.key}>
                {column.header}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {items.map((item) => (
            <TableRow key={getRowKey(item)}>
              {columns.map((column) => (
                <TableCell className={column.cellClassName} key={column.key}>
                  {column.render(item)}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
