if ([0, -0, +0].includes(value)) use();
if ([null, true].includes(value)) use();
if ([, 'first', 'second'].includes(value)) use();
if (['first', 'second'].includes(value, 1)) use();
