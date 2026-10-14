import React, { useEffect, useRef, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api } from './api';
import { Icon, Segmented, ago, useExit, useToast } from './ui';

type Info = { running: boolean; snapshotAt: number; snapshot: number; staged: number };
type Result = { count: number; skipped: number; mode: string; staged: string };

const countCookies = (t: string) => {
  const s = t.trim(); if (!s) return 0;
  if (/^[[{]/.test(s)) { try { const j = JSON.parse(s); return Array.isArray(j) ? j.length : (j.cookies?.length ?? 1); } catch { return 0; } }
  return s.split('\n').filter(l => l.trim() && (!l.startsWith('#') || l.startsWith('#HttpOnly_')) && l.split('\t').length >= 7).length;
};
const guess = (t: string) => { const s = t.trim(); return !s ? '' : /^[[{]/.test(s) ? 'JSON' : /\t/.test(s) ? 'Netscape' : '未知格式'; };
const domains = (t: string) => {
  const s = t.trim(); const set = new Set<string>();
  try {
    if (/^[[{]/.test(s)) { const j = JSON.parse(s); (Array.isArray(j) ? j : j.cookies || []).forEach((c: any) => c?.domain && set.add(String(c.domain).replace(/^\./, ''))); }
    else s.split('\n').forEach(l => { const c = l.split('\t'); if (c.length >= 7) set.add(c[0].replace('#HttpOnly_', '').replace(/^\./, '')); });
  } catch { /* */ }
  return [...set];
};

/** Import / export a profile's cookies. Running profiles are read and written live over CDP; stopped ones are staged for the next launch. */
export const CookieModal: React.FC<{ profile: { id: string; name: string }; initial?: 'import' | 'export'; onClose: () => void; onStaged?: (staged: string) => void }> = ({ profile, initial = 'import', onClose, onStaged }) => {
  const toast = useToast();
  const [out, close] = useExit(onClose);
  const [mode, setMode] = useState<'import' | 'export'>(initial);
  const [fmtIn, setFmtIn] = useState<'auto' | 'json' | 'netscape'>('auto');
  const [fmtOut, setFmtOut] = useState<'json' | 'netscape'>('json');
  const [text, setText] = useState('');
  const [exp, setExp] = useState('');
  const [expErr, setExpErr] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  const [res, setRes] = useState<Result | null>(null);
  const [info, setInfo] = useState<Info | null>(null);
  const [drag, setDrag] = useState(false);
  const file = useRef<HTMLInputElement>(null);

  const loadInfo = () => api.CookieInfo(profile.id).then(i => setInfo(i as Info)).catch(() => setInfo(null));
  useEffect(() => { loadInfo(); /* eslint-disable-next-line */ }, [profile.id]);
  useEffect(() => {
    if (mode !== 'export') return;
    setBusy(true); setExpErr('');
    api.ExportCookies(profile.id, fmtOut).then(t => setExp(t || '')).catch((e: any) => { setExp(''); setExpErr(String(e?.message || e)); }).finally(() => setBusy(false));
  }, [mode, fmtOut, profile.id]);

  const readFile = async (fs: FileList | null) => {
    if (!fs?.length) return;
    const t = await fs[0].text(); setText(t); setErr(''); setRes(null);
    if (/\.txt$/i.test(fs[0].name) && fmtIn === 'json') setFmtIn('auto');
  };
  const doImport = async () => {
    setBusy(true); setErr(''); setRes(null);
    try {
      const r = (await api.ImportCookies(profile.id, fmtIn, text)) as Result;
      setRes(r);
      if (r.mode === 'staged') onStaged?.(r.staged);
      toast(r.mode === 'live' ? `已写入 ${r.count} 条 Cookie` : `已暂存 ${r.count} 条 Cookie，下次启动时写入`);
      setText(''); loadInfo();
    } catch (e: any) { setErr(String(e?.message || e)); }
    finally { setBusy(false); }
  };
  const copy = async () => { try { await navigator.clipboard.writeText(exp); toast('已复制到剪贴板'); } catch { toast('复制失败，请手动选择文本复制', 'bad'); } };
  const download = () => {
    const safe = profile.name.replace(/[\\/:*?"<>|\s]+/g, '_') || 'cookies';
    const blob = new Blob([exp], { type: fmtOut === 'json' ? 'application/json' : 'text/plain' });
    const a = document.createElement('a'); a.href = URL.createObjectURL(blob); a.download = `${safe}-cookies.${fmtOut === 'json' ? 'json' : 'txt'}`;
    document.body.appendChild(a); a.click(); a.remove(); setTimeout(() => URL.revokeObjectURL(a.href), 2000);
  };

  const nIn = countCookies(text); const nOut = countCookies(exp);
  const doms = domains(mode === 'import' ? text : exp);

  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal tall" style={{ width: 'min(720px, 94vw)' }}>
        <div className="dhead">
          <div className="glyph"><Icon.lock s={18} /></div>
          <div style={{ minWidth: 0 }}><h3>Cookie 导入 / 导出</h3><div className="sub ellip">{profile.name}</div></div>
          <div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div>
        </div>
        <div className="dbody">
          <div className={`ck-status ${info?.running ? 'live' : ''}`}>
            <span className={`ck-dot ${info?.running ? 'on' : ''}`} />
            {!info ? <span className="sub">正在读取环境状态…</span> : info.running
              ? <span><b>运行中</b><span className="sub"> · 导入立即写入浏览器，导出读取实时 Cookie</span></span>
              : <span><b>未运行</b><span className="sub"> · 导入的 Cookie 下次启动时写入{info.snapshotAt ? ` · 导出使用 ${ago(info.snapshotAt)}关闭时的快照（${info.snapshot} 条）` : ' · 还没有关闭时的快照'}</span></span>}
            {info && !info.running && info.staged > 0 && <span className="badge warn" style={{ marginLeft: 'auto' }}>待写入 <span className="num">{info.staged}</span></span>}
          </div>

          <div className="ck-bar">
            <Segmented value={mode} onChange={m => { setMode(m); setErr(''); setRes(null); }} items={[['import', '导入', <Icon.upload s={14} />], ['export', '导出', <Icon.download s={14} />]]} />
            <div className="spacer" />
            <span className="sub" style={{ fontSize: 12 }}>格式</span>
            {mode === 'import'
              ? <Segmented className="compact" value={fmtIn} onChange={setFmtIn} items={[['auto', '自动'], ['json', 'JSON'], ['netscape', 'Netscape']]} />
              : <Segmented className="compact" value={fmtOut} onChange={setFmtOut} items={[['json', 'JSON'], ['netscape', 'Netscape']]} />}
          </div>

          {mode === 'import' ? <>
            <div className={'dropzone' + (drag ? ' on' : '')} onDragOver={e => { e.preventDefault(); setDrag(true); }} onDragLeave={() => setDrag(false)} onDrop={e => { e.preventDefault(); setDrag(false); readFile(e.dataTransfer.files); }}>
              <textarea className="ck-text" value={text} spellCheck={false} onChange={e => { setText(e.target.value); setErr(''); setRes(null); }} autoFocus
                placeholder={'粘贴 Cookie-Editor / EditThisCookie 导出的 JSON，或 cookies.txt（Netscape）内容，也可以把文件拖到这里\n\n[ { "domain": ".facebook.com", "name": "c_user", "value": "1000…", "path": "/", "secure": true } ]'} />
            </div>
            <div className="ck-tools">
              <button type="button" className="btn sm" onClick={() => file.current?.click()}><Icon.folder s={13} /> 选择文件</button>
              <button type="button" className="btn sm" onClick={async () => { try { const t = await navigator.clipboard.readText(); if (t) { setText(t); setErr(''); setRes(null); } } catch { toast('无法读取剪贴板，请直接 Ctrl+V', 'bad'); } }}><Icon.copy s={13} /> 从剪贴板粘贴</button>
              {text && <button type="button" className="btn sm ghost" onClick={() => { setText(''); setErr(''); }}>清空</button>}
              <input ref={file} type="file" accept=".json,.txt,text/plain,application/json" hidden onChange={e => { readFile(e.target.files); e.target.value = ''; }} />
              <div className="spacer" />
              {text.trim() && <span className="sub num" style={{ fontSize: 12 }}>{guess(text)} · {nIn ? `约 ${nIn} 条` : '未识别到 Cookie'}{doms.length ? ` · ${doms.slice(0, 3).join('、')}${doms.length > 3 ? ` 等 ${doms.length} 个域名` : ''}` : ''}</span>}
            </div>
            {err && <div className="notice bad" style={{ marginTop: 12, marginBottom: 0 }}><Icon.alert /><span className="grow">{err}</span></div>}
            {res && <div className={`notice ${res.mode === 'live' ? 'ok' : 'warn'}`} style={{ marginTop: 12, marginBottom: 0 }}>
              <Icon.check /><span className="grow"><b>{res.mode === 'live' ? `已写入 ${res.count} 条 Cookie` : `已暂存 ${res.count} 条 Cookie`}</b>
                <span className="sub"> · {res.mode === 'live' ? '刷新页面即可生效' : '下次启动这个环境时写入'}{res.skipped ? ` · 跳过 ${res.skipped} 条无效记录` : ''}</span></span>
            </div>}
          </> : <>
            <div className="ck-out">
              {busy && <div className="ck-loading"><ThinkingOrb state="searching" size={32} theme="dark" /><span>正在读取 Cookie…</span></div>}
              <textarea className="ck-text" readOnly value={expErr ? '' : exp} spellCheck={false} placeholder={busy ? '' : '这个环境还没有 Cookie'} />
            </div>
            <div className="ck-tools">
              <button type="button" className="btn sm" disabled={!exp || busy} onClick={copy}><Icon.copy s={13} /> 复制</button>
              <button type="button" className="btn sm" disabled={!exp || busy} onClick={download}><Icon.download s={13} /> 下载 .{fmtOut === 'json' ? 'json' : 'txt'}</button>
              <div className="spacer" />
              {!busy && exp && <span className="sub num" style={{ fontSize: 12 }}>{nOut} 条{doms.length ? ` · ${doms.length} 个域名` : ''}{info && !info.running ? (info.snapshotAt ? ' · 来自快照' : ' · 来自待写入数据') : ' · 实时'}</span>}
            </div>
            {expErr && <div className="notice bad" style={{ marginTop: 12, marginBottom: 0 }}><Icon.alert /><span className="grow">导出失败：{expErr}</span></div>}
          </>}
        </div>
        <div className="dfoot">
          <span className="sub" style={{ fontSize: 11.5 }}>{mode === 'import' ? '同名 Cookie（域名 + 路径 + 名称相同）会被覆盖' : 'Cookie 等同于登录凭据，请妥善保管导出的文件'}</span>
          <div className="spacer" />
          <button className="btn" onClick={() => close()}>{res ? '完成' : '关闭'}</button>
          {mode === 'import' && <button className="btn primary" disabled={busy || !text.trim()} onClick={doImport}>{busy ? <><ThinkingOrb state="connecting" size={20} theme="dark" /> 导入中…</> : <><Icon.upload s={15} /> {info?.running ? '立即写入' : '暂存，下次启动写入'}</>}</button>}
        </div>
      </div>
    </div>
  );
};
