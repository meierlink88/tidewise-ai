import type { ReactNode } from 'react';

export interface DataTableColumn<T> {
  key: string;
  header: string;
  render: (item: T) => ReactNode;
}

interface DataTableProps<T> {
  columns: DataTableColumn<T>[];
  emptyText: string;
  getRowKey: (item: T) => string;
  items: T[];
}

export default function DataTable<T>({ columns, emptyText, getRowKey, items }: DataTableProps<T>) {
  if (items.length === 0) {
    return <div className="ui-table-empty">{emptyText}</div>;
  }

  return (
    <div className="ui-data-table-wrap">
      <table className="ui-data-table">
        <thead>
          <tr>
            {columns.map((column) => <th key={column.key}>{column.header}</th>)}
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={getRowKey(item)}>
              {columns.map((column) => <td key={column.key}>{column.render(item)}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
