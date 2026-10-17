type Site = { id: string; crewId: string; name: string };
const sites = new Map<string, Site>();
export function get(query: { siteId: string; crewId: string }): { id: string; name: string } {
  const site = sites.get(query.siteId);
  if (!site || site.crewId !== query.crewId) throw new Error("not found");
  return { id: site.id, name: site.name };
}
