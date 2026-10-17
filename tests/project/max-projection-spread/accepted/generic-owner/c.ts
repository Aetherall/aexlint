import type { Box, B } from "./file.js"; export const c = (box: Box<B>) => ({ x: box.get().x(), y: box.get().y() });
