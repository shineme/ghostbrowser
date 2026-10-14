import React, { useCallback, useEffect, useRef, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api } from './api';
import { Icon, Avatar, ago, useToast, useConfirm, useExit } from './ui';

// 最近删除 (recycle bin): profiles deleted on this device stay here with their
// data folder for Settings.trashDays (default 30) before they are purged.
export type TrashItem = {
  id: string; name: string; group?: string; fingerprint?: any; trashedAt: number; expiresAt: number;
  size: number; hasData: boolean; cloudNote: string; restoreAsNew: boolean;
};

const size = (b?: number) => !b || b < 0 ? '—' : b >= 1 << 30 ? `${(b / (1 << 30)).toFixed(1)} GB` : b >= 1 << 20 ? `${(b / (1 << 20)).toFixed(1)} MB` : b >= 1024 ? `${Math.round(b / 1024)} KB` : `${b} B`;
const pad = (n: number) => String(n).padStart(2, '0');
const exact = (ts: number) => { const d = new Date(ts * 1000); return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`; };
const daysLeft = (exp: number) => Math.max(0, Math.ceil((exp - Date.now() / 1000) / 86400));

/** Small count used by the 最近删除 button on the profiles toolbar. */
export const useTrashCount = (dep: unknown) => {
  const [n, setN] = useState(0);
  const refresh = useCallback(() => { api.ListTrash().then((l: any[]) => setN((l || []).length)).catch(() => {}); }, []);
  useEffect(refresh, [refresh, dep]);
  return [n, refresh] as const;
};

export const TrashModal: React.FC<{ onClose: () => void; onChanged: () => void }> = ({ onClose, onChanged }) => {
  const toast = useToast();
  const confirm = useConfirm();
  const asking = useRef(false); // the confirm dialog handles Escape itself
  const ask = async (o: Parameters<typeof confirm>[0]) => { asking.current = true; try { return await confirm(o); } finally { setTimeout(() => { asking.current = false; }, 0); } };
  const [out, close] = useExit(onClose);
  const [items, setItems] = useState<TrashItem[] | null>(null);
  const [err, setErr] = useState('');
  const [days, setDays] = useState(30);
  const [busy, setBusy] = useState<Set<string>>(new Set());
  const [emptying, setEmptying] = useState(false);

  const load = useCallback(async () => {
    try { setItems(((await api.ListTrash()) || []) as any); setErr(''); }
    catch (e: any) { setErr(String(e?.message || e)); setItems([]); }
  }, []);
  useEffect(() => { load(); api.TrashDays().then(setDays).catch(() => {}); }, [load]);
  useEffect(() => {
    const k = (e: KeyboardEvent) => { if (e.key === 'Escape' && !emptying && !asking.current) close(); };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  });

  const mark = (id: string, on: boolean) => setBusy(s => { const n = new Set(s); on ? n.add(id) : n.delete(id); return n; });

  const restore = async (it: TrashItem) => {
    mark(it.id, true);
    try {
      const r: any[] = (await api.RestoreFromTrash([it.id])) || [];
      const e = r.find(x => x.error);
      if (e) toast(`「${it.name}」恢复失败：${e.error}`, 'bad');
      else toast(it.restoreAsNew ? `已恢复「${it.name}」（云端副本已删除，作为新的本地环境）` : `已恢复「${it.name}」`);
      await load(); onChanged();
    } catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { mark(it.id, false); }
  };

  const purge = async (it: TrashItem) => {
    const ok = await ask({ title: `彻底删除「${it.name}」？`, body: <>环境记录和数据目录（Cookie、缓存、扩展状态{it.size ? `，约 ${size(it.size)}` : ''}）会被永久删除，<b>无法恢复</b>。{it.cloudNote === '云端副本保留' ? '云端副本不受影响。' : ''}</>, ok: '彻底删除', danger: true });
    if (!ok) return;
    mark(it.id, true);
    try { await api.PurgeTrash([it.id]); toast(`已彻底删除「${it.name}」`); await load(); onChanged(); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { mark(it.id, false); }
  };

  const empty = async () => {
    if (!items?.length) return;
    const total = items.reduce((s, x) => s + (x.size || 0), 0);
    const ok = await ask({ title: `清空最近删除？`, body: <>{items.length} 个环境{total ? `（约 ${size(total)}）` : ''}会被永久删除，<b>无法恢复</b>。</>, ok: '清空', danger: true });
    if (!ok) return;
    setEmptying(true);
    try { const n = await api.EmptyTrash(); toast(`已彻底删除 ${n} 个环境`); await load(); onChanged(); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { setEmptying(false); }
  };

  const total = (items || []).reduce((s, x) => s + (x.size || 0), 0);

  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && !emptying && close()}>
      <div className="modal tall trash">
        <div className="dhead">
          <div className="glyph trash-glyph"><Icon.trash s={18} /></div>
          <div className="trash-h"><h3>最近删除</h3><div className="sub">删除的环境连同数据目录保留 {days} 天，可随时恢复；到期后自动彻底删除</div></div>
          <div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" disabled={emptying} onClick={() => close()} aria-label="关闭"><Icon.x /></button></div>
        </div>
        <div className="dbody trash-body">
          {err && <div className="notice bad trash-err"><Icon.alert s={15} /><span>{err}</span></div>}
          {items === null ? <div className="trash-load"><ThinkingOrb state="connecting" size={32} theme="dark" /><span>正在读取…</span></div>
            : items.length === 0 ? <div className="empty trash-empty">
              <div className="halo"><ThinkingOrb state="breathing" size={64} theme="dark" /></div>
              <h3>最近删除是空的</h3>
              <div>在环境列表中删除的环境会先放到这里，保留 {days} 天，误删可以一键恢复。</div>
            </div>
              : <div className="trash-list" role="list">
                {items.map((it, i) => {
                  const left = daysLeft(it.expiresAt); const b = busy.has(it.id);
                  return <div key={it.id} role="listitem" className={`trash-row ${b ? 'busy' : ''}`} style={{ ['--i' as any]: Math.min(i, 12) }}>
                    <Avatar seed={it.id} os={it.fingerprint?.os || 'windows'} />
                    <div className="trash-main">
                      <div className="trash-name" title={it.name}>{it.name}</div>
                      <div className="trash-meta">
                        <span title={exact(it.trashedAt)}>删除于 {ago(it.trashedAt)}</span>
                        <span className="dot" />
                        <span className="num">{it.hasData ? size(it.size) : '无数据目录'}</span>
                        {it.group && <><span className="dot" /><span className="trash-grp" title={it.group}>{it.group}</span></>}
                      </div>
                    </div>
                    <div className="trash-tags">
                      <span className={`badge ${it.cloudNote === '仅本机' ? 'local' : it.restoreAsNew ? 'unknown' : 'cloud'}`} title={it.restoreAsNew ? '云端副本已删除，恢复后作为新的本地环境，自动同步会重新上传' : undefined}>{it.restoreAsNew ? <Icon.cloud /> : it.cloudNote === '仅本机' ? <Icon.laptop /> : <Icon.cloud />}{it.cloudNote}</span>
                      <span className={`trash-left ${left <= 3 ? 'warn' : ''}`} title={`${exact(it.expiresAt)} 自动彻底删除`}>{left > 0 ? `剩 ${left} 天` : '即将删除'}</span>
                    </div>
                    <div className="trash-acts">
                      <button className="btn sm" disabled={b || emptying} onClick={() => restore(it)}>{b ? <ThinkingOrb state="weaving" size={20} theme="dark" /> : <Icon.refresh s={13} />} 恢复</button>
                      <button className="btn sm ghost danger icon" disabled={b || emptying} onClick={() => purge(it)} title="彻底删除" aria-label={`彻底删除 ${it.name}`}><Icon.trash s={13} /></button>
                    </div>
                  </div>;
                })}
              </div>}
        </div>
        <div className="dfoot">
          <span className="trash-sum">{items && items.length > 0 ? <>共 <b className="num">{items.length}</b> 个环境{total ? <> · 占用 <b className="num">{size(total)}</b></> : null}</> : ' '}</span>
          <div className="spacer" />
          <button className="btn" disabled={emptying} onClick={() => close()}>关闭</button>
          <button className="btn solid-bad" disabled={emptying || !items?.length} onClick={empty}>
            {emptying ? <><ThinkingOrb state="weaving" size={20} theme="dark" /> 正在清空…</> : <><Icon.trash s={14} /> 清空回收站</>}
          </button>
        </div>
      </div>
    </div>
  );
};
