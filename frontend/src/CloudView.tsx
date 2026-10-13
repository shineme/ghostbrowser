import React, { useEffect, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, Profile, Settings } from './api';
import { Icon, useToast } from './ui';

export const CloudView: React.FC<{ profiles: Profile[]; settings: Settings | null; reload: () => void }> = ({ profiles, settings, reload }) => {
  const toast = useToast();
  const [busy, setBusy] = useState<Set<string>>(new Set());
  const [remote,setRemote]=useState<{id:string;name:string;version:number}[]>([]);
  const [connected,setConnected]=useState(false);
  const [error,setError]=useState('');
  const configured = !!settings?.syncServer && !!settings?.syncPassword && !!settings?.syncToken;
  const refreshCloud=async()=>{try{await api.TestSync();setRemote(await api.ListCloud());setConnected(true);setError('');}catch(e){setConnected(false);setError(String(e));}};
  useEffect(()=>{if(configured)void refreshCloud();},[settings?.syncServer,settings?.syncToken,configured]);

  const push = async (p: {id:string;name:string}) => { mark(p.id, true); try { await api.PushProfile(p.id); toast(`已上传 ${p.name}`); reload();await refreshCloud(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { mark(p.id, false); } };
  const pull = async (p: {id:string;name:string}) => { mark(p.id, true); try { await api.PullProfile(p.id); toast(`已拉取 ${p.name}`); reload();await refreshCloud(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { mark(p.id, false); } };
  const mark = (id: string, on: boolean) => setBusy(s => { const n = new Set(s); on ? n.add(id) : n.delete(id); return n; });

  if (!configured) return (
    <div className="empty">
      <ThinkingOrb state="connecting" size={64} theme="dark" />
      <h3>尚未配置同步服务器</h3>
      <div>到「设置 → 云端同步」填入你的 VPS 地址、访问令牌和同步密码后，<br />即可把环境加密上传到自己的服务器。Cookies 在本地加密，服务器拿不到明文。</div>
    </div>
  );

  const cloud = profiles.filter(p => p.storage === 'cloud');
  const rows=[...profiles.map(p=>({...p,hasLocal:true})),...remote.filter(r=>!profiles.some(p=>p.id===r.id)).map(r=>({id:r.id,name:r.name,group:'仅在云端',storage:'cloud',cloudVersion:r.version,syncedAt:0,hasLocal:false}))];

  return (
    <>
      <div className="panel" style={{ padding: 16, marginBottom: 18, display: 'flex', alignItems: 'center', gap: 14 }}>
        <Icon.cloud s={22} />
        <div><div className="nm">同步服务器</div><div className="sub"><code className="mono">{settings!.syncServer}</code> · 本机 {settings!.deviceName || settings!.deviceId}</div></div>
        <div className="spacer" style={{ flex: 1 }} />
        <button className="btn sm" onClick={refreshCloud}>刷新云端</button>
        <span className={`badge ${connected?'ok':''}`}>{connected?'已连接 · 端到端加密':'尚未连接'}</span>
      </div>
      {error && <div className="panel" style={{padding:14,marginBottom:16}}>{error}</div>}
      <div className="sub" style={{marginBottom:14}}>先停止环境再上传或拉取。其它设备的云端环境可直接拉取，完整指纹和环境信息会一起恢复。</div>

      <div className="toolbar"><div className="nm">全部环境</div><div className="sub" style={{ marginLeft: 8 }}>{cloud.length} 个云端 · {profiles.length - cloud.length} 个本地</div></div>
      <div className="panel" style={{ overflow: 'hidden' }}>
        <table>
          <thead><tr><th>环境</th><th>存储</th><th>云端版本</th><th>最近同步</th><th style={{ textAlign: 'right' }}>操作</th></tr></thead>
          <tbody>
            {rows.map(p => <tr key={p.id}>
              <td><div className="nm">{p.name}</div><div className="sub">{p.group || '未分组'}</div></td>
              <td><span className={`badge ${p.storage}`}>{p.storage === 'cloud' ? '云端' : '本地'}</span></td>
              <td className="sub">{p.cloudVersion ? `v${p.cloudVersion}` : '—'}</td>
              <td className="sub">{p.syncedAt ? new Date(p.syncedAt * 1000).toLocaleString() : '—'}</td>
              <td><div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                {busy.has(p.id) ? <span style={{ width: 26, display: 'grid', placeItems: 'center' }}><ThinkingOrb state="weaving" size={20} theme="dark" /></span> : <>
                  {p.hasLocal && <button className="btn sm" onClick={() => push(p)}><Icon.upload s={13} /> 上传</button>}
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
