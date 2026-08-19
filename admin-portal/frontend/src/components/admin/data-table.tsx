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
  onRowActivate?: (item: T, trigger: HTMLTableRowElement) => void;
  rowAccessibleName?: (item: T) => string;
  scrollAreaLabel: string;
  selectedRowKey?: string;
  tableClassName?: string;
}

export function DataTable<T>({
  className,
  columns,
  emptyText,
  getRowKey,
  items,
  onRowActivate,
  rowAccessibleName,
  scrollAreaLabel,
  selectedRowKey,
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
          {items.map((item) => {
            const rowKey = getRowKey(item);
            const interactive = onRowActivate !== undefined;
            return (
              <TableRow
                aria-label={rowAccessibleName?.(item)}
                aria-selected={interactive ? rowKey === selectedRowKey : undefined}
                className={cn(
                  interactive &&
                    'cursor-pointer focus-visible:bg-muted/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring data-[selected=true]:bg-muted'
                )}
                data-selected={interactive ? String(rowKey === selectedRowKey) : undefined}
                key={rowKey}
                onClick={
                  interactive
                    ? (event) => {
                        event.currentTarget.focus();
                        onRowActivate(item, event.currentTarget);
                      }
                    : undefined
                }
                onKeyDown={
                  interactive
                    ? (event) => {
                        if (event.key === 'Enter' || event.key === ' ') {
                          event.preventDefault();
                          onRowActivate(item, event.currentTarget);
                        }
                      }
                    : undefined
                }
                tabIndex={interactive ? 0 : undefined}
              >
                {columns.map((column) => (
                  <TableCell className={column.cellClassName} key={column.key}>
                    {column.render(item)}
                  </TableCell>
                ))}
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
