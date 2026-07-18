import type { ReactNode } from 'react';

interface TabItem<T extends string> {
  id: T;
  label: string;
}

interface TabsProps<T extends string> {
  active: T;
  items: TabItem<T>[];
  onChange: (id: T) => void;
}

export default function Tabs<T extends string>({ active, items, onChange }: TabsProps<T>) {
  return (
    <div className="ui-tabs" role="tablist" aria-label="数据采集中心标签">
      {items.map((item) => (
        <button
          aria-selected={active === item.id}
          className={`ui-tab ${active === item.id ? 'active' : ''}`}
          key={item.id}
          onClick={() => onChange(item.id)}
          role="tab"
          type="button"
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}

interface TabPanelProps {
  children: ReactNode;
  label: string;
}

export function TabPanel({ children, label }: TabPanelProps) {
  return (
    <section aria-label={label} className="ui-tab-panel" role="tabpanel">
      {children}
    </section>
  );
}
