import React, { useCallback, useEffect, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, Profile } from './api';
import { Icon, useToast, useConfirm, useExit, Avatar, ago, Segmented, Field, SelectField } from './ui';

// ---- share: icons not in the shared Icon set ----
const SV = (s = 16) => ({ width: s, height: s, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 1.8, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const });
export const ShIc = {
  share: ({ s = 16 }: { s?: number }) => <svg {...SV(s)}><circle cx="18" cy="5" r="3" /><circle cx="6" cy="12" r="3" /><circle cx="18" cy="19" r="3" /><path d="m8.6 13.5 6.8 4" /><path d="m15.4 6.5-6.8 4" /></svg>,
  key: ({ s = 16 }: { s?: number }) => <svg {...SV(s)}><circle cx="7.5" cy="15.5" r="4.5" /><path d="m10.7 12.3 9.8-9.8" /><path d="m16 7 3 3" /><path d="m19 4 2 2" /></svg>,
  users: ({ s = 16 }: { s?: number }) => <svg {...SV(s)}><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M22 21v-2a4 4 0 0 0-3-3.9" /><path d="M16 3.1a4 4 0 0 1 0 7.8" /></svg>,
  inbox: ({ s = 16 }: { s?: number }) => <svg {...SV(s)}><path d="M22 12h-6l-2 3h-4l-2-3H2" /><path d="M5.5 5.1 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.5-6.9A2 2 0 0 0 16.8 4H7.2a2 2 0 0 0-1.7 1.1Z" /></svg>,
  send: ({ s = 16 }: { s?: number }) => <svg {...SV(s)}><path d="m22 2-7 20-4-9-9-4Z" /><path d="M22 2 11 13" /></svg>,
  clock: ({ s = 16 }: { s?: number }) => <svg {...SV(s)}><circle cx="12" cy="12" r="10" /><path d="M12 6v6l4 2" /></svg>,
  logout: ({ s = 16 }: { s?: number }) => <svg {...SV(s)}><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" /><path d="m16 17 5-5-5-5" /><path d="M21 12H9" /></svg>,
};

export type ShareInfo = { id: string; from: string; to: string; name: string; mode: 'session' | 'config'; note: string; os: string; createdAt: number; expiresAt: number; updatedAt: number; size: number; status: string; localId?: string };
type ShareList = { supported: boolean; ready: boolean; self: string; out: ShareInfo[]; in: ShareInfo[]; removed: number; error?: string };

const errText = (e: any) => String(e?.message || e);
const size = (b: number) => !b ? '—' : b >= 1 << 20 ? `${(b / (1 << 20)).toFixed(1)} MB` : b >= 1024 ? `${Math.round(b / 1024)} KB` : `${b} B`;

/** "剩余 6 天" / "剩余 5 小时" / "已过期". */
export const shareLeft = (ts: number): [string, 'ok' | 'warn' | 'fail'] => {
  const d = ts - Date.now() / 1000;
  if (d <= 0) return ['已过期', 'fail'];
  if (d < 3600) return [`剩余 ${Math.max(1, Math.round(d / 60))} 分钟`, 'warn'];
  if (d < 86400) return [`剩余 ${Math.round(d / 3600)} 小时`, 'warn'];
  const n = Math.floor(d / 86400);
  return [`剩余 ${n} 天`, n <= 2 ? 'warn' : 'ok'];
};

const STATUS: Record<string, [string, string]> = {
  pending: ['待接受', 'warn'], accepted: ['已接受', 'ok'], declined: ['已拒绝', 'unknown'], revoked: ['已撤销', 'fail'],
  left: ['对方已退出', 'unknown'], expired: ['已过期', 'fail'],
};
const live = (s: ShareInfo) => s.status === 'pending' || s.status === 'accepted';

