// 终端：open Claude Code / Codex CLI / Gemini CLI or PowerShell in a new window,
// optionally inheriting a profile's timezone, language and proxy (terminal.go).
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, Profile, onEvent, TerminalTool, TerminalSession } from './api';
import { Icon, Pick, Segmented, Avatar, useToast, useConfirm, ago } from './ui';
import { TimezoneSelect, SearchSelect, LANGS } from './locales';

const openURL = (u: string) => { const rt = (window as any).runtime; if (rt?.BrowserOpenURL) rt.BrowserOpenURL(u); else window.open(u, '_blank'); };

/** Simple per-tool glyphs (no brand logos): a badge with a mark. */
export const ToolGlyph: React.FC<{ id: string; size?: number }> = ({ id, size = 40 }) => {
  const mark = {
    claude: <path d="M12 3.5v17M3.5 12h17M6 6l12 12M18 6L6 18" strokeWidth="2.1" />,
    codex: <><path d="M8 8l-4 4 4 4" /><path d="M16 8l4 4-4 4" /><path d="M13.5 5.5l-3 13" /></>,
    gemini: <path d="M12 3c.6 4.8 4.2 8.4 9 9-4.8.6-8.4 4.2-9 9-.6-4.8-4.2-8.4-9-9 4.8-.6 8.4-4.2 9-9z" fill="currentColor" stroke="none" />,
    shell: <><path d="M5 8l5 4-5 4" /><path d="M12 17h7" /></>,
  }[id] || <path d="M5 8l5 4-5 4" />;
  return <span className={`tg tg-${id}`} style={{ width: size, height: size }}>
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">{mark}</svg>
  </span>;
};

type Prefs = { tool: string; dir: string; envMode: string; profileId: string; tz: string; lang: string; host: string };
type Info = { prefs: Prefs; wt: boolean; home: string; os: string };

