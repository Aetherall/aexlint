import { readFileSync } from "node:fs";
import { relative } from "node:path";
import type { Context, Diagnostic, Plugin, Rule } from "@oxlint/plugins";

export interface BaselineEntry {
  file: string;
  code: string;
  message: string;
}

const header = "# Diagnostics recorded by aexlint check --write-baseline\n";

export function parseBaseline(text: string): BaselineEntry[] {
  const entries: BaselineEntry[] = [];
  for (const line of text.split(/\r?\n/)) {
    if (!line.trim() || line.startsWith("#")) continue;
    const [file, code, message] = line.split("\t");
    if (!file || !code || message === undefined) {
      throw new Error(`Malformed baseline line: ${line}`);
    }
    entries.push({ file, code, message: JSON.parse(message) });
  }
  return entries;
}

export function formatBaseline(entries: BaselineEntry[]): string {
  const lines = entries.map(({ file, code, message }) =>
    [file, code, JSON.stringify(message)].join("\t"),
  );
  return header + lines.toSorted().join("\n") + (lines.length ? "\n" : "");
}

export function projectPath(root: string, file: string): string {
  return relative(root, file).replaceAll("\\", "/");
}

export function findingOf(message: string): string {
  return message.split("\nhelp: ")[0] ?? message;
}

function messageOf(rule: Rule, diagnostic: Diagnostic): string {
  const template = diagnostic.messageId
    ? (rule.meta?.messages?.[diagnostic.messageId] ?? "")
    : (diagnostic.message ?? "");
  const data = diagnostic.data;
  if (!data) return template;
  return template.replace(/\{\{([^{}]+)\}\}/gu, (match, key: string) => {
    const value = data[key.trim()];
    return value === undefined ? match : String(value);
  });
}

function loadBudgets(path: string | undefined): Map<string, Map<string, number>> {
  const budgets = new Map<string, Map<string, number>>();
  if (!path) return budgets;
  for (const { file, code, message } of parseBaseline(readFileSync(path, "utf8"))) {
    const key = `${file}\n${code}`;
    const messages = budgets.get(key) ?? new Map<string, number>();
    messages.set(message, (messages.get(message) ?? 0) + 1);
    budgets.set(key, messages);
  }
  return budgets;
}

export function withBaseline(plugin: Plugin): Plugin {
  const budgets = loadBudgets(process.env.AEXLINT_BASELINE);
  const root = process.cwd();
  const wrap = (name: string, rule: Rule): Rule => {
    const create = rule.create;
    if (!create) return rule;
    const code = `${plugin.meta?.name}(${name})`;
    return {
      ...rule,
      create(context: Context) {
        const budget = budgets.get(`${projectPath(root, context.filename)}\n${code}`);
        if (!budget) return create(context);
        const remaining = new Map(budget);
        const report = (diagnostic: Diagnostic) => {
          const message = findingOf(messageOf(rule, diagnostic));
          const count = remaining.get(message) ?? 0;
          if (count > 0) remaining.set(message, count - 1);
          else context.report(diagnostic);
        };
        return create(
          new Proxy(Object.create(context) as Context, {
            get: (_, key) => (key === "report" ? report : Reflect.get(context, key)),
          }),
        );
      },
    };
  };
  const rules = Object.entries(plugin.rules).map(([name, rule]) => [name, wrap(name, rule)]);
  return { ...plugin, rules: Object.fromEntries(rules) };
}
