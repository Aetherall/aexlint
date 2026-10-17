function callbackLocals() {
items.map((item) => {
  const normalized = normalize(item);
  return encode(normalized, options);
});
}
function inlineHandler() {
on("click", { once: true, handler: () => close(dialog) });
}
export const styles = StyleSheet.create({
  box: { color: Color.primary, margin: Spacing.small, radius: Radius.medium },
});
function hookDependencies() {
const visible = useMemo(() => filter(items), [items, query, sortOrder, filters, locale]);
}
