import { RootPortal, View } from '@tarojs/components';
import type { ReactNode } from 'react';

export function ReportOverlayHost({ children }: Readonly<{ children?: ReactNode }>) {
  return (
    <RootPortal>
      <View className='report-overlay-host' catchMove onClick={(event) => event.stopPropagation()}>
        {children}
      </View>
    </RootPortal>
  );
}
