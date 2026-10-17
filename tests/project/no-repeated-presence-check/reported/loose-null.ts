type Recipient = { name: string };
function name(u?: Recipient) {
  if (u == null) return;
  return u.name;
}
export const f = (recipient: Recipient | undefined) => (recipient != null ? name(recipient) : undefined);
