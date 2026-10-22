import React, { useCallback, useEffect, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, onEvent, Profile, Settings } from './api';
import { InlineTransfer } from './Transfers';
import { Icon, useToast, useConfirm, useExit, Avatar, ago, Segmented, MoreMenu } from './ui';
import { DeleteModal, DelTarget } from './DeleteModal';
import { SharingView } from './SharingView'; // share
import { DevicesCard } from './DevicesCard';
import { RecoveryCard } from './RecoveryKeyModal';

type CloudOnlyItem = { id: string; name: string; version: number; updatedAt: number; deviceName: string; size: number; lockedBy: string; localDeleted: boolean; deletedVersion: number; deletedAt: number; cloudNewer: boolean; pendingDelete: boolean; pendingError?: string };

const size = (b?: number) => b == null || b < 0 ? '—' : b >= 1 << 30 ? `${(b / (1 << 30)).toFixed(1)} GB` : b >= 1 << 20 ? `${(b / (1 << 20)).toFixed(1)} MB` : b >= 1024 ? `${Math.round(b / 1024)} KB` : `${b} B`;
const ACTION: Record<string, string> = { push: '上传', pull: '拉取', restore: '恢复版本', 'push-global': '上传全局数据', 'pull-global': '拉取全局数据', repair: '修复云端' };

type Info = { features: string[]; keepVersions: number; maxBody: number; usedBytes: number; freeBytes: number; user: string; protocol: number };
type LogRow = { time: number; id: string; name: string; action: string; ok: boolean; error: string; bytes: number; ms: number; version: number };

export const syncState = (p: Profile, busy: Set<string>): [string, string] =>
  busy.has(p.id) ? ['同步中', 'unknown']
    : p.storage !== 'cloud' ? ['仅本地', 'local']
      : p.lastRun > p.syncedAt ? ['待上传', 'warn']
        : ['已同步', 'ok'];

