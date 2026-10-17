declare const value: { id: string } | undefined; const jsx = <div>{value !== undefined ? <span /> : null}</div>;
export {};
