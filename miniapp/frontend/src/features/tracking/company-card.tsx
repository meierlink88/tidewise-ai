import { Button, Image, Text, View } from '@tarojs/components';
import { useState } from 'react';
import Taro from '@tarojs/taro';
import type { Company } from './contract';
import { companyReportUrl } from '../company-report/route';
import arrowIcon from '../../assets/icons/report-arrow-right-light-gold.svg';
import radarIcon from '../../assets/icons/tracking-radar.svg';
import { PreviewSheet } from '../design-preview/shell';
import '../design-preview/preview.scss';
import './company-card.scss';

export function CompanyCard({
  company,
  busy,
  pending,
  onFollow,
  onRemove
}: {
  company: Company;
  busy: boolean;
  pending: boolean;
  onFollow?: () => void;
  onRemove?: () => void;
}) {
  const [sheet, setSheet] = useState<'company' | null>(null);
  async function openReport() {
    try {
      await Taro.navigateTo({ url: companyReportUrl(company.symbol) });
    } catch {
      void Taro.showToast({ title: '打开失败，请重试', icon: 'none' });
    }
  }
  return (
    <View className='preview-company'>
      <View className='preview-between preview-company-heading'>
        <Button className='tidewise-button preview-company-title' onClick={() => void openReport()}>
          {company.title}
        </Button>
        {onRemove ? (
          <Button
            className={`tidewise-button preview-company-track${busy ? ' company-action--busy' : ''}`}
            disabled={busy}
            onClick={onRemove}
            ariaLabel={'取消跟踪' + company.stock_name}
          >
            <Text>{pending ? '处理中' : '取消跟踪'}</Text>
          </Button>
        ) : company.is_followed ? (
          <View className='preview-company-tracking'>
            <Image
              src={radarIcon}
              className='company-tracking-radar'
              mode='aspectFit'
              aria-hidden
            />
            <Text>跟踪中</Text>
          </View>
        ) : (
          <Button
            className={`tidewise-button preview-company-track${busy ? ' company-action--busy' : ''}`}
            disabled={busy}
            onClick={onFollow}
            ariaLabel={'跟踪' + company.stock_name}
          >
            <Text>{pending ? '添加中' : '＋ 跟踪'}</Text>
          </Button>
        )}
      </View>
      <View className='preview-tags'>
        {company.industry_label && (
          <Text className='preview-tag industry'>{company.industry_label}</Text>
        )}
        {company.concepts.slice(0, 2).map((tag, index) => (
          <Text className='preview-tag' key={tag + index}>
            {tag}
          </Text>
        ))}
        {company.concepts.length > 2 && (
          <Button
            className='tidewise-button preview-muted'
            ariaLabel='查看全部公司资料'
            onClick={() => setSheet('company')}
          >
            +{company.concepts.length - 2}
          </Button>
        )}
      </View>
      <View className='preview-between'>
        <Text className='preview-muted'>
          {company.stock_name} · {company.symbol}
        </Text>
        <Button
          className='tidewise-button preview-company-report'
          ariaLabel={'查看' + company.stock_name + '洞察报告'}
          onClick={() => void openReport()}
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
      {sheet && (
        <PreviewSheet title='公司资料' close={() => setSheet(null)}>
          <Text className='preview-heading'>{company.title}</Text>
          <Text className='preview-muted'>
            {company.stock_name} · {company.symbol}
          </Text>
          <View className='preview-card'>
            <Text>{company.industry_path || '暂无行业资料'}</Text>
            <View className='preview-tags'>
              {company.concepts.map((tag, index) => (
                <Text className='preview-tag' key={tag + index}>
                  {tag}
                </Text>
              ))}
            </View>
            <Text className='preview-note'>主题按资料来源顺序展示，不代表权重或排名。</Text>
          </View>
        </PreviewSheet>
      )}
    </View>
  );
}
