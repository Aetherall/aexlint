if (node.isContainer()) use(node);
if (isContainer(node)) use(node);
if (node.type === 'Array') use(node);
if (node.type === 'Array') one(); if (node.type === 'Object') two();
if (permissions.includes('manage')) edit();
