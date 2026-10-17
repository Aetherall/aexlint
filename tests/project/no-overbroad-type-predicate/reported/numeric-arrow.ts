type Item = { code: 9401 | 9402 };
const selected = (value: Item): value is Item => value.code === 9401;
