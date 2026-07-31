import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../../lib/utils';

const buttonVariants = cva(
  'inline-flex min-h-9 shrink-0 items-center justify-center gap-2 rounded-md border border-transparent px-4 text-sm font-medium whitespace-nowrap transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground hover:bg-primary/90',
        destructive: 'bg-destructive text-white hover:bg-destructive/90',
        outline: 'border-border bg-background text-foreground hover:bg-accent',
        secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
        ghost: 'text-foreground hover:bg-accent'
      },
      size: {
        default: 'h-10',
        sm: 'h-9 px-3',
        icon: 'size-9 p-0'
      }
    },
    defaultVariants: {
      variant: 'default',
      size: 'default'
    }
  }
);

type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
  };

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ asChild = false, className, size, variant, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button';
    return (
      <Comp
        className={cn(buttonVariants({ className, size, variant }))}
        ref={ref}
        type={asChild ? undefined : (props.type ?? 'button')}
        {...props}
      />
    );
  }
);
Button.displayName = 'Button';

interface LegacyButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
}

function LegacyButton({
  className,
  type = 'button',
  variant = 'primary',
  ...props
}: LegacyButtonProps) {
  return (
    <button className={cn('ui-button', `ui-button-${variant}`, className)} type={type} {...props} />
  );
}

export default LegacyButton;
export { Button, buttonVariants };
