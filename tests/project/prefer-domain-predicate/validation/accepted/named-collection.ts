function check(value) { const allowed = ['first', 'second']; if (!allowed.includes(value)) throw new Error('invalid'); }
