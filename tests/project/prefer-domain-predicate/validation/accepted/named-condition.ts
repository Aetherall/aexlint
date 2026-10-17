function check(value) { const invalid = value < 0 || value > 10; if (invalid) throw new Error('range'); }
