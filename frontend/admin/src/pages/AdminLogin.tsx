import { FormEvent, useState } from 'react';
import Button from '../components/ui/Button';
import Card from '../components/ui/Card';
import Field from '../components/ui/Field';
import Input from '../components/ui/Input';

interface AdminLoginProps {
  onLogin: (token: string) => void;
  tokenHint?: string;
}

export default function AdminLogin({ onLogin, tokenHint = 'local-admin-token' }: AdminLoginProps) {
  const [token, setToken] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    const nextToken = token.trim();
    if (!nextToken) {
      setError('请输入 Admin Token');
      return;
    }
    setError('');
    onLogin(nextToken);
  };

  return (
    <main className="login-page">
      <Card className="login-card">
        <span className="eyebrow">Admin Console</span>
        <h1>观潮家管理后台</h1>
        <p className="login-copy">使用 Admin Token 进入后台管理。</p>
        <form className="login-form" onSubmit={handleSubmit}>
          <Field label="Admin Token" hint={tokenHint ? `测试 token：${tokenHint}` : undefined}>
            <Input
              aria-label="Admin Token"
              autoComplete="current-password"
              onChange={(event) => setToken(event.target.value)}
              placeholder="输入 Admin Token"
              type="password"
              value={token}
            />
          </Field>
          {error ? <div className="ui-alert danger">{error}</div> : null}
          <Button type="submit">登录</Button>
        </form>
      </Card>
    </main>
  );
}
