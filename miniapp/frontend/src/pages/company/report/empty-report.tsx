import { Button, ScrollView, Text, View } from '@tarojs/components';

export function EmptyReport({
  symbol,
  stockName,
  companyName,
  onBack
}: {
  symbol?: string;
  stockName?: string;
  companyName?: string;
  onBack: () => void;
}) {
  return (
    <ScrollView scrollY className='company-report-empty'>
      <View className='company-report-empty__content'>
        {symbol && (
          <View className='company-report-empty__identity'>
            <View className='company-report-empty__company'>
              <View className='company-report-empty__stock-row'>
                <Text className='company-report-empty__stock-name'>{stockName || '股票代码'}</Text>
                <Text className='company-report-empty__symbol'>{symbol}</Text>
              </View>
              {companyName && (
                <Text className='company-report-empty__company-name'>{companyName}</Text>
              )}
            </View>
          </View>
        )}
        <View className='company-report-empty__card'>
          <View className='company-report-empty__illustration' aria-hidden>
            <View className='company-report-empty__halo' />
            <View className='company-report-empty__document'>
              <View className='company-report-empty__fold' />
              <View className='company-report-empty__line' />
              <View className='company-report-empty__line' />
              <View className='company-report-empty__line short' />
            </View>
          </View>
          <Text className='company-report-empty__title'>暂无报告</Text>
          <Text className='company-report-empty__description'>当前暂无可展示的洞察报告内容。</Text>
          <Button
            className='tidewise-button company-report-empty__back'
            ariaLabel='返回股票列表'
            onClick={onBack}
          >
            返回列表
          </Button>
          <Text className='company-report-empty__hint'>返回后可查看其他股票的洞察报告</Text>
        </View>
      </View>
    </ScrollView>
  );
}
