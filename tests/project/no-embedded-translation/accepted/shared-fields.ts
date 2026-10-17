type Seat = { section: "strings"; id: string; chair: number } | { section: "winds"; id: string; row: number };

declare function reserve(id: string): void;

export function book(seat: Seat): void {
  let target = "";
  if (seat.section === "strings") target = seat.id;
  reserve(target);
}
