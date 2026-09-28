import { useMemo } from 'react';
import Taro from '@tarojs/taro';
import { Button, View } from '@tarojs/components';
import { NavigationBar } from '../../../platform/navigation-bar';
import { getHomeChromeMetrics } from '../../../platform/system-ui';
import InvestmentReport from './legacy/InvestmentReport';
import './legacy/legacy.scss';
import './index.scss';

export default function CompanyReportPage() {
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);
  async function back() {
    try {
      if (Taro.getCurrentPages().length > 1) await Taro.navigateBack();
      else
        await Taro.switchTab({
          url: '/pages/company/index'
        });
    } catch {
      void Taro.showToast({
        title: '返回失败，请重试',
        icon: 'none'
      });
    }
  }
  return (
    <View className='company-report-screen legacy-report'>
      <NavigationBar
        title='投资推理 · 样例'
        chrome={chrome}
        leading={
          <Button
            className='tidewise-button preview-close'
            ariaLabel='返回上一页'
            onClick={() => void back()}
          >
            ‹
          </Button>
        }
      />
      <InvestmentReport />
    </View>
  );
}
