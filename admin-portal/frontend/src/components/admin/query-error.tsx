import { Button } from '../ui/Button';

interface QueryErrorProps {
  message: string;
  onRetry: () => void;
  retrying?: boolean;
}

export default function QueryError({ message, onRetry, retrying = false }: QueryErrorProps) {
  return (
    <div
      className='flex flex-wrap items-center justify-between gap-3 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200'
      role='alert'
    >
      <span>{message}</span>
      <Button disabled={retrying} onClick={onRetry} size='sm' variant='outline'>
        {retrying ? '重试中…' : '重试'}
      </Button>
    </div>
  );
}
