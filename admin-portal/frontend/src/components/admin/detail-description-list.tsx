import type { ReactNode } from 'react';

export function DetailSection({ children, title }: { children: ReactNode; title: string }) {
  return (
    <section className='border-b py-5 last:border-b-0'>
      <h3 className='mb-4 text-sm font-semibold'>{title}</h3>
      {children}
    </section>
  );
}

export function DetailList({ children }: { children: ReactNode }) {
  return <dl className='mt-4 grid grid-cols-1 gap-x-7 gap-y-4 sm:grid-cols-2'>{children}</dl>;
}

export function DetailItem({
  children,
  full = false,
  label
}: {
  children: ReactNode;
  full?: boolean;
  label: string;
}) {
  return (
    <div className={full ? 'sm:col-span-2' : undefined}>
      <dt className='mb-1 text-[11px] font-medium text-muted-foreground'>{label}</dt>
      <dd className='text-sm leading-6 break-words'>{children}</dd>
    </div>
  );
}

export function nullableDetailValue(value: string | null): string {
  return value === null ? '—' : value;
}
