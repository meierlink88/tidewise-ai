import { useState } from 'react';
import { Input, Text } from '@tarojs/components';
import { companies } from '../../features/design-preview/fixtures';
import { PreviewPage } from '../../features/design-preview/shell';
import { PreviewCompanyRow } from '../../features/design-preview/company-row';

export default function CompanyPage() {
  const [query, setQuery] = useState('');
  const items = companies.filter((c) =>
    (c.fullName + c.name + c.symbol + c.initials + c.industry + c.tags.join(''))
      .toUpperCase()
      .includes(query.trim().toUpperCase())
  );
  return (
    <PreviewPage
      title='公司洞察'
      subtitle='从公司出发，看清价值与风险'
      filters={
        <Input
          className='preview-search'
          value={query}
          onInput={(e) => setQuery(e.detail.value)}
          placeholder='公司名称 / 代码 / 拼音首字母'
        />
      }
    >
      {items.map((c) => (
        <PreviewCompanyRow company={c} key={c.id} />
      ))}
      {!items.length && (
        <Text className='preview-note'>没有找到公司，试试公司名称、股票代码或拼音首字母。</Text>
      )}
      <Text className='preview-note'>
        设计模拟 · 示例跟踪不保存到账户。真实跟踪请进入“我的跟踪”。
      </Text>
      <Text className='preview-note'>报告沿用原始时点，行情并非实时数据</Text>
    </PreviewPage>
  );
}
