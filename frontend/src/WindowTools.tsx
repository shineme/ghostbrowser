// Window arrange + group-control syncer (群控同步器) for running profiles.
// Lives in the profiles view's run strip: two buttons, an arrange popover with
// a live layout preview, and the 多窗口同步器 dialog (SyncerDialog.tsx).
import React, { useEffect, useRef, useState } from 'react';
import { api, onEvent, Profile } from './api';
import { Icon, Segmented, useToast, Floating, Tip } from './ui';
import { SyncerDialog } from './SyncerDialog';

type Mode = 'grid' | 'cascade' | 'stack';
type Rect = { x: number; y: number; w: number; h: number };
type Layout = { mode: string; screen: Rect; cols: number; rows: number; cells: Rect[] };
type Follower = { id: string; name: string; ok: boolean; events: number; error?: string; latencyMs?: number; tabs?: number };
export type SyncOptions = { click: boolean; input: boolean; key: boolean; scroll: boolean; tabs: boolean; delayMin: number; delayMax: number };
export type SyncerState = { running: boolean; master: string; masterName: string; followers: Follower[]; events: number; lastEvent: string; startedAt: number; reason?: string; options?: SyncOptions; tabs?: number };

const MODE_LABEL: Record<Mode, string> = { grid: '网格', cascade: '层叠', stack: '重叠' };
const MODE_HINT: Record<Mode, string> = {
  grid: '平铺占满屏幕，每个窗口互不遮挡',
  cascade: '等大窗口斜向错开，标题栏都可见',
  stack: '所有窗口叠在同一位置，适合配合同步器',
};
const ACCENTS = ['#7c8cff', '#b07cff', '#4fe0d0', '#e58bff', '#3ee6a8', '#ffc857', '#60a5fa', '#ff8a9a'];
const initials = (n: string) => (n.match(/[A-Za-z0-9\u4e00-\u9fa5]+/g) || [n]).slice(0, 2).map(w => w[0]).join('').toUpperCase();
const errMsg = (e: any) => String(e?.message || e || '未知错误');

// ------------------------------------------------------------------ preview
const LayoutPreview: React.FC<{ layout: Layout | null; names: string[]; mode: Mode }> = ({ layout, names, mode }) => {
  const W = 316;
  const scr = layout?.screen && layout.screen.w > 0 ? layout.screen : { x: 0, y: 0, w: 1920, h: 1040 };
  const k = W / scr.w; const H = Math.round(scr.h * k);
  const cells = layout?.cells || [];
  // z-order mirrors the backend: cascade raises the last window, stack the first.
  const z = (i: number) => mode === 'stack' ? cells.length - i : i + 1;
  return (
    <div className="wt-screen" style={{ width: W, height: H + 12 }}>
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
const ArrangePop: React.FC<{ anchor: React.RefObject<HTMLElement | null>; running: Profile[]; selected: string[]; onClose: () => void }> = ({ anchor, running, selected, onClose }) => {
  const toast = useToast();
  const [mode, setMode] = useState<Mode>(() => (localStorage.getItem('gb.arrangeMode') as Mode) || 'grid');
  const [scope, setScope] = useState<'all' | 'sel'>(selected.length ? 'sel' : 'all');
  const [layout, setLayout] = useState<Layout | null>(null);
  const [busy, setBusy] = useState(false);
  const targets = scope === 'sel' && selected.length ? running.filter(p => selected.includes(p.id)) : running;

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

  return (
    <Floating open anchor={anchor} onClose={onClose} placement="bottom-end" gap={8} className="wt-pop">
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
    </Floating>);
};

// ------------------------------------------------------------------ strip buttons
export const WindowTools: React.FC<{ running: Profile[]; selected: string[]; bulk?: boolean }> = ({ running, selected }) => {
  const toast = useToast();
  const btnRef = useRef<HTMLButtonElement>(null);
  const [arr, setArr] = useState(false);
  const [panel, setPanel] = useState(false);
  const [st, setSt] = useState<SyncerState>({ running: false, master: '', masterName: '', followers: [], events: 0, lastEvent: '', startedAt: 0 });
  const userStop = useRef(false);
  const wasRunning = useRef(false);

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
      <Tip tip={none ? '没有正在运行的环境' : ''} placement="bottom-end" className="wt-tip">
        <button ref={btnRef} className={`btn sm ${arr ? 'on' : ''}`} disabled={none} onClick={() => setArr(o => !o)}><Icon.grid s={13} /> 排列窗口</button>
      </Tip>
      {st.running
        ? <button className="wt-pill" onClick={() => setPanel(p => !p)} title="打开同步器"><i className="wt-pulse" />同步中 · <b className="num">{okN}</b> 台</button>
        : <Tip tip={running.length < 2 ? '至少需要两个正在运行的环境' : ''} placement="bottom-end" className="wt-tip">
          <button className={`btn sm ${panel ? 'on' : ''}`} disabled={running.length < 2} onClick={() => setPanel(p => !p)}><Icon.bolt s={13} /> 同步器</button>
        </Tip>}
      {arr && <ArrangePop anchor={btnRef} running={running} selected={selected} onClose={() => setArr(false)} />}
      {panel && <SyncerDialog running={running} st={st} onClose={() => setPanel(false)} onStop={stop} />}
    </div>
  );
};
