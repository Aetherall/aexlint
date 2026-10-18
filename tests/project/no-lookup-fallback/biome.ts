export type Biome = "tundra" | "steppe" | "taiga" | "savanna" | "mangrove";
export interface Region { biome: Biome; name: string }
export enum Season { Sprout = "sprout", Bloom = "bloom", Harvest = "harvest", Dormancy = "dormancy", Rains = "rains" }
