import { QueryClient } from "@tanstack/react-query";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";
import { adminQueryOptions } from "./query-options";

afterEach(() => vi.restoreAllMocks());

describe("admin query options", () => {
  it("forwards TanStack Query cancellation to API reads", async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    let receivedSignal: AbortSignal | undefined;
    let resolveRequestStarted!: () => void;
    const requestStarted = new Promise<void>((resolve) => {
      resolveRequestStarted = resolve;
    });
    vi.spyOn(api, "getTenant").mockImplementation((_id, signal) => {
      receivedSignal = signal;
      resolveRequestStarted();
      return new Promise((_resolve, reject) => {
        signal?.addEventListener(
          "abort",
          () => reject(new DOMException("aborted", "AbortError")),
          { once: true },
        );
      });
    });

    const request = queryClient.fetchQuery(adminQueryOptions.tenant("tenant-1"));
    await requestStarted;
    await queryClient.cancelQueries({ queryKey: adminQueryOptions.tenant("tenant-1").queryKey });

    expect(receivedSignal?.aborted).toBe(true);
    await expect(request).rejects.toThrow("CancelledError");
  });
});
