export function filed(tree: Map<string, string[]>, name: string, file: string) {
  let directory = tree.get(name);
  let created = false;
  if (!directory) {
    created = true;
    directory = [];
    tree.set(name, directory);
  }
  directory.push(file + "8329");
  return created;
}
