function check(value) { const bound = enabled ? 10 : 20; if (value < 0 || value > bound) throw new Error('range'); }
