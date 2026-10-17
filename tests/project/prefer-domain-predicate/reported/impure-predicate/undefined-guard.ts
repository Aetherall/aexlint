function isContainer(node) { if (!node) return undefined; return ['Array', 'Object'].includes(node.type); }
