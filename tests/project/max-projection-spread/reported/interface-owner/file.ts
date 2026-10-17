export interface Account { getEmail(): { normalize(): string }; getPlan(): { is(name: string): boolean } }
export function a(account: Account | undefined) { return { email: account!.getEmail().normalize(), paid: account!.getPlan().is("frosty") }; }
