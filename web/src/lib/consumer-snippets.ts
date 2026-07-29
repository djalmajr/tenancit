export const consumerSnippets = {
  identify: `curl -H "Authorization: Bearer <token>" \\
  "/v1/identify?hostname=<tenant-hostname>"`,
  byTenantId: `curl -H "Authorization: Bearer <token>" \\
  -H 'If-None-Match: "<etag>"' \\
  "/v1/resolve?tenantId=<tenant-slug>"`,
  byHostname: `curl -H "Authorization: Bearer <token>" \\
  "/v1/resolve?hostname=<tenant-hostname>"`,
  resource: `curl -H "Authorization: Bearer <token>" \\
  "/v1/resolve/<tenant-hostname>/resources/<resource-alias>"`,
} as const;
