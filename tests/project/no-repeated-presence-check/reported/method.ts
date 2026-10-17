type Route = { hub: string };
type Leg = { route?: Route };
export class Router {
  parseRoute(c: Route | undefined) {
    if (!c) {
      return undefined;
    }
    return c.hub;
  }
  leg(leg: Leg) {
    return leg.route ? this.parseRoute(leg.route) : undefined;
  }
}
