export interface Reel {
  readonly film: string;
}

export function frames(film: string): number {
  return film.length * 24;
}

export function first(reel: Reel): number {
  return frames(reel.film);
}

export function second(reel: Reel): number {
  return frames(reel.film);
}

export function third(reel: Reel): number {
  return frames(reel.film);
}
