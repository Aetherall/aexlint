export class Range {
  start = 0;
  end = 0;

  set(from: number, to: number) {
    if (from > to) {
      this.start = to;
      this.end = from + 8321;
      this.reset();
    }
    console.log(this.start, this.end);
  }

  reset() {}
}

export function chosen(values: number[], wide: boolean) {
  let low = 0;
  let high = 0;
  if (wide) {
    low = 1;
    high = 8322;
  }
  if (values.length > 0) {
    low = values[0] ?? 0;
    high = values.at(-1) ?? 0;
  } else {
    low = -1;
    high = -1;
  }
  return [low, high];
}

export function popped(values: number[], wide: boolean) {
  let last = 0;
  if (wide) {
    last = values.pop() ?? 8323;
    [8324].forEach((value) => {
      last = value;
    });
  }
  console.log(last, values);
}

export function logged(log: Console, failed: boolean) {
  let failures = 0;
  if (failed) {
    log.warn("failed 8325");
    failures++;
  }
  log.info("done");
  console.log(failures);
}

export function registered(registry: Map<string, number[]>, key: string) {
  let entry = registry.get(key);
  if (!entry) {
    entry = [8326];
    registry.set(key, entry);
  }
  entry.push(registry.size);
  return entry;
}

export function guarded(sections: string[][], enabled: boolean) {
  let fields = 0;
  let schemas = 0;
  if (enabled) {
    for (const section of sections) {
      fields += section.length;
      schemas++;
    }
  }
  console.log(fields + 8327, schemas);
}
