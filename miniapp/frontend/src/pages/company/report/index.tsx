import { useMemo } from 'react';
import Taro, { useRouter } from '@tarojs/taro';
import { Button, View } from '@tarojs/components';
import { useResearchAccess } from '../../../features/identity/use-research-access';
import { ResearchAccessState } from '../../../features/identity/research-access-state';
import { NavigationBar } from '../../../platform/navigation-bar';
import { getHomeChromeMetrics } from '../../../platform/system-ui';
import {
  parseCompanyReportSymbol,
  parseCompanyReportNames
} from '../../../features/company-report/route';
import { EmptyReport } from './empty-report';
import StandardReport from './standard/StandardReport';
import standardReport from '../../../features/design-preview/report-data/standard-report.json';
import './legacy/legacy.scss';
import './index.scss';

export default function CompanyReportPage() {
  const params = useRouter().params;
  const symbol = parseCompanyReportSymbol(params.symbol);
  const names = symbol ? parseCompanyReportNames(params.display) : {};
  const access = useResearchAccess('companyReport', symbol, names);
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
  if (!access.allowed) return <ResearchAccessState error={access.error} retry={access.retry} />;
  return (
    <View className='company-report-screen legacy-report'>
      <NavigationBar
        title='洞察报告'
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
      {symbol === '603179.SH' ? (
        <StandardReport report={standardReport} />
      ) : (
        <EmptyReport {...names} symbol={symbol} onBack={() => void back()} />
      )}
    </View>
  );
}
