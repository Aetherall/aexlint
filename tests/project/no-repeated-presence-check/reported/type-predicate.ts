type Telegram = { medium: "telegram" };
type Postcard = { medium: "postcard" };
type Post = Telegram | Postcard;
function isTelegram(m: Post | undefined): m is Telegram {
  return m !== undefined && m.medium === "telegram";
}
export const f = (m: Post | undefined) => (m !== undefined ? isTelegram(m) : false);
