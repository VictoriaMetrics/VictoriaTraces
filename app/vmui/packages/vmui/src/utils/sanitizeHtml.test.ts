import { expect } from "vitest";
import { sanitizeHtml } from "./sanitizeHtml";

describe("sanitizeHtml", () => {
  it("returns an empty string for empty input", () => {
    expect(sanitizeHtml("")).toBe("");
  });

  it("keeps plain text untouched", () => {
    expect(sanitizeHtml("hello world")).toBe("hello world");
  });

  it("keeps allowed tags with allowed attributes", () => {
    const input = "<a href=\"https://example.com\"><code>copy</code></a> copies fields.";
    expect(sanitizeHtml(input)).toBe(
      "<a href=\"https://example.com\"><code>copy</code></a> copies fields."
    );
  });

  it("preserves target=\"_blank\" and forces safe rel", () => {
    const input = "<a href=\"https://example.com\" target=\"_blank\" rel=\"external\">link</a>";
    const output = sanitizeHtml(input);
    expect(output).toContain("target=\"_blank\"");
    expect(output).toContain("href=\"https://example.com\"");
    expect(output).toMatch(/rel="[^"]*noopener[^"]*"/);
    expect(output).toMatch(/rel="[^"]*noreferrer[^"]*"/);
    expect(output).toMatch(/rel="[^"]*external[^"]*"/);
  });

  it("strips script tags entirely, including their content", () => {
    expect(sanitizeHtml("<script>alert(1)</script>hello")).toBe("hello");
  });

  it("drops disallowed tags but keeps their inner text", () => {
    expect(sanitizeHtml("<img src=\"x\" onerror=\"alert(1)\">hello")).toBe("hello");
    expect(sanitizeHtml("<div onclick=\"alert(1)\">hello</div>")).toBe("hello");
  });

  it("drops event handler attributes on allowed tags", () => {
    const output = sanitizeHtml("<a href=\"https://example.com\" onclick=\"alert(1)\">click</a>");
    expect(output).not.toContain("onclick");
    expect(output).toContain("click");
  });

  it("drops javascript: URLs", () => {
    const output = sanitizeHtml("<a href=\"javascript:alert(1)\">click</a>");
    expect(output).not.toContain("javascript:");
  });

  it("drops data: URLs", () => {
    const output = sanitizeHtml("<a href=\"data:text/html,<script>alert(1)</script>\">click</a>");
    expect(output).not.toContain("data:");
  });

  it("allows relative and anchor hrefs", () => {
    expect(sanitizeHtml("<a href=\"#anchor\">a</a>")).toContain("href=\"#anchor\"");
    expect(sanitizeHtml("<a href=\"/docs\">a</a>")).toContain("href=\"/docs\"");
  });

  it("strips unknown attributes such as style", () => {
    const output = sanitizeHtml("<a href=\"https://example.com\" style=\"color:red\">a</a>");
    expect(output).not.toContain("style");
  });
});
