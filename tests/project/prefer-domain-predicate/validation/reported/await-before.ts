async function check(value) { const changed = await work(); if (value < 0 || value > 10) throw new Error(); }
