class Format { checkIntegrity() { if (!['png', 'jpeg'].includes(this.format)) throw new UnknownFormatError(this.format); } }
