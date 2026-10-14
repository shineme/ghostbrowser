package main

import "fmt"

// recWorld is the isolated world the recorder's capture script lives in.
// Page scripts share the DOM with it but never see its globals.
const recWorld = "__gb_rec"

// recCaptureScript is injected into every page of the recorded browser (main
// frame only) in recWorld. It queues user actions in window.__gbRec.q; Go
// drains the queue with Runtime.evaluate (no Runtime.enable, no bindings).
// overlay/scroll are the initial options; drain() keeps them in sync.
func recCaptureScript(overlay, scroll bool) string {
	return fmt.Sprintf(`(() => {
if (window.top !== window) return;
// The initial about:blank of a new tab hands its window to the first real
// page, listeners and all: install again per document and silence the old.
const old = window.__gbRec;
if (old && old.doc === document && !old.dead) return;
if (old) { old.dead = true; try { old.unmount(); } catch (e) {} }
const R = window.__gbRec = { doc: document, q: [], n: 0, paused: false, overlay: %t, scroll: %t, mode: '', dead: false, lastEnter: 0, lastLabel: null, lastUser: 0 };
const send = (o) => { if (R.dead || R.q.length >= 3000) return; o.ts = Date.now(); R.q.push(o); };
const HTML = 'http://www.w3.org/1999/xhtml';
let host = null, root = null, ui = {};
const inOverlay = (e) => !!host && (e.target === host || (e.composedPath && e.composedPath().includes(host)));
// ---------- selectors ----------
const esc = (s) => (window.CSS && CSS.escape) ? CSS.escape(s) : String(s).replace(/[^\w-]/g, '\\$&');
const cstr = (s) => String(s).replace(/\\/g, '\\\\').replace(/"/g, '\\"');
const bad = (v) => !v || v.length > 48 || /\d{3,}|^\d|[:{}]|[a-f0-9]{8,}|^(css|sc|jsx|ember|react|mui|ng|v)-[\w-]*\d/i.test(v);
const all = (s) => { try { return document.querySelectorAll(s); } catch (e) { return []; } };
const uniq = (s, el) => { const n = all(s); return n.length === 1 && n[0] === el; };
const byText = (t) => { let best = null, bl = Infinity; for (const e of document.body ? document.body.querySelectorAll('*') : []) { if (host && (e === host)) continue; const s = (e.innerText || '').trim(); if (s && s.includes(t) && (s.length < bl || (s.length === bl && best.contains(e)))) { best = e; bl = s.length; } } return best; };
const CLICKABLE = 'a,button,label,summary,option,[role=button],[role=link],[role=tab],[role=menuitem],[role=option],[role=checkbox],[role=radio],input[type=submit],input[type=button]';
const cssPath = (el) => {
  const parts = [];
  for (let e = el; e && e.nodeType === 1 && e !== document.documentElement; e = e.parentElement) {
    if (e !== el && e.id && !bad(e.id) && uniq('#' + esc(e.id), e)) { parts.unshift('#' + esc(e.id)); return parts.join(' > '); }
    let p = e.tagName.toLowerCase();
    const cls = Array.from(e.classList).filter(c => !bad(c) && c.length <= 32).slice(0, 2);
    if (cls.length) p += '.' + cls.map(esc).join('.');
    const par = e.parentElement;
    if (par) {
      const sameTag = Array.from(par.children).filter(c => c.tagName === e.tagName);
      const sameAll = sameTag.filter(c => cls.every(k => c.classList.contains(k)));
      if (sameAll.length > 1) p += ':nth-of-type(' + (sameTag.indexOf(e) + 1) + ')';
    }
    parts.unshift(p);
    if (uniq(parts.join(' > '), el)) return parts.join(' > ');
  }
  return parts.join(' > ');
};
const xpath = (el) => {
  const parts = [];
  for (let e = el; e && e.nodeType === 1; e = e.parentElement) {
    if (e.id && !bad(e.id) && !e.id.includes('"') && uniq('#' + esc(e.id), e)) { parts.unshift('//*[@id="' + e.id + '"]'); return parts.join('/'); }
    const tag = e.tagName.toLowerCase();
    const sib = e.parentElement ? Array.from(e.parentElement.children).filter(c => c.tagName === e.tagName) : [e];
    parts.unshift(sib.length > 1 ? tag + '[' + (sib.indexOf(e) + 1) + ']' : tag);
  }
  return '/' + parts.join('/');
};
// strong: a stable attribute that needs no fallback.
const strong = (el) => {
  const tag = el.tagName.toLowerCase();
  if (el.id && !bad(el.id) && uniq('#' + esc(el.id), el)) return '#' + esc(el.id);
  for (const a of ['data-testid', 'data-test-id', 'data-test', 'data-qa', 'data-cy']) {
    const v = el.getAttribute(a); if (v) { const s = '[' + a + '="' + cstr(v) + '"]'; if (uniq(s, el)) return s; }
  }
  const name = el.getAttribute('name');
  if (name && !bad(name)) {
    let s = tag + '[name="' + cstr(name) + '"]'; if (uniq(s, el)) return s;
    if (el.type === 'radio' || el.type === 'checkbox') { s += '[value="' + cstr(el.value) + '"]'; if (uniq(s, el)) return s; }
  }
  const al = el.getAttribute('aria-label');
  if (al && al.length <= 60) { const s = tag + '[aria-label="' + cstr(al) + '"]'; if (uniq(s, el)) return s; }
  for (const at of Array.from(el.attributes)) {
    if (at.name.startsWith('data-') && at.value && at.value.length <= 40 && !bad(at.value)) { const s = tag + '[' + at.name + '="' + cstr(at.value) + '"]'; if (uniq(s, el)) return s; }
  }
  if ((tag === 'input' || tag === 'textarea') && el.placeholder) { const s = tag + '[placeholder="' + cstr(el.placeholder) + '"]'; if (uniq(s, el)) return s; }
  return '';
};
const selCache = new WeakMap();
const S = (el) => {
  const c = selCache.get(el); if (c && Date.now() - c.t < 1500) return c.s;
  let s = strong(el);
  if (!s) {
    let primary = '';
    if (el.matches(CLICKABLE)) {
      const t = (el.innerText || '').trim();
      if (t && t.length <= 30 && !/[\n|]/.test(t)) { const r = byText(t); if (r && (r === el || el.contains(r))) primary = 'text:' + t; }
    }
    if (!primary) { const p = cssPath(el); if (uniq(p, el)) primary = p; }
    s = primary ? primary + ' || xpath:' + xpath(el) : 'xpath:' + xpath(el);
  }
  selCache.set(el, { s, t: Date.now() });
  return s;
};
R.sel = S;
// ---------- what was on the page at load ----------
const born = new WeakSet();
const snap = () => { for (const e of document.querySelectorAll('*')) born.add(e); };
if (document.readyState === 'loading') { document.addEventListener('DOMContentLoaded', snap, { once: true }); addEventListener('load', () => setTimeout(snap, 0), { once: true }); } else snap();
// ---------- targets ----------
const htmlEl = (t) => { let el = t instanceof Element ? t : (t && t.parentElement) || null; while (el && el.namespaceURI !== HTML) el = el.parentElement; return el; };
const norm = (t) => { const el = htmlEl(t); if (!el) return null; return el.closest('a[href],button,input,select,textarea,label,summary,[role=button],[role=link],[role=checkbox],[role=radio],[role=tab],[role=menuitem],[role=option],[onclick]') || el; };
const textField = (el) => (el.tagName === 'INPUT' && !/^(checkbox|radio|file|submit|button|reset|image|range|color|hidden)$/i.test(el.type)) || el.tagName === 'TEXTAREA' || el.isContentEditable;
const skipClick = (el) => textField(el) || el.tagName === 'SELECT' || el.tagName === 'OPTION' || (el.tagName === 'INPUT' && el.type === 'file');
const user = () => { R.lastUser = Date.now(); };
// ---------- capture ----------
addEventListener('pointerdown', (e) => {
  if (!e.isTrusted || inOverlay(e) || e.button !== 0) return;
  user();
  if (R.mode) { e.preventDefault(); e.stopImmediatePropagation(); return; }
  const el = norm(e.target); if (!el || skipClick(el)) return;
  if (el.tagName === 'LABEL' && el.control && skipClick(el.control)) return;
  send({ t: 'down', sel: S(el), dyn: !born.has(el) });
}, true);
for (const ty of ['mousedown', 'mouseup', 'pointerup']) addEventListener(ty, (e) => { if (R.mode && e.isTrusted && !inOverlay(e)) { e.preventDefault(); e.stopImmediatePropagation(); } }, true);
addEventListener('click', (e) => {
  if (!e.isTrusted || inOverlay(e)) return;
  if (R.mode) {
    e.preventDefault(); e.stopImmediatePropagation();
    const el = htmlEl(e.target); if (!el) return;
    const mode = R.mode; R.mode = ''; paint();
    send({ t: mode, sel: S(el), text: (el.innerText || el.value || '').trim().slice(0, 60) });
    flash(el, '#a78bfa'); toast(mode === 'extract' ? '已添加「提取文本」' : '已添加「等待元素」');
    return;
  }
  if (e.detail >= 2) return;
  const el = norm(e.target); if (!el) return;
  if (e.detail === 0 && Date.now() - R.lastEnter < 150) return; // implicit submit after Enter
  if (R.lastLabel && R.lastLabel.ctl === el && Date.now() - R.lastLabel.t < 300) return; // label forwarded the click
  if (el.tagName === 'LABEL' && el.control) { R.lastLabel = { ctl: el.control, t: Date.now() }; if (skipClick(el.control)) return; }
  if (skipClick(el)) return;
  send({ t: 'click', sel: S(el), dyn: !born.has(el) });
  flash(el, '#f43f5e');
}, true);
addEventListener('dblclick', (e) => {
  if (!e.isTrusted || inOverlay(e) || R.mode) return;
  const el = norm(e.target); if (!el || skipClick(el)) return;
  send({ t: 'dbl', sel: S(el) });
}, true);
addEventListener('input', (e) => {
  if (!e.isTrusted || inOverlay(e)) return;
  let el = e.target; if (!(el instanceof Element) || el.tagName === 'SELECT') return;
  let v;
  if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') { if (!textField(el)) return; v = el.value; }
  else { el = el.closest('[contenteditable]:not([contenteditable=false])') || el; if (!el.isContentEditable) return; while (el.parentElement && el.parentElement.isContentEditable) el = el.parentElement; v = el.innerText; }
  user();
  const c = selCache.get(el); if (c) c.t = Date.now(); // keep one selector while typing
  send({ t: 'input', sel: S(el), v, pw: el.type === 'password' });
}, true);
addEventListener('change', (e) => {
  if (!e.isTrusted || inOverlay(e)) return;
  const el = e.target; if (!(el instanceof Element)) return;
  if (el.tagName === 'SELECT') send({ t: 'select', sel: S(el), v: el.value });
  else if (el.tagName === 'INPUT' && el.type === 'file') send({ t: 'upload', sel: S(el), files: Array.from(el.files || []).map(f => f.name) });
}, true);
addEventListener('keydown', (e) => {
  if (!e.isTrusted || inOverlay(e)) return;
  user();
  if (R.mode && e.key === 'Escape') { R.mode = ''; paint(); return; }
  if (e.key !== 'Enter' || e.isComposing || e.keyCode === 229) return;
  const el = e.target;
  if (el instanceof Element && el.tagName === 'INPUT' && textField(el)) { R.lastEnter = Date.now(); send({ t: 'enter', sel: S(el) }); }
}, true);
for (const ty of ['wheel', 'touchmove']) addEventListener(ty, (e) => { if (e.isTrusted) user(); }, { capture: true, passive: true });
let st = 0;
addEventListener('scroll', (e) => {
  if (!R.scroll || e.target !== document || Date.now() - R.lastUser > 1500) return;
  clearTimeout(st); st = setTimeout(() => send({ t: 'scroll', x: Math.round(scrollX), y: Math.round(scrollY) }), 350);
}, { capture: true, passive: true });
addEventListener('mousemove', (e) => { if (R.mode && ui.hl) { const el = htmlEl(e.target); if (el && !inOverlay(e)) box(ui.hl, el); } }, { capture: true, passive: true });
// ---------- overlay (closed shadow root, created lazily by drain) ----------
const CSS_ = ':host{all:initial}*{box-sizing:border-box;font-family:system-ui,-apple-system,"Segoe UI","Microsoft YaHei","PingFang SC",sans-serif}' +
  '.bar{position:fixed;right:16px;bottom:16px;display:flex;align-items:center;gap:6px;padding:6px 6px 6px 12px;border-radius:999px;background:rgba(17,17,24,.92);color:#ececf1;font-size:12px;line-height:1;box-shadow:0 8px 30px rgba(0,0,0,.45),0 0 0 1px rgba(255,255,255,.08);backdrop-filter:blur(10px);z-index:2147483647;user-select:none}' +
  '.dot{width:9px;height:9px;border-radius:50%%;background:#f43f5e;box-shadow:0 0 0 0 rgba(244,63,94,.6);animation:p 1.4s infinite}.paused .dot{background:#f59e0b;animation:none}' +
  '@keyframes p{0%%{box-shadow:0 0 0 0 rgba(244,63,94,.55)}70%%{box-shadow:0 0 0 7px rgba(244,63,94,0)}100%%{box-shadow:0 0 0 0 rgba(244,63,94,0)}}' +
  '.lbl{font-weight:600;letter-spacing:.02em}.n{font-variant-numeric:tabular-nums;color:#a1a1b5;margin-right:4px}' +
  'button{all:unset;cursor:pointer;padding:6px 10px;border-radius:999px;color:#d4d4e0;background:rgba(255,255,255,.06);font-size:12px;transition:background .15s}button:hover{background:rgba(255,255,255,.14)}button.on{background:#7c3aed;color:#fff}button.stop{background:rgba(244,63,94,.18);color:#fda4af}button.stop:hover{background:rgba(244,63,94,.3)}' +
  '.sep{width:1px;height:16px;background:rgba(255,255,255,.12);margin:0 2px}' +
  '.hl,.fl{position:fixed;pointer-events:none;border-radius:4px;z-index:2147483646;display:none}.hl{outline:2px solid #a78bfa;background:rgba(167,139,250,.12)}.fl{outline:2px solid #f43f5e;transition:opacity .45s;opacity:0}' +
  '.tip{position:fixed;right:16px;bottom:58px;padding:7px 12px;border-radius:8px;background:rgba(17,17,24,.92);color:#ececf1;font-size:12px;opacity:0;transform:translateY(4px);transition:opacity .2s,transform .2s;pointer-events:none;z-index:2147483647}.tip.show{opacity:1;transform:none}' +
  '@media (prefers-reduced-motion:reduce){.dot{animation:none}.tip,.fl{transition:none}}';
const box = (b, el) => { const r = el.getBoundingClientRect(); Object.assign(b.style, { display: 'block', left: r.left + 'px', top: r.top + 'px', width: r.width + 'px', height: r.height + 'px' }); };
let ft = 0, tt = 0;
const flash = (el, c) => { if (!ui.fl) return; ui.fl.style.outlineColor = c; box(ui.fl, el); ui.fl.style.transition = 'none'; ui.fl.style.opacity = '1'; clearTimeout(ft); ft = setTimeout(() => { ui.fl.style.transition = ''; ui.fl.style.opacity = '0'; }, 250); };
const toast = (s) => { if (!ui.tip) return; ui.tip.textContent = s; ui.tip.classList.add('show'); clearTimeout(tt); tt = setTimeout(() => ui.tip.classList.remove('show'), 1600); };
const paint = () => {
  if (!ui.bar) return;
  ui.bar.classList.toggle('paused', R.paused);
  ui.lbl.textContent = R.paused ? '已暂停' : '录制中';
  ui.n.textContent = R.n + ' 步';
  ui.pause.textContent = R.paused ? '继续' : '暂停';
  ui.ex.classList.toggle('on', R.mode === 'extract'); ui.as.classList.toggle('on', R.mode === 'assert');
  if (!R.mode && ui.hl) ui.hl.style.display = 'none';
  if (R.mode) toast(R.mode === 'extract' ? '点击要提取文字的元素（Esc 取消）' : '点击要等待出现的元素（Esc 取消）');
};
const mount = () => {
  if (host && host.isConnected) return;
  if (!host) {
    host = document.createElement('div');
    root = host.attachShadow({ mode: 'closed' });
    const st = document.createElement('style'); st.textContent = CSS_; root.appendChild(st);
    const mk = (tag, cls, txt) => { const e = document.createElement(tag); if (cls) e.className = cls; if (txt) e.textContent = txt; return e; };
    ui.bar = mk('div', 'bar'); ui.hl = mk('div', 'hl'); ui.fl = mk('div', 'fl'); ui.tip = mk('div', 'tip');
    ui.lbl = mk('span', 'lbl'); ui.n = mk('span', 'n');
    ui.ex = mk('button', '', '提取文本'); ui.as = mk('button', '', '等待元素'); ui.pause = mk('button', '', '暂停'); ui.stop = mk('button', 'stop', '停止');
    ui.ex.title = '点击页面元素，把它的文字保存到变量'; ui.as.title = '点击页面元素，回放时等它出现再继续';
    ui.bar.append(mk('span', 'dot'), ui.lbl, ui.n, ui.ex, ui.as, mk('span', 'sep'), ui.pause, ui.stop);
    ui.ex.onclick = () => { R.mode = R.mode === 'extract' ? '' : 'extract'; paint(); };
    ui.as.onclick = () => { R.mode = R.mode === 'assert' ? '' : 'assert'; paint(); };
    ui.pause.onclick = () => send({ t: 'ctl', a: R.paused ? 'resume' : 'pause' });
    ui.stop.onclick = () => send({ t: 'ctl', a: 'stop' });
    root.append(ui.hl, ui.fl, ui.tip, ui.bar);
  }
  (document.documentElement || document).appendChild(host);
  paint();
};
const unmount = () => { if (host) host.remove(); };
R.unmount = unmount;
R.drain = (n, paused, overlay, scroll) => {
  if (R.n !== n || R.paused !== paused) { R.n = n; R.paused = paused; paint(); }
  R.scroll = scroll;
  if (overlay !== R.overlay) { R.overlay = overlay; if (!overlay) { R.mode = ''; unmount(); } }
  if (R.overlay && document.documentElement) mount();
  const q = R.q; R.q = [];
  return q.length ? JSON.stringify(q) : '';
};
R.destroy = () => { R.dead = true; R.mode = ''; R.q = []; unmount(); };
})();`, overlay, scroll)
}

