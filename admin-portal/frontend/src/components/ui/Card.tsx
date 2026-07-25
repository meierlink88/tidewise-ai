import type { HTMLAttributes, ReactNode } from 'react';

interface CardProps extends HTMLAttributes<HTMLElement> {
  children: ReactNode;
  muted?: boolean;
}

export default function Card({ children, className = '', muted = false, ...props }: CardProps) {
  return (
    <article className={`ui-card ${muted ? 'ui-card-muted' : ''} ${className}`.trim()} {...props}>
      {children}
    </article>
  );
}
