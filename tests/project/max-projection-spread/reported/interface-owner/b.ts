import type { Account } from "./file.js"; export const b = (x: Account) => ({ email: x.getEmail().normalize(), paid: x.getPlan().is("frosty") });
