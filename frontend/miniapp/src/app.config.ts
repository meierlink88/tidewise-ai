import { appPages } from './constants/app-navigation';

export default defineAppConfig({
  pages: [...appPages],
  window: {
    backgroundTextStyle: 'light',
    navigationBarBackgroundColor: '#0f172a',
    navigationBarTitleText: '观潮家',
    navigationBarTextStyle: 'white'
  }
});
