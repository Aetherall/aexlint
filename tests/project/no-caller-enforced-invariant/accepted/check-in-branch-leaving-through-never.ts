import { Credential } from "./credential.js";
declare function reject(reason: string): never;
export function retry(command: { expiresAt: string; retry: boolean }) {
  const expiresAt = new Date(command.expiresAt);
  if (command.retry) {
    if (expiresAt <= new Date()) throw new Error("future");
    reject("retry");
  }
  return Credential.create(expiresAt);
}
