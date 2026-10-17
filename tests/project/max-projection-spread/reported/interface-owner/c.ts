import type { Account } from "./file.js"; export const c = (y: Account) => ({ paid: y.getPlan().is("frosty"), email: y.getEmail().normalize() });