const ModeBadge: React.FC<{ mode: string }> = ({ mode }) => mode === 'session'
  ? <span className="badge cloud sh-mode" title="包含 Cookies、本地存储等登录状态"><Icon.lock s={11} /> 含登录状态</span>
  : <span className="badge local sh-mode" title="只有指纹、代理等配置，不含登录状态"><Icon.settings s={11} /> 仅配置</span>;

export const SharingView: React.FC<{ reload: () => void; goSecurity: () => void }> = ({ reload, goSecurity }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [tab, setTab] = useState<'in' | 'out'>('in');
  const [data, setData] = useState<ShareList | null>(null);
  const [busy, setBusy] = useState('');
  const [extend, setExtend] = useState<ShareInfo | null>(null);
  const load = useCallback(() => {
    api.ShareList().then((l: any) => { setData(l); if (l?.removed) { toast(`已删除 ${l.removed} 个已结束分享的本机副本`); reload(); } })
      .catch((e: any) => setData({ supported: false, ready: false, self: '', out: [], in: [], removed: 0, error: errText(e) }));
  }, [reload, toast]);
  useEffect(() => { load(); }, [load]);

  const act = async (s: ShareInfo, f: () => Promise<any>, ok: string, after?: () => void) => {
    setBusy(s.id);
    try { await f(); toast(ok, 'ok'); after?.(); load(); } catch (e: any) { toast(errText(e), 'bad'); } finally { setBusy(''); }
  };
  const accept = (s: ShareInfo) => act(s, () => api.AcceptShare(s.id), `已接受「${s.name}」，已添加到环境列表`, reload);
  const decline = async (s: ShareInfo) => { if (await ask({ title: `拒绝「${s.name}」？`, body: `${s.from} 会看到你已拒绝，分享包会从服务器删除。`, ok: '拒绝', danger: true })) act(s, () => api.DeclineShare(s.id), '已拒绝'); };
  const leave = async (s: ShareInfo) => { if (await ask({ title: `退出「${s.name}」？`, body: '本机的共享环境副本会被删除（包括你在里面产生的登录状态），之后需要对方重新分享。', ok: '退出并删除', danger: true })) act(s, () => api.LeaveShare(s.id), '已退出分享', reload); };
  const revoke = async (s: ShareInfo) => { if (await ask({ title: `撤销分享给 ${s.to} 的「${s.name}」？`, body: '对方本机的副本会在下次同步（最多几分钟）或下次启动该环境时被删除。已经打开的浏览器在关闭后删除。', ok: '撤销', danger: true })) act(s, () => api.RevokeShare(s.id), '已撤销'); };

  if (!data) return <div className="panel sh-panel"><div className="empty" style={{ padding: 60 }}><ThinkingOrb state="connecting" size={32} theme="dark" /><div>正在读取分享…</div></div></div>;
  if (!data.supported) return <div className="panel sh-panel"><div className="empty" style={{ padding: 56 }}>
    <div className="halo"><ThinkingOrb state="breathing" size={32} theme="dark" /></div><h3>无法使用环境分享</h3><div className="sh-maxw">{data.error || '同步服务器不支持分享'}</div>
    <div className="actions"><button className="btn" onClick={() => { setData(null); load(); }}><Icon.refresh s={14} /> 重试</button></div></div></div>;

  const list = tab === 'in' ? data.in : data.out;
  const pendingIn = data.in.filter(s => s.status === 'pending').length;
  const liveOut = data.out.filter(live).length;
  return (
    <>
      {!data.ready && <div className="notice warn sh-notice"><ShIc.key s={15} /><div className="grow">分享需要账号密钥：设置恢复密钥后，本机会在服务器上发布你的分享公钥，别人才能分享给你，你也能接受分享。</div><button className="btn sm" onClick={goSecurity}>去设置</button></div>}
      <div className="toolbar sh-bar">
        <Segmented className="compact" value={tab} onChange={setTab}
          items={[['in', pendingIn ? `收到的 · ${pendingIn} 待接受` : `收到的${data.in.length ? ` · ${data.in.length}` : ''}`, <ShIc.inbox s={13} />], ['out', `发出的${liveOut ? ` · ${liveOut} 进行中` : ''}`, <ShIc.send s={13} />]]} />
        <div className="sub" style={{ color: 'var(--txt-3)' }}>你在此服务器上的用户名 <b className="sh-me">{data.self || '—'}</b></div>
        <div style={{ flex: 1 }} />
        <button className="btn sm ghost" onClick={load}><Icon.refresh s={13} /> 刷新</button>
      </div>
      {data.error && <div className="notice bad"><Icon.alert s={14} /> {data.error}</div>}
      <div className="panel list sh-panel">
        {list.length === 0 ? <div className="empty" style={{ padding: 52 }}>
          <div className="halo"><ThinkingOrb state="breathing" size={32} theme="dark" /></div>
          {tab === 'in' ? <><h3>还没有人分享环境给你</h3><div className="sh-maxw">同一同步服务器上的其他用户可以把环境分享给 <b>{data.self}</b>。你会在这里看到，并选择接受或拒绝。</div></>
            : <><h3>你还没有分享过环境</h3><div className="sh-maxw">在「环境」列表的操作栏点击分享图标，选择接收者、是否包含登录状态和有效期。服务器只保存加密后的分享包。</div></>}
        </div> : <div className="sh-list">{list.map((s, i) => {
          const [lt, lc] = shareLeft(s.expiresAt);
          const [st, sc] = STATUS[s.status] || [s.status, 'unknown'];
          const who = tab === 'in' ? s.from : s.to;
          return <div key={s.id} className={`sh-row rise ${live(s) ? '' : 'ended'}`} style={{ ['--i' as any]: i }}>
            <Avatar seed={s.id} os={s.os || 'windows'} />
            <div className="sh-main">
              <div className="sh-title"><span className="nm sh-ell" title={s.name}>{s.name}</span><ModeBadge mode={s.mode} /></div>
              <div className="sub sh-meta">
                <span className="sh-who"><ShIc.users s={12} /> {tab === 'in' ? '来自' : '分享给'} <b>{who}</b></span>
                <span>· {ago(s.createdAt)}</span>
                {s.size > 0 && <span className="num">· {size(s.size)}</span>}
              </div>
              {s.note && <div className="sh-note" title={s.note}>“{s.note}”</div>}
            </div>
            <div className="sh-state">
              <span className={`badge ${sc}`}><i className="d" />{st}</span>
              {live(s) && <span className={`sh-left ${lc}`} title={new Date(s.expiresAt * 1000).toLocaleString()}><ShIc.clock s={12} /> {lt}</span>}
              {!live(s) && <span className="sh-left mute">{new Date(s.updatedAt * 1000).toLocaleDateString()}</span>}
            </div>
            <div className="sh-acts">
              {busy === s.id ? <span className="sh-spin"><ThinkingOrb state="weaving" size={20} theme="dark" /></span> : tab === 'in' ? (
                s.status === 'pending' ? <>
                  <button className="btn sm primary" disabled={!data.ready || !!busy} title={data.ready ? '' : '请先设置恢复密钥'} onClick={() => accept(s)}><Icon.check s={13} /> 接受</button>
                  <button className="btn sm ghost" disabled={!!busy} onClick={() => decline(s)}>拒绝</button>
                </> : s.status === 'accepted' ? <>
                  {s.localId ? <span className="sub sh-added"><Icon.check s={12} /> 已在环境列表</span> : <button className="btn sm" disabled={!!busy} onClick={() => accept(s)}><Icon.download s={13} /> 重新下载</button>}
                  <button className="btn sm ghost danger" disabled={!!busy} onClick={() => leave(s)}><ShIc.logout s={13} /> 退出</button>
                </> : <span className="sub">本机副本已移除</span>
              ) : live(s) ? <>
                <button className="btn sm" disabled={!!busy} onClick={() => setExtend(s)}><ShIc.clock s={13} /> 续期</button>
                <button className="btn sm ghost danger" disabled={!!busy} onClick={() => revoke(s)}>撤销</button>
              </> : <span className="sub">已结束</span>}
            </div>
          </div>;
        })}</div>}
      </div>
      {extend && <ExtendModal share={extend} onClose={() => setExtend(null)} onDone={load} />}
    </>
  );
};

