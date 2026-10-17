function check(value) { const minimum = limits.minimum + 1; if (value < minimum || value > 10) throw new Error('range'); }
