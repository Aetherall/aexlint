import { Credential } from "./credential.js";
export function after(command: { expiresAt: string }) {
  const expiresAt = new Date(command.expiresAt);
  const created = Credential.create(expiresAt);
  if (expiresAt <= new Date()) throw new Error("future");
  return created;
}
export function unrelated(command: { expiresAt: string; label: string }) {
  if (command.label.length > 20) throw new Error("label");
  return Credential.create(new Date(command.expiresAt));
}
export function narrowed(command: { expiresAt?: Date }) {
  if (!command.expiresAt) throw new Error("missing");
  return Credential.create(command.expiresAt);
}
