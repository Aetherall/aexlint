if (node?.type === 'Array' || node?.type === 'Object') use(node);
if ((node.type as string) === 'Array' || node.type! === 'Object') use(node);
if (node.parent.type === 'Array' || node.parent.type === 'Object') use(node);
if (node['type'] === 'Array' || node.type === 'Object') use(node);
if (node['a.b'].type === 'Array' || node['a.b'].type === 'Object') use(node);
