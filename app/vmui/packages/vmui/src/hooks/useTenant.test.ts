import { describe, expect, it } from "vitest";
import { buildTraceLocation } from "./useTenant";

describe("buildTraceLocation", () => {
  it("keeps the tenant and trace id, drops other params, and adds span_id only when given", () => {
    const source = new URLSearchParams("g0.range_input=5m&accountID=193&projectID=20");

    expect(buildTraceLocation(source, "abc")).toEqual({
      pathname: "/trace",
      search: "accountID=193&projectID=20&trace_id=abc",
    });
    expect(buildTraceLocation(source, "abc", "span1").search).toBe(
      "accountID=193&projectID=20&trace_id=abc&span_id=span1"
    );
  });
});
