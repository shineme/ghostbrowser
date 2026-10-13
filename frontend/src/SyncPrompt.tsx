import React, { useEffect, useState } from 'react';
import { api, onEvent } from './api';
import { Icon, ago } from './ui';

type Item = { id: string; name: string; conflict: boolean; cloudVersion: number; localVersion: number; cloudUpdatedAt: number; cloudDevice: string; localLastRun: number; size: number };
type Prompt = { items: Item[]; seconds: number; action: string; deadline: number; held: boolean };

const mb = (b: number) => b ? `${(b / 1048576).toFixed(b > 1e8 ? 0 : 1)} MB` : '';

// Start-up question: the cloud holds newer copies of local profiles.
export const SyncPrompt: React.FC = () => {
  const [p, setP] = useState<Prompt | null>(null);
  const [pick, setPick] = useState<Set<string>>(new Set());
  const [held, setHeld] = useState(false);
  const [now, setNow] = useState(Date.now());

  const open = (pr: Prompt | null) => {
    setP(pr);
    if (pr) { setPick(new Set(pr.action === 'keep' ? [] : pr.items.map(i => i.id))); setHeld(!!pr.held); }
  };
  useEffect(() => { api.PendingSyncPrompt().then((x: any) => x && open(x)).catch(() => {}); }, []);
  useEffect(() => onEvent('sync:event', (e: any) => {
    if (e.phase === 'prompt' && e.prompt) open(e.prompt);
    if (e.phase === 'prompt-closed') setP(null);
  }), []);
  useEffect(() => { if (!p || held) return; const t = setInterval(() => setNow(Date.now()), 200); return () => clearInterval(t); }, [p, held]);

  if (!p) return null;
  const left = Math.max(0, Math.ceil((p.deadline - now) / 1000));
  const hold = () => { if (!held) { setHeld(true); api.SyncHold(); } };
  const decide = async (ids: string[]) => { setP(null); await api.SyncDecide(ids); };
  const toggle = (id: string) => { hold(); setPick(s => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; }); };
  const auto = held ? '' : `（${left} 秒后${p.action === 'keep' ? '保留本地' : '自动拉取'}）`;

  return (
    <div className="overlay center" style={{ zIndex: 80 }}>
      <div className="modal" style={{ width: 'min(640px, 94vw)' }} onMouseDown={hold}>
        <div className="dhead"><div className="glyph"><Icon.cloud s={18} /></div>
          <div><h3>云端有更新的数据</h3><div className="sub">要用云端版本覆盖本地吗？{auto}</div></div>
        </div>
        <div className="dbody">
          {!held && <div className="sync-countdown"><i style={{ animationDuration: `${p.seconds}s` }} /></div>}
          <div className="panel list" style={{ maxHeight: 320, overflow: 'auto' }}>
            <table>
              <thead><tr><th style={{ width: 28 }} /><th>环境</th><th>云端</th><th>本地</th></tr></thead>
              <tbody>{p.items.map(it => <tr key={it.id} className={pick.has(it.id) ? 'sel' : ''} onClick={() => toggle(it.id)} style={{ cursor: 'pointer' }}>
                <td><input type="checkbox" checked={pick.has(it.id)} readOnly /></td>
                <td><div className="nm">{it.name}</div>{it.conflict && <span className="badge warn" title="本机上次运行后的改动还没上传；拉取会用云端版本替换它">本地也有未上传改动</span>}</td>
                <td className="num"><div>v{it.cloudVersion} · {ago(it.cloudUpdatedAt)}</div><div className="sub">{[it.cloudDevice, mb(it.size)].filter(Boolean).join(' · ')}</div></td>
                <td className="num"><div>v{it.localVersion}</div>{it.id !== '_global' && <div className="sub">上次运行 {ago(it.localLastRun)}</div>}</td>
              </tr>)}</tbody>
            </table>
          </div>
          <div className="sub" style={{ marginTop: 8 }}>勾选的拉取云端覆盖本地（本地旧版本会留一份备份）；未勾选的保留本地，并上传覆盖云端。</div>
        </div>
        <div className="dfoot">
          <button className="btn" onClick={() => decide([])}>全部保留本地</button>
          <div className="spacer" />
          <button className="btn primary" onClick={() => decide([...pick])}>{pick.size === p.items.length ? '全部拉取' : `拉取选中 ${pick.size} 个`}{!held && p.action !== 'keep' ? ` · ${left}s` : ''}</button>
        </div>
      </div>
    </div>
  );
};
