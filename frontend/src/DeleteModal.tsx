import React, { useEffect, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, Settings } from './api';
import { Icon, useToast, useConfirm, useExit } from './ui';

// Cloud-aware delete dialog: "只删本地" or "同时删云端".
export type DelTarget = { id: string; name: string; inCloud: boolean; running?: boolean };
type Mode = 'local' | 'cloud';
type Result = { id: string; name: string; local: boolean; cloud?: string; error?: string; message?: string };

export const syncConfigured = (s: Settings | null) => !!s?.syncServer && !!s?.syncToken;

export const DeleteModal: React.FC<{ targets: DelTarget[]; cloudOnly?: boolean; onClose: () => void; onDone: () => void }> = ({ targets, cloudOnly, onClose, onDone }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [out, close] = useExit(onClose);
  const [st, setSt] = useState<Settings | null>(null);
  const [mode, setMode] = useState<Mode>(cloudOnly ? 'cloud' : 'local');
  const [busy, setBusy] = useState(false);
  const [days, setDays] = useState(30); // recycle bin retention
  useEffect(() => { api.TrashDays().then(setDays).catch(() => {}); }, []);
  useEffect(() => {
    api.GetSettings().then((s: any) => {
      setSt(s);
      if (!cloudOnly) setMode(syncConfigured(s) ? (s.deleteMode === 'local' ? 'local' : 'cloud') : 'local');
    }).catch(() => setSt({} as any));
  }, [cloudOnly]);
  const sync = syncConfigured(st);
  const n = targets.length;
  const one = n === 1 ? targets[0] : null;
  const onCloud = targets.filter(t => t.inCloud).length;
  const running = targets.filter(t => t.running).length;
  const effective: Mode = sync ? mode : 'local';

  useEffect(() => {
    const k = (e: KeyboardEvent) => { if (e.key === 'Escape' && !busy) close(); };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  });

  const summarize = (rs: Result[]) => {
    const del = rs.filter(r => r.local || r.cloud === 'deleted').length;
    const queued = rs.filter(r => r.cloud === 'queued').length;
    const msg = rs.find(r => r.message)?.message;
    if (cloudOnly) toast(queued ? `服务器暂时连不上，已排队：联网后自动删除云端副本` : `已从云端删除 ${rs.filter(r => r.cloud === 'deleted').length} 个环境`, queued ? 'bad' : undefined);
    else if (effective === 'cloud') toast(queued ? `本机副本已移到最近删除；${queued} 个环境的云端删除已排队，联网后自动完成` : `已删除 ${del} 个环境（本机副本在最近删除保留 ${days} 天），其他设备同步时会一并删除`, queued ? 'bad' : undefined);
    else toast(onCloud ? `已将 ${del} 个环境移到最近删除，云端副本保留` : `已将 ${del} 个环境移到最近删除，${days} 天内可恢复`);
    if (msg) toast(msg, 'bad');
  };

  const go = async () => {
    setBusy(true);
    try {
      if (!cloudOnly && sync) api.SetDeleteMode(mode).catch(() => {});
      const ids = targets.map(t => t.id);
      let rs: Result[] = (await api.DeleteProfilesMode(ids, effective, false)) || [];
      const locked = rs.filter(r => r.cloud === 'locked');
      if (locked.length) {
        const force = await ask({
          title: locked.length === 1 ? `「${locked[0].name}」正在被另一台设备同步` : `${locked.length} 个环境正在被另一台设备同步`,
          body: `${locked[0].error || ''} 强制删除会中断那台设备的上传，它上传的内容也会被丢弃。`, ok: '强制删除', danger: true,
        });
        if (force) {
          const again: Result[] = (await api.DeleteProfilesMode(locked.map(r => r.id), effective, true)) || [];
          rs = rs.filter(r => r.cloud !== 'locked').concat(again);
        } else {
          rs = rs.filter(r => r.cloud !== 'locked');
          toast(`已跳过 ${locked.length} 个正在同步的环境`, 'bad');
        }
      }
      if (rs.length) summarize(rs);
      onDone();
      close();
    } catch (e: any) {
      toast(String(e?.message || e), 'bad');
      setBusy(false);
    }
  };

  const title = cloudOnly ? (one ? `从云端删除「${one.name}」？` : `从云端删除 ${n} 个环境？`) : (one ? `删除环境「${one.name}」？` : `删除 ${n} 个环境？`);
  const opts: [Mode, string, React.ReactNode, React.ReactNode][] = [
    ['local', '只删本地', <Icon.laptop s={17} />, <>本机的环境和数据目录移到<b>「最近删除」</b>，保留 {days} 天可恢复。云端副本保留，<b>不会再被自动下载回来</b>，可随时在「云端同步 → 仅云端」中恢复到本机。其他设备不受影响。</>],
    ['cloud', '同时删云端', <Icon.cloud s={17} />, <>同时从你的同步服务器删除数据和全部历史版本。其他设备下次同步时会<b>自动删除各自的副本</b>，旧副本也无法再上传回来。服务器只留一条删除记录（名称、时间、设备，不含数据），90 天后清除。本机副本仍先放进「最近删除」。</>],
  ];

  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && !busy && close()}>
      <div className="modal cdel">
        <div className="dhead">
          <div className="glyph bad"><Icon.trash s={18} /></div>
          <div className="cdel-h"><h3 title={title}>{title}</h3><div className="sub">{cloudOnly ? '本机没有这个环境的副本' : `环境和数据目录（Cookie、缓存、扩展状态）先移到「最近删除」，保留 ${days} 天`}</div></div>
          <div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" disabled={busy} onClick={() => close()}><Icon.x /></button></div>
        </div>
        <div className="dbody">
          {n > 1 && <div className="cdel-names">
            {targets.slice(0, 6).map(t => <span key={t.id} className="cdel-name" title={t.name}>{t.name}</span>)}
            {n > 6 && <span className="cdel-name more">等 {n} 个</span>}
          </div>}
          {!st ? <div className="cdel-load"><ThinkingOrb state="connecting" size={20} theme="dark" /></div>
            : cloudOnly ? <div className="cdel-note bad"><Icon.alert s={15} /><span>将从同步服务器删除数据和全部历史版本，所有设备都不会再下载它，<b>无法恢复</b>。服务器只保留一条不含数据的删除记录（90 天）。</span></div>
              : sync ? <div className="cdel-opts" role="radiogroup">
                {opts.map(([v, label, ic, desc], i) => <button key={v} role="radio" aria-checked={mode === v} className={`cdel-opt ${v} ${mode === v ? 'on' : ''}`} style={{ ['--i' as any]: i }} onClick={() => setMode(v)} disabled={busy}>
                  <span className="cdel-ic">{ic}</span>
                  <span className="cdel-tx"><span className="cdel-t">{label}{v === 'cloud' && <span className="badge fail cdel-tag">所有设备</span>}</span><span className="cdel-d">{desc}</span></span>
                  <span className="cdel-radio" />
                </button>)}
              </div>
                : <div className="cdel-note"><Icon.alert s={15} /><span>环境会移到<b>「最近删除」</b>，{days} 天内可在环境列表的「最近删除」中恢复，到期后自动彻底删除。尚未配置云端同步，只影响本机。</span></div>}
          {st && !cloudOnly && sync && effective === 'cloud' && onCloud < n && <div className="cdel-hint">{onCloud === 0 ? (n === 1 ? '这个环境从未上传到云端，' : '这些环境都从未上传到云端，') + '只会删除本机副本。' : `其中 ${n - onCloud} 个从未上传到云端，只会删除本机副本。`}</div>}
          {running > 0 && <div className="cdel-hint warn"><i className="d" />{running === n && n === 1 ? '该环境正在运行，' : `${running} 个环境正在运行，`}删除前会先关闭浏览器。</div>}
        </div>
        <div className="dfoot"><div className="spacer" />
          <button className="btn" disabled={busy} onClick={() => close()}>取消</button>
          <button className="btn solid-bad cdel-go" disabled={busy || !st} onClick={go} autoFocus>
            {busy ? <><ThinkingOrb state="weaving" size={20} theme="dark" /> 正在删除…</> : <><Icon.trash s={14} /> {cloudOnly ? '删除云端' : effective === 'cloud' ? (n > 1 ? `删除 ${n} 个（含云端）` : '删除（含云端）') : (n > 1 ? `删除 ${n} 个` : '删除')}</>}
          </button>
        </div>
      </div>
    </div>
  );
};
