function check(value) { const changed = delete state.value; if (value < 0 || value > 10) throw new Error(); }
