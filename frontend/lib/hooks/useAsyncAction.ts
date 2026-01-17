import { useState, useCallback } from 'react';

interface AsyncActionState<T> {
  isLoading: boolean;
  error: Error | null;
  data: T | null;
}

export function useAsyncAction<T = unknown>() {
  const [state, setState] = useState<AsyncActionState<T>>({
    isLoading: false,
    error: null,
    data: null,
  });

  const execute = useCallback(async (action: () => Promise<T>): Promise<T> => {
    setState(prev => ({ ...prev, isLoading: true, error: null }));
    try {
      const result = await action();
      setState({ isLoading: false, error: null, data: result });
      return result;
    } catch (e) {
      const error = e instanceof Error ? e : new Error(String(e));
      setState(prev => ({ ...prev, isLoading: false, error }));
      throw error;
    }
  }, []);

  const reset = useCallback(() => {
    setState({ isLoading: false, error: null, data: null });
  }, []);

  return { ...state, execute, reset };
}