export class Site {
  private readonly parcels: string[] = [];
  hasParcel(name: string): boolean { return this.parcels.includes(name); }
}
export class Logger { warn(message: string): void { console.log(message); } }
export function join(site: Site, logger: Logger, input: { name: string }) {
  if (input.name.length > 20) throw new Error("long");
  logger.warn(input.name);
  return site.hasParcel(input.name);
}