export function TerminalView({ profiles, proxyName }: { profiles: Profile[]; proxyName: Record<string, any> }) {
  const toast = useToast();
  const ask = useConfirm();
  const [tools, setTools] = useState<TerminalTool[] | null>(null);
  const [toolErr, setToolErr] = useState('');
  const [info, setInfo] = useState<Info | null>(null);
  const [pr, setPr] = useState<Prefs>({ tool: 'claude', dir: '', envMode: 'none', profileId: '', tz: '', lang: '', host: 'console' });
  const [dirState, setDirState] = useState<{ ok: boolean; msg: string } | null>(null);
  const [launching, setLaunching] = useState(false);
  const [launchErr, setLaunchErr] = useState('');
  const [sessions, setSessions] = useState<TerminalSession[]>([]);
  const [closing, setClosing] = useState('');
  const [copied, setCopied] = useState('');

  const detect = useCallback(async () => {
    setTools(null); setToolErr('');
    try { setTools(await api.TerminalTools()); } catch (e: any) { setToolErr(String(e?.message || e)); setTools([]); }
  }, []);
  useEffect(() => {
    detect();
    api.TerminalInfo().then((i: Info) => { setInfo(i); setPr(p => ({ ...p, ...i.prefs, envMode: i.prefs.envMode || 'none', host: i.prefs.host || 'console' })); }).catch(() => setInfo({ prefs: pr, wt: false, home: '', os: 'windows' }));
    api.ListTerminals().then(l => setSessions(l || []));
    return onEvent('terminal:changed', (l: TerminalSession[]) => setSessions(l || []));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [detect]);

  const set = (patch: Partial<Prefs>) => { setPr(p => ({ ...p, ...patch })); setLaunchErr(''); };
  // Validate the typed folder (debounced).
  useEffect(() => {
    if (!info) return;
    const d = pr.dir.trim();
    if (!d) { setDirState(null); return; }
    let live = true;
    const t = setTimeout(async () => {
      try { const abs = await api.CheckTerminalDir(d); if (live) setDirState({ ok: true, msg: abs }); }
      catch (e: any) { if (live) setDirState({ ok: false, msg: String(e?.message || e) }); }
    }, 350);
    return () => { live = false; clearTimeout(t); };
  }, [pr.dir, info]);

  const browse = async () => { const d = await api.PickTerminalDir(pr.dir || info?.home || ''); if (d) set({ dir: d }); };
  const tool = tools?.find(t => t.id === pr.tool);
  const prof = profiles.find(p => p.id === pr.profileId);
  const px = prof ? (prof.inlineProxy && prof.inlineProxy.host ? prof.inlineProxy : proxyName[prof.proxyId]) : null;
  const needsRelay = !!px && px.type !== 'none' && (!!px.user || String(px.type).startsWith('socks'));
  const langOpts = useMemo(() => LANGS.map(([v, l]) => [v, `${v} · ${l}`] as [string, string]), []);

  const preview: [string, string, string][] = []; // [key, value, title]
  if (pr.envMode === 'profile' && prof) {
    const f = prof.fingerprint;
    if (f.timezone) preview.push(['TZ', f.timezone + (f.timezoneFromIP ? '（跟随 IP）' : ''), '时区']);
    if ((f.languages || [])[0]) preview.push(['LANG', posix(f.languages[0]), '语言']);
    if (px && px.type !== 'none' && px.host) preview.push(['PROXY', needsRelay ? `本地中转 → ${String(px.type).toUpperCase()} ${px.host}:${px.port}` : `${String(px.type).toUpperCase()} ${px.host}:${px.port}`, needsRelay ? '带账号密码或 SOCKS 的代理经本地中转，CLI 拿到的是 http://127.0.0.1 代理' : 'HTTP_PROXY / HTTPS_PROXY / ALL_PROXY']);
    else preview.push(['PROXY', '直连（该环境未设置代理）', '']);
  } else if (pr.envMode === 'custom') {
    if (pr.tz) preview.push(['TZ', pr.tz, '时区']);
    if (pr.lang) preview.push(['LANG', posix(pr.lang), '语言']);
  }

  const blocked = !tool || !tool.installed || (dirState !== null && !dirState.ok) || (pr.envMode === 'profile' && !prof) || (pr.envMode === 'custom' && !pr.tz && !pr.lang);
  const launch = async () => {
    if (blocked || launching) return;
    setLaunching(true); setLaunchErr('');
    try {
      const s = await api.LaunchTerminal({ ...pr, dir: pr.dir.trim() } as any);
      toast(`已打开 ${s.name}${s.profileName ? ` · ${s.profileName}` : ''}`);
      setSessions(l => l.some(x => x.id === s.id) ? l : [s, ...l]);
    } catch (e: any) { setLaunchErr(String(e?.message || e)); }
    finally { setLaunching(false); }
  };
  const close = async (s: TerminalSession) => {
    if (s.tracked && !await ask({ title: `关闭「${s.name}」终端？`, body: '终端窗口和其中运行的程序会被结束，未保存的会话内容会丢失。', ok: '关闭终端', danger: true })) return;
    setClosing(s.id);
    try { await api.CloseTerminal(s.id); setSessions(l => l.filter(x => x.id !== s.id)); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); setSessions(l => l.filter(x => x.id !== s.id)); }
    finally { setClosing(''); }
  };
  const copy = (t: TerminalTool) => { navigator.clipboard?.writeText(t.installCmd).then(() => { setCopied(t.id); setTimeout(() => setCopied(''), 1600); }); };

  const installed = (tools || []).filter(t => t.installed).length;

  return (
    <div className="term">
      <div className="term-head">
        <div className="section-title" style={{ margin: 0 }}>选择工具</div>
        <span className="sub" style={{ fontSize: 12 }}>{tools === null ? '正在检测 PATH 中的命令行工具…' : `已安装 ${installed} / ${tools.length}`}</span>
        <div className="spacer" />
        <button className="btn ghost sm" onClick={detect} disabled={tools === null}><Icon.refresh s={13} /> 重新检测</button>
      </div>

      {toolErr && <div className="notice bad"><Icon.alert /><span className="grow">检测失败：{toolErr}</span><button className="btn sm" onClick={detect}>重试</button></div>}

      <div className="term-grid">
        {tools === null ? [0, 1, 2, 3].map(i => <div key={i} className="term-card panel skel"><span className="skel-b" style={{ width: 40, height: 40, borderRadius: 11 }} /><span className="skel-b" style={{ width: '60%' }} /><span className="skel-b" style={{ width: '85%' }} /></div>)
          : tools.map((t, i) => <div key={t.id} role="button" tabIndex={0} className={`term-card panel spot rise ${pr.tool === t.id ? 'on' : ''} ${t.installed ? '' : 'missing'}`} style={{ ['--i' as any]: i }}
            onClick={() => set({ tool: t.id })} onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); set({ tool: t.id }); } }}>
            <div className="tc-top">
              <ToolGlyph id={t.id} />
              <span className="tc-radio" />
            </div>
            <div className="tc-name">{t.name}</div>
            <div className="tc-desc" title={t.desc}>{t.desc}</div>
            <div className="tc-foot">
              {t.installed
                ? <span className="badge ok" title={t.path}><span className="d" />{t.version ? <span className="num">v{t.version}</span> : '已就绪'}</span>
                : <span className="badge warn"><Icon.alert s={12} /> 未安装</span>}
              <code className="tc-bin">{t.bin}</code>
            </div>
            {!t.installed && t.installCmd && <div className="tc-install" onClick={e => e.stopPropagation()}>
              <code title={t.installCmd}>{t.installCmd}</code>
              <button className="btn ghost icon" title="复制安装命令" onClick={() => copy(t)}>{copied === t.id ? <Icon.check s={13} /> : <Icon.copy s={13} />}</button>
              <button className="btn ghost icon" title="打开官方安装指南" onClick={() => openURL(t.installUrl)}><Icon.globe s={13} /></button>
            </div>}
          </div>)}
      </div>

      <div className="term-body">
        <div className="panel term-form">
          <div className="tf-row">
            <div className="tf-l"><b>工作目录</b><small>终端打开时所在的文件夹</small></div>
            <div className="tf-c">
              <div className={`tf-dir ${dirState && !dirState.ok ? 'err' : ''}`}>
                <Icon.folder s={15} />
                <input type="text" className="mono" value={pr.dir} onChange={e => set({ dir: e.target.value })} placeholder={info?.home ? `留空 = 用户目录（${info.home}）` : '留空 = 用户目录'} spellCheck={false} />
                {pr.dir && <button className="btn ghost icon" title="清空" onClick={() => set({ dir: '' })}><Icon.x s={13} /></button>}
                <button className="btn sm" onClick={browse}>浏览…</button>
              </div>
              {dirState && <div className={`tf-hint ${dirState.ok ? 'ok' : 'bad'}`}>{dirState.ok ? <Icon.check s={12} /> : <Icon.alert s={12} />}<span>{dirState.msg}</span></div>}
            </div>
          </div>

          <div className="tf-row">
            <div className="tf-l"><b>环境变量</b><small>让 CLI 的时区、语言、出口 IP 与环境一致</small></div>
            <div className="tf-c">
              <Segmented className="compact" value={pr.envMode} onChange={v => set({ envMode: v })} items={[['none', '不继承'], ['profile', '继承环境'], ['custom', '自定义']]} />
              {pr.envMode === 'profile' && <div className="tf-sub">
                {profiles.length === 0 ? <div className="tf-hint bad"><Icon.alert s={12} /><span>还没有环境，先在「环境」页创建一个</span></div> : <>
                  <Pick label="环境" value={pr.profileId} onChange={v => set({ profileId: v })} icon={<Icon.layers />}
                    options={[['', '选择一个环境'] as [string, string], ...profiles.map(p => [p.id, p.name || p.id] as [string, string])]} />
                  {prof && <span className="tf-prof"><Avatar seed={prof.fingerprint.noiseSeed || prof.id} os={prof.fingerprint.os} /><span className="ell">{prof.name}</span></span>}
                </>}
              </div>}
              {pr.envMode === 'custom' && <div className="tf-custom">
                <div><label>时区 (TZ)</label><TimezoneSelect value={pr.tz} onChange={v => set({ tz: v })} /></div>
                <div><label>语言 (LANG / LC_ALL)</label><SearchSelect value={pr.lang} options={langOpts} onChange={v => set({ lang: v })} placeholder="搜索语言，如 ja / English" /></div>
              </div>}
              {preview.length > 0 && <div className="tf-env">{preview.map(([k, v, t]) => <span key={k} className="envchip" title={t}><b>{k}</b><span className="ell">{v}</span></span>)}</div>}
            </div>
          </div>

          <div className="tf-row">
            <div className="tf-l"><b>窗口</b><small>{pr.host === 'wt' ? 'Windows Terminal 新窗口' : '独立的 PowerShell 控制台窗口'}</small></div>
            <div className="tf-c">
              <Segmented className="compact" value={info?.wt ? pr.host : 'console'} onChange={v => set({ host: v })} items={info?.wt ? [['console', 'PowerShell 窗口'], ['wt', 'Windows Terminal']] : [['console', 'PowerShell 窗口']]} />
              {info && !info.wt && <div className="tf-hint"><span>未检测到 Windows Terminal (wt.exe)，使用 PowerShell 窗口</span></div>}
              {info?.wt && pr.host === 'wt' && <div className="tf-hint"><span>Windows Terminal 由它自己管理窗口：GhostBrowser 无法替你关闭它，「关闭」只会停止代理中转</span></div>}
            </div>
          </div>

          {launchErr && <div className="notice bad" style={{ margin: '4px 0 0' }}><Icon.alert /><span className="grow">{launchErr}</span><button className="btn sm" onClick={launch}>重试</button></div>}

          <div className="tf-go">
            <span className="sub ell" style={{ fontSize: 12 }}>
              {!tool ? '请选择一个工具' : !tool.installed ? `${tool.name} 未安装，先按卡片上的命令安装` : pr.envMode === 'profile' && !prof ? '请选择要继承的环境' : pr.envMode === 'custom' && !pr.tz && !pr.lang ? '请至少设置时区或语言' : dirState && !dirState.ok ? '工作目录无效' : `将在新窗口运行 ${tool.id === 'shell' ? tool.bin : tool.bin}`}
            </span>
            <button className="btn primary" disabled={blocked || launching} onClick={launch}>
              {launching ? <><ThinkingOrb state="connecting" size={20} theme="dark" /> 正在打开…</> : <><Icon.terminal s={15} /> 启动 {tool?.name || ''}</>}
            </button>
          </div>
        </div>

        <div className="panel term-live">
          <div className="tl-head"><b>运行中的终端</b><span className="count num">{sessions.length}</span></div>
          {sessions.length === 0 ? <div className="tl-empty">
            <Icon.terminal s={26} />
            <div>还没有从这里打开的终端</div>
            <small>启动后会出现在这里，可随时关闭；关闭窗口时代理中转也会一起停止</small>
          </div> : <div className="tl-list">{sessions.map(s => <div key={s.id} className="tl-item">
            <ToolGlyph id={s.tool} size={32} />
            <div className="tl-main">
              <div className="tl-t"><b>{s.name}</b>{s.profileName && <span className="tl-prof ell" title={s.profileName}>{s.profileName}</span>}<span className="tl-ago">{ago(s.started)}</span></div>
              <div className="tl-dir mono ell" title={s.dir}>{s.dir}</div>
              <div className="tl-env">
                {s.tz && <span className="envchip sm"><b>TZ</b><span className="ell">{s.tz}</span></span>}
                {s.lang && <span className="envchip sm"><b>LANG</b><span className="ell">{s.lang}</span></span>}
                {s.proxy && <span className="envchip sm" title={s.relay ? `本地中转 ${s.relay}` : ''}><b>PROXY</b><span className="ell">{s.relay ? `中转 → ${s.proxy}` : s.proxy}</span></span>}
                {!s.tz && !s.lang && !s.proxy && <span className="envchip sm dim"><span>系统环境</span></span>}
                {!s.tracked && <span className="envchip sm dim" title="Windows Terminal 自己管理这个窗口"><span>Windows Terminal</span></span>}
              </div>
            </div>
            <button className="btn ghost icon danger" title={s.tracked ? '关闭终端' : '移除并停止代理中转'} disabled={closing === s.id} onClick={() => close(s)}>{closing === s.id ? <ThinkingOrb state="breathing" size={20} theme="dark" /> : <Icon.x s={15} />}</button>
          </div>)}</div>}
        </div>
      </div>
    </div>
  );
}

const posix = (tag: string) => {
  const p = tag.replace('_', '-').split('-'); const r = p.slice(1).find(x => x.length === 2);
  return r ? `${p[0].toLowerCase()}_${r.toUpperCase()}.UTF-8` : `${p[0].toLowerCase()}.UTF-8`;
};