// ---------------------------------------------------------------- expiry picker

type DaysKey = '1' | '7' | '30' | '90' | 'custom';
const DaysPicker: React.FC<{ value: DaysKey; custom: string; onChange: (v: DaysKey) => void; onCustom: (v: string) => void }> = ({ value, custom, onChange, onCustom }) => (
  <>
    <Segmented className="block" value={value} onChange={onChange} items={[['1', '1 天'], ['7', '7 天'], ['30', '30 天'], ['90', '90 天'], ['custom', '自定义']]} />
    {value === 'custom' && <div className="sh-custom"><input type="number" min={1} max={366} value={custom} onChange={e => onCustom(e.target.value)} /><span>天（1–366）</span></div>}
  </>
);
const daysOf = (k: DaysKey, custom: string) => k === 'custom' ? Math.floor(Number(custom)) : Number(k);
const daysOk = (n: number) => Number.isFinite(n) && n >= 1 && n <= 366;
const until = (n: number) => daysOk(n) ? new Date(Date.now() + n * 86400000).toLocaleDateString() : '—';

const ExtendModal: React.FC<{ share: ShareInfo; onClose: () => void; onDone: () => void }> = ({ share, onClose, onDone }) => {
  const toast = useToast();
  const [out, close] = useExit(onClose);
  const [k, setK] = useState<DaysKey>('30');
  const [custom, setCustom] = useState('14');
  const [busy, setBusy] = useState(false);
  const n = daysOf(k, custom);
  const go = async () => {
    setBusy(true);
    try { await api.ExtendShare(share.id, n); toast(`已续期到 ${until(n)}`, 'ok'); onDone(); close(); } catch (e: any) { toast(errText(e), 'bad'); } finally { setBusy(false); }
  };
  return <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
    <div className="modal">
      <div className="dhead"><div className="glyph"><ShIc.clock s={18} /></div><div className="sh-h"><h3>续期分享</h3><div className="sub sh-ell">{share.name} · 分享给 {share.to}</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
      <div className="dbody">
        <Field label="从今天起有效" hint={`当前 ${shareLeft(share.expiresAt)[0]}；续期后到期日为 ${until(n)}`}><DaysPicker value={k} custom={custom} onChange={setK} onCustom={setCustom} /></Field>
      </div>
      <div className="dfoot"><div className="spacer" /><button className="btn" onClick={() => close()}>取消</button><button className="btn primary" disabled={busy || !daysOk(n)} onClick={go}>{busy ? <ThinkingOrb state="weaving" size={20} theme="dark" /> : <Icon.check s={15} />} 续期</button></div>
    </div>
  </div>;
};

