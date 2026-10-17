function check(value) { const callbacks = [() => value < 0 || value > 10]; if (!value) throw new Error(); }
