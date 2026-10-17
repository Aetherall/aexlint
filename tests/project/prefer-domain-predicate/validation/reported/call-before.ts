function check(value) { const result = performEdit(value); if (value < 0 || value > 10) throw new Error(); }
