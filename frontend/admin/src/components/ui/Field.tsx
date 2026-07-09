import type { ReactNode } from 'react';

interface FieldProps {
  children: ReactNode;
  hint?: string;
  label: string;
}

export default function Field({ children, hint, label }: FieldProps) {
  return (
    <label className="ui-field">
      <span className="ui-field-label">{label}</span>
      {children}
      {hint ? <span className="ui-field-hint">{hint}</span> : null}
    </label>
  );
}
