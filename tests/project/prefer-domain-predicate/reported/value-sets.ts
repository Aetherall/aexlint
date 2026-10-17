if (node.type === 'Array' || node.type === 'Object') use(node);
if (node.type !== 'Array' && node.type !== 'Object') use(node);
if (node.type === 'Array' || 'Object' === node.type) use(node);
if (value === `first` || value === `second`) use(value);
if (!(node.type === 'Array' || node.type === 'Object')) use(node);
if (node.type === 'Array' || (node.type === 'Object' || node.type === 'Other')) use(node);