// ---------------------------------------------------------------- share dialog

export const ShareDialog: React.FC<{ profile: Profile; running: boolean; onClose: () => void; onSecurity?: () => void }> = ({ profile, running, onClose, onSecurity }) => {
  const toast = useToast();
  const [out, close] = useExit(onClose);
  const [users, setUsers] = useState<string[] | null>(null);
  const [err, setErr] = useState('');
  const [to, setTo] = useState('');
  const [mode, setMode] = useState<'session' | 'config'>('config');
  const [k, setK] = useState<DaysKey>('7');
  const [custom, setCustom] = useState('14');
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [ready, setReady] = useState(true);
  useEffect(() => {
    api.ShareList().then((l: any) => setReady(!!l?.ready)).catch(() => {});
    api.ShareUsers().then((u: any) => { setUsers(u || []); if (u?.length) setTo(u[0]); }).catch((e: any) => { setUsers([]); setErr(errText(e)); });
  }, []);
  const n = daysOf(k, custom);
  const blocked = mode === 'session' && running;
  const go = async () => {
    setBusy(true);
    try { await api.CreateShare({ profileId: profile.id, to, mode, days: n, note: note.trim() } as any); toast(`已分享「${profile.name}」给 ${to}`, 'ok'); close(); }
    catch (e: any) { toast(errText(e), 'bad'); } finally { setBusy(false); }
  };
  return <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && !busy && close()}>
    <div className="modal sh-dialog">
      <div className="dhead"><div className="glyph"><ShIc.share s={18} /></div><div className="sh-h"><h3>分享环境</h3><div className="sub sh-ell" title={profile.name}>{profile.name}</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" disabled={busy} onClick={() => close()}><Icon.x /></button></div></div>
      <div className="dbody">
        {!ready ? <div className="sh-blank"><ShIc.key s={26} /><h4>先设置恢复密钥</h4><div className="sub">分享使用账号密钥加密：设置恢复密钥后才能分享或接收环境。</div>{onSecurity && <button className="btn sm primary" onClick={() => { close(); onSecurity(); }}>去设置</button>}</div>
          : users === null ? <div className="sh-blank"><ThinkingOrb state="connecting" size={32} theme="dark" /><div className="sub">正在读取服务器上的用户…</div></div>
            : err ? <div className="notice bad"><Icon.alert s={14} /> {err}</div>
              : users.length === 0 ? <div className="sh-blank"><ShIc.users s={26} /><h4>没有可以接收分享的用户</h4><div className="sub">对方需要是同一台同步服务器上的用户（ghostsync.php 的 GHOST_USERS），并在客户端设置过恢复密钥。</div></div>
                : <>
                  <Field label="接收者"><SelectField value={to} options={users.map(u => [u, u] as [string, string])} onChange={setTo} /></Field>
                  <Field label="分享内容">
                    <Segmented className="block" value={mode} onChange={setMode} items={[['config', '仅配置', <Icon.settings s={13} />], ['session', '包含登录状态', <Icon.lock s={13} />]]} />
                    <div className={`sh-mode-hint ${mode}`}>{mode === 'config'
                      ? '指纹、代理、启动页等配置。对方启动后是一个全新的、未登录的浏览器。'
                      : '额外包含 Cookies、本地存储等浏览器数据：对方打开即是已登录状态。请只分享给信任的人。'}</div>
                  </Field>
                  {blocked && <div className="notice warn"><Icon.alert s={14} /> 该环境正在运行，请先关闭浏览器再分享登录状态。</div>}
                  <Field label="有效期" hint={`到期日 ${until(n)}，到期后对方的副本自动删除；可随时撤销或续期`}><DaysPicker value={k} custom={custom} onChange={setK} onCustom={setCustom} /></Field>
                  <Field label="备注（可选）"><input type="text" maxLength={200} value={note} onChange={e => setNote(e.target.value)} placeholder="例如：帮忙回一下这周的消息" /></Field>
                  <div className="sh-fine"><Icon.shield s={13} /> 分享包在本机用对方的公钥加密后上传，服务器无法解密。对方的配置为只读，他在浏览器里产生的数据只留在他本机。</div>
                </>}
      </div>
      <div className="dfoot"><div className="spacer" /><button className="btn" disabled={busy} onClick={() => close()}>取消</button>
        <button className="btn primary" disabled={busy || !ready || !to || !users?.length || blocked || !daysOk(n)} onClick={go}>{busy ? <><ThinkingOrb state="weaving" size={20} theme="dark" /> 加密上传中…</> : <><ShIc.send s={14} /> 分享</>}</button></div>
    </div>
  </div>;
};
