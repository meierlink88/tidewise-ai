import { useState } from 'react';
import Taro from '@tarojs/taro';
import { Button, Image, Text, View } from '@tarojs/components';
import arrowIcon from '../../assets/icons/report-arrow-right-light-gold.svg';
import type { Company } from './fixtures';
import { PreviewSheet } from './shell';

// These IDs are design fixture IDs, never stock IDs accepted by TrackingPort.
export function PreviewCompanyRow({ company }: { company: Company }) {
  const [followed, setFollowed] = useState(false);
  const [detail, setDetail] = useState(false);
  async function open() {
    if (!company.report) {
      setDetail(true);
      return;
    }
    try {
      await Taro.navigateTo({
        url: '/pages/company/report/index'
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
      {detail && (
        <PreviewSheet title='洞察报告' close={() => setDetail(false)}>
          <Text className='preview-muted'>设计模拟</Text>
          <Text className='preview-heading'>{company.fullName}</Text>
          <Text>
            {company.name} · {company.symbol}
          </Text>
          <View className='preview-card'>
            <Text className='preview-heading'>暂无投研推理报告</Text>
            <Text>当前原型未保留这家公司的研究正文。</Text>
          </View>
        </PreviewSheet>
      )}
    </View>
  );
}
