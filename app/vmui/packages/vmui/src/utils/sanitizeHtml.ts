const ALLOWED_TAGS = new Set(["a", "code", "b", "i", "em", "strong", "br", "span"]);

const DROP_SUBTREE_TAGS = new Set(["script", "style", "iframe", "object", "embed", "template", "noscript", "svg", "math"]);

const ALLOWED_ATTRIBUTES: Record<string, Set<string>> = {
  a: new Set(["href", "target", "rel", "class"]),
  span: new Set(["class"]),
};

const SAFE_HREF_PATTERN = /^(?:https?:|mailto:|\/|#)/i;
const SAFE_REL_TOKENS = new Set(["noopener", "noreferrer", "external", "help", "nofollow"]);
const SAFE_CLASS_TOKEN = /^[a-zA-Z0-9_-]+$/;

const sanitizeAttributeValue = (name: string, value: string): string | null => {
  switch (name) {
    case "href": {
      const trimmed = value.trim();
      return SAFE_HREF_PATTERN.test(trimmed) ? trimmed : null;
    }
    case "target":
      return value === "_blank" ? "_blank" : null;
    case "rel": {
      const tokens = value.split(/\s+/).filter(t => SAFE_REL_TOKENS.has(t));
      return tokens.length ? tokens.join(" ") : null;
    }
    case "class": {
      const tokens = value.split(/\s+/).filter(t => SAFE_CLASS_TOKEN.test(t));
      return tokens.length ? tokens.join(" ") : null;
    }
    default:
      return null;
  }
};

const sanitizeNode = (node: ChildNode, target: Node) => {
  if (node.nodeType === Node.TEXT_NODE) {
    target.appendChild(node.cloneNode());
    return;
  }

  if (node.nodeType !== Node.ELEMENT_NODE) {
    // drop comments, processing instructions, etc.
    return;
  }

  const el = node as Element;
  const tagName = el.tagName.toLowerCase();

  if (DROP_SUBTREE_TAGS.has(tagName)) {
    return;
  }

  if (!ALLOWED_TAGS.has(tagName)) {
    // Unwrap disallowed elements: keep sanitizing their children as if
    // they were inlined into the parent, but drop the element itself.
    Array.from(el.childNodes).forEach(child => sanitizeNode(child, target));
    return;
  }

  const clean = document.createElement(tagName);
  const allowedAttrs = ALLOWED_ATTRIBUTES[tagName];
  if (allowedAttrs) {
    Array.from(el.attributes).forEach(attr => {
      if (!allowedAttrs.has(attr.name)) return;
      const value = sanitizeAttributeValue(attr.name, attr.value);
      if (value !== null) clean.setAttribute(attr.name, value);
    });
  }

  if (tagName === "a" && clean.getAttribute("target") === "_blank") {
    const relTokens = new Set((clean.getAttribute("rel") || "").split(/\s+/).filter(Boolean));
    relTokens.add("noopener");
    relTokens.add("noreferrer");
    clean.setAttribute("rel", Array.from(relTokens).join(" "));
  }

  target.appendChild(clean);
  Array.from(el.childNodes).forEach(child => sanitizeNode(child, clean));
};

/**
 * Sanitizes an HTML string against a narrow allowlist of tags/attributes
 * before it is ever assigned to `dangerouslySetInnerHTML`. Anything not
 * explicitly allowed (scripts, event handlers, `javascript:`/`data:` URLs,
 * unknown tags, etc.) is stripped.
 */
export const sanitizeHtml = (html: string): string => {
  if (!html) return "";

  const doc = new DOMParser().parseFromString(html, "text/html");
  const container = document.createElement("div");
  Array.from(doc.body.childNodes).forEach(child => sanitizeNode(child, container));
  return container.innerHTML;
};
