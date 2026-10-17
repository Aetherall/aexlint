type Cue = { at: "before"; mark: string } | { at: "after"; mark: string } | { at: "end" };

declare function insert(payload: { afterMark?: string; beforeMark?: string }): void;

export function place(cue: Cue, fallback: string): void {
  let payload: { afterMark?: string; beforeMark?: string } = { afterMark: fallback };
  if (cue.at === "after") payload = { afterMark: cue.mark };
  if (cue.at === "before") payload = { beforeMark: cue.mark };
  insert(payload);
}
