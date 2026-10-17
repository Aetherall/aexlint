if (a) { if (b) { if (c) { if (d) work(); } } }

function carried(items) { for (const item of items) { if (item.length > 0) { item.map((value) => { if (value < 0) throw new Error("negative"); return value; }); } } }

if (a) { for (const x of xs) { f(() => (x ? 1 : 2)); } }

switch (kind) { case 1: for (const x of xs) { if (x) { if (y) {} } } }

for (const x of xs) { try { work(x); } catch (error) { if (a) { if (b) {} } } }

if (a) { if (b) { if (c) {} else if (d) { if (e) {} } } }

if (a) { if (b) { if (c) { if (d) { if (e) work(); } } } }

if (a) { if (b) { if (c) { if (d) {} } } } if (e) { if (f) { if (g) { if (h) {} } } }

for (const x of xs) {
  for (const y of x) {
    for (const z of y) {
      for (const w of z) use(w);
    }
  }
}

function m(a, b, c, d) { const inner = { run() { if (a) { if (b) { if (c) { if (d) {} } } } } }; return inner; }

for (const a of as) { for (const b of a) { for (const c of b) { if (!c.x) continue; if (!c.y) continue; if (!c.z) continue; use(c); } } }

if (a) { if (b) { if (c) { if (d) {} } else { if (e) {} } } }
