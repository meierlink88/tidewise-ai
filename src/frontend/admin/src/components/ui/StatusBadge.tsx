interface StatusBadgeProps {
  children: string;
  tone?: 'success' | 'danger' | 'neutral';
}

export default function StatusBadge({ children, tone = 'neutral' }: StatusBadgeProps) {
  return <span className={`ui-status ui-status-${tone}`}>{children}</span>;
}
