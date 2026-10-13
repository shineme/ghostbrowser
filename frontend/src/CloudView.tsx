import React, { useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, Profile, Settings } from './api';
import { Icon, useToast, Avatar } from './ui';

export const CloudView: React.FC<{ profiles: Profile[]; settings: Settings | null; reload: () => void }> = ({ profiles, settings, reload }) => {
  const toast = useToast();
  const [busy, setBusy] = useState<Set<string>>(new Set());
  const configured = !!settings?.syncServer && !!settings?.syncPassword;

  const push = async (p: Profile) => { mark(p.id, true); try { await api.PushProfile(p.id); toast(`已上传 ${p.name}`); reload(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { mark(p.id, false); } };
  const pull = async (p: Profile) => { mark(p.id, true); try { await api.PullProfile(p.id); toast(`已拉取 ${p.name}`); reload(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { mark(p.id, false); } };
  const [syncing, setSyncing] = useState(false);
  const syncAll = async () => { setSyncing(true); try { const r: any = await api.SyncNow(); toast(r.error ? `同步失败：${r.error}` : (r.message || '同步完成'), r.error || r.failed ? 'bad' : undefined); reload(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { setSyncing(false); } };
  const mark = (id: string, on: boolean) => setBusy(s => { const n = new Set(s); on ? n.add(id) : n.delete(id); return n; });

  if (!configured) return (
    <div className="empty">
      <div className="halo"><ThinkingOrb state="connecting" size={64} theme="dark" /></div>
      <h3>尚未配置同步服务器</h3>
      <div>到「设置 → 云端同步」填入你的 VPS 地址、访问令牌和同步密码后，<br />即可把环境加密上传到自己的服务器。Cookies 在本地加密，服务器拿不到明文。</div>
    </div>
  );

  const cloud = profiles.filter(p => p.storage === 'cloud');

  return (
    <>
      <div className="panel hero spot">
        <div className="orbit"><Icon.cloud s={26} /></div>
        <div>
          <h4>同步服务器 <span className="badge ok" style={{ marginLeft: 8, verticalAlign: 2 }}><i className="d" />已连接</span></h4>
          <div className="sub"><code className="mono">{settings!.syncServer}</code> · 本机 {settings!.deviceName || settings!.deviceId}</div>
        </div>
        <div className="flow">
          <span className="node"><Icon.laptop />本机</span><span className="wire" />
          <span className="node"><Icon.lock />AES-256-GCM</span><span className="wire" />
          <span className="node"><Icon.server />你的 VPS</span>
        </div>
      </div>

      <div className="toolbar"><div className="nm" style={{ fontWeight: 650 }}>全部环境</div><div className="sub" style={{ color: 'var(--txt-3)' }}>{cloud.length} 个云端 · {profiles.length - cloud.length} 个本地 · 自动同步{settings!.autoSyncOff ? '已关闭' : '已开启'}</div><div style={{ flex: 1 }} /><button className="btn sm" onClick={syncAll} disabled={syncing}><Icon.refresh s={14} /> {syncing ? '同步中…' : '立即同步（服务器优先）'}</button></div>
      <div className="panel list">
        <table>
          <thead><tr><th>环境</th><th>存储</th><th>云端版本</th><th>最近同步</th><th style={{ textAlign: 'right' }}>操作</th></tr></thead>
          <tbody>
            {profiles.map((p, i) => <tr key={p.id} className="rise" style={{ ['--i' as any]: i }}>
              <td><div className="cell-id"><Avatar seed={p.fingerprint.noiseSeed || p.id} os={p.fingerprint.os} /><div><div className="nm">{p.name}</div><div className="sub">{p.group || '未分组'}</div></div></div></td>
              <td><span className={`badge ${p.storage}`}>{p.storage === 'cloud' ? <><Icon.cloud s={12} /> 云端</> : <><Icon.laptop s={12} /> 本地</>}</span></td>
              <td className="sub num">{p.cloudVersion ? `v${p.cloudVersion}` : '—'}</td>
              <td className="sub">{p.syncedAt ? new Date(p.syncedAt * 1000).toLocaleString() : '—'}</td>
              <td><div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                {busy.has(p.id) ? <span style={{ width: 26, display: 'grid', placeItems: 'center' }}><ThinkingOrb state="weaving" size={20} theme="dark" /></span> : <>
                  <button className="btn sm" onClick={() => push(p)}><Icon.upload s={13} /> 上传</button>
                  {p.storage === 'cloud' && <button className="btn sm" onClick={() => pull(p)}><Icon.download s={13} /> 拉取</button>}
                </>}
              </div></td>
            </tr>)}
          </tbody>
        </table>
      </div>
    </>
  );
};
