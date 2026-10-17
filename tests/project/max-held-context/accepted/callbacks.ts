function save(items) { return perform(async () => { await Promise.all(items.map(async (item) => Promise.all([item].map(async (inner) => store.save(inner))))); }); }

const value = items.map((item) => (item ? 1 : 2));
