import type { Box, B } from "./file.js"; export const b = (box: Box<B>) => ({ x: box.get().x(), y: box.get().y() });
