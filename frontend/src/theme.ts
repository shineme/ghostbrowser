// 外观主题: 'dark' (default) | 'gray' | 'light'. Applied as <html data-theme>,
// cached in localStorage so the first paint already has the right colours.
export type ThemeID = 'dark' | 'gray' | 'light';
const KEY = 'gb.theme';
export const normTheme = (t?: string): ThemeID => (t === 'light' || t === 'gray' ? t : 'dark');
export function applyTheme(t?: string, animate = false) {
  const id = normTheme(t);
  const el = document.documentElement;
  if ((el.dataset.theme || 'dark') === id) { try { localStorage.setItem(KEY, id); } catch {} return; }
  if (animate) { el.classList.add('theme-anim'); setTimeout(() => el.classList.remove('theme-anim'), 420); }
  if (id === 'dark') delete el.dataset.theme; else el.dataset.theme = id;
  try { localStorage.setItem(KEY, id); } catch {}
  window.dispatchEvent(new CustomEvent('gb:theme', { detail: id }));
}
export function bootTheme() { try { applyTheme(localStorage.getItem(KEY) || 'dark'); } catch {} }
// thinking-orbs / border-beam only know light and dark
export const fxTheme = (): 'dark' | 'light' => (document.documentElement.dataset.theme === 'light' ? 'light' : 'dark');
