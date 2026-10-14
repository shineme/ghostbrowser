// Shared RPA vocabulary for the editor, list and run drawer: step catalog,
// parameter schemas, summaries and immutable tree helpers.
import React from 'react';

export type RStep = {
  id: string; type: string; name?: string; disabled?: boolean; continueOnError?: boolean;
  params: Record<string, string>; children?: RStep[]; else?: RStep[];
};
export type RWorkflow = {
  id: string; name: string; description: string; steps: RStep[]; vars: Record<string, string>;
  createdAt: number; updatedAt: number; lastRunAt: number; lastStatus: string; lastRunId: string; runCount: number;
};
export type RIssue = { stepId: string; message: string };
export type RStepEvent = { stepId: string; index: number; type: string; status: string; log: string; ms: number; iter?: number; vars?: Record<string, string>; shot?: string };
export type RTarget = { profileId: string; profileName: string; status: string; error: string; startedAt: number; endedAt: number; autoStarted: boolean; steps: RStepEvent[]; vars: Record<string, string>; cur?: RStepEvent };
export type RRun = { id: string; workflowId: string; workflowName: string; status: string; startedAt: number; endedAt: number; concurrency: number; stepCount: number; targets: RTarget[] };
export type REvent = { runId: string; profileId: string; phase: 'step' | 'target' | 'run'; status: string; stepId?: string; type?: string; log?: string; index: number; ms?: number; iter?: number; vars?: Record<string, string>; shot?: string; error?: string };

