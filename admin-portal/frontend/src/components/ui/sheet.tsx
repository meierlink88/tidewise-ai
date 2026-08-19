import * as DialogPrimitive from '@radix-ui/react-dialog';
import * as React from 'react';
import { X } from 'lucide-react';
import { cn } from '../../lib/utils';

const Sheet = DialogPrimitive.Root;
const SheetTrigger = DialogPrimitive.Trigger;
const SheetClose = DialogPrimitive.Close;

const SheetContent = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Content> & {
    closeLabel?: string;
    side?: 'left' | 'right';
  }
>(({ children, className, closeLabel = '关闭', side = 'left', ...props }, ref) => (
  <DialogPrimitive.Portal>
    <DialogPrimitive.Overlay className='sheet-overlay fixed inset-0 z-50 bg-black/50' />
    <DialogPrimitive.Content
      data-side={side}
      ref={ref}
      className={cn(
        'sheet-content fixed inset-y-0 z-50 flex w-[min(27.5rem,92vw)] flex-col bg-background p-6 text-foreground shadow-xl outline-none',
        side === 'left' ? 'left-0 border-r' : 'right-0 border-l',
        className
      )}
      {...props}
    >
      {children}
      <SheetClose
        aria-label={closeLabel}
        className='absolute right-4 top-4 inline-flex size-8 items-center justify-center rounded-md text-muted-foreground outline-none hover:bg-accent hover:text-accent-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50'
      >
        <X aria-hidden='true' className='size-4' />
      </SheetClose>
    </DialogPrimitive.Content>
  </DialogPrimitive.Portal>
));
SheetContent.displayName = DialogPrimitive.Content.displayName;

const SheetTitle = DialogPrimitive.Title;
const SheetDescription = DialogPrimitive.Description;

export { Sheet, SheetClose, SheetContent, SheetDescription, SheetTitle, SheetTrigger };
