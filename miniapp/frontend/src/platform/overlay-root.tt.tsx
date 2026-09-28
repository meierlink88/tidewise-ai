import type { ReactNode } from 'react';

// Douyin has no RootPortal. Fixed overlays remain in the page root layer.
export function OverlayRoot({ children }: { children?: ReactNode }) {
  return <>{children}</>;
}
