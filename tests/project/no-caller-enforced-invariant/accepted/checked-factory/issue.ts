import { Credential } from "./credential.js";
export function issue(command: { expiresAt: string }) {
  const expiresAt = new Date(command.expiresAt);
  if (Number.isNaN(expiresAt.getTime()) || expiresAt <= new Date()) throw new Error("future");
  return Credential.create(expiresAt);
}
