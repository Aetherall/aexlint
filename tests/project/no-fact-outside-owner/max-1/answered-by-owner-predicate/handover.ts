export class Handover {
  refused?: { at: number };
  isRefused() { return this.refused !== undefined; }
}
