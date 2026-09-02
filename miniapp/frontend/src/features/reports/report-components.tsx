import { Button, Image, Text, View } from '@tarojs/components';
import fileTextIcon from '../../assets/icons/file-text.svg';
import reportActivityCoolingIcon from '../../assets/icons/report-activity-cooling.svg';
import reportActivityDivergingIcon from '../../assets/icons/report-activity-diverging.svg';
import reportActivityPendingIcon from '../../assets/icons/report-activity-pending.svg';
import reportActivityWarmingIcon from '../../assets/icons/report-activity-warming.svg';
import reportConfidenceIcon from '../../assets/icons/report-confidence.svg';
import reportWindowClockIcon from '../../assets/icons/report-window-clock.svg';
import type { ReportConfidence, ReportNature, ReportResult } from './contract';

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
  const resultIcon = {
    warming: reportActivityWarmingIcon,
    cooling: reportActivityCoolingIcon,
    diverging: reportActivityDivergingIcon,
    stable: reportActivityPendingIcon,
    mixed: reportActivityDivergingIcon,
    pending: reportActivityPendingIcon
  }[result.code];
  return (
    <View className='report-impact-signals'>
      <View className={`report-result-chip report-result-chip--${result.code}`}>
        <Image className='report-result-chip__icon' src={resultIcon} mode='aspectFit' />
        <Text>{result.label}</Text>
      </View>
      {nature ? (
        <Text className={`report-nature-chip report-nature-chip--${nature.code}`}>
          {nature.label}
        </Text>
      ) : null}
      <View className='report-signal-meta'>
        <Image className='report-signal-meta__icon' src={reportConfidenceIcon} mode='aspectFit' />
        <Text>置信 {confidence.label}</Text>
      </View>
      <View className='report-signal-meta'>
        <Image className='report-signal-meta__icon' src={reportWindowClockIcon} mode='aspectFit' />
        <Text>{timeWindow}</Text>
      </View>
    </View>
  );
}

export function ReportEvidenceButton({ label, onClick }: { label: string; onClick: () => void }) {
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
