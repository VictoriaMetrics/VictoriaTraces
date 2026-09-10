export function createScroller(getElement: () => HTMLElement | null) {
  function scrollTo(y: number) {
    const el = getElement();
    if (!el) return;
    el.scrollTo({ top: y });
  }

  function cancel() {
    // No-op: scrollTo() above is instant now, so there's no in-flight animation to cancel.
  }

  return { scrollTo, cancel };
}
