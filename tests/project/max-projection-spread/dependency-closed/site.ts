export class Brightness { constructor(private readonly value: string) {} is(value: string) { return this.value === value; } }
export class SiteId { constructor(private readonly value: string) {} serialize() { return this.value; } }
export class Site {
  constructor(readonly id: SiteId, private readonly brightness: Brightness) {}
  getBrightness() { return this.brightness; }
}
