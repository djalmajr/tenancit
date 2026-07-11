import { QueryClient } from "@tanstack/react-query";
import { ADMIN_TOKEN_CHANGE_EVENT, ADMIN_TOKEN_KEY, ApiError } from "@/lib/api";
import { adminQueryKeys } from "@/lib/query-keys";

export const QUERY_STALE_TIME_MS = 30_000;

export function shouldRetryQuery(failureCount: number, error: unknown): boolean {
  if (error instanceof ApiError && error.status < 500) return false;
  return failureCount < 2;
}

export function createAppQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: shouldRetryQuery,
        staleTime: QUERY_STALE_TIME_MS,
      },
    },
  });
}

export function registerAdminQueryCacheInvalidation(queryClient: QueryClient): () => void {
  const clearCache = () => {
    void queryClient.cancelQueries({ queryKey: adminQueryKeys.all });
    // Reset each observer synchronously so no protected value survives in a
    // mounted hook. This is Query.reset(), not resetQueries(): it cannot refetch.
    queryClient.getQueryCache()
      .findAll({ queryKey: adminQueryKeys.all })
      .forEach((query) => query.reset());
    queryClient.clear();
  };
  const clearCacheFromStorage = (event: StorageEvent) => {
    if (event.key === ADMIN_TOKEN_KEY) clearCache();
  };

  window.addEventListener(ADMIN_TOKEN_CHANGE_EVENT, clearCache);
  window.addEventListener("storage", clearCacheFromStorage);
  return () => {
    window.removeEventListener(ADMIN_TOKEN_CHANGE_EVENT, clearCache);
    window.removeEventListener("storage", clearCacheFromStorage);
  };
}