// ---- icons (same stroke language as ui.tsx) ----
type IP = { s?: number };
const S = (p: IP): React.SVGProps<SVGSVGElement> => ({ xmlns: 'http://www.w3.org/2000/svg', viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2, strokeLinecap: 'round', strokeLinejoin: 'round', width: p.s || 16, height: p.s || 16 });
export const RI = {
  link: (p: IP) => <svg {...S(p)}><path d="M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7" /><path d="M14 11a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7" /></svg>,
  pointer: (p: IP) => <svg {...S(p)}><path d="m4 4 7 17 2.5-7.5L21 11Z" /><path d="m14 14 6 6" /></svg>,
  type: (p: IP) => <svg {...S(p)}><rect x="2" y="6" width="20" height="12" rx="2.5" /><path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M7 14h10" /></svg>,
  hourglass: (p: IP) => <svg {...S(p)}><path d="M6 2h12M6 22h12M7 2v4a5 5 0 0 0 10 0V2M7 22v-4a5 5 0 0 1 10 0v4" /></svg>,
  clock: (p: IP) => <svg {...S(p)}><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></svg>,
  scroll: (p: IP) => <svg {...S(p)}><rect x="7" y="2" width="10" height="20" rx="5" /><path d="M12 6v4" /><path d="m9.5 15.5 2.5 2.5 2.5-2.5" /></svg>,
  quote: (p: IP) => <svg {...S(p)}><path d="M4 6h16M4 12h10M4 18h7" /><path d="m17 15 3 3-3 3" /></svg>,
  code: (p: IP) => <svg {...S(p)}><path d="m8 7-5 5 5 5M16 7l5 5-5 5M14 4l-4 16" /></svg>,
  camera: (p: IP) => <svg {...S(p)}><path d="M3 8a2 2 0 0 1 2-2h2l2-2h6l2 2h2a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z" /><circle cx="12" cy="13" r="3.5" /></svg>,
  upload: (p: IP) => <svg {...S(p)}><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z" /><path d="M14 2v6h6M12 18v-6M9 15l3-3 3 3" /></svg>,
  tabs: (p: IP) => <svg {...S(p)}><path d="M3 8V6a2 2 0 0 1 2-2h4l2 3h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z" /><path d="M3 9h18" /></svg>,
  tabx: (p: IP) => <svg {...S(p)}><rect x="3" y="4" width="18" height="16" rx="2.5" /><path d="m9.5 10 5 5M14.5 10l-5 5" /></svg>,
  branch: (p: IP) => <svg {...S(p)}><circle cx="6" cy="5" r="2.2" /><circle cx="6" cy="19" r="2.2" /><circle cx="18" cy="8" r="2.2" /><path d="M6 7.2v9.6M18 10.2c0 4-6 3-11 6.5" /></svg>,
  repeat: (p: IP) => <svg {...S(p)}><path d="m17 2 4 4-4 4" /><path d="M3 11V9a3 3 0 0 1 3-3h15M7 22l-4-4 4-4" /><path d="M21 13v2a3 3 0 0 1-3 3H3" /></svg>,
  exit: (p: IP) => <svg {...S(p)}><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4M10 17l5-5-5-5M15 12H3" /></svg>,
  up: (p: IP) => <svg {...S(p)}><path d="m6 15 6-6 6 6" /></svg>,
  down: (p: IP) => <svg {...S(p)}><path d="m6 9 6 6 6-6" /></svg>,
  chev: (p: IP) => <svg {...S(p)}><path d="m9 6 6 6-6 6" /></svg>,
  flow: (p: IP) => <svg {...S(p)}><rect x="3" y="3" width="7" height="6" rx="1.5" /><rect x="14" y="15" width="7" height="6" rx="1.5" /><path d="M6.5 9v3a3 3 0 0 0 3 3h4.5" /><path d="m12 12.5 2.5 2.5L12 17.5" /></svg>,
  eyeoff: (p: IP) => <svg {...S(p)}><path d="M9.9 4.2A10 10 0 0 1 12 4c5 0 9 4.5 10 8a13 13 0 0 1-2.2 3.7M6.6 6.6A13 13 0 0 0 2 12c1 3.5 5 8 10 8a10 10 0 0 0 5.4-1.6" /><path d="m2 2 20 20M9.9 9.9a3 3 0 0 0 4.2 4.2" /></svg>,
  history: (p: IP) => <svg {...S(p)}><path d="M3 12a9 9 0 1 0 3-6.7L3 8" /><path d="M3 3v5h5M12 7v5l3 2" /></svg>,
  dot: (p: IP) => <svg {...S(p)}><circle cx="12" cy="12" r="4" fill="currentColor" stroke="none" /></svg>,
  braces: (p: IP) => <svg {...S(p)}><path d="M8 3H7a2 2 0 0 0-2 2v5a2 2 0 0 1-2 2 2 2 0 0 1 2 2v5a2 2 0 0 0 2 2h1M16 3h1a2 2 0 0 1 2 2v5a2 2 0 0 0 2 2 2 2 0 0 0-2 2v5a2 2 0 0 1-2 2h-1" /></svg>,
  image: (p: IP) => <svg {...S(p)}><rect x="3" y="3" width="18" height="18" rx="2.5" /><circle cx="9" cy="9" r="2" /><path d="m21 15-5-5L5 21" /></svg>,
};

// ---- catalog ----
export type FieldDef = {
  k: string; label: string;
  kind: 'text' | 'selector' | 'num' | 'code' | 'area' | 'bool' | 'seg' | 'pick' | 'var';
  opts?: [string, string][]; ph?: string; hint?: string; unit?: string; half?: boolean;
  show?: (p: Record<string, string>) => boolean;
};
export type StepDef = { type: string; label: string; group: string; icon: keyof typeof RI; desc: string; defaults: Record<string, string>; fields: FieldDef[]; container?: boolean };

const condShow = (...kinds: string[]) => (p: Record<string, string>) => kinds.includes(p.kind || 'exists');
const COND_KIND: [string, string][] = [['exists', '元素存在'], ['not_exists', '元素不存在'], ['contains', '文本包含'], ['equals', '值相等'], ['js', 'JS 结果为真']];
const condFields = (gate: (p: Record<string, string>) => boolean = () => true): FieldDef[] => [
  { k: 'kind', label: '判断方式', kind: 'pick', opts: COND_KIND, show: gate },
  { k: 'selector', label: '元素选择器', kind: 'selector', show: p => gate(p) && condShow('exists', 'not_exists')(p) },
  { k: 'left', label: '左值', kind: 'text', ph: '{{price}}', half: true, show: p => gate(p) && condShow('equals', 'contains')(p) },
  { k: 'right', label: '右值', kind: 'text', ph: '$19.99', half: true, show: p => gate(p) && condShow('equals', 'contains')(p) },
  { k: 'code', label: '判断代码', kind: 'code', ph: "return document.querySelectorAll('.item').length > 0;", hint: '返回真值即视为成立；可用 vars.名字 读取变量', show: p => gate(p) && condShow('js')(p) },
];

