// 多窗口同步器: master + followers, per-type switches, random delay, tab sync,
// window/tab actions and batch URL. Opened from the run strip (WindowTools).
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { api, Profile } from './api';
import { Icon, useToast, useExit, CountUp, avatarStyle } from './ui';
import type { SyncerState } from './WindowTools';

type Opts = { click: boolean; input: boolean; key: boolean; scroll: boolean; tabs: boolean; delayMin: number; delayMax: number };
type Prefs = Opts & { master: string; followers: string[]; newTab: boolean };
type ActKey = 'grid' | 'stack' | 'size' | 'show' | 'unify' | 'closeCurrent' | 'closeOthers' | 'closeBlank' | 'open';

const DEF: Prefs = { click: true, input: true, key: true, scroll: true, tabs: true, delayMin: 0, delayMax: 120, master: '', followers: [], newTab: false };
const EV_LABEL: Record<string, string> = { click: '点击', scroll: '滚动', key: '按键', text: '输入', select: '选择', nav: '跳转', tabnew: '新建标签', tabswitch: '切换标签', tabclose: '关闭标签' };
const initials = (n: string) => (n.match(/[A-Za-z0-9\u4e00-\u9fa5]+/g) || [n]).slice(0, 2).map(w => w[0]).join('').toUpperCase();
const errMsg = (e: any) => String(e?.message || e || '未知错误');
const elapsed = (ms: number) => {
  const s = Math.max(0, Math.floor(ms / 1000)); const h = Math.floor(s / 3600), m = Math.floor(s % 3600 / 60), ss = s % 60;
  const p = (x: number) => String(x).padStart(2, '0');
  return h ? `${h}:${p(m)}:${p(ss)}` : `${p(m)}:${p(ss)}`;
};
const clampMs = (v: string) => Math.max(0, Math.min(10000, parseInt(v.replace(/\D/g, '') || '0', 10)));

