import { createHash } from "node:crypto";
import {
  existsSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  renameSync,
  rmSync,
  statSync,
  utimesSync,
  writeFileSync,
} from "node:fs";
import { join, sep } from "node:path";
import type { Context, Diagnostic, Rule } from "@oxlint/plugins";
import { messageOf } from "./baseline.ts";

interface Finding {
  message: string;
  range: [number, number];
}

function rangeOf(context: Context, { node, loc }: Diagnostic): [number, number] {
  if (node) return [...node.range];
  if (!loc) return [0, 0];
  const start = "start" in loc ? loc.start : loc;
  const end = ("start" in loc && loc.end) || start;
  return [context.sourceCode.getIndexFromLoc(start), context.sourceCode.getIndexFromLoc(end)];
}

interface Kept {
  text: string;
  findings: Record<string, Finding[]>;
}

const unusedFor = 7 * 24 * 60 * 60 * 1000;

function hashOf(parts: Iterable<string | Buffer>): string {
  const hash = createHash("sha256");
  for (const part of parts) hash.update(part).update("\0");
  return hash.digest("hex");
}

function* implementation(): Iterable<string | Buffer> {
  const root = new URL("../", import.meta.url);
  yield readFileSync(new URL(import.meta.resolve("oxlint/package.json")));
  for (const folder of ["src", "rules"]) {
    const entries = readdirSync(new URL(folder, root), { recursive: true, withFileTypes: true });
    for (const entry of entries) {
      if (!entry.isFile() || !/(?<!\.d)\.[jt]s$/u.test(entry.name)) continue;
      const path = join(entry.parentPath, entry.name);
      yield path;
      yield readFileSync(path);
    }
  }
}

function open(directory: string): { kept: Map<string, Kept>; save: () => void } {
  const name = `syntax-${hashOf(implementation()).slice(0, 32)}.json`;
  const path = join(directory, name);
  const kept = new Map<string, Kept>();
  try {
    for (const [file, entry] of Object.entries(JSON.parse(readFileSync(path, "utf8")))) {
      kept.set(file, entry as Kept);
    }
    utimesSync(path, new Date(), new Date());
  } catch {
    kept.clear();
  }
  const save = () => {
    mkdirSync(directory, { recursive: true });
    for (const file of kept.keys()) if (!existsSync(file)) kept.delete(file);
    const written = `${path}.${process.pid}`;
    writeFileSync(written, JSON.stringify(Object.fromEntries(kept)));
    renameSync(written, path);
    for (const other of readdirSync(directory)) {
      if (!/^syntax-[0-9a-f]+\.json$/u.test(other)) continue;
      const used = statSync(join(directory, other)).mtimeMs;
      if (Date.now() - used > unusedFor) rmSync(join(directory, other), { force: true });
    }
  };
  return { kept, save };
}

/**
 * Keeps the findings of rules that read nothing but the linted file, and reports them again while that file's text,
 * the rule's options and this package are unchanged. Files are named from the working directory, so that another
 * checkout of the same files reuses them.
 */
export function keepingFindings(rules: Record<string, Rule>): Record<string, Rule> {
  const directory = process.env.AEXLINT_FINDINGS;
  if (!directory) return rules;
  const { kept, save } = open(directory);
  let changed = false;
  process.on("exit", () => {
    if (changed) save();
  });
  const root = process.cwd() + sep;
  let current: { file: string; entry: Kept } | undefined;
  const entryOf = (context: Context): Kept => {
    const path = context.filename;
    const file = path.startsWith(root) ? path.slice(root.length) : path;
    if (current?.file === file) return current.entry;
    const text = hashOf([context.sourceCode.text]);
    let entry = kept.get(file);
    if (entry?.text !== text) {
      entry = { text, findings: {} };
      kept.set(file, entry);
    }
    current = { file, entry };
    return entry;
  };
  const wrap = (name: string, rule: Rule): Rule => {
    const create = rule.create;
    if (!create) return rule;
    return {
      ...rule,
      create(context: Context) {
        const entry = entryOf(context);
        const key = `${name}\n${JSON.stringify(context.options)}`;
        const known = entry.findings[key];
        if (known) {
          return {
            Program() {
              for (const { message, range } of known) context.report({ message, node: { range } });
            },
          };
        }
        const found: Finding[] = [];
        const report = (diagnostic: Diagnostic) => {
          const message = messageOf(rule, diagnostic);
          found.push({ message, range: rangeOf(context, diagnostic) });
          context.report(diagnostic);
        };
        const visitor = create(
          new Proxy(Object.create(context) as Context, {
            get: (_, property) => (property === "report" ? report : Reflect.get(context, property)),
          }),
        );
        const exit = visitor["Program:exit"];
        return {
          ...visitor,
          "Program:exit"(node) {
            exit?.(node);
            entry.findings[key] = found;
            changed = true;
          },
        };
      },
    };
  };
  return Object.fromEntries(Object.entries(rules).map(([name, rule]) => [name, wrap(name, rule)]));
}
