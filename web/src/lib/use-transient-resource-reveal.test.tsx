import { QueryClient } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { StrictMode, type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api, setAdminToken, type TenantResource } from "./api";
import { adminQueryKeys } from "./query-keys";
import { useTransientResourceReveal } from "./use-transient-resource-reveal";

const maskedResources: TenantResource[] = [
  {
    alias: "postgres",
    definitionId: "definition-1",
    definitionKey: "postgres",
    fields: [
      {
        dataType: "string",
        isOverride: true,
        isSecret: true,
        key: "password",
        label: "Password",
        origin: "local",
        required: true,
        value: "••••••••••••",
      },
    ],
    id: "resource-1",
    linked: false,
    name: "Postgres",
    status: "active",
  },
];

const revealedResources: TenantResource[] = [
  {
    ...maskedResources[0],
    fields: [{ ...maskedResources[0].fields[0], value: "cleartext-secret" }],
  },
];

afterEach(() => {
  localStorage.clear();
  vi.restoreAllMocks();
});

describe("useTransientResourceReveal", () => {
  it("remains usable after the StrictMode setup-cleanup-setup cycle", async () => {
    vi.spyOn(api, "listTenantResources").mockResolvedValue(revealedResources);
    const wrapper = ({ children }: { children: ReactNode }) => (
      <StrictMode>{children}</StrictMode>
    );
    const { result } = renderHook(() => useTransientResourceReveal("tenant-1"), { wrapper });

    await act(() => result.current.show());

    expect(result.current.resources).toEqual(revealedResources);
  });

  it("keeps cleartext outside QueryCache and erases local state on hide", async () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(adminQueryKeys.tenantResources("tenant-1"), maskedResources);
    vi.spyOn(api, "listTenantResources").mockResolvedValue(revealedResources);
    const { result } = renderHook(() => useTransientResourceReveal("tenant-1"));

    await act(() => result.current.show());

    expect(result.current.resources).toEqual(revealedResources);
    expect(JSON.stringify(queryClient.getQueryCache().getAll().map((query) => query.state.data)))
      .not.toContain("cleartext-secret");

    act(() => result.current.hide());
    expect(result.current.resources).toBeUndefined();
    expect(result.current.isRevealed).toBe(false);
  });

  it("ignores a late reveal response after hide", async () => {
    let resolveReveal!: (resources: TenantResource[]) => void;
    const deferred = new Promise<TenantResource[]>((resolve) => {
      resolveReveal = resolve;
    });
    vi.spyOn(api, "listTenantResources").mockReturnValue(deferred);
    const { result } = renderHook(() => useTransientResourceReveal("tenant-1"));

    let request!: Promise<boolean>;
    act(() => {
      request = result.current.show();
    });
    act(() => result.current.hide());
    await act(async () => {
      resolveReveal(revealedResources);
      await deferred;
    });
    await request;

    expect(result.current.resources).toBeUndefined();
    expect(result.current.isRevealed).toBe(false);
  });

  it("aborts and erases cleartext on credential or tenant change", async () => {
    const signals: AbortSignal[] = [];
    vi.spyOn(api, "listTenantResources").mockImplementation((_id, _reveal, signal) => {
      if (signal) signals.push(signal);
      return Promise.resolve(revealedResources);
    });
    const { rerender, result } = renderHook(
      ({ tenantId }) => useTransientResourceReveal(tenantId),
      { initialProps: { tenantId: "tenant-1" } },
    );

    await act(() => result.current.show());
    expect(result.current.resources).toEqual(revealedResources);

    act(() => setAdminToken("next-credential"));
    expect(result.current.resources).toBeUndefined();
    expect(result.current.isRevealed).toBe(false);

    await act(() => result.current.show());
    rerender({ tenantId: "tenant-2" });
    await waitFor(() => expect(result.current.resources).toBeUndefined());
    expect(signals.every((signal) => signal.aborted)).toBe(true);
  });

  it("aborts an in-flight reveal when unmounted", () => {
    let signal: AbortSignal | undefined;
    vi.spyOn(api, "listTenantResources").mockImplementation((_id, _reveal, nextSignal) => {
      signal = nextSignal;
      return new Promise(() => undefined);
    });
    const { result, unmount } = renderHook(() => useTransientResourceReveal("tenant-1"));

    act(() => {
      void result.current.show();
    });
    unmount();

    expect(signal?.aborted).toBe(true);
  });
});
