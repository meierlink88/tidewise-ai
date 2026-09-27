import { Children, useState, type ReactNode } from 'react';
import { Button, RichText, View } from '@tarojs/components';

// All input is checked-in prototype text, never HTML from an API response.
export function ReportHtml({ value, className }: { value: string; className?: string }) {
  const nodes = value.replace(/<(p|h[1-6]|li|table|td|th)(\s[^>]*)?>/g, (tag, name: string) => {
    if (tag.includes('style=')) return tag;
    const style =
      name === 'table'
        ? 'width:100%;border-collapse:collapse'
        : ['td', 'th'].includes(name)
          ? 'padding:6px;border:1px solid #e4e8ef;vertical-align:top'
          : 'margin:8px 0;line-height:1.7';
    return `<${name} style="${style}">`;
  });
  return <RichText className={className} nodes={nodes} />;
}
export function Disclosure({ children, className }: { children?: ReactNode; className?: string }) {
  const [open, setOpen] = useState(false);
  const [heading, ...body] = Children.toArray(children);
  return (
    <View className={(className || '') + (open ? ' is-open' : '')}>
      <Button
        ariaLabel={open ? '收起内容' : '展开内容'}
        onClick={() => setOpen(!open)}
        className='tidewise-button'
      >
        {heading}
      </Button>
      {open && <View>{body}</View>}
    </View>
  );
}