export const STEP_DEFS: StepDef[] = [
  { type: 'goto', label: '打开网址', group: '页面', icon: 'link', desc: '在当前标签页打开链接', defaults: { url: 'https://', timeout: '30000' }, fields: [
    { k: 'url', label: '网址', kind: 'text', ph: 'https://www.example.com', hint: '可用 {{变量}}；省略协议时自动补 https://' },
    { k: 'timeout', label: '超时', kind: 'num', unit: 'ms', half: true },
  ] },
  { type: 'delay', label: '延时', group: '页面', icon: 'clock', desc: '固定或随机等待一段时间', defaults: { ms: '1000', maxMs: '' }, fields: [
    { k: 'ms', label: '等待', kind: 'num', unit: 'ms', half: true },
    { k: 'maxMs', label: '随机上限', kind: 'num', unit: 'ms', half: true, ph: '留空 = 固定', hint: '填写后在两者之间随机，更像真人' },
  ] },
  { type: 'scroll', label: '滚动', group: '页面', icon: 'scroll', desc: '滚动页面或滚动到元素', defaults: { mode: 'by', selector: '', x: '0', y: '600' }, fields: [
    { k: 'mode', label: '方式', kind: 'seg', opts: [['by', '相对滚动'], ['to', '滚动到位置'], ['el', '滚动到元素']] },
    { k: 'selector', label: '元素选择器', kind: 'selector', show: p => p.mode === 'el' },
    { k: 'x', label: '水平 X', kind: 'num', unit: 'px', half: true, show: p => p.mode !== 'el' },
    { k: 'y', label: '垂直 Y', kind: 'num', unit: 'px', half: true, show: p => p.mode !== 'el' },
  ] },
  { type: 'screenshot', label: '截图', group: '页面', icon: 'camera', desc: '保存当前页面截图', defaults: { fullPage: 'false', saveAs: '' }, fields: [
    { k: 'fullPage', label: '截取整页', kind: 'bool', hint: '关闭则只截可视区域' },
    { k: 'saveAs', label: '保存路径到变量', kind: 'var', ph: '可选，如 shot' },
  ] },
  { type: 'click', label: '点击', group: '元素', icon: 'pointer', desc: '点击按钮、链接等元素', defaults: { selector: '', double: 'false' }, fields: [
    { k: 'selector', label: '元素选择器', kind: 'selector' },
    { k: 'double', label: '双击', kind: 'bool' },
  ] },
  { type: 'fill', label: '输入文本', group: '元素', icon: 'type', desc: '在输入框里填写内容', defaults: { selector: '', text: '', clear: 'true' }, fields: [
    { k: 'selector', label: '输入框选择器', kind: 'selector' },
    { k: 'text', label: '内容', kind: 'area', ph: '支持 {{变量}}' },
    { k: 'clear', label: '先清空原内容', kind: 'bool' },
  ] },
  { type: 'wait_for', label: '等待元素', group: '元素', icon: 'hourglass', desc: '等元素出现、可见或消失', defaults: { selector: '', state: 'visible', timeout: '10000' }, fields: [
    { k: 'selector', label: '元素选择器', kind: 'selector', hint: '留空则单纯等待「超时」时长' },
    { k: 'state', label: '等到', kind: 'seg', opts: [['visible', '可见'], ['present', '存在'], ['hidden', '消失']] },
    { k: 'timeout', label: '超时', kind: 'num', unit: 'ms', half: true },
  ] },
  { type: 'upload', label: '上传文件', group: '元素', icon: 'upload', desc: '给文件输入框选择本地文件', defaults: { selector: 'input[type=file]', files: '' }, fields: [
    { k: 'selector', label: '文件输入框', kind: 'selector' },
    { k: 'files', label: '文件路径', kind: 'area', ph: 'C:\\图片\\主图.jpg\nC:\\图片\\细节.jpg', hint: '每行一个，也可用 ; 分隔' },
  ] },
  { type: 'extract_text', label: '提取文本', group: '数据', icon: 'quote', desc: '读取元素文本或属性存入变量', defaults: { selector: '', var: '', attr: '', trim: 'true' }, fields: [
    { k: 'selector', label: '元素选择器', kind: 'selector' },
    { k: 'var', label: '保存到变量', kind: 'var', ph: 'price', half: true },
    { k: 'attr', label: '读取', kind: 'text', ph: '文本（留空）/ value / html / href', half: true },
    { k: 'trim', label: '去掉首尾空白', kind: 'bool' },
  ] },
  { type: 'execute_js', label: '执行 JS', group: '数据', icon: 'code', desc: '在页面里运行一段脚本', defaults: { code: 'return document.title;', saveAs: '' }, fields: [
    { k: 'code', label: '代码', kind: 'code', hint: '函数体或表达式；变量通过 vars 对象读取，不做 {{}} 替换' },
    { k: 'saveAs', label: '返回值存入变量', kind: 'var', ph: '可选', half: true },
  ] },
  { type: 'select_tab', label: '切换标签页', group: '标签页', icon: 'tabs', desc: '切到第 N 个标签页', defaults: { index: '1' }, fields: [
    { k: 'index', label: '标签页序号', kind: 'num', half: true, hint: '从 1 开始，按打开顺序' },
  ] },
  { type: 'close_tab', label: '关闭标签页', group: '标签页', icon: 'tabx', desc: '关闭当前或指定标签页', defaults: { index: '0' }, fields: [
    { k: 'index', label: '标签页序号', kind: 'num', half: true, hint: '0 = 当前标签页' },
  ] },
  { type: 'condition', label: '条件判断', group: '流程控制', icon: 'branch', desc: '满足条件走「则」，否则走「否则」', container: true, defaults: { kind: 'exists', selector: '' }, fields: condFields() },
  { type: 'loop', label: '循环', group: '流程控制', icon: 'repeat', desc: '按次数或条件重复执行', container: true, defaults: { mode: 'count', count: '3', indexVar: 'i' }, fields: [
    { k: 'mode', label: '循环方式', kind: 'seg', opts: [['count', '按次数'], ['while', '条件成立时']] },
    { k: 'count', label: '次数', kind: 'num', half: true, show: p => (p.mode || 'count') === 'count' },
    { k: 'max', label: '最多轮数', kind: 'num', half: true, ph: '1000', show: p => p.mode === 'while' },
    { k: 'indexVar', label: '轮次变量', kind: 'var', ph: 'loop_index', half: true, hint: '从 1 开始计数' },
    ...condFields(p => p.mode === 'while'),
  ] },
  { type: 'break_loop', label: '跳出循环', group: '流程控制', icon: 'exit', desc: '立即结束所在的循环', defaults: {}, fields: [] },
];
export const GROUPS = ['页面', '元素', '数据', '标签页', '流程控制'];
export const DEF: Record<string, StepDef> = Object.fromEntries(STEP_DEFS.map(d => [d.type, d]));
export const typeLabel = (t: string) => DEF[t]?.label || t;
export const StepIcon: React.FC<{ type: string; s?: number }> = ({ type, s = 15 }) => { const I = RI[DEF[type]?.icon || 'dot']; return <I s={s} />; };
// Category tint per group, used for the step glyph.
export const GROUP_TONE: Record<string, string> = { 页面: 'g-page', 元素: 'g-el', 数据: 'g-data', 标签页: 'g-tab', 流程控制: 'g-flow' };
export const toneOf = (t: string) => GROUP_TONE[DEF[t]?.group] || 'g-page';

