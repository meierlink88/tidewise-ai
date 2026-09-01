import { Button, Image, Text, View } from '@tarojs/components';
import fileTextIcon from '../../assets/icons/file-text.svg';
import type {
  ReportConfidence,
  ReportNature,
  ReportResult
} from './contract';

export function ReportImpactSignals({
  result,
  confidence,
  timeWindow,
  nature
}: {
  result: ReportResult;
  confidence: ReportConfidence;
  timeWindow: string;
  nature?: ReportNature;
}) {
  return (
    <View className='report-impact-signals'>
      <Text className={`report-result-chip report-result-chip--${result.code}`}>{result.label}</Text>
      {nature ? <Text className='report-nature-chip'>{nature.label}</Text> : null}
      <Text className='report-signal-meta'>置信 {confidence.label}</Text>
      <Text className='report-signal-meta'>{timeWindow}</Text>
    </View>
  );
}

export function ReportEvidenceButton({
  label,
  onClick
}: {
  label: string;
  onClick: () => void;
}) {
  return (
    <Button
      className='tidewise-button report-evidence-button'
      hoverClass='report-evidence-button--pressed'
      ariaLabel={label}
      onClick={(event) => {
        event.stopPropagation();
        onClick();
      }}
    >
      <Image className='report-evidence-icon' src={fileTextIcon} mode='aspectFit' />
    </Button>
  );
}

export function ReportStatePanel({
  title,
  description,
  actionLabel,
  onAction,
  busy = false
}: {
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
  busy?: boolean;
}) {
  return (
    <View className='report-state-panel' ariaLabel={title}>
      {busy ? <View className='report-loading-mark' /> : null}
      <Text className='report-state-panel__title'>{title}</Text>
      <Text className='report-state-panel__description'>{description}</Text>
      {actionLabel && onAction ? (
        <Button
          className='tidewise-button report-state-panel__action'
          hoverClass='report-state-panel__action--pressed'
          onClick={onAction}
        >
          {actionLabel}
        </Button>
      ) : null}
    </View>
  );
}
