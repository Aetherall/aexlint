function check(value) { if (value == null) return; if (value < 0 || value > 10) throw new Error('range'); }
