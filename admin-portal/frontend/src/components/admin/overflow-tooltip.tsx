import { useEffect, useId, useState, type FocusEvent, type MouseEvent } from 'react';
import { createPortal } from 'react-dom';
import { cn } from '../../lib/utils';

interface OverflowTooltipProps {
  className?: string;
  value: string;
}

interface TooltipPosition {
  left: number;
  top: number;
}

export function OverflowTooltip({ className, value }: OverflowTooltipProps) {
  const tooltipId = useId();
  const [position, setPosition] = useState<TooltipPosition>();

  useEffect(() => {
    if (!position) return;
    const close = () => setPosition(undefined);
    window.addEventListener('resize', close);
    window.addEventListener('scroll', close, true);
    return () => {
      window.removeEventListener('resize', close);
      window.removeEventListener('scroll', close, true);
    };
  }, [position]);

  const open = (target: HTMLElement) => {
    const rect = target.getBoundingClientRect();
    setPosition({
      left: Math.max(16, Math.min(rect.left, window.innerWidth - 528)),
      top: rect.top - 8
    });
  };

  return (
    <>
      <span
        aria-describedby={position ? tooltipId : undefined}
        className={cn(
          'block truncate outline-none focus-visible:ring-2 focus-visible:ring-ring',
          className
        )}
        onBlur={() => setPosition(undefined)}
        onFocus={(event: FocusEvent<HTMLSpanElement>) => open(event.currentTarget)}
        onMouseEnter={(event: MouseEvent<HTMLSpanElement>) => open(event.currentTarget)}
        onMouseLeave={() => setPosition(undefined)}
        tabIndex={0}
      >
        {value}
      </span>
      {position
        ? createPortal(
            <span
              className='pointer-events-none fixed z-[100] max-w-[min(32rem,calc(100vw-2rem))] -translate-y-full whitespace-normal break-words rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm leading-5 font-medium text-white shadow-2xl dark:border-slate-200 dark:bg-slate-100 dark:text-slate-950'
              id={tooltipId}
              role='tooltip'
              style={position}
            >
              {value}
            </span>,
            document.body
          )
        : null}
    </>
  );
}
