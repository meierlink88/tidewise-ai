import { View } from '@tarojs/components';
import type { ReactNode } from 'react';
import { OverlayRoot } from './overlay-root';

export function ReportOverlayHost({ children }: Readonly<{ children?: ReactNode }>) {
  return (
    <OverlayRoot>
      <View className='report-overlay-host' catchMove onClick={(event) => event.stopPropagation()}>
        {children}
      </View>
    </OverlayRoot>
  );
}
