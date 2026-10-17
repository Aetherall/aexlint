if (left.type === 'Array' || right.type === 'Object') use(left);
if (node.type === 'Array' || node.parent.type === 'Object') use(node);
if (read().type === 'Array' || read().type === 'Object') use();
if (node[first] === 'Array' || node[second] === 'Object') use();
if (node['a.b'].type === 'Array' || node.a.b.type === 'Object') use();
