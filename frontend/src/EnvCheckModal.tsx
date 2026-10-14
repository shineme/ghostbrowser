import React, { useCallback, useEffect, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api } from './api';
import { Icon, Segmented, useToast, useConfirm, useExit, ago } from './ui';

// 环境检测 / 一键修复 / 修复记录 / 系统时钟
export type EnvItem = { key: string; label: string; status: 'pass' | 'warn' | 'fail' | 'unknown'; actual: string; expected: string; detail: string; fixable: boolean; fixHint: string };
export type ClockReport = { status: 'pass' | 'fail' | 'unknown'; skewSecs: number; source: string; localTime: number; serverTime: number; error: string; detail: string };
export type EnvReport = { profileId: string; profileName: string; kernel: string; checkedAt: number; items: EnvItem[]; pass: number; warn: number; fail: number; fixable: string[]; clock: ClockReport };
type Change = { item: string; label: string; field: string; old: string; new: string };
type Entry = { id: string; at: number; kind: 'repair' | 'restore'; items: string[]; changes: Change[]; skipped: string[]; restored: boolean; restoredAt?: number; restoreOf?: string };

const ITEM_CN: Record<string, string> = { language: '语言', timezone: '时区', geo: '地理位置', webrtc: 'WebRTC', seed: '噪声种子', fonts: '字体', webdriver: 'webdriver', clock: '系统时钟', exit: '出口 IP', ua: 'UA' };
const ST: Record<string, [string, string]> = { pass: ['ok', '通过'], warn: ['warn', '注意'], fail: ['fail', '异常'], unknown: ['unknown', '未确认'] };
const fmtTime = (t: number) => { const d = new Date(t * 1000); const p = (n: number) => String(n).padStart(2, '0'); return `${d.getMonth() + 1}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`; };
const errText = (e: any) => String(e?.message || e || '未知错误');

const Badge: React.FC<{ s: string }> = ({ s }) => { const [c, t] = ST[s] || ST.unknown; return <span className={`badge ${c}`}><i className="d" />{t}</span>; };

