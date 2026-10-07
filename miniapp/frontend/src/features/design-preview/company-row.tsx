import { useState } from 'react';
import Taro from '@tarojs/taro';
import { Button, Image, Text, View } from '@tarojs/components';
import arrowIcon from '../../assets/icons/report-arrow-right-light-gold.svg';
import type { Company } from './fixtures';
import { companyReportUrl } from '../company-report/route';

// These IDs are design fixture IDs, never stock IDs accepted by TrackingPort.
export function PreviewCompanyRow({ company }: { company: Company }) {
  const [followed, setFollowed] = useState(false);
  async function open() {
    try {
      await Taro.navigateTo({
        url: companyReportUrl(company.symbol, {
          stockName: company.name,
          companyName: company.fullName
        })
      });
    } catch {
      void Taro.showToast({
        title: '打开失败，请重试',
        icon: 'none'
      });
    }
  }
  return (
    <View className='preview-company'>
      <View className='preview-between preview-company-heading'>
        <Button className='tidewise-button preview-company-title' onClick={() => void open()}>
          {company.fullName}
        </Button>
        {followed ? (
          <Text className='preview-company-tracking'>✓ 跟踪中</Text>
        ) : (
          <Button
            className='tidewise-button preview-company-track'
            ariaLabel={'模拟跟踪' + company.name}
            onClick={() => setFollowed(true)}
          >
            <Text aria-hidden>＋</Text>
            <Text>跟踪</Text>
          </Button>
        )}
      </View>
      <View className='preview-tags'>
        <Text className='preview-tag industry'>{company.industry}</Text>
        {company.tags.map((t) => (
          <Text className='preview-tag' key={t}>
            {t}
          </Text>
        ))}
        {company.extraTagCount && <Text className='preview-muted'>+{company.extraTagCount}</Text>}
      </View>
      <View className='preview-between'>
        <Text className='preview-muted'>
          {company.name} · {company.symbol}
        </Text>
        <Button
          className='tidewise-button preview-company-report'
          ariaLabel={'查看' + company.name + '洞察报告'}
          onClick={() => void open()}
        >
          <Text>洞察报告</Text>
          <Image
            src={arrowIcon}
            className='preview-company-report-arrow'
            mode='scaleToFill'
            aria-hidden
          />
        </Button>
      </View>
    </View>
  );
}