const clip = (s = '', n = 48) => (s.length > n ? s.slice(0, n - 1) + '…' : s);
const condText = (p: Record<string, string>) => {
  switch (p.kind || 'exists') {
    case 'not_exists': return `${p.selector || '…'} 不存在`;
    case 'js': return `JS：${clip((p.code || '').replace(/\s+/g, ' '), 40)}`;
    case 'equals': return `${p.left || '…'} = ${p.right || '""'}`;
    case 'contains': return `${p.left || '…'} 包含 ${p.right || '""'}`;
    default: return `${p.selector || '…'} 存在`;
  }
};
/** One-line description shown on a step row. */
export const summary = (s: RStep): string => {
  const p = s.params || {};
  switch (s.type) {
    case 'goto': return p.url || '未填写网址';
    case 'click': return (p.selector || '未填写选择器') + (p.double === 'true' ? ' · 双击' : '');
    case 'fill': return `${p.selector || '…'} ← ${clip(p.text || '""', 30)}`;
    case 'wait_for': return p.selector ? `${p.selector} ${({ hidden: '消失', present: '存在' } as any)[p.state] || '可见'}` : `等待 ${p.timeout || 10000} ms`;
    case 'delay': return p.maxMs ? `${p.ms || 0} ~ ${p.maxMs} ms 随机` : `${p.ms || 1000} ms`;
    case 'scroll': return p.mode === 'el' ? `到 ${p.selector || '…'}` : `${p.mode === 'to' ? '到' : '滚动'} (${p.x || 0}, ${p.y || 0})`;
    case 'extract_text': return `${p.selector || '…'} → {{${p.var || '?'}}}`;
    case 'execute_js': return clip((p.code || '').replace(/\s+/g, ' '), 52) + (p.saveAs ? ` → {{${p.saveAs}}}` : '');
    case 'screenshot': return (p.fullPage === 'true' ? '整页' : '可视区域') + (p.saveAs ? ` → {{${p.saveAs}}}` : '');
    case 'upload': return `${(p.files || '').split(/[\n;|]/).filter(x => x.trim()).length} 个文件 → ${p.selector || '…'}`;
    case 'select_tab': return `第 ${p.index || 1} 个标签页`;
    case 'close_tab': return p.index && p.index !== '0' ? `第 ${p.index} 个标签页` : '当前标签页';
    case 'condition': return `如果 ${condText(p)}`;
    case 'loop': return (p.mode === 'while' ? `当 ${condText(p)}` : `重复 ${p.count || 0} 次`) + ` · {{${p.indexVar || 'loop_index'}}}`;
    case 'break_loop': return '结束当前循环';
  }
  return '';
};

