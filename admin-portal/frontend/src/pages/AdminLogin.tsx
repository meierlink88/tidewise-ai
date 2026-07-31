import { FormEvent, useRef, useState } from 'react';
import ThemeToggle from '../components/admin/theme-toggle';
import { Button } from '../components/ui/Button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../components/ui/Card';
import { Input } from '../components/ui/Input';

interface AdminLoginProps {
  onLogin: (token: string) => void;
  tokenHint?: string;
}

export default function AdminLogin({ onLogin, tokenHint = 'local-admin-token' }: AdminLoginProps) {
  const [token, setToken] = useState('');
  const [error, setError] = useState('');
  const tokenInputRef = useRef<HTMLInputElement>(null);
  const hintId = 'admin-token-hint';
  const errorId = 'admin-token-error';

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    const nextToken = token.trim();
    if (!nextToken) {
      setError('请输入 Admin Token');
      tokenInputRef.current?.focus();
      return;
    }
    setError('');
    onLogin(nextToken);
  };

  return (
    <main className='relative grid min-h-full place-items-center overflow-auto bg-background px-4 py-12 text-foreground'>
      <div className='absolute right-4 top-4'>
        <ThemeToggle />
      </div>
      <Card className='w-full max-w-md'>
        <CardHeader>
          <span className='text-xs font-semibold uppercase tracking-[0.16em] text-muted-foreground'>
            Admin Console
          </span>
          <CardTitle className='text-3xl'>观潮家管理后台</CardTitle>
          <CardDescription>使用现有 Admin Token 进入管理后台。</CardDescription>
        </CardHeader>
        <CardContent>
          <form className='grid gap-5' onSubmit={handleSubmit}>
            <div className='grid gap-2'>
              <label className='text-sm font-medium' htmlFor='admin-token'>
                Admin Token
              </label>
              <Input
                aria-describedby={error ? errorId : tokenHint ? hintId : undefined}
                aria-invalid={error ? true : undefined}
                autoComplete='current-password'
                id='admin-token'
                onChange={(event) => setToken(event.target.value)}
                placeholder='输入 Admin Token'
                ref={tokenInputRef}
                type='password'
                value={token}
              />
              {tokenHint && !error ? (
                <p className='text-xs text-muted-foreground' id={hintId}>
                  测试 token：{tokenHint}
                </p>
              ) : null}
              {error ? (
                <p className='text-sm font-medium text-destructive' id={errorId} role='alert'>
                  {error}
                </p>
              ) : null}
            </div>
            <Button className='w-full' type='submit'>
              登录
            </Button>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
