import test from 'node:test';
import assert from 'node:assert/strict';
import {
  serviceIconHtml,
  avatarHtml,
  genIconButtonHtml,
  presetCardHtml,
  iconClass,
  avatarClass,
} from './icons.js';

test('serviceIconHtml shows the text badge when there is no picture', () => {
  const out = serviceIconHtml({ iconBg: 'bg-red-700', iconText: 'N' });

  assert.ok(out.includes('icon-red'), 'the colour class is applied');
  assert.ok(out.includes('>N<'), 'the badge text is shown');
  assert.ok(!out.includes('data-icon-file'), 'no picture holder is created');
});

test('serviceIconHtml points at the picture when one exists', () => {
  const out = serviceIconHtml({ iconBg: 'bg-blue-600', iconText: 'G1', iconFile: 'sub-1.png' });

  assert.ok(out.includes('data-icon-file="sub-1.png"'), 'the picture holder is created');
  assert.ok(out.includes('icon-blue'), 'the colour class is still applied');
});

test('serviceIconHtml escapes the file name', () => {
  const out = serviceIconHtml({ iconBg: 'bg-blue-600', iconText: 'X', iconFile: 'a" onload="x' });

  assert.ok(!out.includes('" onload="'), 'the attribute cannot be broken out of');
  assert.ok(out.includes('&quot;'));
});

test('serviceIconHtml escapes the badge text', () => {
  const out = serviceIconHtml({ iconBg: 'bg-blue-600', iconText: '<img src=x>' });

  assert.ok(!out.includes('<img'), 'the payload is rendered as text');
});

test('serviceIconHtml falls back to the brand colour', () => {
  assert.equal(iconClass('bg-unknown-500'), 'icon-brand');
  assert.equal(iconClass(undefined), 'icon-brand');
});

test('avatarHtml shows the initials without a picture', () => {
  const out = avatarHtml({ color: 'blue', initials: 'MC' });

  assert.ok(out.includes('avatar-blue'), 'the colour class is applied');
  assert.ok(out.includes('>MC<'), 'the initials are shown');
  assert.ok(!out.includes('data-icon-file'));
});

test('avatarHtml points at the picture when one exists', () => {
  const out = avatarHtml({ color: 'rose', initials: 'MS', iconFile: 'acc-abc.png' });

  assert.ok(out.includes('data-icon-file="acc-abc.png"'));
  assert.ok(out.includes('avatar-rose'));
});

test('avatarClass falls back to the brand colour', () => {
  assert.equal(avatarClass('chartreuse'), 'avatar-brand');
  assert.equal(avatarClass(undefined), 'avatar-brand');
});

test('genIconButtonHtml stays empty when no image service is configured', () => {
  const out = genIconButtonHtml({ id: 7, name: 'Netflix' }, false);

  assert.equal(out, '', 'no dead action is rendered');
});

test('genIconButtonHtml carries the subscription id and name', () => {
  const out = genIconButtonHtml({ id: 7, name: 'Netflix Premium' }, true);

  assert.ok(out.includes('data-gen-icon="7"'));
  assert.ok(out.includes('generateIconForSubscription(7,'));
  assert.ok(out.includes('Netflix Premium'));
});

test('genIconButtonHtml escapes a name with a quote', () => {
  const out = genIconButtonHtml({ id: 1, name: "it's" }, true);

  // The raw quote must not reach the handler; it has to be escaped instead.
  assert.ok(!out.includes("'it's'"), 'the raw quote is escaped');
  assert.ok(out.includes("\\'"), 'the quote is escaped');
});

test('presetCardHtml shows the name, price and plan', () => {
  const out = presetCardHtml({
    name: 'ChatGPT Plus',
    plan: 'Gói cá nhân GPT-4o',
    category: 'AI',
    price: 20,
    cycle: 'tháng',
  });

  assert.ok(out.includes('ChatGPT Plus'));
  assert.ok(out.includes('$20.00/tháng'));
  assert.ok(out.includes('Gói cá nhân GPT-4o'));
  assert.ok(out.includes('AI'));
});

test('presetCardHtml escapes a hostile preset name', () => {
  const out = presetCardHtml({
    name: "'); alert(1); ('",
    plan: '<script>x</script>',
    category: 'AI',
    price: 1,
    cycle: 'tháng',
  });

  assert.ok(!out.includes('alert(1); (') || out.includes("\\'"), 'the handler stays intact');
  assert.ok(!out.includes('<script>'), 'the plan is rendered as text');
});
