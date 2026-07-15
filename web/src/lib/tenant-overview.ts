import type { TenantResource } from "@/lib/api";

export type TenantReadiness = "ready" | "attention" | "incomplete";

export interface TenantOverviewSummary {
  activeResourceCount: number;
  attentionCodes: Array<
    | "inactive_tenant"
    | "missing_domain"
    | "missing_active_resource"
    | "inactive_resources"
    | "incomplete_resources"
  >;
  incompleteResourceCount: number;
  inactiveResourceCount: number;
  readiness: TenantReadiness;
  readyRequirementCount: number;
  totalResourceCount: number;
}

export function summarizeTenantOverview(input: {
  domainCount: number;
  resources: TenantResource[];
  tenantStatus: string;
}): TenantOverviewSummary {
  const tenantIsActive = input.tenantStatus === "active";
  const hasDomain = input.domainCount > 0;
  const activeResourceCount = input.resources.filter((resource) => resource.status === "active").length;
  const inactiveResourceCount = input.resources.length - activeResourceCount;
  const incompleteResourceCount = input.resources.filter((resource) =>
    resource.fields.some((field) => field.required && field.value.trim() === ""),
  ).length;
  const hasActiveResource = activeResourceCount > 0;
  const readyRequirementCount = [tenantIsActive, hasDomain, hasActiveResource].filter(Boolean).length;
  const attentionCodes: TenantOverviewSummary["attentionCodes"] = [];

  if (!tenantIsActive) attentionCodes.push("inactive_tenant");
  if (!hasDomain) attentionCodes.push("missing_domain");
  if (!hasActiveResource) attentionCodes.push("missing_active_resource");
  if (inactiveResourceCount > 0) attentionCodes.push("inactive_resources");
  if (incompleteResourceCount > 0) attentionCodes.push("incomplete_resources");

  const hasBlockingIssue = !tenantIsActive || !hasDomain || !hasActiveResource || incompleteResourceCount > 0;
  const readiness: TenantReadiness = hasBlockingIssue
    ? "incomplete"
    : inactiveResourceCount > 0
      ? "attention"
      : "ready";

  return {
    activeResourceCount,
    attentionCodes,
    incompleteResourceCount,
    inactiveResourceCount,
    readiness,
    readyRequirementCount,
    totalResourceCount: input.resources.length,
  };
}
