import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/preact";
import { useChartRun } from "./useChartRun";
import { TraceChartRun } from "./types";

interface Props {
  isActive: boolean;
  run: TraceChartRun | null;
}

const makeRun = (id: number, query = "*"): TraceChartRun => ({
  id, query, startNs: 1_000n, endNs: 2_000n, extraClauses: [],
});

const setup = (initial: Props) => {
  const load = vi.fn();
  const hook = renderHook((props: Props) => useChartRun(props.isActive, props.run, load), { initialProps: initial });
  return { load, ...hook };
};

describe("TraceExplorer/charts/useChartRun", () => {
  it("does nothing before the first run is committed", () => {
    const { load } = setup({ isActive: true, run: null });
    expect(load).not.toHaveBeenCalled();
  });

  it("loads an active chart once per run and passes the run snapshot", () => {
    const run = makeRun(1, "a:b");
    const { load, rerender } = setup({ isActive: true, run });
    expect(load).toHaveBeenCalledTimes(1);
    expect(load).toHaveBeenCalledWith(run);

    rerender({ isActive: true, run: { ...run } });
    expect(load).toHaveBeenCalledTimes(1);
  });

  it("skips runs while inactive and catches up with the latest one when activated", () => {
    const { load, rerender } = setup({ isActive: false, run: makeRun(1) });
    expect(load).not.toHaveBeenCalled();

    rerender({ isActive: false, run: makeRun(2) });
    expect(load).not.toHaveBeenCalled();

    const latest = makeRun(3);
    rerender({ isActive: true, run: latest });
    expect(load).toHaveBeenCalledTimes(1);
    expect(load).toHaveBeenCalledWith(latest);
  });

  it("re-loads an active chart for every new run", () => {
    const { load, rerender } = setup({ isActive: true, run: makeRun(1) });
    rerender({ isActive: true, run: makeRun(2) });
    expect(load).toHaveBeenCalledTimes(2);
  });
});
