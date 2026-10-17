type Item = { code: 9401 | 9402 };
const chosen = 9401;
function selected(value: Item): value is Item { return value.code === chosen; }
