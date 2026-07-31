import StatusAlert from './status-alert';

interface QueryErrorProps {
  message: string;
  onRetry: () => void;
  retrying?: boolean;
}

export default function QueryError({ message, onRetry, retrying = false }: QueryErrorProps) {
  return (
    <StatusAlert
      actionLabel={retrying ? '重试中…' : '重试'}
      actionDisabled={retrying}
      onAction={onRetry}
      tone='destructive'
    >
      {message}
    </StatusAlert>
  );
}
