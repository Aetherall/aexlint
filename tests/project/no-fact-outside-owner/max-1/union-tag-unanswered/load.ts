export type PalletLoad = { load: "pallet"; name: string };
export type CourierLoad = { load: "courier"; body: string };
export type Load = PalletLoad | CourierLoad;
