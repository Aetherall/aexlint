if (user.rank === 'chief' || user.credentials.includes('manage')) edit();
if (user.rank === 'chief' || user.credentials.has('manage')) edit();