// ---- tree helpers (immutable) ----
export const uid = () => Math.random().toString(36).slice(2, 10) + Date.now().toString(36).slice(-4);
export const newStep = (type: string): RStep => {
  const d = DEF[type];
  const s: RStep = { id: uid(), type, params: { ...(d?.defaults || {}) } };
  if (type === 'condition') { s.children = []; s.else = []; }
  if (type === 'loop') s.children = [];
  return s;
};
export const walk = (list: RStep[] = [], fn: (s: RStep, depth: number, inLoop: boolean) => void, depth = 0, inLoop = false) => {
  for (const s of list) {
    fn(s, depth, inLoop);
    walk(s.children, fn, depth + 1, inLoop || s.type === 'loop');
    walk(s.else, fn, depth + 1, inLoop);
  }
};
export const flat = (list: RStep[]) => { const out: RStep[] = []; walk(list, s => out.push(s)); return out; };
export const countSteps = (list: RStep[]) => flat(list).length;

const mapLists = (list: RStep[], fn: (l: RStep[]) => RStep[]): RStep[] =>
  fn(list).map(s => ({ ...s, children: s.children ? mapLists(s.children, fn) : s.children, else: s.else ? mapLists(s.else, fn) : s.else }));
export const updateStep = (list: RStep[], id: string, fn: (s: RStep) => RStep) => mapLists(list, l => l.map(s => (s.id === id ? fn(s) : s)));
export const removeStep = (list: RStep[], id: string) => mapLists(list, l => l.filter(s => s.id !== id));
export const moveStep = (list: RStep[], id: string, dir: -1 | 1) => mapLists(list, l => {
  const i = l.findIndex(s => s.id === id);
  if (i < 0 || i + dir < 0 || i + dir >= l.length) return l;
  const n = [...l]; [n[i], n[i + dir]] = [n[i + dir], n[i]]; return n;
});
export const insertAfter = (list: RStep[], afterId: string, step: RStep) => mapLists(list, l => {
  const i = l.findIndex(s => s.id === afterId);
  if (i < 0) return l; const n = [...l]; n.splice(i + 1, 0, step); return n;
});
/** Append into a container branch; parentId "" = top level. */
export const insertInto = (list: RStep[], parentId: string, branch: 'children' | 'else', step: RStep): RStep[] =>
  !parentId ? [...list, step] : updateStep(list, parentId, s => ({ ...s, [branch]: [...(s[branch] || []), step] }));
