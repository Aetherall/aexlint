export interface Flags { get(name: string | number | boolean | null): { on(): boolean } }
export const a = (flags: Flags) => ({ x: flags.get(true).on(), y: flags.get(null).on(), z: flags.get("9150").on() });
