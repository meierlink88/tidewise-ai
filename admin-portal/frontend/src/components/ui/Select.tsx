import * as SelectPrimitive from '@radix-ui/react-select';
import { Check, ChevronDown } from 'lucide-react';
import { cn } from '../../lib/utils';

interface SelectOption {
  label: string;
  value: string;
}

interface SelectProps {
  'aria-describedby'?: string;
  ariaLabel: string;
  className?: string;
  id?: string;
  onValueChange: (value: string) => void;
  options: SelectOption[];
  value: string;
}

function Select({
  'aria-describedby': ariaDescribedBy,
  ariaLabel,
  className,
  id,
  onValueChange,
  options,
  value
}: SelectProps) {
  return (
    <SelectPrimitive.Root onValueChange={onValueChange} value={value}>
      <SelectPrimitive.Trigger
        aria-label={ariaLabel}
        aria-describedby={ariaDescribedBy}
        className={cn(
          'flex h-9 w-full items-center justify-between gap-2 rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-xs outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50',
          className
        )}
        id={id}
      >
        <SelectPrimitive.Value />
        <SelectPrimitive.Icon asChild>
          <ChevronDown aria-hidden='true' className='size-4 opacity-50' />
        </SelectPrimitive.Icon>
      </SelectPrimitive.Trigger>
      <SelectPrimitive.Portal>
        <SelectPrimitive.Content
          className='relative z-50 max-h-72 min-w-[8rem] overflow-hidden rounded-md border bg-background text-foreground shadow-md'
          position='popper'
          sideOffset={4}
        >
          <SelectPrimitive.Viewport className='p-1'>
            {options.map((option) => (
              <SelectPrimitive.Item
                className='relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50'
                key={option.value}
                value={option.value}
              >
                <span className='absolute left-2 flex size-4 items-center justify-center'>
                  <SelectPrimitive.ItemIndicator>
                    <Check className='size-4' />
                  </SelectPrimitive.ItemIndicator>
                </span>
                <SelectPrimitive.ItemText>{option.label}</SelectPrimitive.ItemText>
              </SelectPrimitive.Item>
            ))}
          </SelectPrimitive.Viewport>
        </SelectPrimitive.Content>
      </SelectPrimitive.Portal>
    </SelectPrimitive.Root>
  );
}

export { Select, type SelectOption };
