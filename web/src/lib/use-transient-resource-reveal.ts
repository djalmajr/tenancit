import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import {
  ADMIN_TOKEN_KEY,
  ADMIN_TOKEN_CHANGE_EVENT,
  api,
  getAdminToken,
  type TenantResource,
} from "./api";

export interface TransientResourceReveal {
  error: unknown;
  hide: () => void;
  isLoading: boolean;
  isRevealed: boolean;
  resources: TenantResource[] | undefined;
  show: () => Promise<boolean>;
}

export function useTransientResourceReveal(tenantId: string): TransientResourceReveal {
  const [resources, setResources] = useState<TenantResource[]>();
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<unknown>();
  const controllerRef = useRef<AbortController | undefined>(undefined);
  const epochRef = useRef(0);
  const mountedRef = useRef(true);
  const tenantRef = useRef(tenantId);

  const cancelAndErase = useCallback((updateState = true) => {
    epochRef.current += 1;
    controllerRef.current?.abort();
    controllerRef.current = undefined;
    if (!updateState || !mountedRef.current) return;
    setResources(undefined);
    setError(undefined);
    setIsLoading(false);
  }, []);

  const hide = useCallback(() => cancelAndErase(), [cancelAndErase]);

  const show = useCallback(async () => {
    cancelAndErase();
    const requestEpoch = epochRef.current;
    const credential = getAdminToken();
    const controller = new AbortController();
    controllerRef.current = controller;
    setIsLoading(true);

    try {
      const revealed = await api.listTenantResources(tenantId, true, controller.signal);
      const isCurrent = mountedRef.current
        && !controller.signal.aborted
        && requestEpoch === epochRef.current
        && tenantRef.current === tenantId
        && getAdminToken() === credential;
      if (!isCurrent) return false;
      setResources(revealed);
      setError(undefined);
      return true;
    } catch (revealError) {
      const wasCancelled = controller.signal.aborted
        || (revealError instanceof DOMException && revealError.name === "AbortError");
      if (wasCancelled || !mountedRef.current || requestEpoch !== epochRef.current) return false;
      setResources(undefined);
      setError(revealError);
      return false;
    } finally {
      if (mountedRef.current && requestEpoch === epochRef.current) setIsLoading(false);
    }
  }, [cancelAndErase, tenantId]);

  useLayoutEffect(() => {
    tenantRef.current = tenantId;
    cancelAndErase();
  }, [cancelAndErase, tenantId]);

  useEffect(() => {
    const eraseForCredentialChange = () => cancelAndErase();
    const eraseForStorageChange = (event: StorageEvent) => {
      if (event.key === ADMIN_TOKEN_KEY) cancelAndErase();
    };
    window.addEventListener(ADMIN_TOKEN_CHANGE_EVENT, eraseForCredentialChange);
    window.addEventListener("storage", eraseForStorageChange);
    return () => {
      window.removeEventListener(ADMIN_TOKEN_CHANGE_EVENT, eraseForCredentialChange);
      window.removeEventListener("storage", eraseForStorageChange);
    };
  }, [cancelAndErase]);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      cancelAndErase(false);
    };
  }, [cancelAndErase]);

  return {
    error,
    hide,
    isLoading,
    isRevealed: resources !== undefined,
    resources,
    show,
  };
}