// recDrainExpr installs the script when this world lacks it and drains the queue.
func recDrainExpr(n int, paused, overlay, scroll bool) string {
	return fmt.Sprintf(`(() => { %s const R = window.__gbRec; if (!R || R.dead || R.doc !== document) return ''; return R.drain(%d, %t, %t, %t); })()`, recCaptureScript(overlay, scroll), n, paused, overlay, scroll)
}

// recDestroyExpr removes the overlay and silences capture in a page.
const recDestroyExpr = `(() => { const R = window.__gbRec; if (R && R.destroy) R.destroy(); return ''; })()`

// recEnterJS is the execute_js code a recorded Enter key press replays as:
// key events for page handlers, then the form's implicit submission.
func recEnterJS(sel string) string {
	return `const el = (() => { const s = ` + jsq(sel) + `; for (const p of s.split(' || ')) { try { const e = p.startsWith('xpath:') ? document.evaluate(p.slice(6), document, null, 9, null).singleNodeValue : document.querySelector(p); if (e) return e; } catch (e) {} } return null; })();
if (!el) throw new Error('找不到元素：' + ` + jsq(sel) + `);
el.focus();
const o = { key: 'Enter', code: 'Enter', keyCode: 13, which: 13, bubbles: true, cancelable: true };
const go = el.dispatchEvent(new KeyboardEvent('keydown', o));
el.dispatchEvent(new KeyboardEvent('keypress', o));
el.dispatchEvent(new KeyboardEvent('keyup', o));
if (go && el.form) { if (el.form.requestSubmit) el.form.requestSubmit(); else el.form.submit(); }
return '回车';`
}
