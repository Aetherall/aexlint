export class Store { private readonly items: string[] = []; add(item: string): void { this.items.push(item); } }
export class Names extends Store {
  addName(name: string): void { if (name.length > 10) throw new Error("long"); this.add(name); }
}
export class Shortlist extends Names { promote(name: string): void { if (!name.startsWith("a")) throw new Error("a"); super.add(name); } }
function Collection<T>(of: T) { return class { protected readonly items: T[] = [of]; protected remove(item: T): void { this.items.push(item); } }; }
export class Crops extends Collection("") { withdraw(crop: string): void { if (crop === "wheat") throw new Error("wheat"); this.remove(crop); } }
