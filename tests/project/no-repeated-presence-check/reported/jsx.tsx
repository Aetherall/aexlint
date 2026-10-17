import { values, label, create, logged, nonEmpty, pair, only, upper, withDefault, many, show, external, wrap, now, stamped, type Manifest } from "../items.js";
declare function List(props: { codes: string[] }): null;
export const View = ({ property }: { property: Manifest | undefined }) => <List codes={property ? values(property) : []} />;
