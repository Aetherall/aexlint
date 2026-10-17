type Gauge = { litres: number };
type Reading = { exitStatus: string; samples: { type: string; text: string }[]; gauge: Gauge };
type Experiment = (channel: string, options: { max: number }) => Promise<Reading>;
export async function estimate(input: { channel: string; max: number }, simulate: Experiment): Promise<{ reading: string; gauge: Gauge }> {
  const recorded = await simulate(input.channel, { max: input.max });
  if (recorded.exitStatus !== "done") throw new Error("failed");
  const reading = recorded.samples.filter((b) => b.type === "text").map((b) => b.text).join("\n");
  if (!reading.trim()) throw new Error("empty");
  return { reading, gauge: recorded.gauge };
}
