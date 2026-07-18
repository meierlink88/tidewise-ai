import type { ButtonHTMLAttributes, ReactNode } from 'react';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
}

export default function Button({ children, className = '', variant = 'primary', type = 'button', ...props }: ButtonProps) {
  return (
    <button className={`ui-button ui-button-${variant} ${className}`.trim()} type={type} {...props}>
      {children}
    </button>
  );
}
