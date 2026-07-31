import { Card, CardContent } from '../ui/Card';

interface MetricCardProps {
  label: string;
  value: number;
}

export default function MetricCard({ label, value }: MetricCardProps) {
  return (
    <Card>
      <CardContent className='grid gap-2 p-5'>
        <span className='text-sm text-muted-foreground'>{label}</span>
        <strong className='text-3xl font-semibold tabular-nums'>{value}</strong>
      </CardContent>
    </Card>
  );
}