/** Same rules as the backend's normalizeURL, for instant feedback. */
export const normURL = (raw: string): { url?: string; err?: string } => {
  let s = raw.trim();
  if (!s) return { err: '请输入网址' };
  if (/^(about:blank|chrome:\/\/|edge:\/\/)/i.test(s)) return { url: s };
  if (/\s/.test(s)) return { err: '网址里不能有空格' };
  if (s.startsWith('//')) s = 'https:' + s; else if (!/^[a-z][a-z0-9+.-]*:\/\//i.test(s)) s = 'https://' + s;
  let u: URL; try { u = new URL(s); } catch { return { err: '网址格式不正确' }; }
  if (!/^https?:$/.test(u.protocol)) return { err: '只支持 http / https 网址' };
  if (!u.hostname || (u.hostname !== 'localhost' && !u.hostname.includes('.') && !/^\[/.test(u.hostname))) return { err: '网址格式不正确' };
  return { url: s };
};

// ------------------------------------------------------------------ glyphs
const S = (s = 15) => ({ width: s, height: s, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 1.9, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const });
const G = {
  sync: (s?: number) => <svg {...S(s)}><path d="M20 11a8 8 0 0 0-14.3-4.9L4 8" /><path d="M4 3v5h5" /><path d="M4 13a8 8 0 0 0 14.3 4.9L20 16" /><path d="M20 21v-5h-5" /></svg>,
  stack: (s?: number) => <svg {...S(s)}><rect x="7" y="7" width="13" height="13" rx="2" /><path d="M4 16V6a2 2 0 0 1 2-2h10" /></svg>,
  size: (s?: number) => <svg {...S(s)}><path d="M15 3h6v6" /><path d="M9 21H3v-6" /><path d="M21 3l-7 7" /><path d="M3 21l7-7" /></svg>,
  eye: (s?: number) => <svg {...S(s)}><path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12Z" /><circle cx="12" cy="12" r="3" /></svg>,
  link: (s?: number) => <svg {...S(s)}><path d="M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7" /><path d="M14 11a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7" /></svg>,
  tabx: (s?: number) => <svg {...S(s)}><rect x="3" y="5" width="18" height="15" rx="2" /><path d="M3 9h18" /><path d="m10 12.5 4 4m0-4-4 4" /></svg>,
  tabs: (s?: number) => <svg {...S(s)}><rect x="3" y="5" width="18" height="15" rx="2" /><path d="M3 9h7V5" /><path d="M10 9h4V5" /></svg>,
  blank: (s?: number) => <svg {...S(s)}><rect x="3" y="5" width="18" height="15" rx="2" strokeDasharray="3 3" /><path d="m9.5 11 5 5m0-5-5 5" /></svg>,
  mouse: (s?: number) => <svg {...S(s)}><rect x="6" y="3" width="12" height="18" rx="6" /><path d="M12 7v4" /></svg>,
  type: (s?: number) => <svg {...S(s)}><path d="M4 7V5h16v2" /><path d="M12 5v14" /><path d="M9 19h6" /></svg>,
  key: (s?: number) => <svg {...S(s)}><rect x="2" y="6" width="20" height="12" rx="2" /><path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M7 14h10" /></svg>,
  wheel: (s?: number) => <svg {...S(s)}><path d="M12 3v18" /><path d="m8 7 4-4 4 4" /><path d="m8 17 4 4 4-4" /></svg>,
  crown: (s?: number) => <svg {...S(s)}><path d="m3 7 4.5 4L12 4l4.5 7L21 7l-2 12H5L3 7Z" /></svg>,
  chev: (s?: number) => <svg {...S(s)}><path d="M7 10l5 5 5-5" /></svg>,
  clock: (s?: number) => <svg {...S(s)}><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></svg>,
  info: (s?: number) => <svg {...S(s)}><circle cx="12" cy="12" r="9" /><path d="M12 11v5M12 8h.01" /></svg>,
};

// ------------------------------------------------------------------ small parts
const useOutside = (open: boolean, close: () => void, ref: React.RefObject<HTMLElement | null>) => {
  useEffect(() => {
    if (!open) return;
    const down = (e: MouseEvent) => { if (!ref.current?.contains(e.target as Node)) close(); };
    const key = (e: KeyboardEvent) => { if (e.key === 'Escape') { e.stopPropagation(); close(); } };
    window.addEventListener('mousedown', down); window.addEventListener('keydown', key, true);
    return () => { window.removeEventListener('mousedown', down); window.removeEventListener('keydown', key, true); };
  }, [open]);
};

const Ava: React.FC<{ p?: Profile; id: string; name: string; sm?: boolean }> = ({ p, id, name, sm }) =>
  <span className={`sd-ava ${sm ? 'sm' : ''}`} style={avatarStyle(p?.fingerprint?.noiseSeed || id)}>{initials(name)}</span>;

/** Hover tooltip with the reason a control is disabled. */
const Tip: React.FC<{ tip?: string; children: React.ReactNode; wide?: boolean }> = ({ tip, children, wide }) =>
  <span className={`sd-tip ${wide ? 'wide' : ''}`} data-tip={tip || ''}>{children}</span>;

const MasterPick: React.FC<{ running: Profile[]; value: string; onChange: (id: string) => void; disabled?: string }> = ({ running, value, onChange, disabled }) => {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useOutside(open, () => setOpen(false), ref);
  const cur = running.find(p => p.id === value);
  return (
    <Tip tip={disabled} wide>
      <div className={`sd-pick ${open ? 'open' : ''} ${disabled ? 'dis' : ''}`} ref={ref}>
        <button type="button" className="sd-pick-btn" disabled={!!disabled} onClick={() => setOpen(o => !o)}>
          {cur ? <><Ava p={cur} id={cur.id} name={cur.name} sm /><span className="sd-pick-v">{cur.name}</span><span className="sd-crown">{G.crown(11)}主控</span></>
            : <span className="sd-pick-ph">选择主控环境</span>}
          <span className="sd-chev">{G.chev(14)}</span>
        </button>
        {open && <div className="sd-menu">
          {running.map(p => <button type="button" key={p.id} className={p.id === value ? 'on' : ''} onClick={() => { onChange(p.id); setOpen(false); }}>
            <span className="radio" /><Ava p={p} id={p.id} name={p.name} sm /><span className="sd-o">{p.name}</span>
          </button>)}
          {!running.length && <div className="sd-menu-empty">没有正在运行的环境</div>}
        </div>}
      </div>
    </Tip>
  );
};

const FollowPick: React.FC<{ running: Profile[]; master: string; value: string[]; onChange: (ids: string[]) => void; disabled?: string }> = ({ running, master, value, onChange, disabled }) => {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useOutside(open, () => setOpen(false), ref);
  const opts = running.filter(p => p.id !== master);
  const sel = opts.filter(p => value.includes(p.id));
  const toggle = (id: string) => onChange(value.includes(id) ? value.filter(x => x !== id) : [...value, id]);
  const all = sel.length === opts.length && opts.length > 0;
  return (
    <Tip tip={disabled} wide>
      <div className={`sd-pick ${open ? 'open' : ''} ${disabled ? 'dis' : ''}`} ref={ref}>
        <button type="button" className="sd-pick-btn" disabled={!!disabled} onClick={() => setOpen(o => !o)}>
          {sel.length ? <span className="sd-stackav">{sel.slice(0, 4).map((p, i) => <span key={p.id} style={{ zIndex: 9 - i }}><Ava p={p} id={p.id} name={p.name} sm /></span>)}</span> : null}
          {sel.length ? <span className="sd-pick-v">{sel.length === 1 ? sel[0].name : <>已选 <b className="num">{sel.length}</b> 个受控环境</>}</span>
            : <span className="sd-pick-ph">{opts.length ? '至少选择一个受控环境' : '没有其他正在运行的环境'}</span>}
          <span className="sd-chev">{G.chev(14)}</span>
        </button>
        {open && <div className="sd-menu">
          {opts.length > 1 && <div className="sd-menu-h"><span>受控环境 <em className="num">{sel.length}/{opts.length}</em></span>
            <button type="button" onClick={() => onChange(all ? [] : opts.map(p => p.id))}>{all ? '全不选' : '全选'}</button></div>}
          {opts.map(p => { const on = value.includes(p.id); return <button type="button" key={p.id} className={on ? 'on' : ''} onClick={() => toggle(p.id)}>
            <span className={`chk ${on ? 'on' : ''}`}><Icon.check s={11} /></span><Ava p={p} id={p.id} name={p.name} sm /><span className="sd-o">{p.name}</span>
          </button>; })}
          {!opts.length && <div className="sd-menu-empty">至少需要两个正在运行的环境</div>}
        </div>}
      </div>
    </Tip>
  );
};

const Switch: React.FC<{ on: boolean; label: string; icon: React.ReactNode; onChange: (v: boolean) => void }> = ({ on, label, icon, onChange }) =>
  <button type="button" role="switch" aria-checked={on} className={`sd-sw ${on ? 'on' : ''}`} onClick={() => onChange(!on)}>
    <span className="sd-sw-ic">{icon}</span><span className="sd-sw-l">{label}</span><span className="sd-sw-track"><i /></span>
  </button>;

type ActState = 'idle' | 'busy' | 'ok' | 'bad';
const ActBtn: React.FC<{ icon: React.ReactNode; label: string; state: ActState; tip?: string; onClick: () => void; tone?: string }> = ({ icon, label, state, tip, onClick, tone = '' }) =>
  <Tip tip={tip}>
    <button type="button" className={`sd-act ${tone} is-${state}`} disabled={!!tip || state === 'busy'} onClick={onClick}>
      <span className="sd-act-ic">{state === 'busy' ? <span className="sd-spin" /> : state === 'ok' ? <Icon.check s={14} /> : state === 'bad' ? <Icon.alert s={14} /> : icon}</span>
      <span className="sd-act-l">{label}</span>
    </button>
  </Tip>;

// ------------------------------------------------------------------ dialog
export const SyncerDialog: React.FC<{ running: Profile[]; st: SyncerState; onClose: () => void; onStop: () => Promise<void> | void }> = ({ running, st, onClose, onStop }) => {
  const toast = useToast();
  const [out, close] = useExit(onClose, 220);
  const [p, setP] = useState<Prefs>(DEF);
  const [loaded, setLoaded] = useState(false);
  const [acts, setActs] = useState<Partial<Record<ActKey | 'start', ActState>>>({});
  const [url, setUrl] = useState('');
  const [urlTouched, setUrlTouched] = useState(false);
  const [, tick] = useState(0);
  const [pulse, setPulse] = useState(0);
  const byId = useMemo(() => Object.fromEntries(running.map(x => [x.id, x])), [running]);
  const save = useRef<any>(0);

  useEffect(() => {
    api.GetSyncerPrefs().then((v: any) => {
      const n: Prefs = { ...DEF, ...(v || {}), followers: v?.followers || [] };
      setP(n); setLoaded(true);
    }).catch(() => setLoaded(true));
  }, []);
  // Keep selections valid as browsers start/stop; default to the first running.
  const runIds = running.map(x => x.id).join();
  useEffect(() => {
    if (!loaded) return;
    setP(cur => {
      const ids = running.map(x => x.id);
      let master = st.running && st.master ? st.master : cur.master;
      if (!ids.includes(master)) master = ids[0] || '';
      let followers = (st.running ? st.followers.map(f => f.id) : cur.followers).filter(id => id !== master && ids.includes(id));
      if (!followers.length && !cur.followers.length) followers = ids.filter(id => id !== master);
      return master === cur.master && followers.join() === cur.followers.join() ? cur : { ...cur, master, followers };
    });
  }, [loaded, runIds, st.running]);
  // Persist (debounced); the backend applies options to a running syncer at once.
  useEffect(() => {
    if (!loaded) return;
    clearTimeout(save.current);
    save.current = setTimeout(() => { api.SaveSyncerPrefs(p as any).catch(() => {}); }, 260);
    return () => clearTimeout(save.current);
  }, [p, loaded]);
  useEffect(() => { if (!st.running) return; const t = setInterval(() => tick(x => x + 1), 1000); return () => clearInterval(t); }, [st.running]);
  useEffect(() => { if (st.running && st.events) setPulse(x => x + 1); }, [st.events]);
  useEffect(() => {
    const k = (e: KeyboardEvent) => { if (e.key === 'Escape') close(); };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  }, [close]);

  const set = <K extends keyof Prefs>(k: K, v: Prefs[K]) => setP(cur => ({ ...cur, [k]: v }));
  const flash = (k: ActKey | 'start', s: ActState) => {
    setActs(a => ({ ...a, [k]: s }));
    if (s === 'ok' || s === 'bad') setTimeout(() => setActs(a => (a[k] === s ? { ...a, [k]: 'idle' } : a)), 1600);
  };

  const masterOk = !!p.master && !!byId[p.master];
  const fol = p.followers.filter(id => byId[id] && id !== p.master);
  const windows = [p.master, ...fol].filter(id => byId[id]);
  const lockReason = st.running ? '同步进行中不能更换窗口，请先停止同步' : '';
  const needMaster = !masterOk ? '请先选择主控窗口' : '';
  const needFol = !fol.length ? '请至少选择一个受控窗口' : '';
  const needAny = !windows.length ? '请先选择窗口' : '';
  const nu = normURL(url);
  const urlErr = urlTouched && url.trim() ? nu.err : '';

  const report = (label: string, res: any) => {
    const failed: string[] = res?.failed || [];
    if (failed.length && !res?.done) toast(`${label}失败：${failed.join('；')}`, 'bad');
    else if (failed.length) toast(`${label}：${res.done}/${res.total} 个窗口完成，${failed.length} 个失败：${failed.join('；')}`, 'bad');
    else toast(`${label}：已完成 ${res?.done ?? 0} 个窗口`);
    return !failed.length || !!res?.done;
  };
  const run = async (k: ActKey, label: string, fn: () => Promise<any>, arrange = false) => {
    flash(k, 'busy');
    try {
      const res = await fn();
      const ok = arrange
        ? (res?.failed?.length ? (toast(`${label}：${res.arranged}/${res.total} 个窗口完成，${res.failed.length} 个失败：${res.failed.join('；')}`, 'bad'), !!res.arranged) : (toast(`${label}：已排列 ${res?.arranged ?? 0} 个窗口`), true))
        : report(label, res);
      flash(k, ok ? 'ok' : 'bad');
    } catch (e) { toast(`${label}失败：${errMsg(e)}`, 'bad'); flash(k, 'bad'); }
  };
  const tab = (k: ActKey, label: string, extra?: { url?: string; newTab?: boolean }) =>
    run(k, label, () => api.SyncTabAction(p.master, fol, k, extra?.url || '', !!extra?.newTab));

  const start = async () => {
    flash('start', 'busy');
    try {
      await api.SaveSyncerPrefs(p as any);
      await api.StartSync(p.master, fol);
      flash('start', 'ok');
    } catch (e) { toast(`无法开始同步：${errMsg(e)}`, 'bad'); flash('start', 'bad'); }
  };
  const stop = async () => { flash('start', 'busy'); try { await onStop(); } finally { flash('start', 'idle'); } };
  const open = () => {
    setUrlTouched(true);
    if (nu.err) return;
    tab('open', p.newTab ? '在新标签页打开' : '批量访问', { url: nu.url, newTab: p.newTab });
  };

  const okN = st.followers.filter(f => f.ok).length;
  const startTip = st.running ? '' : needMaster || needFol;
  const delayTxt = p.delayMax > 0 ? `${p.delayMin}–${p.delayMax} ms` : '无延迟';
  const offTypes = (['click', 'input', 'key', 'scroll'] as const).filter(k => !p[k]).length;

  return createPortal(
    <div className={`overlay center sd-ov ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className={`modal sd ${st.running ? 'live' : ''}`} role="dialog" aria-label="多窗口同步器">
        <div className="dhead sd-head">
          <div className={`glyph sd-glyph ${st.running ? 'spin' : ''}`}>{G.sync(19)}</div>
          <div className="grow sd-title">
            <h3>多窗口同步器</h3>
            <div className="sub">{st.running ? <>正在同步 · <b>{st.masterName || '主控'}</b> → {okN} 个受控窗口</> : '在主控窗口操作，受控窗口同步执行'}</div>
          </div>
          <div className={`sd-live ${st.running ? 'on' : ''}`}>
            <i className="sd-dot" />{st.running ? <><span>同步中</span><b className="num">{elapsed(Date.now() - st.startedAt)}</b></> : <span>未开始</span>}
          </div>
          <button className="btn ghost icon" title="关闭（Esc）" onClick={() => close()}><Icon.x /></button>
        </div>

        <div className="dbody sd-body">
          <div className="sd-note"><span className="sd-note-ic">{G.info(16)}</span><div>
            <b>主控窗口的网页操作、地址跳转与标签页新建、切换、关闭会复制到受控窗口。</b>
            <span>所有操作仅在本机通过调试协议传递，不经过任何服务器；输入法候选、文件选择框与 iframe 内的操作不会同步。</span>
          </div></div>

          <div className="sd-pickers">
            <div className="sd-field"><label>主控窗口</label>
              <MasterPick running={running} value={p.master} disabled={lockReason} onChange={id => setP(c => ({ ...c, master: id, followers: c.followers.filter(f => f !== id).concat(c.master && c.master !== id && byId[c.master] ? [c.master] : []) }))} />
            </div>
            <div className="sd-field"><label>受控窗口</label>
              <FollowPick running={running} master={p.master} value={fol} disabled={lockReason} onChange={ids => set('followers', ids)} />
            </div>
          </div>

          <div className="sd-card sd-opts">
            <div className="sd-card-h"><span>同步内容</span><em>{st.running ? '修改后立即生效' : offTypes ? `已关闭 ${offTypes} 类事件` : '全部同步'}</em></div>
            <div className="sd-sws">
              <Switch on={p.click} label="同步点击" icon={G.mouse(14)} onChange={v => set('click', v)} />
              <Switch on={p.input} label="同步输入" icon={G.type(14)} onChange={v => set('input', v)} />
              <Switch on={p.key} label="同步按键" icon={G.key(14)} onChange={v => set('key', v)} />
              <Switch on={p.scroll} label="同步滚动" icon={G.wheel(14)} onChange={v => set('scroll', v)} />
              <Switch on={p.tabs} label="同步标签页" icon={G.tabs(14)} onChange={v => set('tabs', v)} />
            </div>
            <div className="sd-delay">
              <div className="sd-delay-l"><b>{G.clock(14)}操作延迟</b><span>每个受控窗口对每个操作独立随机等待，模拟真人节奏</span></div>
              <div className="sd-range">
                <label className="sd-ms"><input type="text" inputMode="numeric" aria-label="最小延迟" value={String(p.delayMin)} onChange={e => set('delayMin', clampMs(e.target.value))}
                  onBlur={() => p.delayMin > p.delayMax && setP(c => ({ ...c, delayMin: c.delayMax, delayMax: c.delayMin }))} /><span>ms</span></label>
                <span className="sd-to">至</span>
                <label className="sd-ms"><input type="text" inputMode="numeric" aria-label="最大延迟" value={String(p.delayMax)} onChange={e => set('delayMax', clampMs(e.target.value))}
                  onBlur={() => p.delayMin > p.delayMax && setP(c => ({ ...c, delayMin: c.delayMax, delayMax: c.delayMin }))} /><span>ms</span></label>
              </div>
            </div>
          </div>

          {st.running && <div className="sd-card sd-livecard">
            <div className="sd-stats">
              <div><span>同步事件</span><b><CountUp n={st.events} ms={500} /></b></div>
              <div><span>受控在线</span><b className="num">{okN}<small>/{st.followers.length}</small></b></div>
              <div><span>主控标签页</span><b className="num">{st.tabs ?? 1}</b></div>
              <div className="sd-lastev" key={pulse}><span>最近动作</span><b>{EV_LABEL[st.lastEvent] || '等待操作'}</b></div>
            </div>
            <div className="sd-chips">
              {st.followers.map(f => <div key={f.id} className={`sd-chip ${f.ok ? '' : 'bad'}`} title={f.ok ? `${f.name} · 已同步 ${f.events} 个事件` : `${f.name}：${f.error || '已断开'}`}>
                <Ava p={byId[f.id]} id={f.id} name={f.name} sm />
                <span className="sd-chip-n">{f.name}</span>
                {f.ok ? <><span className="sd-chip-m num">{f.latencyMs ? `${f.latencyMs} ms` : '—'}</span><span className="sd-chip-m num">{f.tabs ?? 1} 标签</span><i className="sd-chip-dot" key={`${f.id}-${f.events}`} /></>
                  : <span className="sd-chip-err">{f.error || '已断开'}</span>}
              </div>)}
            </div>
            {st.reason && <div className="sd-warn"><Icon.alert s={13} /><span title={st.reason}>{st.reason}</span></div>}
          </div>}

          <div className="sd-acts">
            {st.running
              ? <ActBtn tone="stop" icon={<Icon.stop s={13} />} label="停止同步" state={acts.start || 'idle'} onClick={stop} />
              : <ActBtn tone="go" icon={<Icon.play s={13} />} label="开始同步" state={acts.start || 'idle'} tip={startTip} onClick={start} />}
            <ActBtn icon={<Icon.grid s={14} />} label="网格平铺" state={acts.grid || 'idle'} tip={needAny} onClick={() => run('grid', '网格平铺', () => api.ArrangeWindows(windows, 'grid'), true)} />
            <ActBtn icon={G.stack(14)} label="重叠排列" state={acts.stack || 'idle'} tip={needAny} onClick={() => run('stack', '重叠排列', () => api.ArrangeWindows(windows, 'stack'), true)} />
            <ActBtn icon={G.size(14)} label="统一大小" state={acts.size || 'idle'} tip={needMaster || needFol} onClick={() => tab('size', '统一大小')} />
            <ActBtn icon={G.eye(14)} label="显示窗口" state={acts.show || 'idle'} tip={needAny} onClick={() => tab('show', '显示窗口')} />
            <ActBtn icon={G.link(14)} label="统一标签页" state={acts.unify || 'idle'} tip={needMaster || needFol} onClick={() => tab('unify', '统一标签页')} />
            <ActBtn icon={G.tabx(14)} label="关闭当前标签" state={acts.closeCurrent || 'idle'} tip={needAny} onClick={() => tab('closeCurrent', '关闭当前标签')} />
            <ActBtn icon={G.tabs(14)} label="关闭其他标签" state={acts.closeOthers || 'idle'} tip={needAny} onClick={() => tab('closeOthers', '关闭其他标签')} />
            <ActBtn icon={G.blank(14)} label="关闭空白标签" state={acts.closeBlank || 'idle'} tip={needAny} onClick={() => tab('closeBlank', '关闭空白标签')} />
          </div>

          <div className="sd-card sd-batch">
            <div className="sd-batch-l"><b>批量访问网址</b><span>在全部选中窗口（{windows.length} 个）打开同一网页</span></div>
            <div className="sd-batch-r">
              <div className={`sd-url ${urlErr ? 'err' : ''}`}>
                <input type="text" value={url} placeholder="https://example.com" spellCheck={false}
                  onChange={e => setUrl(e.target.value)} onBlur={() => setUrlTouched(true)} onKeyDown={e => { if (e.key === 'Enter') open(); }} />
                <span className="sd-url-hint">{urlErr ? <em>{urlErr}</em> : nu.url && nu.url !== url.trim() ? <>将打开 <b>{nu.url}</b></> : ''}</span>
              </div>
              <button type="button" className={`sd-check ${p.newTab ? 'on' : ''}`} onClick={() => set('newTab', !p.newTab)}>
                <span className={`chk ${p.newTab ? 'on' : ''}`}><Icon.check s={11} /></span>新标签页
              </button>
              <Tip tip={needAny || (!url.trim() ? '请输入网址' : '')}>
                <button type="button" className={`btn primary sd-open is-${acts.open || 'idle'}`} disabled={!!needAny || !url.trim() || acts.open === 'busy'} onClick={open}>
                  {acts.open === 'busy' ? <span className="sd-spin light" /> : acts.open === 'ok' ? <Icon.check s={14} /> : null}打开
                </button>
              </Tip>
            </div>
          </div>
        </div>

        <div className="dfoot sd-foot">
          <span className="sd-foot-note">{G.info(13)}选项与所选窗口会自动保存 · 当前延迟 <b className="num">{delayTxt}</b></span>
          <div className="spacer" />
          <button className="btn" onClick={() => close()}>关闭</button>
        </div>
      </div>
    </div>, document.body);
};
