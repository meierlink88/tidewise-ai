export type ResourceState<T> =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'ready'; data: T }
  | { status: 'empty' }
  | { status: 'error'; message: string };

export type ResourceAction<T> =
  | { type: 'load' }
  | { type: 'resolve'; data: T | null }
  | { type: 'reject'; message: string };

export function createInitialResourceState<T>(): ResourceState<T> {
  return { status: 'idle' };
}

export function resourceStateReducer<T>(_state: ResourceState<T>, action: ResourceAction<T>): ResourceState<T> {
  if (action.type === 'load') return { status: 'loading' };
  if (action.type === 'reject') return { status: 'error', message: action.message };
  return action.data === null ? { status: 'empty' } : { status: 'ready', data: action.data };
}