export const ClockCard: React.FC<{ clock: ClockReport | null; busy: boolean; onRecheck: () => void }> = ({ clock, busy, onRecheck }) => {
  const st = clock?.status || 'unknown';
  const skew = clock?.skewSecs || 0;
  const big = !clock || st === 'unknown' ? '尚未确认' : `${skew > 0 ? '+' : ''}${skew}`;
  // Meter: 0 .. 300 s, 2-min threshold at 40%.
  const pct = Math.min(100, Math.abs(skew) / 300 * 100);
  return (
    <div className={`ec-clock ${st}`}>
      <div className="ec-clock-ic"><Icon.alert s={16} /></div>
      <div className="ec-clock-main">
        <div className="ec-clock-h"><b>系统时钟</b><Badge s={st} />{clock?.source && <span className="ec-src" title={clock.source}>参考 {clock.source.replace(/^https?:\/\//, '').replace(/\/.*$/, '')}</span>}</div>
        <div className="ec-clock-d" title={clock?.detail}>{busy && !clock ? '正在比对 HTTPS 服务器时间…' : clock?.detail || '尚未确认'}</div>
        {st !== 'unknown' && <div className="ec-meter" aria-hidden><i style={{ width: `${pct}%` }} /><span className="ec-th" title="2 分钟" /></div>}
      </div>
      <div className="ec-clock-v">
        <div className={`ec-skew num ${st === 'unknown' ? 'na' : ''}`}>{big}{st !== 'unknown' && <small>秒</small>}</div>
        <button className="btn sm ghost" disabled={busy} onClick={onRecheck}>{busy ? <ThinkingOrb state="connecting" size={20} theme="dark" /> : <Icon.refresh s={13} />} 重新比对</button>
      </div>
    </div>
  );
};

export const EnvCheckModal: React.FC<{ profileId: string; name: string; onClose: () => void; onChanged?: () => void }> = ({ profileId, name, onClose, onChanged }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [out, close] = useExit(onClose);
  const [tab, setTab] = useState<'check' | 'log'>('check');
  const [rep, setRep] = useState<EnvReport | null>(null);
  const [err, setErr] = useState('');
  const [loading, setLoading] = useState(true);
  const [fixing, setFixing] = useState<string>(''); // '' | '*' | item key
  const [clock, setClock] = useState<ClockReport | null>(null);
  const [clockBusy, setClockBusy] = useState(false);
  const [log, setLog] = useState<Entry[] | null>(null);
  const [logErr, setLogErr] = useState('');
  const [restoring, setRestoring] = useState('');

  const run = useCallback(async () => {
    setLoading(true); setErr('');
    try { const r: EnvReport = await (api as any).CheckEnvironment(profileId); setRep(r); setClock(r.clock); }
    catch (e) { setErr(errText(e)); }
    finally { setLoading(false); }
  }, [profileId]);
  const loadLog = useCallback(async () => {
    setLogErr('');
    try { setLog((await (api as any).RepairHistory(profileId)) || []); } catch (e) { setLogErr(errText(e)); setLog([]); }
  }, [profileId]);
  useEffect(() => { run(); loadLog(); }, [run, loadLog]);
  useEffect(() => {
    const k = (e: KeyboardEvent) => { if (e.key === 'Escape' && !fixing && !restoring) close(); };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  });

  const recheckClock = async () => { setClockBusy(true); try { setClock(await (api as any).CheckSystemClock()); } catch (e) { toast(errText(e), 'bad'); } finally { setClockBusy(false); } };

  const repair = async (items: string[], key: string) => {
    setFixing(key);
    try {
      const r: any = await (api as any).RepairEnvironment(profileId, items);
      if (r.changed > 0) toast(`已修复 ${r.entry?.items?.length || items.length} 项，修改 ${r.changed} 个字段，可在「修复记录」中恢复`);
      else toast('没有需要修改的配置', 'bad');
      if (r.skipped?.length) toast(r.skipped[0], 'bad');
      onChanged?.();
      await Promise.all([run(), loadLog()]);
    } catch (e) { toast(errText(e), 'bad'); }
    finally { setFixing(''); }
  };

  const restore = async (e: Entry) => {
    const ok = await ask({ title: `恢复 ${fmtTime(e.at)} 的修复？`, body: `将把 ${e.changes.length} 个字段改回修复前的值（${e.items.map(i => ITEM_CN[i] || i).join('、')}）。之后对其他设置的修改不受影响。`, ok: '恢复' });
    if (!ok) return;
    setRestoring(e.id);
    try { await (api as any).RestoreRepair(profileId, e.id); toast('已恢复到修复前的配置'); onChanged?.(); await Promise.all([loadLog(), run()]); }
    catch (x) { toast(errText(x), 'bad'); }
    finally { setRestoring(''); }
  };

  const items = (rep?.items || []).filter(i => i.key !== 'clock');
  const fixable = rep?.fixable || [];
  const busy = !!fixing || !!restoring;
  const logCount = log?.length || 0;

  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && !busy && close()}>
      <div className="modal tall envchk">
        <div className="dhead">
          <div className="glyph"><Icon.shield s={18} /></div>
          <div className="cdel-h"><h3 title={`环境检测 · ${name}`}>环境检测 · {name}</h3>
            <div className="sub">{rep ? `${rep.kernel === 'chromium' ? '开源内核' : rep.kernel === 'firefox' ? 'Firefox 内核' : '定制内核'} · 检测于 ${ago(rep.checkedAt)}` : '对照出口 IP 检查指纹配置的一致性'}</div></div>
          <div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" disabled={busy} onClick={() => close()} aria-label="关闭"><Icon.x /></button></div>
        </div>
        <div className="ec-tabs">
          <Segmented className="compact" value={tab} onChange={setTab} items={[['check', '检测结果', <Icon.shield s={14} />], ['log', logCount ? `修复记录 · ${logCount}` : '修复记录', <Icon.refresh s={14} />]]} />
        </div>
        <div className="dbody ec-body">
          {tab === 'check' ? <>
            {err ? <div className="notice bad"><Icon.alert /><span className="grow">检测失败：{err}</span><button className="btn sm" onClick={run}>重试</button></div> : null}
            <div className="ec-sum">
              {loading && !rep ? <div className="ec-sum-load"><ThinkingOrb state="connecting" size={32} theme="dark" /><div><b>正在检测…</b><div className="sub">查询出口 IP、比对系统时钟、检查指纹一致性</div></div></div>
                : rep ? <>
                  <div className="ec-cnt ok"><b className="num">{rep.pass}</b><span>通过</span></div>
                  <div className="ec-cnt warn"><b className="num">{rep.warn}</b><span>注意</span></div>
                  <div className="ec-cnt fail"><b className="num">{rep.fail}</b><span>异常</span></div>
                  <div className="spacer" />
                  <button className="btn sm ghost" disabled={loading || busy} onClick={run}>{loading ? <ThinkingOrb state="connecting" size={20} theme="dark" /> : <Icon.refresh s={13} />} 重新检测</button>
                  <button className="btn sm primary" disabled={!fixable.length || loading || busy} onClick={() => repair(fixable, '*')} title={fixable.length ? fixable.map(k => ITEM_CN[k] || k).join('、') : '没有可自动修复的项目'}>
                    {fixing === '*' ? <><ThinkingOrb state="weaving" size={20} theme="dark" /> 修复中…</> : <><Icon.sparkles s={13} /> {fixable.length ? `一键修复 ${fixable.length} 项` : '无需修复'}</>}
                  </button>
                </> : null}
            </div>
            <ClockCard clock={clock} busy={clockBusy || (loading && !clock)} onRecheck={recheckClock} />
            <div className={`ec-list ${loading && rep ? 'dim' : ''}`}>
              {loading && !rep ? Array.from({ length: 6 }).map((_, i) => <div key={i} className="ec-row skel" style={{ ['--i' as any]: i }}><i /><i /><i /></div>)
                : items.map((it, i) => <div key={it.key} className={`ec-row ${it.status}`} style={{ ['--i' as any]: i }}>
                  <div className="ec-st"><Badge s={it.status} /></div>
                  <div className="ec-what"><div className="ec-label">{it.label}</div><div className="ec-detail" title={it.detail}>{it.detail}</div></div>
                  <div className="ec-vals">
                    <div className="ec-kv"><span>当前</span><b title={it.actual}>{it.actual || '—'}</b></div>
                    <div className="ec-kv"><span>期望</span><b title={it.expected}>{it.expected || '—'}</b></div>
                  </div>
                  <div className="ec-act">
                    {it.status === 'pass' ? <span className="ec-okmark"><Icon.check s={14} /></span>
                      : it.fixable ? <button className="btn sm" disabled={busy || loading} title={it.fixHint} onClick={() => repair([it.key], it.key)}>{fixing === it.key ? <ThinkingOrb state="weaving" size={20} theme="dark" /> : '修复'}</button>
                        : <span className="ec-manual" title={it.key === 'ua' ? '请在编辑里重新生成指纹' : it.key === 'exit' ? '请检查或更换代理' : '只能检测'}>仅检测</span>}
                  </div>
                </div>)}
            </div>
            {rep && <div className="ec-foot-note"><Icon.alert s={13} />出口 IP、UA 一致性、系统时钟只能检测，无法自动修复；webdriver 按配置判断，不读取运行中的浏览器。修复前会自动备份原值。</div>}
          </> : <>
            {logErr && <div className="notice bad"><Icon.alert /><span className="grow">读取修复记录失败：{logErr}</span><button className="btn sm" onClick={loadLog}>重试</button></div>}
            {log === null ? <div className="cdel-load"><ThinkingOrb state="connecting" size={32} theme="dark" /></div>
              : log.length === 0 && !logErr ? <div className="empty ec-empty"><div className="halo"><ThinkingOrb state="breathing" size={64} theme="dark" /></div><h3>还没有修复记录</h3><div>一键修复或单项修复后，这里会记录修改的字段和原值，可随时恢复。</div></div>
                : <div className="ec-log">{log.map((e, i) => <div key={e.id} className={`ec-entry ${e.kind} ${e.restored ? 'restored' : ''}`} style={{ ['--i' as any]: i }}>
                  <div className="ec-entry-h">
                    <span className={`badge ${e.kind === 'restore' ? 'cy' : e.restored ? 'unknown' : 'ok'}`}>{e.kind === 'restore' ? '恢复' : e.restored ? '已恢复' : '修复'}</span>
                    <b className="num">{fmtTime(e.at)}</b>
                    <span className="ec-items" title={e.items.map(k => ITEM_CN[k] || k).join('、')}>{e.items.map(k => <span key={k} className="ec-chip">{ITEM_CN[k] || k}</span>)}</span>
                    <span className="spacer" />
                    {e.kind === 'repair' && !e.restored && <button className="btn sm ghost" disabled={busy} onClick={() => restore(e)}>{restoring === e.id ? <ThinkingOrb state="weaving" size={20} theme="dark" /> : <Icon.refresh s={13} />} 恢复</button>}
                    {e.restored && e.restoredAt && <span className="sub">恢复于 {fmtTime(e.restoredAt)}</span>}
                  </div>
                  <div className="ec-changes">{e.changes.map((c, j) => <div key={j} className="ec-chg">
                    <span className="ec-f">{c.label || c.field}</span>
                    <span className="ec-old" title={c.old}>{c.old}</span>
                    <span className="ec-arrow">→</span>
                    <span className="ec-new" title={c.new}>{c.new}</span>
                  </div>)}</div>
                  {e.skipped?.length > 0 && <div className="ec-skip" title={e.skipped.join('\n')}><Icon.alert s={12} />{e.skipped.join('；')}</div>}
                </div>)}</div>}
          </>}
        </div>
      </div>
    </div>
  );
};
