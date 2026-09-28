import { Button, Image, Input, Text, View } from '@tarojs/components';
import { useResearchAccess } from '../../features/identity/use-research-access';
import { ResearchAccessState } from '../../features/identity/research-access-state';
import trackingIcon from '../../assets/icons/bookmark-light.svg';
import { openTracking } from '../../platform/tracking';
import './index.scss';
import searchIcon from '../../assets/company-search.svg';
import { PreviewPage } from '../../features/design-preview/shell';
import { CompanyCard } from '../../features/tracking/company-card';
import { useTracking } from '../../features/tracking/use-tracking';
import { openLogin } from '../../platform/identity';

export default function CompanyPage() {
  const access = useResearchAccess('company');
  const directory = useTracking(undefined, 'directory', access.allowed);
  const loadMore = () => {
    if (directory.hasMore && directory.searchStatus === 'ready' && !directory.pending)
      void directory.loadMore();
  };
  if (!access.allowed) return <ResearchAccessState error={access.error} retry={access.retry} />;
  return (
    <PreviewPage
      simulation={false}
      onLoadMore={loadMore}
      headerSearch={
        <View className='company-header-actions'>
          <View className='preview-search'>
            <Image src={searchIcon} className='preview-search-icon' aria-hidden />
            <Input
              className='preview-search-input'
              value={directory.query}
              maxlength={64}
              confirmType='search'
              onInput={(e) => directory.setQuery(e.detail.value)}
              onConfirm={() => void directory.retry()}
              placeholder='名称 / 代码 / 拼音'
            />
            {directory.query && (
              <Button
                className='tidewise-button company-search-clear'
                ariaLabel='清空搜索'
                onClick={() => directory.setQuery('')}
              >
                ×
              </Button>
            )}
          </View>
          <Button
            className='tidewise-button company-tracking-entry'
            ariaLabel='我的跟踪'
            onClick={() => void openTracking()}
          >
            <Image src={trackingIcon} className='company-tracking-entry__icon' aria-hidden />
          </Button>
        </View>
      }
    >
      {directory.error && (
        <View className='preview-card'>
          <Text>{directory.error}</Text>
          <Button
            className='tidewise-button company-directory-action'
            onClick={() => void directory.retry()}
          >
            重试
          </Button>
          {directory.guest && (
            <Button
              className='tidewise-button company-directory-action'
              onClick={() => void openLogin()}
            >
              登录 / 注册
            </Button>
          )}
        </View>
      )}
      {directory.results.map((company) => (
        <CompanyCard
          key={company.id}
          company={company}
          busy={!!directory.pending}
          pending={directory.pending === company.id}
          onFollow={() => {
            if (directory.guest) void openLogin();
            else void directory.change(company, true);
          }}
        />
      ))}
      {directory.searchStatus === 'loading' && <Text className='preview-note'>加载中…</Text>}
      {directory.searchStatus === 'ready' && !directory.results.length && (
        <Text className='preview-note'>
          {directory.query.trim()
            ? '没有找到公司，试试公司名称、股票代码或拼音首字母。'
            : '暂无公司资料'}
        </Text>
      )}
      {!directory.hasMore && directory.results.length > 0 && directory.searchStatus === 'ready' && (
        <Text className='preview-note'>
          已展示全部{directory.query.trim() ? '匹配结果' : '公司'}
        </Text>
      )}
    </PreviewPage>
  );
}
