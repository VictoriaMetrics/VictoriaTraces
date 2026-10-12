import { describe, it, expect } from "vitest";
import { buildExcludeClause } from "./index";

describe("TraceExplorer/utils", () => {
  describe("buildExcludeClause", () => {
    it("negates an exact match, quoting field names that contain LogsQL separators", () => {
      expect(buildExcludeClause("resource_attr:service.name", "frontend"))
        .toBe("-\"resource_attr:service.name\":\"frontend\"");
    });

    it("leaves plain field names unquoted and escapes quotes inside the value", () => {
      expect(buildExcludeClause("name", "GET \"x\"")).toBe("-name:\"GET \\\"x\\\"\"");
    });
  });
});
