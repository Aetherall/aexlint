type Cell = { text: string; separator: string };

export function dependent(cells: Cell[]): string {
  let out = "";
  for (const [index, cell] of cells.entries()) {
    if (index > 0) out += cell.separator;
    out += cell.text;
  }
  return out;
}

export function header(names: string[]): string {
  let out = "";
  for (const [index, name] of names.entries()) {
    if (index === 0) out += "Names: ";
    out += name;
  }
  return out;
}

export function tracked(names: string[]): string {
  let out = "";
  let emitted = false;
  for (const name of names) {
    if (!name) continue;
    if (emitted) out += ", ";
    out += name;
    emitted = true;
  }
  return out;
}

export function elsewhere(names: string[], log: string[]): string {
  let out = "";
  for (const [index, name] of names.entries()) {
    if (index > 0) log.push("next");
    out += name;
  }
  return out;
}

export function joinedAlready(names: string[]): string {
  return names.filter((name) => name.length > 0).join(", ");
}

export function longer(names: string[]): string {
  let text = "";
  for (const name of names) text += name + ",";
  return text.slice(0, -2);
}

export function closed(names: string[]): string {
  let text = "";
  for (const name of names) text += name + ", ";
  text += "end";
  return text.slice(0, -2);
}

export function middle(people: { name: string; age: number }[]): string {
  let text = "";
  for (const person of people) text += person.name + ", " + person.age;
  return text.slice(0, -2);
}

export function inside(names: string[]): string[] {
  const out: string[] = [];
  let text = "";
  for (const name of names) {
    text += name + ", ";
    out.push(text.slice(0, -2));
  }
  return out;
}

export function unknownValues(values: unknown[]): string {
  let out = "";
  values.forEach((value, index) => {
    if (index > 0) out += String(value);
    out += "|";
  });
  return out;
}
