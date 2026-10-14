import React, { useCallback, useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { ThinkingOrb } from 'thinking-orbs';
import { api } from './api';
import { Icon, useToast, useExit, Field, Segmented } from './ui';
import { ShIc } from './SharingView';

type RState = { supported: boolean; configured: boolean; unlocked: boolean; legacy: boolean; kid: string; rkCreated: number; pwChanged: number; shareReady: boolean; user: string; error?: string };
type Mode = 'setup' | 'rotate' | 'unlock' | 'password';
const errText = (e: any) => String(e?.message || e);
const day = (ts: number) => ts ? new Date(ts * 1000).toLocaleDateString() : '—';

/** 账号密钥 card: master key status, recovery key, password change. */
export const RecoveryCard: React.FC = () => {
  const [st, setSt] = useState<RState | null>(null);
  const [modal, setModal] = useState<Mode | ''>('');
  const load = useCallback(() => { api.RecoveryStatus().then((s: any) => setSt(s)).catch((e: any) => setSt({ supported: false, error: errText(e) } as RState)); }, []);
  useEffect(() => { load(); }, [load]);
  return (
    <div className="panel dv-card rk-card">
      <div className="dv-head">
        <div className="dv-glyph rk"><ShIc.key s={18} /></div>
        <div className="dv-h"><h4>账号密钥与恢复</h4><div className="sub">云端数据用账号主密钥加密；同步密码和恢复密钥都只是用来解开它的钥匙。</div></div>
        {st?.configured && (st.unlocked ? <span className="badge ok"><i className="d" />已解锁</span> : <span className="badge warn"><i className="d" />本机未解锁</span>)}
      </div>
      {!st ? <div className="dv-blank"><ThinkingOrb state="connecting" size={32} theme="dark" /><div className="sub">正在检查…</div></div>
        : st.error ? <div className="notice bad dv-in"><Icon.alert s={14} /><div className="grow">{st.error}</div><button className="btn sm" onClick={() => { setSt(null); load(); }}>重试</button></div>
          : !st.supported ? <div className="notice warn dv-in"><Icon.alert s={14} /><div className="grow">服务器版本过旧，不支持恢复密钥：请更新 ghostsync.php。</div></div>
            : !st.configured ? <div className="rk-setup">
              <div className="rk-steps">
                <div><b>1</b><span>生成一个恢复密钥，只显示一次，请离线保存</span></div>
                <div><b>2</b><span>忘记同步密码时，用恢复密钥在新设备上解锁云端数据</span></div>
                <div><b>3</b><span>之后修改同步密码无需重新加密云端数据，也能启用环境分享</span></div>
              </div>
              <div className="sub rk-fine">已有云端数据会保持可读：老账号的主密钥就是当前同步密码，不会重新上传任何环境。</div>
              <button className="btn primary" onClick={() => setModal('setup')}><ShIc.key s={15} /> 设置恢复密钥</button>
            </div>
              : !st.unlocked ? <div className="rk-setup">
                <div className="dv-self pending"><Icon.lock s={22} /><div><b>本机还没有解锁账号主密钥</b><div className="sub">同步密码可能已在其他设备修改。用新的同步密码，或用恢复密钥解锁（会设置新的同步密码）。</div></div></div>
                <button className="btn primary" onClick={() => setModal('unlock')}><Icon.lock s={15} /> 解锁本机</button>
              </div>
                : <div className="rk-grid">
                  <div className="rk-kv"><span>密钥指纹</span><b className="mono">{st.kid.replace(/(.{4})/g, '$1 ').trim()}</b></div>
                  <div className="rk-kv"><span>恢复密钥生成于</span><b>{day(st.rkCreated)}</b></div>
                  <div className="rk-kv"><span>同步密码更新于</span><b>{day(st.pwChanged)}</b></div>
                  <div className="rk-kv"><span>环境分享</span><b>{st.shareReady ? <span className="rk-ok">已启用 · 用户名 {st.user}</span> : '未启用'}</b></div>
                  {st.legacy && <div className="sub rk-fine rk-wide">此账号从旧版本迁移：主密钥沿用最初的同步密码，旧数据无需重新加密。修改密码后，旧密码仍能解密已有数据，如果怀疑密码泄露，请同时更换服务器令牌。</div>}
                  <div className="rk-acts rk-wide">
                    <button className="btn sm" onClick={() => setModal('password')}><Icon.lock s={13} /> 修改同步密码</button>
                    <button className="btn sm ghost" onClick={() => setModal('rotate')}><Icon.refresh s={13} /> 轮换恢复密钥</button>
                  </div>
                </div>}
      {/* portal: .panel's backdrop-filter would trap a fixed overlay inside the card */}
      {modal && createPortal(<RecoveryKeyModal mode={modal} onClose={() => setModal('')} onDone={load} />, document.body)}
    </div>
  );
};

/** Shows a recovery key once, with copy / download and a "我已保存" confirm. */
const KeyReveal: React.FC<{ rk: string; saved: boolean; setSaved: (v: boolean) => void }> = ({ rk, saved, setSaved }) => {
  const toast = useToast();
  const [copied, setCopied] = useState(false);
  const copy = async () => { try { await navigator.clipboard.writeText(rk); setCopied(true); toast('已复制恢复密钥'); setTimeout(() => setCopied(false), 1800); } catch { toast('复制失败，请手动抄写', 'bad'); } };
  const download = async () => { try { const p: any = await api.SaveRecoveryKeyFile(rk); if (p) toast('已保存为文本文件', 'ok'); } catch (e: any) { toast(errText(e), 'bad'); } };
  return <>
    <div className="rk-warn"><Icon.alert s={15} /><div>这是唯一一次显示。服务器和 GhostBrowser 都无法找回它：丢失恢复密钥且忘记同步密码，云端数据将<b>永久无法解密</b>。</div></div>
    <div className="rk-key" aria-label="恢复密钥">{rk.split('-').map((g, i) => <span key={i} className="mono">{g}</span>)}</div>
    <div className="rk-tools">
      <button className="btn sm" onClick={copy}>{copied ? <Icon.check s={13} /> : <Icon.copy s={13} />} {copied ? '已复制' : '复制'}</button>
      <button className="btn sm" onClick={download}><Icon.download s={13} /> 下载 .txt</button>
    </div>
    <label className={`rk-confirm ${saved ? 'on' : ''}`}><input type="checkbox" checked={saved} onChange={e => setSaved(e.target.checked)} /><span className="rk-box">{saved && <Icon.check s={12} />}</span>我已把恢复密钥保存在安全的地方</label>
  </>;
};

export const RecoveryKeyModal: React.FC<{ mode: Mode; onClose: () => void; onDone: () => void }> = ({ mode, onClose, onDone }) => {
  const toast = useToast();
  const [out, close] = useExit(onClose);
  const [rk, setRk] = useState('');
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [how, setHow] = useState<'password' | 'recovery'>('password');
  const [pw, setPw] = useState('');
  const [pw2, setPw2] = useState('');
  const [pw3, setPw3] = useState('');
  const [rkIn, setRkIn] = useState('');
  const showing = !!rk;
  const title = showing ? '保存你的恢复密钥' : ({ setup: '设置恢复密钥', rotate: '轮换恢复密钥', unlock: '解锁本机', password: '修改同步密码' } as const)[mode];
  const sub = showing ? '用它可以在任何设备上解锁云端数据' : ({ setup: '启用账号主密钥，生成一次性显示的恢复密钥', rotate: '旧的恢复密钥会立即失效', unlock: '解开账号主密钥后即可同步', password: '只重新包装主密钥，不会重新加密云端数据' } as const)[mode];
  const run = async (f: () => Promise<any>) => {
    setBusy(true); setErr('');
    try { return await f(); } catch (e: any) { setErr(errText(e)); return undefined; } finally { setBusy(false); }
  };
  const submit = async () => {
    if (mode === 'setup') { const k = await run(() => api.SetupRecoveryKey()); if (k) { setRk(k); onDone(); } return; }
    if (mode === 'rotate') { const k = await run(() => api.RotateRecoveryKey(pw)); if (k) { setRk(k); onDone(); } return; }
    if (mode === 'unlock') {
      if (how === 'recovery' && pw2 !== pw3) { setErr('两次输入的新密码不一致'); return; }
      const ok = await run(async () => { how === 'password' ? await api.UnlockSyncWithPassword(pw) : await api.UnlockSyncWithRecoveryKey(rkIn, pw2); return true; });
      if (ok) { toast('本机已解锁', 'ok'); onDone(); close(); }
      return;
    }
    if (pw2 !== pw3) { setErr('两次输入的新密码不一致'); return; }
    const ok = await run(async () => { await api.ChangeSyncPassword(pw, pw2); return true; });
    if (ok) { toast('同步密码已修改。其他设备会继续正常同步，新设备请用新密码解锁', 'ok'); onDone(); close(); }
  };
  const canSubmit = !busy && (mode === 'setup' || (mode === 'rotate' && pw.length > 0)
    || (mode === 'unlock' && (how === 'password' ? pw.length > 0 : rkIn.trim().length >= 32 && pw2.length >= 8 && pw3.length >= 8))
    || (mode === 'password' && pw.length > 0 && pw2.length >= 8 && pw3.length >= 8));
  const dismissable = !showing && !busy;
  const strength = pw2.length === 0 ? '' : pw2.length < 8 ? '至少 8 位' : pw2.length < 12 ? '可用，建议 12 位以上' : '强度良好';
  return <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && dismissable && close()}>
    <div className="modal rk-modal">
      <div className="dhead"><div className="glyph"><ShIc.key s={18} /></div><div><h3>{title}</h3><div className="sub">{sub}</div></div>
        <div style={{ marginLeft: 'auto' }}>{dismissable && <button className="btn ghost icon" onClick={() => close()}><Icon.x /></button>}</div></div>
      <div className="dbody">
        {showing ? <KeyReveal rk={rk} saved={saved} setSaved={setSaved} /> : <>
          {mode === 'setup' && <div className="rk-intro">
            <div className="rk-flow"><span className="node"><Icon.lock s={13} />同步密码</span><span className="wire" /><span className="node hi"><ShIc.key s={13} />账号主密钥</span><span className="wire" /><span className="node"><Icon.shield s={13} />恢复密钥</span></div>
            <div className="sub">两把钥匙都能解开同一个主密钥，服务器只保存加密后的密钥包，看不到密码、主密钥或恢复密钥。</div>
          </div>}
          {mode === 'unlock' && <Field><Segmented className="block" value={how} onChange={v => { setHow(v); setErr(''); }} items={[['password', '用同步密码'], ['recovery', '用恢复密钥']]} /></Field>}
          {(mode === 'rotate' || mode === 'password' || (mode === 'unlock' && how === 'password')) &&
            <Field label={mode === 'password' ? '当前同步密码' : '同步密码'}><input type="password" autoFocus value={pw} onChange={e => setPw(e.target.value)} onKeyDown={e => e.key === 'Enter' && canSubmit && submit()} /></Field>}
          {mode === 'unlock' && how === 'recovery' && <Field label="恢复密钥" hint="8 组，每组 4 位；不区分大小写，横线可省略">
            <input type="text" className="mono rk-input" autoFocus spellCheck={false} value={rkIn} placeholder="XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX" onChange={e => setRkIn(e.target.value.toUpperCase())} /></Field>}
          {(mode === 'password' || (mode === 'unlock' && how === 'recovery')) && <>
            <Field label="新同步密码" hint={strength}><input type="password" value={pw2} onChange={e => setPw2(e.target.value)} /></Field>
            <Field label="再次输入新密码"><input type="password" value={pw3} onChange={e => setPw3(e.target.value)} onKeyDown={e => e.key === 'Enter' && canSubmit && submit()} /></Field>
          </>}
          {err && <div className="notice bad"><Icon.alert s={14} /> {err}</div>}
        </>}
      </div>
      <div className="dfoot">
        {showing ? <><div className="spacer" /><button className="btn primary" disabled={!saved} onClick={() => close()}><Icon.check s={15} /> 我已保存，完成</button></>
          : <><div className="spacer" /><button className="btn" disabled={busy} onClick={() => close()}>取消</button>
            <button className="btn primary" disabled={!canSubmit} onClick={submit}>{busy ? <ThinkingOrb state="weaving" size={20} theme="dark" /> : mode === 'setup' || mode === 'rotate' ? <ShIc.key s={15} /> : <Icon.check s={15} />} {({ setup: '生成恢复密钥', rotate: '生成新密钥', unlock: '解锁', password: '修改密码' } as const)[mode]}</button></>}
      </div>
    </div>
  </div>;
};
