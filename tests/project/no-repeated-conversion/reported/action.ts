type Reminder = { type: "tandem" | "unicycle"; color: string; crewId: string };
declare function createForCrew(input: { color: string; crewId: string }): void;
declare function createPersonal(input: { color: string }): void;
export function paste(reminder: Reminder) {
  if (reminder.type === "tandem") createForCrew({ color: reminder.color, crewId: reminder.crewId });
  else createPersonal({ color: reminder.color });
}
export function assign(details: Reminder) {
  if (details.type === "tandem") createForCrew({ color: details.color, crewId: details.crewId });
  else createPersonal({ color: details.color });
}
