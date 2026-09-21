import { useMemo } from 'react';
import Taro from '@tarojs/taro';
import { Button, Text, View } from '@tarojs/components';
import { NavigationBar } from '../../../platform/navigation-bar';
import { getHomeChromeMetrics } from '../../../platform/system-ui';
import {
  copyPrivacyContact,
  leaveProfile,
  openProfileInformation
} from '../../../platform/identity';
import { privacySections, privacyVersion } from '../../../features/identity/privacy';
import '../index.scss';
import './index.scss';

export default function InformationPage() {
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);
  const about = Taro.getCurrentInstance().router?.params.section === 'about';
  const version = useMemo(() => {
    if (process.env.TARO_ENV !== 'weapp') return '';
    try {
      return Taro.getAccountInfoSync().miniProgram.version || '';
    } catch {
      return '';
    }
  }, []);
  return (
    <View className={`profile-page account-page${about ? ' profile-page--about' : ''}`}>
      <View className='profile-page__header account-header'>
        <NavigationBar
          title={about ? '关于观潮家' : '隐私政策'}
          chrome={chrome}
          leading={
            <Button
              className='tidewise-button profile-page__back'
              aria-label='返回上一页'
              onClick={() => void leaveProfile()}
            >
              <View className='profile-page__chevron' />
            </Button>
          }
        />
      </View>
      {about ? (
        <View className='about-page__body'>
          <View className='about-page__brand'>
            <View className='about-page__mark'>观</View>
            <Text className='about-page__name'>观潮家</Text>
            <Text className='about-page__tagline'>读懂全球政经变化</Text>
            {version && <Text className='about-page__version'>V{version}</Text>}
          </View>
          <Button
            className='about-page__privacy'
            onClick={() => void openProfileInformation('privacy')}
          >
            隐私政策
          </Button>
        </View>
      ) : (
        <View className='profile-page__body profile-information'>
          <Text className='profile-information__title'>观潮家隐私政策</Text>
          <Text className='profile-information__lead'>版本日期：{privacyVersion}</Text>
          {privacySections.map(([heading, text]) => (
            <View key={heading} className='profile-information__section'>
              <Text className='profile-information__heading'>{heading}</Text>
              <Text className='profile-information__text' selectable>
                {text}
              </Text>
            </View>
          ))}
          <Button className='profile-page__button' onClick={() => void copyPrivacyContact()}>
            复制隐私联系邮箱
          </Button>
        </View>
      )}
    </View>
  );
}
