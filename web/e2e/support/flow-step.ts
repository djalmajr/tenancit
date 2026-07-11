import { test } from "@playwright/test";

export async function flowStep(
  flowId: string,
  stepNumber: number,
  title: string,
  actionAndAssertion: () => Promise<void>,
) {
  if (!flowId.trim()) throw new Error("flowStep requires a flow id");
  if (!Number.isInteger(stepNumber) || stepNumber < 1) {
    throw new Error(`flowStep requires a positive integer, received ${stepNumber}`);
  }
  if (!title.trim()) throw new Error(`${flowId}#${stepNumber} requires a title`);

  await test.step(`[${flowId}#${stepNumber}] ${title}`, actionAndAssertion);
}
