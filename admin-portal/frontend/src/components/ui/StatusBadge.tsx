interface StatusBadgeProps {
  children: string;
  tone?: 'success' | 'running' | 'danger' | 'neutral';
}

export function StatusBadge({ children, tone = 'neutral' }: StatusBadgeProps) {
  return (
    <Badge
      variant={
        tone === 'success'
          ? 'success'
          : tone === 'running'
            ? 'running'
            : tone === 'danger'
              ? 'destructive'
              : 'secondary'
      }
    >
      {children}
    </Badge>
  );
}
import { Badge } from './badge';
