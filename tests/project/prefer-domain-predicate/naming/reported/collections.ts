if (['first', 'second'].includes(value)) use();
if (new Set(['first', 'second']).has(value)) use();
if (policy.allowed.includes(node.type) && node.state === 'ready') use();
if (new Set(allowed).has(node.type) && node.state === 'ready') use();
if (allowed.includes(node.type) && privileged.includes(node.type)) use();
if (user.rank === 'chief' || user.credentials.includes('manage')) edit();
