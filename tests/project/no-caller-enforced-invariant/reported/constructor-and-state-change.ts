export class Account {
  constructor(readonly email: string) {}
  private name = "";
  rename(name: string): void { this.name = name; }
}
export function open(input: { email: string; name: string }) {
  if (!input.email.includes("@")) throw new Error("email");
  const account = new Account(input.email);
  if (input.name.trim() === "") throw new Error("name");
  account.rename(input.name);
  return account;
}