export const CloudView: React.FC<{ profiles: Profile[]; settings: Settings | null; reload: () => void }> = ({ profiles, settings, reload }) => {
  const toast = useToast();
  const [busy, setBusy] = useState<Set<string>>(new Set());
  const [info, setInfo] = useState<Info | null>(null);
  const [infoErr, setInfoErr] = useState('');
  const [log, setLog] = useState<LogRow[]>([]);
  const [hist, setHist] = useState<Profile | null>(null);
  const [tab, setTab] = useState<'list' | 'only' | 'log' | 'share' | 'security'>('list');
  useEffect(() => { const f = (e: any) => setTab(e.detail); window.addEventListener('gb:cloud-tab', f); return () => window.removeEventListener('gb:cloud-tab', f); }, []); // share
  const [only, setOnly] = useState<CloudOnlyItem[] | null>(null);
  const [onlyErr, setOnlyErr] = useState('');
  const [delT, setDelT] = useState<DelTarget[] | null>(null);
  const configured = !!settings?.syncServer && !!settings?.syncPassword;

  const loadLog = useCallback(() => { api.SyncLog().then((l: any) => setLog(l || [])).catch(() => {}); }, []);
  const loadInfo = useCallback(() => { api.SyncServerInfo().then((i: any) => { setInfo(i); setInfoErr(''); }).catch((e: any) => setInfoErr(String(e?.message || e))); }, []);
  const loadOnly = useCallback(() => { api.CloudOnly().then((r: any) => { setOnly(r?.items || []); setOnlyErr(r?.error || ''); }).catch((e: any) => { setOnly([]); setOnlyErr(String(e?.message || e)); }); }, []);
  useEffect(() => { if (configured) { loadInfo(); loadLog(); loadOnly(); } }, [configured, loadInfo, loadLog, loadOnly]);
  useEffect(() => { api.SyncBusy().then((b: any) => setBusy(new Set(b || []))).catch(() => {}); }, []);
  useEffect(() => onEvent('sync:event', (e: any) => {
    if (e.phase === 'busy') setBusy(s => new Set(s).add(e.id));
    if (e.phase === 'idle') setBusy(s => { const n = new Set(s); n.delete(e.id); return n; });
    if (e.phase === 'item' || e.phase === 'done') loadLog();
    if (e.phase === 'done' || e.phase === 'deleted' || e.phase === 'deleted-cloud') loadOnly();
  }), [loadLog, loadOnly]);
  const [onlyBusy, setOnlyBusy] = useState('');
  const restoreLocal = async (it: CloudOnlyItem) => {
    setOnlyBusy(it.id);
    try { await api.RestoreLocalDeleted(it.id); toast(`已恢复「${it.name}」到本机`); reload(); loadOnly(); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { setOnlyBusy(''); loadLog(); }
  };
  const retryDeletes = async () => { const left: any = await api.RetryCloudDeletes(); toast(left ? `仍有 ${left} 个云端删除未完成，稍后会自动重试` : '排队的云端删除已全部完成', left ? 'bad' : undefined); loadOnly(); };

  const mark = (id: string, on: boolean) => setBusy(s => { const n = new Set(s); on ? n.add(id) : n.delete(id); return n; });
  const run = async (p: Profile, f: () => Promise<any>, ok: string) => { mark(p.id, true); try { await f(); toast(ok); reload(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { mark(p.id, false); loadLog(); } };
  const [syncing, setSyncing] = useState(false);
  const syncAll = async () => { setSyncing(true); try { const r: any = await api.SyncNow(); toast(r.error ? `同步失败：${r.error}` : (r.message || '同步完成'), r.error || r.failed ? 'bad' : undefined); reload(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { setSyncing(false); loadLog(); loadInfo(); } };
  const ask = useConfirm();
  const repair = async (p: Profile) => {
    if (!await ask({ title: `修复「${p.name}」的云端数据？`, body: '用本机的这份环境数据强制重新上传：先检查服务器缺少哪些数据分块，全部补传后提交一个新的云端版本（即使本机没有改动）。请在上传它的那台设备上操作；拉取时提示「云端缺少数据分块」就用这个。', ok: '修复云端' })) return;
    mark(p.id, true);
    try { const r: any = await api.RepairCloud(p.id); toast(r?.message || '已修复', 'ok'); reload(); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { mark(p.id, false); loadLog(); }
  };
  const check = async (p: Profile) => {
    mark(p.id, true);
    try { const r: any = await api.CheckCloud(p.id); toast(r?.message || '检查完成', r?.missing ? 'bad' : 'ok'); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { mark(p.id, false); }
  };
  const [gBusy, setGBusy] = useState(false);
  const repairGlobal = async () => { setGBusy(true); try { const r: any = await api.RepairGlobal(); toast(r?.message || '全局数据已重新上传', 'ok'); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { setGBusy(false); loadLog(); } };
  const retry = async (r: LogRow) => { try { await api.RetrySync(r.id, r.action); toast('重试成功', 'ok'); reload(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { loadLog(); } };

  if (!configured) return (
    <div className="empty">
      <div className="halo"><ThinkingOrb state="connecting" size={64} theme="dark" /></div>
      <h3>尚未配置同步服务器</h3>
      <div>到「设置 → 云端同步」填入你的 VPS 地址、访问令牌和同步密码后，<br />即可把环境加密上传到自己的服务器。Cookies 在本地加密，服务器拿不到明文。</div>
    </div>
  );

  const cloud = profiles.filter(p => p.storage === 'cloud');
  const has = (f: string) => !!info?.features?.includes(f);
  const fails = log.filter(l => !l.ok).length;

  return (
    <>
      <div className="panel hero spot">
        <div className="orbit"><Icon.cloud s={26} /></div>
        <div>
          <h4>同步服务器 {infoErr ? <span className="badge fail" style={{ marginLeft: 8, verticalAlign: 2 }}><i className="d" />连接失败</span> : <span className="badge ok" style={{ marginLeft: 8, verticalAlign: 2 }}><i className="d" />{info ? '已连接' : '检测中'}</span>}</h4>
          <div className="sub"><code className="mono">{settings!.syncServer}</code> · 本机 {settings!.deviceName || settings!.deviceId}{info?.user && info.user !== 'owner' ? ` · 用户 ${info.user}` : ''}</div>
          {infoErr ? <div className="sub" style={{ color: '#ffb3bc' }}>{infoErr}</div> : info && <div className="sub num" style={{ marginTop: 4 }}>
            已用 {size(info.usedBytes)} · 磁盘剩余 {size(info.freeBytes)} · 单次上传上限 {info.maxBody ? size(info.maxBody) : '不限'} · 保留 {info.keepVersions || 1} 个版本 · {has('chunks') ? '增量同步' : '整包同步（建议更新 ghostsync.php）'}
          </div>}
        </div>
        <div className="flow">
          <span className="node"><Icon.laptop />本机</span><span className="wire" />
          <span className="node"><Icon.lock />AES-256-GCM</span><span className="wire" />
          <span className="node"><Icon.server />你的 VPS</span>
        </div>
      </div>

      <div className="toolbar">
        <Segmented className="compact" value={tab} onChange={t => { setTab(t); if (t === 'log') loadLog(); }}
          items={[['list', '全部环境'], ['only', only?.length ? `仅云端 · ${only.length}` : '仅云端'], ['share', '分享'], ['security', '设备与密钥'], ['log', fails ? `同步日志 · ${fails} 失败` : '同步日志']]} />
        <div className="sub" style={{ color: 'var(--txt-3)' }}>{cloud.length} 个云端 · {profiles.length - cloud.length} 个本地 · 自动同步{settings!.autoSyncOff ? '已关闭' : '已开启'}{' · 登录状态：'}{(settings as any).syncSessionStorage === false ? '仅 Cookie' : 'Cookie + LocalStorage + IndexedDB'}</div>
        <div style={{ flex: 1 }} />
        {tab === 'only' && only?.some(i => i.pendingDelete) && <button className="btn sm ghost" onClick={retryDeletes}><Icon.refresh s={13} /> 重试云端删除</button>}
        {tab === 'list' && <button className="btn sm ghost" onClick={repairGlobal} disabled={gBusy} title="把本机的设置、代理和指纹库强制重新上传为新的云端版本"><Icon.upload s={13} /> {gBusy ? '上传中…' : '重新上传全局数据'}</button>}
        {tab === 'log' && log.length > 0 && <button className="btn sm ghost" onClick={async () => { await api.ClearSyncLog(); loadLog(); }}><Icon.trash s={13} /> 清空日志</button>}
        <button className="btn sm" onClick={syncAll} disabled={syncing}><Icon.refresh s={14} /> {syncing ? '同步中…' : '立即同步（服务器优先）'}</button>
      </div>

      {tab === 'share' ? <SharingView reload={reload} goSecurity={() => setTab('security')} />
        : tab === 'security' ? <div className="dv-grid"><DevicesCard /><RecoveryCard /></div>
        : tab === 'only' ? <div className="panel list cdel-only">
        {onlyErr && <div className="notice bad" style={{ margin: 12 }}><Icon.alert s={14} /> 无法连接同步服务器，下面只显示本机记录：{onlyErr}</div>}
        {only === null ? <div className="empty" style={{ padding: 40 }}><ThinkingOrb state="connecting" size={32} theme="dark" /></div>
          : only.length === 0 ? <div className="empty" style={{ padding: 44 }}><div className="halo"><ThinkingOrb state="breathing" size={32} theme="dark" /></div><h3>云端没有本机缺少的环境</h3><div>用「只删本地」删除的环境会出现在这里，可以随时恢复到本机或从云端删除。</div></div>
            : <table>
              <colgroup><col /><col style={{ width: '27%' }} /><col style={{ width: 84 }} /><col style={{ width: '19%' }} /><col style={{ width: 228 }} /></colgroup>
              <thead><tr><th>环境</th><th>状态</th><th>云端版本</th><th>云端更新</th><th style={{ textAlign: 'right' }}>操作</th></tr></thead>
              <tbody>{only.map((it, i) => <tr key={it.id} className="rise" style={{ ['--i' as any]: i }}>
                <td><div className="cell-id"><Avatar seed={it.id} os="windows" /><div style={{ minWidth: 0 }}><div className="nm cdel-ell" title={it.name}>{it.name || it.id}</div><div className="sub num">{it.id}</div></div></div></td>
                <td><div className="cdel-badges">
                  {it.pendingDelete ? <span className="badge warn" title={it.pendingError || ''}><i className="d" />云端删除排队中</span>
                    : it.localDeleted ? <span className="badge local"><Icon.cloud s={12} /> 仅云端（已在本机删除）</span>
                      : <span className="badge unknown"><Icon.cloud s={12} /> 仅云端（尚未下载）</span>}
                  {it.cloudNewer && !it.pendingDelete && <span className="badge ok cdel-new" title={`本机删除时为 v${it.deletedVersion}`}><i className="d" />云端有更新</span>}
                </div>
                  {it.pendingDelete && it.pendingError && <div className="sub cdel-err" title={it.pendingError}>{it.pendingError}</div>}
                  {it.localDeleted && !it.pendingDelete && it.deletedAt > 0 && <div className="sub cdel-ell">{ago(it.deletedAt)}在本机删除</div>}
                </td>
                <td className="sub num">{it.version ? `v${it.version}` : '—'}</td>
                <td className="sub"><div className="cdel-ell">{it.updatedAt ? <>{new Date(it.updatedAt * 1000).toLocaleString()}</> : '—'}</div>{it.updatedAt ? <>{it.deviceName && <div className="cdel-ell" title={it.deviceName}>来自 {it.deviceName}</div>}</> : null}</td>
                <td><div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                  {onlyBusy === it.id ? <span style={{ width: 26, display: 'grid', placeItems: 'center' }}><ThinkingOrb state="weaving" size={20} theme="dark" /></span> : it.pendingDelete ? <span className="sub">联网后自动重试</span> : <>
                    <button className="btn sm" disabled={!!onlyBusy || !it.version} onClick={() => restoreLocal(it)}><Icon.download s={13} /> 恢复到本机</button>
                    <button className="btn sm ghost danger" disabled={!!onlyBusy} onClick={() => setDelT([{ id: it.id, name: it.name || it.id, inCloud: true }])}><Icon.trash s={13} /> 删除云端</button>
                  </>}
                </div></td>
              </tr>)}</tbody>
            </table>}
      </div> : tab === 'list' ? <div className="panel list">
        <table>
          <thead><tr><th>环境</th><th>状态</th><th>云端版本</th><th>最近同步</th><th style={{ textAlign: 'right' }}>操作</th></tr></thead>
          <tbody>
            {profiles.filter(p => !(p as any).shareId).map((p, i) => {
              const [txt, tone] = syncState(p, busy);
              return <tr key={p.id} className="rise" style={{ ['--i' as any]: i }}>
                <td><div className="cell-id"><Avatar seed={p.fingerprint.noiseSeed || p.id} os={p.fingerprint.os} /><div><div className="nm">{p.name}</div><div className="sub">{p.group || '未分组'}</div></div></div></td>
                <td><InlineTransfer id={p.id} fallback={<span className={`badge ${tone}`}>{tone === 'local' ? <Icon.laptop s={12} /> : <i className="d" />} {txt}</span>} /></td>
                <td className="sub num">{p.cloudVersion ? `v${p.cloudVersion}` : '—'}</td>
                <td className="sub">{p.syncedAt ? new Date(p.syncedAt * 1000).toLocaleString() : '—'}</td>
                <td><div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                  {busy.has(p.id) ? <span style={{ width: 26, display: 'grid', placeItems: 'center' }}><ThinkingOrb state="weaving" size={20} theme="dark" /></span> : <>
                    <button className="btn sm" onClick={() => run(p, () => api.PushProfile(p.id), `已上传 ${p.name}`)}><Icon.upload s={13} /> 上传</button>
                    {p.storage === 'cloud' && <button className="btn sm" onClick={() => run(p, () => api.PullProfile(p.id), `已拉取 ${p.name}`)}><Icon.download s={13} /> 拉取</button>}
                    {p.storage === 'cloud' && has('history') && <button className="btn sm ghost" onClick={() => setHist(p)}><Icon.layers s={13} /> 历史版本</button>}
                    {p.storage === 'cloud' && has('chunks') && <MoreMenu title="云端数据" items={[
                      { label: '检查云端完整性', icon: <Icon.shield s={14} />, onClick: () => check(p) },
                      { label: '修复云端（强制重新上传）', icon: <Icon.upload s={14} />, onClick: () => repair(p) },
                    ]} />}
                  </>}
                </div></td>
              </tr>;
            })}
          </tbody>
        </table>
      </div> : <div className="panel list">
        <table>
          <thead><tr><th>时间</th><th>项目</th><th>操作</th><th>结果</th><th>流量 / 耗时</th><th style={{ textAlign: 'right' }}></th></tr></thead>
          <tbody>
            {log.map((r, i) => <tr key={i}>
              <td className="sub num" title={new Date(r.time * 1000).toLocaleString()}>{ago(r.time)}</td>
              <td><div className="nm">{r.name || r.id}</div>{r.version ? <div className="sub num">v{r.version}</div> : null}</td>
              <td>{ACTION[r.action] || r.action}</td>
              <td>{r.ok ? <span className="badge ok"><i className="d" />成功</span> : <><span className="badge fail"><i className="d" />失败</span><div className="sub" style={{ color: '#ffb3bc', maxWidth: 380, wordBreak: 'break-all' }}>{r.error}</div></>}</td>
              <td className="sub num">{r.bytes ? size(r.bytes) : '—'} · {(r.ms / 1000).toFixed(1)}s</td>
              <td style={{ textAlign: 'right' }}>{!r.ok && r.action !== 'restore' && <button className="btn sm" onClick={() => retry(r)}><Icon.refresh s={13} /> 重试</button>}</td>
            </tr>)}
          </tbody>
        </table>
        {log.length === 0 && <div className="empty" style={{ padding: 40 }}><div>还没有同步记录</div></div>}
      </div>}

      {delT && <DeleteModal targets={delT} cloudOnly onClose={() => setDelT(null)} onDone={() => { loadOnly(); loadLog(); }} />}
      {hist && <HistoryModal p={hist} onClose={() => setHist(null)} onDone={() => { reload(); loadLog(); }} />}
    </>
  );
};

type Ver = { version: number; size: number; updatedAt: number; deviceName: string; current: boolean };

const HistoryModal: React.FC<{ p: Profile; onClose: () => void; onDone: () => void }> = ({ p, onClose, onDone }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [list, setList] = useState<Ver[] | null>(null);
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(0);
  const [out, close] = useExit(onClose);
  useEffect(() => { api.CloudHistory(p.id).then((l: any) => setList(l || [])).catch((e: any) => setErr(String(e?.message || e))); }, [p.id]);
  const restore = async (v: Ver) => {
    if (!await ask({ title: `恢复到 v${v.version}？`, body: '本机的这个环境会被替换成该版本（当前数据保留一份备份），然后作为最新版本上传，其他电脑下次启动时会拉取。', ok: '恢复' })) return;
    setBusy(v.version);
    try { await api.RestoreVersion(p.id, v.version); toast(`已恢复到 v${v.version}`, 'ok'); onDone(); close(); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { setBusy(0); }
  };
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal">
        <div className="dhead"><div className="glyph"><Icon.layers s={18} /></div><div><h3>历史版本</h3><div className="sub">{p.name} · 服务器保留的版本可以随时恢复</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          {err ? <div className="sub" style={{ color: '#ffb3bc' }}>{err}</div> : !list ? <div style={{ display: 'grid', placeItems: 'center', padding: 30 }}><ThinkingOrb state="connecting" size={32} theme="dark" /></div> :
            <table><tbody>{list.map(v => <tr key={v.version}>
              <td className="num"><b>v{v.version}</b>{v.current && <span className="badge ok" style={{ marginLeft: 6 }}>当前</span>}</td>
              <td className="sub">{v.updatedAt ? new Date(v.updatedAt * 1000).toLocaleString() : '—'}</td>
              <td className="sub">{v.deviceName || '—'}</td>
              <td className="sub num">{size(v.size)}</td>
              <td style={{ textAlign: 'right' }}>{!v.current && (busy === v.version ? <ThinkingOrb state="weaving" size={20} theme="dark" /> : <button className="btn sm" disabled={!!busy} onClick={() => restore(v)}><Icon.refresh s={13} /> 恢复</button>)}</td>
            </tr>)}</tbody></table>}
          {list && list.length < 2 && <div className="sub" style={{ marginTop: 8 }}>只有当前版本。之后每次上传，旧版本会自动保留。</div>}
        </div>
      </div>
    </div>
  );
};

