import { act, renderHook } from "@testing-library/preact";
import { useLiveTailingTraces } from "./useLiveTailingTraces";
import { vi } from "vitest";

vi.mock("../../../../../state/common/StateContext", () => ({
  useAppState: () => ({ serverUrl: "http://localhost:8080" }),
}));

vi.mock("../../../../../hooks/useTenant", () => ({
  useTenant: () => ({}),
}));

// Mock dependencies
const mockFetch = vi.fn();
global.fetch = mockFetch;

const createMockStreamResponse = (traces: string[], sendCount: number = 1) => ({
  ok: true,
  body: new ReadableStream({
    async start(controller) {
      for (let i = 0; i < sendCount; i++) {
        traces.forEach((trace) => {
          controller.enqueue(new TextEncoder().encode(trace + "\n"));
        });
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }

      controller.close();
    },
  }),
  text: async () => traces.join("\n"),
});

describe("useLiveTailingTraces", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.clearAllMocks();
  });

  it("should start live tailing and process traces", async () => {
    const query = "*";
    const limit = 10;
    const { result } = renderHook(() => useLiveTailingTraces(query, limit));

    mockFetch.mockResolvedValue(createMockStreamResponse(["{\"traces\":\"test trace\"}"]));

    await act(async () => {
      const started = await result.current.startLiveTailing();
      expect(started).toBe(true);
    });

    expect(mockFetch).toHaveBeenCalledTimes(1);
    expect(mockFetch).toHaveBeenCalledWith(
      "http://localhost:8080/select/tracesql/tail",
      expect.objectContaining({
        method: "POST",
        body: new URLSearchParams({
          query: query.trim(),
        }),
      })
    );
  });

  it("should pause and resume live tailing", () => {
    const query = "*";
    const limit = 10;
    const { result } = renderHook(() => useLiveTailingTraces(query, limit));

    act(() => {
      result.current.pauseLiveTailing();
    });

    expect(result.current.isPaused).toBe(true);

    act(() => {
      result.current.resumeLiveTailing();
    });

    expect(result.current.isPaused).toBe(false);
  });

  it("should stop live tailing", async () => {
    const query = "*";
    const limit = 10;
    const { result } = renderHook(() => useLiveTailingTraces(query, limit));

    act(() => {
      result.current.stopLiveTailing();
    });

    expect(result.current.traces).toHaveLength(0);
  });

  it("should clear traces", () => {
    const query = "*";
    const limit = 10;
    const { result } = renderHook(() => useLiveTailingTraces(query, limit));

    act(() => {
      result.current.clearTraces();
    });

    expect(result.current.traces).toEqual([]);
  });

  it("should handle errors during live tailing", async () => {
    const query = "*";
    const limit = 10;
    const { result } = renderHook(() => useLiveTailingTraces(query, limit));

    mockFetch.mockRejectedValue(new Error("Network error"));

    await act(async () => {
      const started = await result.current.startLiveTailing();
      expect(started).toBe(false);
    });

    expect(result.current.error).toBe("Error: Network error");
    expect(result.current.traces).toHaveLength(0);
  });

  it("should process high load of traces incoming at 100k traces per second", async () => {
    const query = "*";
    const limit = 1000;
    const traceCount = 10000; // High trace rate
    const traces = Array.from({ length: traceCount }, (_, i) => `{"trace": "trace message ${i}"}`);
    const { result } = renderHook(() => useLiveTailingTraces(query, limit));

    mockFetch.mockResolvedValue(createMockStreamResponse(traces, 7));

    await act(async () => {
      const started = await result.current.startLiveTailing();
      expect(started).toBe(true);
    });

    // Wait for traces to process
    await new Promise((resolve) => setTimeout(resolve, 7000));

    // Verify traces are limited and processed correctly
    expect(result.current.traces.length).toBeLessThanOrEqual(limit);
    // After setting flag isLimitedTracesPerUpdate when more than 200 traces received 5 times in a row,
    // we take only the last 200 traces, so we get 800 older traces (9200 - 9999) and 200 new traces (9800-9999)
    expect(result.current.traces[0].trace).toStrictEqual("trace message 9200");
    expect(result.current.traces[799].trace).toStrictEqual("trace message 9999");
    expect(result.current.isLimitedTracesPerUpdate).toBeTruthy();
  }, { timeout: 9000 });
});
