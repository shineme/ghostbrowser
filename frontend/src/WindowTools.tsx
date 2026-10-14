// Window arrange + group-control syncer (群控同步器) for running profiles.
// Lives in the profiles view's run strip: two buttons, an arrange popover with
// a live layout preview, and a floating syncer panel docked bottom-left.
import React, { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { api, onEvent, Profile } from './api';
import { Icon, Segmented, Avatar, useToast, avatarStyle } from './ui';

type Mode = 'grid' | 'cascade' | 'stack';
type Rect = { x: number; y: number; w: number; h: number };
type Layout = { mode: string; screen: Rect; cols: number; rows: number; cells: Rect[] };
type Follower = { id: string; name: string; ok: boolean; events: number; error?: string };
export type SyncerState = { running: boolean; master: string; masterName: string; followers: Follower[]; events: number; lastEvent: string; startedAt: number; reason?: string };

const MODE_LABEL: Record<Mode, string> = { grid: '网格', cascade: '层叠', stack: '重叠' };
const MODE_HINT: Record<Mode, string> = {
  grid: '平铺占满屏幕，每个窗口互不遮挡',
  cascade: '等大窗口斜向错开，标题栏都可见',
  stack: '所有窗口叠在同一位置，适合配合同步器',
};
const EV_LABEL: Record<string, string> = { click: '点击', scroll: '滚动', key: '按键', text: '输入', select: '选择', nav: '跳转' };
const ACCENTS = ['#7c8cff', '#b07cff', '#4fe0d0', '#e58bff', '#3ee6a8', '#ffc857', '#60a5fa', '#ff8a9a'];
const initials = (n: string) => (n.match(/[A-Za-z0-9\u4e00-\u9fa5]+/g) || [n]).slice(0, 2).map(w => w[0]).join('').toUpperCase();
const errMsg = (e: any) => String(e?.message || e || '未知错误');
const elapsed = (ms: number) => {
  const s = Math.max(0, Math.floor(ms / 1000)); const h = Math.floor(s / 3600), m = Math.floor(s % 3600 / 60), ss = s % 60;
  const p = (x: number) => String(x).padStart(2, '0');
  return h ? `${h}:${p(m)}:${p(ss)}` : `${p(m)}:${p(ss)}`;
};

/** Closes on outside mousedown / Escape. `refs` are the elements counted as inside. */
const useDismiss = (open: boolean, close: () => void, refs: React.RefObject<HTMLElement | null>[]) => {
  useEffect(() => {
    if (!open) return;
    const down = (e: MouseEvent) => { if (!refs.some(r => r.current?.contains(e.target as Node))) close(); };
    const key = (e: KeyboardEvent) => { if (e.key === 'Escape') close(); };
    window.addEventListener('mousedown', down); window.addEventListener('keydown', key);
    return () => { window.removeEventListener('mousedown', down); window.removeEventListener('keydown', key); };
  }, [open]);
};

// ------------------------------------------------------------------ preview
const LayoutPreview: React.FC<{ layout: Layout | null; names: string[]; mode: Mode }> = ({ layout, names, mode }) => {
  const W = 316;
  const scr = layout?.screen && layout.screen.w > 0 ? layout.screen : { x: 0, y: 0, w: 1920, h: 1040 };
  const k = W / scr.w; const H = Math.round(scr.h * k);
  const cells = layout?.cells || [];
  // z-order mirrors the backend: cascade raises the last window, stack the first.
  const z = (i: number) => mode === 'stack' ? cells.length - i : i + 1;
  return (
    <div className="wt-screen" style={{ width: W, height: H }}>
      <div className="wt-bar"><i /><i /><i /><span className="num">{scr.w}×{scr.h}</span></div>
      {cells.map((c, i) => {
        const w = c.w * k, h = c.h * k; const col = ACCENTS[i % ACCENTS.length];
        const small = w < 64 || h < 30; const nm = names[i] || `窗口 ${i + 1}`;
        return <div key={i} className="wt-cell" title={nm}
          style={{ left: (c.x - scr.x) * k, top: (c.y - scr.y) * k, width: w, height: h, zIndex: z(i), ['--c' as any]: col }}>
          <span className="wt-dot" />
          {small ? <b className="ini">{initials(nm)}</b> : <b>{nm}</b>}
        </div>;
      })}
      {mode === 'stack' && cells.length > 1 && <span className="wt-stackn num">×{cells.length}</span>}
      {!cells.length && <div className="wt-none">没有可排列的窗口</div>}
    </div>
  );
};

// ------------------------------------------------------------------ arrange popover
const ArrangePop: React.FC<{ anchor: HTMLElement; running: Profile[]; selected: string[]; onClose: () => void; popRef: React.RefObject<HTMLDivElement | null> }> = ({ anchor, running, selected, onClose, popRef }) => {
  const toast = useToast();
  const [mode, setMode] = useState<Mode>(() => (localStorage.getItem('gb.arrangeMode') as Mode) || 'grid');
  const [scope, setScope] = useState<'all' | 'sel'>(selected.length ? 'sel' : 'all');
  const [layout, setLayout] = useState<Layout | null>(null);
  const [busy, setBusy] = useState(false);
  const [pos, setPos] = useState({ top: 0, left: 0 });
  const targets = scope === 'sel' && selected.length ? running.filter(p => selected.includes(p.id)) : running;

  useLayoutEffect(() => {
    const r = anchor.getBoundingClientRect(); const w = 348;
    setPos({ top: r.bottom + 8, left: Math.max(12, Math.min(window.innerWidth - w - 12, r.right - w)) });
  }, [anchor]);
  useEffect(() => { if (!selected.length && scope === 'sel') setScope('all'); }, [selected.length]);
  useEffect(() => {
    let live = true;
    api.ArrangePreview(targets.length, mode).then((l: any) => live && setLayout(l)).catch(() => live && setLayout(null));
    return () => { live = false; };
  }, [targets.length, mode]);

  const apply = async () => {
    setBusy(true);
    try {
      localStorage.setItem('gb.arrangeMode', mode);
      const res: any = await api.ArrangeWindows(scope === 'sel' ? targets.map(p => p.id) : [], mode);
      const failed: string[] = res?.failed || [];
      if (failed.length) toast(`已${MODE_LABEL[mode]}排列 ${res.arranged}/${res.total} 个窗口，${failed.length} 个失败：${failed.join('；')}`, 'bad');
      else toast(`已${MODE_LABEL[mode]}排列 ${res?.arranged ?? targets.length} 个窗口`);
      onClose();
    } catch (e) { toast(`排列失败：${errMsg(e)}`, 'bad'); }
    finally { setBusy(false); }
  };

  return createPortal(
    <div className="wt-pop" ref={popRef} style={pos}>
      <div className="wt-pop-h"><span className="wt-glyph"><Icon.grid s={15} /></span><div><b>排列窗口</b><span>一键整理正在运行的浏览器窗口</span></div></div>
      <Segmented<Mode> className="compact block wt-seg" value={mode} onChange={setMode}
        items={[['grid', '网格', <Icon.grid />], ['cascade', '层叠', <Icon.layers />], ['stack', '重叠', <Icon.copy />]]} />
      <LayoutPreview layout={layout} mode={mode} names={targets.map(p => p.name)} />
      <div className="wt-hint">{MODE_HINT[mode]}{mode === 'grid' && layout && layout.cols > 0 && <span className="num"> · {layout.cols}×{layout.rows}</span>}</div>
      <div className="wt-scope">
        <button className={scope === 'all' ? 'on' : ''} onClick={() => setScope('all')}><span className="radio" />全部运行中<em className="num">{running.length}</em></button>
        <button className={scope === 'sel' ? 'on' : ''} disabled={!selected.length} title={selected.length ? '' : '在列表中勾选正在运行的环境后可用'} onClick={() => setScope('sel')}><span className="radio" />仅已选中<em className="num">{selected.length}</em></button>
      </div>
      <div className="wt-foot">
        <button className="btn ghost sm" onClick={onClose}>取消</button>
        <button className="btn primary sm" disabled={busy || !targets.length} onClick={apply}>{busy ? '排列中…' : <><Icon.check s={13} /> 应用</>}</button>
      </div>
    </div>, document.body);
};

// ------------------------------------------------------------------ syncer panel
const SyncerPanel: React.FC<{ running: Profile[]; st: SyncerState; lift: boolean; onHide: () => void; onStop: () => void }> = ({ running, st, lift, onHide, onStop }) => {
  const toast = useToast();
  const [master, setMaster] = useState<string>(() => st.master && running.some(p => p.id === st.master) ? st.master : running[0]?.id || '');
  const [off, setOff] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState(false);
  const [, tick] = useState(0);
  const [flash, setFlash] = useState(0);
  useEffect(() => { if (!st.running) return; const t = setInterval(() => tick(x => x + 1), 1000); return () => clearInterval(t); }, [st.running]);
  useEffect(() => { if (st.running) setFlash(f => f + 1); }, [st.events]);
  useEffect(() => { if (!running.some(p => p.id === master)) setMaster(running[0]?.id || ''); }, [running.map(p => p.id).join()]);

  const others = running.filter(p => p.id !== master);
  const followers = others.filter(p => !off.has(p.id));
  const toggle = (id: string) => setOff(s => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const allOn = followers.length === others.length;
  const start = async () => {
    setBusy(true);
    try { await api.StartSync(master, followers.map(p => p.id)); }
    catch (e) { toast(`无法开始同步：${errMsg(e)}`, 'bad'); }
    finally { setBusy(false); }
  };
  const byId = Object.fromEntries(running.map(p => [p.id, p]));
  const okN = st.followers.filter(f => f.ok).length;

  return createPortal(
    <div className={`syncp ${lift ? 'lift' : ''} ${st.running ? 'live' : ''}`}>
      <div className="syncp-h">
        <span className={`xorb ${st.running ? 'on' : ''}`}><Icon.bolt s={14} /></span>
        <div className="grow"><b>群控同步器</b><span>{st.running ? `正在同步 · ${okN} 台跟随` : '在主控窗口操作，其余窗口同步执行'}</span></div>
        <button className="ib" title="收起" onClick={onHide}><Icon.x s={14} /></button>
      </div>

      {!st.running ? <>
        <div className="syncp-sec">主控环境<span>你亲手操作的窗口</span></div>
        <div className="syncp-list">
          {running.map(p => <button key={p.id} className={`syncp-row ${p.id === master ? 'on' : ''}`} onClick={() => setMaster(p.id)}>
            <Avatar seed={p.fingerprint.noiseSeed || p.id} os={p.fingerprint.os} />
            <span className="nm">{p.name}</span>
            {p.id === master && <span className="badge ok">主控</span>}
            <span className="radio" />
          </button>)}
        </div>
        <div className="syncp-sec">跟随环境<span className="num">{followers.length} / {others.length}</span>
          {others.length > 0 && <button className="tb-clear" onClick={() => setOff(allOn ? new Set(others.map(p => p.id)) : new Set())}>{allOn ? '全不选' : '全选'}</button>}
        </div>
        <div className="syncp-list">
          {others.length ? others.map(p => <button key={p.id} className={`syncp-row ${off.has(p.id) ? '' : 'on'}`} onClick={() => toggle(p.id)}>
            <span className={`chk ${off.has(p.id) ? '' : 'on'}`}><Icon.check s={11} /></span>
            <span className="avatar xs" style={avatarStyle(p.fingerprint.noiseSeed || p.id)}>{initials(p.name)}</span>
            <span className="nm">{p.name}</span>
          </button>) : <div className="syncp-empty">至少需要两个正在运行的环境</div>}
        </div>
        <div className="syncp-note"><Icon.alert s={13} /><span>只同步每个窗口的<b>第一个标签页</b>，不进入 iframe；输入法候选与文件选择框不会同步。</span></div>
        <div className="syncp-foot">
          <button className="btn ghost sm" onClick={onHide}>取消</button>
          <button className="btn primary sm" disabled={busy || !master || !followers.length} onClick={start}>{busy ? '连接中…' : <><Icon.play s={12} /> 开始同步</>}</button>
        </div>
      </> : <>
        <div className="syncp-master">
          {byId[st.master] ? <Avatar seed={byId[st.master].fingerprint.noiseSeed || st.master} os={byId[st.master].fingerprint.os} /> : <span className="avatar" style={avatarStyle(st.master)}>{initials(st.masterName)}</span>}
          <div className="grow"><span className="lbl"><i className="pulse" />主控</span><b>{st.masterName || '主控环境'}</b></div>
          <div className="syncp-last" key={flash}>
            <span>最近动作</span><b>{EV_LABEL[st.lastEvent] || '等待操作'}</b>
          </div>
        </div>
        <div className="syncp-stats">
          <div><span>同步事件</span><b className="num">{st.events}</b></div>
          <div><span>跟随在线</span><b className="num">{okN}<small>/{st.followers.length}</small></b></div>
          <div><span>已运行</span><b className="num">{elapsed(Date.now() - st.startedAt)}</b></div>
        </div>
        <div className="syncp-sec">跟随环境</div>
        <div className="syncp-list">
          {st.followers.map(f => <div key={f.id} className={`syncp-row static ${f.ok ? '' : 'bad'}`}>
            <span className="avatar xs" style={avatarStyle(byId[f.id]?.fingerprint.noiseSeed || f.id)}>{initials(f.name)}</span>
            <div className="nm2"><span className="nm">{f.name}</span>{!f.ok && <span className="err" title={f.error}>{f.error || '已断开'}</span>}</div>
            <span className="cnt num">{f.events}</span>
            <span className={`badge ${f.ok ? 'ok' : 'fail'}`}>{f.ok ? '同步中' : '已断开'}</span>
          </div>)}
        </div>
        {st.reason && <div className="syncp-note warn"><Icon.alert s={13} /><span title={st.reason}>{st.reason}</span></div>}
        <div className="syncp-foot">
          <button className="btn ghost sm" onClick={onHide}>收起</button>
          <button className="btn sm stop" onClick={onStop}><Icon.stop s={12} /> 停止同步</button>
        </div>
      </>}
    </div>, document.body);
};

// ------------------------------------------------------------------ strip buttons
export const WindowTools: React.FC<{ running: Profile[]; selected: string[]; bulk?: boolean }> = ({ running, selected, bulk = false }) => {
  const toast = useToast();
  const btnRef = useRef<HTMLButtonElement>(null);
  const popRef = useRef<HTMLDivElement>(null);
  const [arr, setArr] = useState(false);
  const [panel, setPanel] = useState(false);
  const [st, setSt] = useState<SyncerState>({ running: false, master: '', masterName: '', followers: [], events: 0, lastEvent: '', startedAt: 0 });
  const userStop = useRef(false);
  const wasRunning = useRef(false);
  useDismiss(arr, () => setArr(false), [btnRef, popRef]);

  useEffect(() => {
    const take = (s: SyncerState) => {
      if (!s) return;
      if (wasRunning.current && !s.running) {
        if (userStop.current) toast(`同步已停止，共同步 ${s.events} 个事件`);
        else toast(s.reason ? `同步已自动停止：${s.reason}` : '同步已自动停止', 'bad');
        userStop.current = false;
      }
      wasRunning.current = s.running;
      setSt({ ...s, followers: s.followers || [] });
    };
    api.SyncerState().then((s: any) => { wasRunning.current = !!s?.running; take(s); }).catch(() => {});
    return onEvent('syncer:event', take);
  }, []);

  const stop = async () => { userStop.current = true; try { const s: any = await api.StopSync(); if (s && wasRunning.current) { /* event will toast */ } } catch (e) { userStop.current = false; toast(errMsg(e), 'bad'); } };
  const none = running.length === 0;
  const okN = st.followers.filter(f => f.ok).length;

  return (
    <div className="wt-tools">
      <span className="wt-tip" data-tip={none ? '没有正在运行的环境' : ''}>
        <button ref={btnRef} className={`btn sm ${arr ? 'on' : ''}`} disabled={none} onClick={() => setArr(o => !o)}><Icon.grid s={13} /> 排列窗口</button>
      </span>
      {st.running
        ? <button className="wt-pill" onClick={() => setPanel(p => !p)} title="打开同步器"><i className="pulse" />同步中 · <b className="num">{okN}</b> 台</button>
        : <span className="wt-tip" data-tip={running.length < 2 ? '至少需要两个正在运行的环境' : ''}>
          <button className={`btn sm ${panel ? 'on' : ''}`} disabled={running.length < 2} onClick={() => setPanel(p => !p)}><Icon.bolt s={13} /> 同步器</button>
        </span>}
      {arr && btnRef.current && <ArrangePop anchor={btnRef.current} running={running} selected={selected} popRef={popRef} onClose={() => setArr(false)} />}
      {panel && <SyncerPanel running={running} st={st} lift={bulk} onHide={() => setPanel(false)} onStop={stop} />}
    </div>
  );
};
