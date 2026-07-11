import { QueryObserver } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api";
import {
  QUERY_STALE_TIME_MS,
  createAppQueryClient,
  registerAdminQueryCacheInvalidation,
  shouldRetryQuery,
} from "@/lib/query-client";

describe("query client policy", () => {
  it("does not retry client or authentication errors", () => {
    expect(shouldRetryQuery(0, new ApiError(401))).toBe(false);
    expect(shouldRetryQuery(0, new ApiError(404))).toBe(false);
  });

  it("bounds retries for transient failures", () => {
    expect(shouldRetryQuery(0, new ApiError(500))).toBe(true);
    expect(shouldRetryQuery(1, new TypeError("network unavailable"))).toBe(true);
    expect(shouldRetryQuery(2, new ApiError(503))).toBe(false);
  });

  it("uses the shared stale-time policy", () => {
    const queryClient = createAppQueryClient();
    expect(queryClient.getDefaultOptions().queries?.staleTime).toBe(QUERY_STALE_TIME_MS);
  });

  it("cancels and clears protected state without refetching the previous identity", () => {
    const queryClient = createAppQueryClient();
    const unregister = registerAdminQueryCacheInvalidation(queryClient);
    const aborted = vi.fn();
    const queryFn = vi.fn(({ signal }: { signal: AbortSignal }) =>
      new Promise<never>((_resolve, reject) => {
        signal.addEventListener("abort", () => {
          aborted();
          reject(new DOMException("aborted", "AbortError"));
        }, { once: true });
      }),
    );
    const observer = new QueryObserver(queryClient, {
      queryFn,
      queryKey: ["admin", "api-clients"],
    });
    const unsubscribe = observer.subscribe(() => undefined);
    queryClient.setQueryData(["admin", "api-clients"], [{ id: "protected" }]);
    queryClient.getMutationCache().build(queryClient, {
      mutationFn: () => Promise.resolve({ token: "one-shot" }),
      mutationKey: ["admin", "mutation"],
    });
    expect(observer.getCurrentResult().data).toEqual([{ id: "protected" }]);

    window.dispatchEvent(new Event("admin-token-change"));

    expect(observer.getCurrentResult().data).toBeUndefined();
    expect(queryClient.getQueryData(["admin", "api-clients"])).toBeUndefined();
    expect(queryClient.getQueryCache().getAll()).toHaveLength(0);
    expect(queryClient.getMutationCache().getAll()).toHaveLength(0);
    expect(aborted).toHaveBeenCalledTimes(1);
    expect(queryFn).toHaveBeenCalledTimes(1);

    const nextIdentityObserver = new QueryObserver(queryClient, {
      queryFn,
      queryKey: ["admin", "api-clients"],
    });
    const unsubscribeNextIdentity = nextIdentityObserver.subscribe(() => undefined);
    expect(queryFn).toHaveBeenCalledTimes(2);

    unsubscribeNextIdentity();
    unsubscribe();
    unregister();
  });
});
