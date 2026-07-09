import type { SelectHTMLAttributes } from 'react';

export default function Select({ children, className = '', ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select className={`ui-input ui-select ${className}`.trim()} {...props}>
      {children}
    </select>
  );
}
