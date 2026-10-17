function check(value) { const label = `value: ${value}`; if (value < 0 || value > 10) throw new Error(label); }
