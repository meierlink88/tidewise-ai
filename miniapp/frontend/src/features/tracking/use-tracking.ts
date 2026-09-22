import { useEffect, useRef, useState } from 'react';
import { useDidHide, useDidShow } from '@tarojs/taro';
import { clearSession, readSession } from '../../platform/identity';
import { trackingAPI } from './api';
import { TrackingError, type Company, type TrackingPort } from './contract';

type Status = 'idle' | 'loading' | 'ready' | 'error';
export function useTracking(port: TrackingPort = trackingAPI) {
  const [query, setQueryState] = useState('');
  const [items, setItems] = useState<Company[]>([]);
  const [results, setResults] = useState<Company[]>([]);
  const [total, setTotal] = useState(0);
  const [cursor, setCursor] = useState('');
  const [more, setMore] = useState(false);
  const [status, setStatus] = useState<Status>('idle');
  const [searchStatus, setSearchStatus] = useState<Status>('idle');
  const [error, setError] = useState('');
  const [guest, setGuest] = useState(true);
  const [pending, setPending] = useState('');
  const mounted = useRef(true);
  const token = useRef('');
  const queryRef = useRef('');
  const listSequence = useRef(0);
  const searchSequence = useRef(0);
  const mutation = useRef(false);
  const listBusy = useRef(false);
  const searchBusy = useRef(false);
  const timer = useRef<ReturnType<typeof setTimeout>>();
  function current(t: string) {
    if (!mounted.current || token.current !== t) return false;
    if ((readSession()?.session_token || '') !== t) {
      refresh();
      return false;
    }
    return true;
  }
  function fail(e: unknown, t: string) {
    if (!current(t)) return;
    if (e instanceof TrackingError && e.expired) {
      try {
        clearSession();
      } catch {
        /* The server has already rejected this session. */
      }
      token.current = '';
      listSequence.current++;
      searchSequence.current++;
      setGuest(true);
      setItems([]);
      setResults([]);
      setTotal(0);
      setCursor('');
      setMore(false);
      setStatus('idle');
      setSearchStatus('idle');
    }
    setError(e instanceof Error ? e.message : '操作未完成，请重试');
  }
  async function loadList(after = '') {
    const t = token.current;
    if (!t) return;
    if (after && listBusy.current) return;
    const seq = ++listSequence.current;
    listBusy.current = true;
    setStatus('loading');
    setError('');
    try {
      const p = await port.list(t, after);
      if (!current(t) || seq !== listSequence.current) return;
      setItems((old) =>
        after ? [...old, ...p.items.filter((x) => !old.some((y) => y.id === x.id))] : p.items
      );
      setTotal(p.total);
      setCursor(p.next_cursor);
      setStatus('ready');
    } catch (e) {
      if (current(t) && seq === listSequence.current) {
        setStatus('error');
        fail(e, t);
      }
    } finally {
      if (seq === listSequence.current) listBusy.current = false;
    }
  }
  async function search(offset = 0) {
    const q = queryRef.current.trim(),
      t = token.current;
    if (!q) return;
    if (offset && searchBusy.current) return;
    const seq = ++searchSequence.current;
    searchBusy.current = true;
    setSearchStatus('loading');
    setError('');
    try {
      const p = await port.search(q, t, offset);
      if (!current(t) || seq !== searchSequence.current) return;
      setResults((old) =>
        offset ? [...old, ...p.items.filter((x) => !old.some((y) => y.id === x.id))] : p.items
      );
      setMore(p.has_more);
      setSearchStatus('ready');
    } catch (e) {
      if (current(t) && seq === searchSequence.current) {
        setSearchStatus('error');
        fail(e, t);
      }
    } finally {
      if (seq === searchSequence.current) searchBusy.current = false;
    }
  }
  function setQuery(value: string) {
    queryRef.current = value;
    setQueryState(value);
    ++searchSequence.current;
    searchBusy.current = false;
    setResults([]);
    setMore(false);
    setError('');
    setSearchStatus(value.trim() ? 'loading' : 'idle');
    clearTimeout(timer.current);
    if (value.trim()) timer.current = setTimeout(() => void search(), 250);
  }
  function refresh() {
    clearTimeout(timer.current);
    mounted.current = true;
    const next = readSession()?.session_token || '';
    const changed = next !== token.current;
    ++listSequence.current;
    ++searchSequence.current;
    listBusy.current = false;
    searchBusy.current = false;
    token.current = next;
    setGuest(!next);
    setError('');
    if (changed || !next) {
      setItems([]);
      setResults([]);
      setTotal(0);
      setCursor('');
      setMore(false);
    }
    if (next) void loadList();
    else setStatus('idle');
    if (queryRef.current.trim()) void search();
  }
  useDidShow(refresh);
  useDidHide(() => {
    mounted.current = false;
    ++listSequence.current;
    ++searchSequence.current;
    clearTimeout(timer.current);
  });
  useEffect(
    () => () => {
      mounted.current = false;
      ++listSequence.current;
      ++searchSequence.current;
      clearTimeout(timer.current);
    },
    []
  );
  async function change(item: Company, add: boolean) {
    if (mutation.current || !token.current || !current(token.current)) return;
    const t = token.current;
    mutation.current = true;
    setPending(item.id);
    setError('');
    // Invalidate in-flight reads that carry the old membership state.
    ++listSequence.current;
    ++searchSequence.current;
    clearTimeout(timer.current);
    try {
      await port.change(t, item.id, add);
      if (!current(t)) return;
      setResults((old) => old.map((x) => (x.id === item.id ? { ...x, is_followed: add } : x)));
      if (!add) setItems((old) => old.filter((x) => x.id !== item.id));
      await loadList();
      if (current(t) && queryRef.current.trim()) await search();
    } catch (e) {
      if (current(t)) {
        setStatus('error');
        setSearchStatus(queryRef.current.trim() ? 'error' : 'idle');
        fail(e, t);
      }
    } finally {
      mutation.current = false;
      if (mounted.current) setPending('');
    }
  }
  return {
    query,
    setQuery,
    items,
    results,
    total,
    guest,
    status,
    searchStatus,
    error,
    pending,
    refresh,
    retry: () => (query.trim() ? search() : loadList()),
    loadMore: () => (query.trim() ? search(results.length) : loadList(cursor)),
    hasMore: query.trim() ? more : !!cursor,
    change
  };
}
