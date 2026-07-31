import Icon from '../ui/Icon';
import { Button } from '../ui/Button';
import { useTheme } from '../../providers/theme-provider';

export default function ThemeToggle() {
  const { theme, setTheme } = useTheme();
  const isDark = theme === 'dark';

  return (
    <Button
      aria-label={isDark ? '切换到浅色主题' : '切换到深色主题'}
      onClick={() => setTheme(isDark ? 'light' : 'dark')}
      size='icon'
      variant='ghost'
    >
      <Icon name={isDark ? 'sun' : 'moon'} />
    </Button>
  );
}
