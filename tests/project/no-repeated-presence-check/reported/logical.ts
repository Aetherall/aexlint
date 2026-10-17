type Address = { postcode: number };
function toLabel(c?: Address) {
  if (!c) return undefined;
  return { at: c.postcode };
}
export const f = (address: Address | undefined) => address && toLabel(address);