export const cloneStep = (s: RStep): RStep => ({ ...s, id: uid(), params: { ...s.params }, children: s.children?.map(cloneStep), else: s.else?.map(cloneStep) });
export const findStep = (list: RStep[], id: string) => flat(list).find(s => s.id === id);
/** Position of a step among its siblings (for disabling ↑/↓). */
export const siblingPos = (list: RStep[], id: string): [number, number] => {
  let r: [number, number] = [-1, 0];
  const go = (l: RStep[]) => { const i = l.findIndex(s => s.id === id); if (i >= 0) r = [i, l.length]; l.forEach(s => { go(s.children || []); go(s.else || []); }); };
  go(list); return r;
};
/** Variables available before step `id`: initial vars plus every var defined earlier in pre-order (and loop index vars of enclosing loops). */
export const varsBefore = (wf: { steps: RStep[]; vars?: Record<string, string> }, id: string): string[] => {
  const out = new Set(Object.keys(wf.vars || {}));
  let done = false;
  const go = (l: RStep[]) => {
    for (const s of l) {
      if (done) return;
      if (s.id === id) { done = true; return; }
      const p = s.params || {};
      if (s.type === 'extract_text' && p.var) out.add(p.var.trim());
      if ((s.type === 'execute_js' || s.type === 'screenshot') && p.saveAs) out.add(p.saveAs.trim());
      if (s.type === 'loop') out.add((p.indexVar || 'loop_index').trim());
      go(s.children || []); go(s.else || []);
    }
  };
  go(wf.steps); out.delete(''); return [...out];
};

export const STATUS: Record<string, [string, string]> = {
  ok: ['成功', 'ok'], failed: ['失败', 'fail'], partial: ['部分成功', 'warn'], cancelled: ['已取消', 'unknown'], running: ['运行中', 'cy'],
  pending: ['排队中', 'unknown'], starting: ['启动中', 'cy'], skipped: ['已跳过', 'unknown'],
};
export const StatusBadge: React.FC<{ s: string; empty?: string }> = ({ s, empty = '未运行' }) => {
  if (!s) return <span className="badge unknown">{empty}</span>;
  const [l, c] = STATUS[s] || [s, 'unknown'];
  return <span className={`badge ${c}`}><span className={`d ${s === 'running' || s === 'starting' ? 'pulse' : ''}`} />{l}</span>;
};
export const dur = (ms: number) => {
  if (ms < 1000) return `${ms} ms`;
  const s = ms / 1000; if (s < 60) return `${s.toFixed(s < 10 ? 1 : 0)} 秒`;
  const m = Math.floor(s / 60); return `${m} 分 ${Math.round(s % 60)} 秒`;
};
export const clock = (ts: number) => { if (!ts) return '—'; const d = new Date(ts * 1000); const p = (n: number) => String(n).padStart(2, '0'); return `${d.getMonth() + 1}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`; };
