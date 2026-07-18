interface IconProps {
  name: 'chart' | 'database' | 'file' | 'log-out' | 'search' | 'settings' | 'table' | 'tag';
  size?: number;
}

const paths: Record<IconProps['name'], string[]> = {
  chart: ['M3 3v18h18', 'M7 15v2', 'M12 9v8', 'M17 12v5'],
  database: ['M4 6c0-2 4-3 8-3s8 1 8 3-4 3-8 3-8-1-8-3Z', 'M4 6v6c0 2 4 3 8 3s8-1 8-3V6', 'M4 12v6c0 2 4 3 8 3s8-1 8-3v-6'],
  file: ['M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8l6 6v12a2 2 0 0 1-2 2Z', 'M14 2v6h6'],
  'log-out': ['M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4', 'M16 17l5-5-5-5', 'M21 12H9'],
  search: ['M21 21l-4.3-4.3', 'M10 18a8 8 0 1 1 0-16 8 8 0 0 1 0 16Z'],
  settings: ['M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z', 'M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 0 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.6V21a2 2 0 0 1-4 0v-.1a1.7 1.7 0 0 0-1-1.6 1.7 1.7 0 0 0-1.9.3l-.1.1A2 2 0 0 1 4.2 17l.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.6-1H3a2 2 0 0 1 0-4h.1a1.7 1.7 0 0 0 1.6-1 1.7 1.7 0 0 0-.3-1.9l-.1-.1A2 2 0 0 1 7 4.2l.1.1a1.7 1.7 0 0 0 1.9.3h.1a1.7 1.7 0 0 0 1-1.6V3a2 2 0 0 1 4 0v.1a1.7 1.7 0 0 0 1 1.6h.1a1.7 1.7 0 0 0 1.9-.3l.1-.1A2 2 0 0 1 19.8 7l-.1.1a1.7 1.7 0 0 0-.3 1.9v.1a1.7 1.7 0 0 0 1.6 1h.1a2 2 0 0 1 0 4H21a1.7 1.7 0 0 0-1.6 1Z'],
  table: ['M3 5h18v14H3Z', 'M3 10h18', 'M9 5v14', 'M15 5v14'],
  tag: ['M12.6 2.6A2 2 0 0 0 11.2 2H4a2 2 0 0 0-2 2v7.2a2 2 0 0 0 .6 1.4l8.7 8.7a2.4 2.4 0 0 0 3.4 0l6.6-6.6a2.4 2.4 0 0 0 0-3.4Z', 'M7.5 7.5h.01']
};

export default function Icon({ name, size = 16 }: IconProps) {
  return (
    <svg className="admin-nav-icon lucide" fill="none" height={size} stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" viewBox="0 0 24 24" width={size} aria-hidden="true">
      {paths[name].map((path) => <path d={path} key={path} />)}
    </svg>
  );
}
