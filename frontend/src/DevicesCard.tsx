import React, { useCallback, useEffect, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api } from './api';
import { Icon, useToast, useConfirm, ago, Toggle, OsIcon } from './ui';

type Dev = { id: string; name: string; os: string; firstSeen: number; lastSeen: number; approved: boolean; revoked: boolean; status: 'approved' | 'pending' | 'revoked'; current: boolean };
type DevList = { supported: boolean; self: string; selfStatus: string; requireApproval: boolean; devices: Dev[]; error?: string };

const osName = (os: string) => ({ windows: 'Windows', darwin: 'macOS', macos: 'macOS', linux: 'Linux' } as Record<string, string>)[os] || os || '未知系统';
const osKey = (os: string) => os === 'darwin' ? 'macos' : os || 'windows';
const errText = (e: any) => String(e?.message || e);

/** 设备管理: devices of this sync account, approval and revocation. */
export const DevicesCard: React.FC = () => {
  const toast = useToast();
  const ask = useConfirm();
  const [data, setData] = useState<DevList | null>(null);
  const [busy, setBusy] = useState('');
  const load = useCallback(() => {
    api.CloudDevices().then((d: any) => setData(d)).catch((e: any) => setData({ supported: false, self: '', selfStatus: '', requireApproval: false, devices: [], error: errText(e) }));
  }, []);
  useEffect(() => { load(); }, [load]);
  const act = async (id: string, f: () => Promise<any>, ok: string) => {
    setBusy(id);
    try { await f(); toast(ok, 'ok'); load(); } catch (e: any) { toast(errText(e), 'bad'); } finally { setBusy(''); }
  };
  const revoke = async (d: Dev) => {
    if (!await ask({ title: `吊销「${d.name || d.id}」？`, body: '该设备之后的所有同步请求都会被服务器拒绝（HTTP 403），但已下载到它本机的环境不会被删除。如果这台设备丢失，建议同时修改同步密码；若要彻底阻止它再注册，请开启「新设备需要批准」或更换令牌。', ok: '吊销', danger: true })) return;
    act(d.id, () => api.RevokeCloudDevice(d.id), '已吊销');
  };
  const setApproval = (on: boolean) => act('__toggle', () => api.SetDeviceApproval(on), on ? '已开启：新设备需在这里批准后才能同步' : '已关闭：新设备可直接同步');

  const pending = data?.devices.filter(d => d.status === 'pending').length || 0;
  return (
    <div className="panel dv-card">
      <div className="dv-head">
        <div className="dv-glyph"><Icon.laptop s={18} /></div>
        <div className="dv-h"><h4>设备管理</h4><div className="sub">登录此同步账号的电脑。吊销后，该设备的请求会被服务器拒绝。</div></div>
        {pending > 0 && <span className="badge warn"><i className="d" />{pending} 台待批准</span>}
        <button className="btn sm ghost icon" title="刷新" onClick={() => { setData(null); load(); }}><Icon.refresh s={14} /></button>
      </div>
      {!data ? <div className="dv-blank"><ThinkingOrb state="connecting" size={32} theme="dark" /><div className="sub">正在读取设备列表…</div></div>
        : !data.supported ? <div className="notice warn dv-in"><Icon.alert s={14} /><div className="grow">{data.error || '服务器版本过旧，不支持设备管理：请更新 ghostsync.php。'}</div></div>
          : data.selfStatus === 'pending' ? <div className="dv-self pending"><ThinkingOrb state="breathing" size={32} theme="dark" /><div><b>本机正在等待批准</b><div className="sub">请在一台已批准的设备上打开「云同步 → 设备与密钥」，批准「{data.devices.find(d => d.current)?.name || data.self}」。批准前本机无法同步。</div></div><button className="btn sm" onClick={load}><Icon.refresh s={13} /> 重新检查</button></div>
            : data.selfStatus === 'revoked' ? <div className="dv-self revoked"><Icon.alert s={22} /><div><b>本机已被吊销</b><div className="sub">{data.error || '此设备不能再访问该同步账号。'}如需恢复，请在其他已批准的设备上重新批准本机。</div></div></div>
              : <>
                {data.error && <div className="notice bad dv-in"><Icon.alert s={14} /> {data.error}</div>}
                <div className="dv-toggle"><Toggle on={data.requireApproval} onChange={setApproval} label="新设备需要批准" hint="开启后，用同一令牌连接的新电脑要先在这里批准才能同步" /></div>
                <div className="dv-list">
                  {data.devices.map((d, i) => <div key={d.id} className={`dv-row rise ${d.status}`} style={{ ['--i' as any]: i }}>
                    <div className="dv-os"><OsIcon os={osKey(d.os)} s={16} /></div>
                    <div className="dv-main">
                      <div className="dv-name"><span className="nm dv-ell" title={d.name || d.id}>{d.name || d.id}</span>{d.current && <span className="badge cloud dv-me">本机</span>}</div>
                      <div className="sub dv-meta"><span>{osName(d.os)}</span><span>· 最近 {d.current ? '刚刚' : ago(d.lastSeen)}</span><span className="dv-first">· 首次 {new Date(d.firstSeen * 1000).toLocaleDateString()}</span></div>
                    </div>
                    <span className={`badge ${d.status === 'approved' ? 'ok' : d.status === 'pending' ? 'warn' : 'fail'}`}><i className="d" />{d.status === 'approved' ? '已批准' : d.status === 'pending' ? '待批准' : '已吊销'}</span>
                    <div className="dv-acts">
                      {busy === d.id ? <span className="dv-spin"><ThinkingOrb state="weaving" size={20} theme="dark" /></span>
                        : d.current ? <span className="sub dv-cur">当前设备</span>
                          : d.status === 'pending' ? <>
                            <button className="btn sm primary" disabled={!!busy} onClick={() => act(d.id, () => api.ApproveCloudDevice(d.id), `已批准「${d.name || d.id}」`)}><Icon.check s={13} /> 批准</button>
                            <button className="btn sm ghost danger" disabled={!!busy} onClick={() => revoke(d)}>拒绝</button>
                          </> : d.status === 'approved' ? <button className="btn sm ghost danger" disabled={!!busy} onClick={() => revoke(d)}>吊销</button>
                            : <>
                              <button className="btn sm" disabled={!!busy} onClick={() => act(d.id, () => api.ApproveCloudDevice(d.id), '已重新批准')}>重新批准</button>
                              <button className="btn sm ghost icon" title="移除记录" disabled={!!busy} onClick={() => act(d.id, () => api.ForgetCloudDevice(d.id), '已移除记录')}><Icon.trash s={14} /></button>
                            </>}
                    </div>
                  </div>)}
                  {data.devices.length <= 1 && <div className="dv-empty sub">目前只有本机。其他电脑用同一服务器地址和令牌连接后会出现在这里。</div>}
                </div>
              </>}
    </div>
  );
};
