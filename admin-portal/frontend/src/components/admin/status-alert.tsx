import type { ReactNode } from 'react';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';

interface StatusAlertProps {
  actionLabel?: string;
  actionDisabled?: boolean;
  children: ReactNode;
  onAction?: () => void;
  role?: 'alert' | 'status';
  tone: 'destructive' | 'success';
}

function StatusAlert({
  actionLabel,
  actionDisabled = false,
  children,
  onAction,
  role = 'alert',
  tone
}: StatusAlertProps) {
  return (
    <div
      className={cn(
        'flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-3 text-sm',
        tone === 'destructive'
          ? 'border-destructive-border bg-destructive-subtle text-destructive-foreground'
          : 'border-success-border bg-success-subtle text-success-foreground'
      )}
      role={role}
    >
      <span>{children}</span>
      {actionLabel && onAction ? (
        <Button disabled={actionDisabled} onClick={onAction} size='sm' variant='ghost'>
          {actionLabel}
        </Button>
      ) : null}
    </div>
  );
}

export default StatusAlert;
