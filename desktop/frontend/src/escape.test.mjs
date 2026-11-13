import test from 'node:test';
import assert from 'node:assert/strict';
import { esc, escJs } from './escape.js';

test('esc neutralises an img onerror payload', () => {
  const out = esc('<img src=x onerror="window.x=1">');
  assert.ok(!out.includes('<'), 'must not contain a raw angle bracket');
  assert.ok(!out.includes('>'), 'must not contain a raw angle bracket');
  assert.ok(out.includes('&lt;img'), 'the payload should be rendered as text');
});

test('esc neutralises a script tag payload', () => {
  const out = esc('</script><script>alert(1)</script>');
  assert.ok(!out.includes('</script>'));
  assert.ok(!out.includes('<script>'));
});

test('esc escapes double quotes so attributes cannot be broken out of', () => {
  const out = esc('a" onmouseover="alert(1)');
  assert.ok(!out.includes('"'), 'no raw double quote may survive');
  assert.ok(out.includes('&quot;'));
});

test('esc escapes single quotes', () => {
  const out = esc("it's");
  assert.ok(!out.includes("'"));
  assert.ok(out.includes('&#39;'));
});

test('esc escapes ampersands before the other entities', () => {
  assert.equal(esc('a & b'), 'a &amp; b');
  assert.equal(esc('&lt;'), '&amp;lt;');
});

test('esc handles empty and missing values', () => {
  assert.equal(esc(''), '');
  assert.equal(esc(null), '');
  assert.equal(esc(undefined), '');
  assert.equal(esc(0), '0');
});

test('esc keeps ordinary Vietnamese text intact', () => {
  assert.equal(esc('Gói cá nhân — Quản lý'), 'Gói cá nhân — Quản lý');
});

test('escJs keeps a single-quoted handler string intact', () => {
  const out = escJs("O'Brien");
  assert.equal(out, "O\\'Brien");
});

test('escJs prevents a value from closing the script tag', () => {
  const out = escJs('</script><img src=x>');
  assert.ok(!out.includes('<'));
  assert.ok(!out.includes('>'));
});

test('escJs escapes backslashes', () => {
  assert.equal(escJs('a\\b'), 'a\\\\b');
});

test('escJs stops a value from breaking out of the handler attribute', () => {
  // The handler lives inside a double-quoted HTML attribute, so a raw double
  // quote would close it and let the value inject a second attribute.
  const out = escJs('x" onmouseover="window.pwned=1" y="');
  assert.ok(!out.includes('"'), 'no raw double quote may survive');
  assert.ok(out.includes('&quot;'));
});

test('escJs escapes ampersands before the other entities', () => {
  // &quot; must not be double-escaped into &amp;quot;.
  assert.equal(escJs('a"b'), 'a&quot;b');
  assert.equal(escJs('a&b'), 'a&amp;b');
});
