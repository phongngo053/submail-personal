// HTML escaping helpers.
//
// Every value that can originate from user input (a Gmail address, a service
// name, a payment note) must pass through esc() before it is interpolated into
// an HTML template. Without this, a JSON backup containing markup would run
// script when rendered.

const ESCAPES = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
};

/**
 * Escapes the five characters that can break out of an HTML text node or a
 * quoted attribute value.
 * @param {unknown} value
 * @returns {string}
 */
export function esc(value) {
  if (value === null || value === undefined) return '';
  return String(value).replace(/[&<>"']/g, (ch) => ESCAPES[ch]);
}

/**
 * Escapes a value for use inside a single-quoted JavaScript string inside an
 * inline event handler. Escaping the quote and the backslash keeps the string
 * intact; escaping angle brackets stops a value from closing the script tag.
 *
 * Double quotes and ampersands are escaped too: the handler sits inside a
 * double-quoted HTML attribute, so a raw `"` would end the attribute and let a
 * crafted value inject a second attribute (for example `onmouseover=`).
 * @param {unknown} value
 * @returns {string}
 */
export function escJs(value) {
  if (value === null || value === undefined) return '';
  return String(value)
    .replace(/\\/g, '\\\\')
    .replace(/'/g, "\\'")
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '\\u003c')
    .replace(/>/g, '\\u003e');
}
