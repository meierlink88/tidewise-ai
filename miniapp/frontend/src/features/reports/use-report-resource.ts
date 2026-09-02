import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ReportResourceSession, type ReportResourceState } from './session';

export interface ReportResource<T> {
  state: ReportResourceState<T>;
  retry: () => Promise<void>;
  refresh: () => Promise<void>;
  snapshot: () => ReportResourceState<T>;
}

export function useReportResource<T>(
  resourceKey: string,
  loader: () => Promise<T>,
  isEmpty: (value: T) => boolean = () => false
): ReportResource<T> {
  const loaderRef = useRef(loader);
  loaderRef.current = loader;
  const emptyRef = useRef(isEmpty);
  emptyRef.current = isEmpty;

  const session = useMemo(
    () => new ReportResourceSession<T>((value) => emptyRef.current(value), resourceKey),
    [resourceKey]
  );
  const [state, setState] = useState<ReportResourceState<T>>(session.snapshot());

  useEffect(() => {
    setState(session.snapshot());
    const unsubscribe = session.subscribe(setState);
    void session.load(() => loaderRef.current());
    return () => {
      unsubscribe();
      session.dispose();
    };
  }, [session]);

  const retry = useCallback(() => session.load(() => loaderRef.current()), [session]);
  const refresh = useCallback(() => session.load(() => loaderRef.current(), true), [session]);
  const snapshot = useCallback(() => session.snapshot(), [session]);

  return { state, retry, refresh, snapshot };
}
