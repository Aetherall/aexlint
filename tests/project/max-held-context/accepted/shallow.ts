for (const x of xs) { if (x) { for (const y of x) use(y); } }

if (a && (b || (c && d))) work();

try { if (a) b(); } catch (error) { if (c) d(); }

switch (kind) { case 1: if (a) { if (b) work(); } break; default: other(); }

if (a) { work(); } if (b) { if (c) { if (d) {} } }
