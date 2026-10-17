type Node = { next?: Node };
export function walk(input: Node): Node { return input.next ? walk(input.next) : { ...input }; }
