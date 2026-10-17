export declare class Brightness { is(value: string): boolean; }
export declare class SiteId { serialize(): string; }
export declare class Site { readonly id: SiteId; getBrightness(): Brightness; }
