export default function SkipToMain() {
  return (
    <a
      className='fixed left-4 top-4 z-[100] -translate-y-24 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow-md transition-transform focus:translate-y-0 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2'
      href='#admin-main-content'
    >
      跳到主要内容
    </a>
  );
}
