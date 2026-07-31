import { createContext, type ReactNode, useContext, useEffect, useMemo, useState } from 'react';

type Theme = 'light' | 'dark' | 'system';

interface ThemeContextValue {
  theme: Theme;
  setTheme: (theme: Theme) => void;
}

const themeStorageKey = 'tidewise_admin_theme';
const ThemeContext = createContext<ThemeContextValue | null>(null);

function resolveTheme(theme: Theme) {
  if (theme !== 'system') {
    return theme;
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    const storedTheme = localStorage.getItem(themeStorageKey);
    return storedTheme === 'light' || storedTheme === 'dark' || storedTheme === 'system'
      ? storedTheme
      : 'system';
  });

  useEffect(() => {
    const root = document.documentElement;
    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const applyTheme = () => {
      root.classList.toggle('dark', resolveTheme(theme) === 'dark');
      root.style.colorScheme = resolveTheme(theme);
    };

    localStorage.setItem(themeStorageKey, theme);
    applyTheme();
    media.addEventListener('change', applyTheme);
    return () => media.removeEventListener('change', applyTheme);
  }, [theme]);

  const value = useMemo(() => ({ theme, setTheme }), [theme]);
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  const value = useContext(ThemeContext);
  if (!value) {
    throw new Error('useTheme must be used within ThemeProvider');
  }
  return value;
}
