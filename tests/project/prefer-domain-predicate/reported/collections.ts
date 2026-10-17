if (['Array', 'Object'].includes(node.type)) use(node);
if (new Set(['Array', 'Object']).has(node.type)) use(node);
if (['Array', 'Object']['includes'](node['type'])) use(node);
if ((['Array', 'Object'] as const).includes(node.type)) use(node);
if (['Array', 'Object'].includes(node.type) || node.type === 'Other') use(node);
