// Markup builders for the icon elements.
//
// The builders are pure functions so they can be unit tested without a
// browser. They all escape the data they interpolate, because the values come
// from a file the user can edit.

import { esc, escJs } from './escape.js';

// iconClassMap translates the legacy Tailwind colour names still stored in the
// data file into the class names the stylesheet uses.
const iconClassMap = {
  'bg-emerald-600': 'icon-emerald',
  'bg-emerald-700': 'icon-emerald',
  'bg-blue-600': 'icon-blue',
  'bg-sky-600': 'icon-sky',
  'bg-red-700': 'icon-red',
  'bg-red-500': 'icon-red',
  'bg-green-600': 'icon-green',
  'bg-amber-600': 'icon-amber',
  'bg-teal-600': 'icon-teal',
  'bg-indigo-700': 'icon-indigo',
  'bg-indigo-600': 'icon-indigo',
  'bg-purple-600': 'icon-purple',
  'bg-slate-900': 'icon-slate',
  'bg-zinc-800': 'icon-zinc',
  'bg-neutral-800': 'icon-neutral',
  'bg-orange-600': 'icon-orange',
};

// avatarClassMap holds the avatar colours, keyed by the persisted `color`.
const avatarClassMap = {
  blue: 'avatar-blue',
  purple: 'avatar-purple',
  amber: 'avatar-amber',
  emerald: 'avatar-emerald',
  rose: 'avatar-rose',
};

/**
 * Returns the colour class of a subscription icon.
 * @param {string} iconBg
 * @returns {string}
 */
export function iconClass(iconBg) {
  return iconClassMap[iconBg] || 'icon-brand';
}

/**
 * Returns the colour class of a Gmail avatar.
 * @param {string} color
 * @returns {string}
 */
export function avatarClass(color) {
  return avatarClassMap[color] || 'avatar-brand';
}

/**
 * Builds the markup of a subscription icon: the AI picture when one exists,
 * the coloured text badge otherwise.
 * @param {{iconBg?: string, iconText?: string, iconFile?: string}} sub
 * @returns {string}
 */
export function serviceIconHtml(sub) {
  const cls = iconClass(sub.iconBg);
  if (sub.iconFile) {
    return `<div class="service-icon ${cls}" data-icon-file="${esc(sub.iconFile)}">${esc(
      sub.iconText || ''
    )}</div>`;
  }
  return `<div class="service-icon ${cls}">${esc(sub.iconText || '')}</div>`;
}

/**
 * Builds the markup of a Gmail account avatar.
 * @param {{color?: string, initials?: string, iconFile?: string}} acc
 * @returns {string}
 */
export function avatarHtml(acc) {
  const cls = avatarClass(acc.color);
  if (acc.iconFile) {
    return `<div class="avatar ${cls}" data-icon-file="${esc(acc.iconFile)}">${esc(
      acc.initials || ''
    )}</div>`;
  }
  return `<div class="avatar ${cls}">${esc(acc.initials || '')}</div>`;
}

/**
 * Builds the button that regenerates the icon of a subscription. It stays
 * hidden when no image service is configured.
 * @param {{id: number, name: string}} sub
 * @param {boolean} serviceReady
 * @returns {string}
 */
export function genIconButtonHtml(sub, serviceReady) {
  if (!serviceReady) return '';
  return `<button type="button" class="link-btn" data-gen-icon="${sub.id}" onclick="generateIconForSubscription(${sub.id}, '${escJs(
    sub.name
  )}')" title="Tạo lại icon bằng AI">⚡ Icon</button>`;
}

/**
 * Builds the card of one preset in the picker.
 * @param {{name: string, plan: string, category: string, price: number, cycle: string}} preset
 * @returns {string}
 */
export function presetCardHtml(preset) {
  const price = Number(preset.price) || 0;
  return `
        <button type="button" class="preset-card" onclick="applyPreset('${escJs(preset.name)}')">
          <span class="preset-name">${esc(preset.name)}</span>
          <span class="preset-meta">${esc(preset.category)} • $${price.toFixed(2)}/${esc(
            preset.cycle
          )}</span>
          <span class="preset-plan">${esc(preset.plan)}</span>
        </button>`;
}
