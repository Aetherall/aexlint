function isContainer(node) { const kind = node.type; if (!kind) { return false; } return ['Array', 'Object'].includes(kind); }
