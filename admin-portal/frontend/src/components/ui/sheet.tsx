import * as DialogPrimitive from '@radix-ui/react-dialog';
import * as React from 'react';
import { cn } from '../../lib/utils';

const Sheet = DialogPrimitive.Root;
const SheetTrigger = DialogPrimitive.Trigger;
const SheetClose = DialogPrimitive.Close;

const SheetContent = React.forwardRef<
  React.ElementRef<typeof DialogPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof DialogPrimitive.Content>
>(({ children, className, ...props }, ref) => (
  <DialogPrimitive.Portal>
    <DialogPrimitive.Overlay className='fixed inset-0 z-50 bg-black/50' />
    <DialogPrimitive.Content
      ref={ref}
      className={cn(
        'fixed inset-y-0 left-0 z-50 flex w-[min(20rem,88vw)] flex-col border-r border-sidebar-border bg-sidebar p-4 text-sidebar-foreground shadow-xl outline-none',
        className
      )}
      {...props}
    >
      {children}
      <SheetClose
        aria-label='关闭导航菜单'
        className='absolute right-3 top-3 inline-flex size-9 items-center justify-center rounded-md text-muted-foreground outline-none hover:bg-sidebar-accent focus-visible:ring-2 focus-visible:ring-ring'
      >
        <span aria-hidden='true'>×</span>
      </SheetClose>
    </DialogPrimitive.Content>
  </DialogPrimitive.Portal>
));
SheetContent.displayName = DialogPrimitive.Content.displayName;

const SheetTitle = DialogPrimitive.Title;
const SheetDescription = DialogPrimitive.Description;

export { Sheet, SheetClose, SheetContent, SheetDescription, SheetTitle, SheetTrigger };
