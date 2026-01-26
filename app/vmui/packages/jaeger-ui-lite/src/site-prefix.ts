// Per the resolution of https://github.com/jaegertracing/jaeger-ui/issues/42,
// package.json#homepage is set to "." and the document MUST have a <base>
// element to define a usable base URL.
import { getAppEnvironment } from './utils/constants';

const baseNode = document.querySelector('base');
// if (!baseNode && getAppEnvironment() !== 'test') {
//   throw new Error('<base> element not found');
// }

if (!baseNode) {
  console.warn('[jaeger-ui] <base> not found, fallback to "/"');
}

const sitePrefix = baseNode ? baseNode.href : `${globalThis.location.origin}/`;

// Configure the webpack publicPath to match the <base>:
// https://webpack.js.org/guides/public-path/#on-the-fly

window.__webpack_public_path__ = sitePrefix;

export default sitePrefix;
