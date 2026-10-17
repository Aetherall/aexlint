function editDocument(doc) { if (doc.state === 'locked' || doc.state === 'archived') throw new AccessDenied(); performEdit(doc); }
