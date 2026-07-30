export const RUNTIME_BASE_PATH_META_NAME = "tenancit-base-path";

interface BuildOIDCLoginURLParams {
  loginURL: string;
  origin: string;
  returnTo: string;
}

export function readRuntimeBasePath(root: ParentNode = document): string {
  const configured = root
    .querySelector<HTMLMetaElement>(`meta[name="${RUNTIME_BASE_PATH_META_NAME}"]`)
    ?.content.trim();
  if (!configured || configured === "/") return "/";
  const segments = configured.slice(1).split("/");
  if (
    !configured.startsWith("/") ||
    configured.startsWith("//") ||
    configured.endsWith("/") ||
    configured.includes("?") ||
    configured.includes("#") ||
    configured.includes("\\") ||
    segments.some((segment) => !segment || segment === "." || segment === "..")
  ) {
    throw new Error(`invalid runtime base path: ${configured}`);
  }
  return configured;
}

export function buildAdminEndpoint(
  endpoint: string,
  root: ParentNode = document,
): string {
  if (!/^\/v1\/(?:admin|auth)(?:[/?]|$)/.test(endpoint)) {
    throw new Error(`expected a local admin or auth endpoint: ${endpoint}`);
  }
  const basePath = readRuntimeBasePath(root);
  return basePath === "/" ? endpoint : `${basePath}${endpoint}`;
}

export function buildOIDCLoginURL({
  loginURL,
  origin,
  returnTo,
}: BuildOIDCLoginURLParams): string {
  const expectedOrigin = new URL(origin).origin;
  const target = new URL(loginURL, `${expectedOrigin}/`);
  if (target.origin !== expectedOrigin) {
    throw new Error("OIDC login endpoint must be same-origin");
  }
  target.searchParams.set("return_to", returnTo || "/");
  return target.toString();
}
