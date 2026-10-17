type Tool = { name: string }; type Payload = Record<string, unknown>;
export class Policy {
  bind(contract: Tool, tool: Tool): Tool {
    this.assertCompatible(contract, tool);
    return { ...tool };
  }
  private settings(input: Record<string, unknown>): Payload {
    if (Object.keys(input).length > 3) throw new Error("x");
    return { ...input };
  }
  assertCompatible(a: Tool, b: Tool): void {
    if (a.name !== b.name) throw new Error("mismatch");
  }
}
