class Note { checkIntegrity(): void { if (typeof this.value !== 'number') throw new Error('number'); if (this.value < 0 || this.value > 10) throw ModelProblem.Invalid.reason(Note, 'range'); } }
