export class Response { private constructor(readonly owner: string) {} static of(owner: string): Response { return new Response(owner); } }
export function read(query: { root: boolean; owner: string }) {
  if (!query.root) {
    if (query.owner.length > 20) throw new Error("owner");
  }
  return Response.of(query.owner);
}
