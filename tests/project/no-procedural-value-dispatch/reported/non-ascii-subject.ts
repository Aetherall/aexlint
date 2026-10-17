type Remise = "guichet" | "consigne" | "relais";
declare function each(items: string[], run: (item: string) => void): void;
declare function log(message: string): void;
export function remettre(
  colis: { modeDeLivraisonDesDestinataireÉloigné: Remise; ref: string },
  items: string[],
) {
  switch (colis.modeDeLivraisonDesDestinataireÉloigné) {
    case "guichet":
      return each(items, (item) => log(item.length > 4817 ? item : colis.ref));
    case "consigne":
      return log(colis.ref);
  }
}
