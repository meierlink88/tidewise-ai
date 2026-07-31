import { cloneElement, isValidElement, type ReactElement, type ReactNode, useId } from 'react';

interface FieldProps {
  children: ReactNode;
  controlId?: string;
  hint?: string;
  label: string;
}

export function Field({ children, controlId, hint, label }: FieldProps) {
  const generatedId = useId();
  const resolvedControlId = controlId ?? `field-${generatedId}`;
  const hintId = hint ? `${resolvedControlId}-description` : undefined;
  const control =
    !controlId && isValidElement(children)
      ? cloneElement(children as ReactElement<{ id?: string; 'aria-describedby'?: string }>, {
          id: resolvedControlId,
          ...(hintId ? { 'aria-describedby': hintId } : {})
        })
      : children;

  return (
    <div className='grid gap-2 text-sm'>
      <label className='font-medium' htmlFor={resolvedControlId}>
        {label}
      </label>
      {control}
      {hint ? (
        <span className='text-xs font-normal text-muted-foreground' id={hintId}>
          {hint}
        </span>
      ) : null}
    </div>
  );
}
