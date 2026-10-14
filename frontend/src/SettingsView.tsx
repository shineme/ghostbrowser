import React, { useEffect, useState } from 'react';
import { api, Settings } from './api';
import { Icon, Field, Toggle, useToast } from './ui';
import type { EnvStatus } from './EnvSetup';
import { KernelInstaller } from './KernelInstall';
import { SessionScopeChips } from './SessionStore';
import { FirefoxSettingsCard, FoxIcon, useFirefoxInfo } from './FirefoxKernel';
import { AppShellSettings } from './AppShell';

const mb = (b: number) => b ? `${(b / 1048576).toFixed(b > 1e8 ? 0 : 1)} MB` : '';

export const SettingsView: React.FC<{ settings: Settings; onSaved: (s: Settings) => void }> = ({ settings, onSaved }) => {
  const toast = useToast();
  const [s, setS] = useState<Settings>({ ...settings });
  const [testing, setTesting] = useState(false);
  const set = (patch: Partial<Settings>) => setS(v => ({ ...v, ...patch }));

  const [defaultKernel, setDefaultKernel] = useState('');
  const [autoChromium, setAutoChromium] = useState('');
  const [chromiumMajor, setChromiumMajor] = useState('');
  useEffect(() => { api.ChromiumPath().then(setAutoChromium).catch(() => {}); api.ChromiumMajor().then(setChromiumMajor).catch(() => {}); }, [(s as any)?.chromiumPath]);
  const [compat, setCompat] = useState<{ version: string; major: number; level: string; message: string } | null>(null);
  useEffect(() => {
    const t = setTimeout(() => { (api as any).ChromiumCompat((s as any)?.chromiumPath || '').then(setCompat).catch(() => setCompat(null)); }, 300);
    return () => clearTimeout(t);
  }, [(s as any)?.chromiumPath]);
  const refreshDefault = () => api.DefaultKernelPath().then(setDefaultKernel).catch(() => setDefaultKernel(''));
  useEffect(() => { refreshDefault(); }, []);
  const save = async () => { const out = await api.SaveSettings(s); onSaved(out); refreshDefault(); };
  const resetKernel = async () => { const f = await api.DefaultKernelPath(); if (f) set({ kernelPath: f }); else toast('未找到已安装的定制 Chromium，请点击“浏览”手动指定', 'bad'); };
  const useKit = async () => { try { const out = await api.UseUndetectableKit(); setS(out); onSaved(out); refreshDefault(); toast('已启用定制 Chromium'); } catch (e) { toast(String(e), 'bad'); } };
  const pickKernel = async () => { const f = await api.PickKernel(); if (f) set({ kernelPath: f }); };
  const pickFolder = async () => { const f = await api.PickFolder(); if (f) set({ profilesDir: f }); };
  const test = async () => { setTesting(true); try { await api.TestSync(); toast('同步服务器连接正常', 'ok'); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { setTesting(false); } };

  const [ffInfo] = useFirefoxInfo((s as any).firefoxPath);
  const [envS, setEnvS] = useState<EnvStatus | null>(null);
  useEffect(() => { api.EnvStatus().then((x: any) => setEnvS(x)).catch(() => {}); return () => {}; }, []);
  const recheck = async (force: boolean) => {
    try {
      const x = await api.EnvRecheck(force);
      setEnvS(x as any);
      toast(force ? '正在重新下载并安装环境…' : '正在检测环境…');
    } catch (e: any) { toast(String(e?.message || e), 'bad'); }
  };

  const envBusy = envS && ['checking', 'downloading', 'verifying', 'extracting'].includes(envS.phase);

  return (
    <div style={{ maxWidth: 780 }}>
      <div className="panel pad setting-card spot rise" style={{ ['--i' as any]: 0 }}>
        <div className="head"><span className="ic"><Icon.cpu /></span><div><h4>浏览器内核</h4><span>定制 Chromium + 原生指纹模板</span></div></div>
        <div className="choice">
          <button className="on" onClick={() => set({ kernelMode: 'undetectable' })}><b><Icon.layers s={15} />定制 Chromium<span className="radio" /></b><small>本地指纹库 + 原生模板，环境缺失时自动下载安装</small></button>
          <button className="ff-tile" onClick={() => document.getElementById('ff-card')?.scrollIntoView({ behavior: 'smooth', block: 'start' })}><b><FoxIcon s={15} />Firefox 内核<span className={`badge ${ffInfo?.path ? 'ok' : 'warn'}`}><span className="d" />{ffInfo ? (ffInfo.path ? ffInfo.version || '已安装' : '未安装') : '检测中…'}</span></b><small>可选：新建环境时在「指纹配置 → 内核」里选择，独立 Firefox 配置目录</small></button>
        </div>

        <div className={`geo-card ${envS?.ready ? '' : envS?.phase === 'failed' ? 'bad' : 'warn'}`}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span className={`badge ${envS?.ready ? 'ok' : envS?.phase === 'failed' ? 'fail' : 'unknown'}`}><span className="d" />{envS?.ready ? '环境就绪' : envS?.phase === 'failed' ? '安装失败' : envBusy ? '安装中' : '待安装'}</span>
            <b style={{ fontSize: 12.5 }}>{envS?.message || '尚未检测'}</b>
          </div>
          {envBusy && envS!.total > 0 && <div className="env-bar" style={{ marginTop: 8 }}><i style={{ width: `${Math.max(envS!.pct, 1)}%` }} /></div>}
          {envBusy && <div className="env-meta num"><span>{mb(envS!.done)}{envS!.total ? ` / ${mb(envS!.total)}` : ''}</span><span>{envS!.speedBps ? `${(envS!.speedBps / 1048576).toFixed(1)} MB/s` : ''}</span></div>}
          {envS?.error && <div className="sub" style={{ color: '#ffb3bc', marginTop: 4, wordBreak: 'break-all' }}>{envS.error}</div>}
          <div className="sub" style={{ marginTop: 4 }}>内核：<code className="mono">{envS?.chromeExe || defaultKernel || '未安装'}</code></div>
          <div className="sub" style={{ marginTop: 2 }}>校验：{envS?.manifestFromServer ? '服务器 manifest.json' : 'EXE 内置校验值'}{envS?.kitDir ? ` · ${envS.kitDir}` : ''}</div>
        </div>

        <div className="row" style={{ marginTop: 10, gap: 10 }}>
          <button className="btn" disabled={!!envBusy} onClick={() => recheck(false)}><Icon.refresh s={14} /> 重新检测并安装</button>
          <button className="btn ghost" disabled={!!envBusy} onClick={() => recheck(true)}><Icon.download s={14} /> 强制重新下载</button>
          {envBusy && <button className="btn ghost" onClick={() => api.EnvCancel()}>取消</button>}
        </div>

        <Field label="浏览器 EXE 路径" hint={`默认路径：${defaultKernel || '启动时自动检测，也可以手动选择'}`}>
          <div className="row"><input type="text" value={s.kernelPath} onChange={e => set({ kernelPath: e.target.value })} placeholder="定制 Chromium 的 chrome.exe" /><button className="btn" style={{ flex: 'none' }} onClick={pickKernel}><Icon.folder s={15} /> 浏览</button><button className="btn ghost" style={{ flex: 'none' }} onClick={resetKernel}><Icon.refresh s={14} /> 恢复默认</button></div>
        </Field>
        <div className="rise">
          <Field label="本地指纹包目录" hint="包含 runtime/chrome 和 fingerprints/library.json"><input type="text" value={s.undetectableKit || ''} onChange={e => set({ undetectableKit: e.target.value })} placeholder="项目目录\\undetectable-kit" /></Field>
          <button className="btn" onClick={useKit}><Icon.layers s={14} /> 使用当前已安装的 Chromium 与指纹库</button>
          <div className="sub" style={{ marginTop: 10 }}>标准 Chrome / Edge 无法运行原生模板。关闭 GhostBrowser 会停止它启动的环境，内置模板已验证 Windows。</div>
        </div>
      </div>

      <div className="panel pad setting-card spot rise" style={{ ['--i' as any]: 1 }}>
        <div className="head"><span className="ic"><Icon.phone /></span><div><h4>开源内核</h4><span>手机、平板、折叠屏、Chromebook、电视、Mac 和 Linux 环境用它启动</span></div></div>
        <Field label="chrome.exe 路径" hint={`留空自动检测：${autoChromium || '没找到已安装的 Chromium / Chrome / Edge'}${(s as any).chromiumPath ? (compat?.version ? ` · 已选内核：Chrome ${compat.version}` : '') : compat?.version ? ` · Chrome ${compat.version}` : chromiumMajor ? ` · Chrome ${chromiumMajor}` : ''}`}>
          <div className="row"><input type="text" value={s.chromiumPath || ''} onChange={e => set({ chromiumPath: e.target.value } as any)} placeholder="Chromium、Chrome for Testing、ungoogled-chromium 或你自己编译的内核" /><button className="btn" style={{ flex: 'none' }} onClick={async () => { const f = await api.PickChromium(); if (f) set({ chromiumPath: f } as any); }}><Icon.folder s={15} /> 浏览</button>{s.chromiumPath && <button className="btn ghost" style={{ flex: 'none' }} onClick={() => set({ chromiumPath: '' } as any)}><Icon.refresh s={14} /> 自动检测</button>}</div>
        </Field>
        {compat && compat.level !== 'ok' && compat.message && <div className={`notice ${compat.level === 'unsupported' ? 'bad' : 'warn'} kc-notice`}><Icon.alert /><span className="grow">{compat.message}</span></div>}
        <KernelInstaller source={(s as any).kernelSource} url={(s as any).kernelUrl} onInstalled={async path => { const out = await api.SaveSettings({ ...s, chromiumPath: path } as any); setS(out); onSaved(out); }} />
        <div className="sub" style={{ marginTop: 12 }}>指纹通过 CDP 注入，不依赖扩展，Chrome 137 以后的正式版也能用。支持 Chrome 110 及以上，推荐 120 以上的 Chromium 或 Chrome for Testing。UA 里的大版本自动跟随内核。需要保持 GhostBrowser 运行。</div>
      </div>

      <FirefoxSettingsCard index={2} value={(s as any).firefoxPath || ''} onChange={v => set({ firefoxPath: v } as any)} onInstalled={async path => { const out = await api.SaveSettings({ ...s, firefoxPath: path } as any); setS(out); onSaved(out); }} />

      <div className="panel pad setting-card spot rise" style={{ ['--i' as any]: 2 }}>
        <div className="head"><span className="ic"><Icon.terminal /></span><div><h4>数据与本地 API</h4><span>供 Puppeteer / Playwright / Selenium 接管</span></div></div>
        <Field label="环境数据目录">
          <div className="row"><input type="text" value={s.profilesDir} onChange={e => set({ profilesDir: e.target.value })} /><button className="btn" style={{ flex: 'none' }} onClick={pickFolder}><Icon.folder s={15} /> 浏览</button></div>
        </Field>
        <Toggle on={s.apiEnabled} onChange={v => set({ apiEnabled: v })} label="启用本地自动化 API" hint="供 Puppeteer / Playwright / Selenium 接管 CDP 端口" />
        <Field label="API 端口"><input type="number" value={s.apiPort} onChange={e => set({ apiPort: +e.target.value })} /></Field>
        {s.apiEnabled && <>
          <Field label="API Key" hint="请求头 X-API-Key 或网址参数 ?key=。网页无法调用本地 API（会校验 Host、Origin 和 Sec-Fetch-Site）。">
            <div className="row"><input type="text" className="mono" readOnly value={s.apiKey || ''} onFocus={e => e.target.select()} />
              <button className="btn" style={{ flex: 'none' }} onClick={() => { navigator.clipboard?.writeText(s.apiKey || ''); toast('已复制 API Key'); }}><Icon.copy s={14} /> 复制</button>
              <button className="btn ghost" style={{ flex: 'none' }} onClick={async () => { const out = await api.ResetAPIKey(); setS(out); onSaved(out); toast('已生成新的 API Key，旧 Key 立即失效'); }}><Icon.refresh s={14} /> 重新生成</button></div>
          </Field>
          <Toggle on={!s.apiKeyOff} onChange={v => set({ apiKeyOff: !v })} label="要求 API Key" hint="关闭后本机任何程序都能直接调用，不建议" />
        </>}
        <Field label="代理自动检测间隔（分钟）" hint="后台定期检测全部代理，失效的会标红并提示受影响的环境；0 = 关闭"><input type="number" min={0} max={1440} value={s.proxyCheckMins ?? 0} onChange={e => set({ proxyCheckMins: Math.max(0, Math.min(1440, Math.round(+e.target.value || 0))) })} /></Field>
        <div className="kv" style={{ alignItems: 'center', borderBottom: 0 }}><span className="k">默认操作系统</span><span className="v"><select style={{ width: 160 }} value={s.defaultOS} onChange={e => set({ defaultOS: e.target.value })}><option value="windows">Windows</option><option value="macos">macOS</option><option value="linux">Linux</option></select></span></div>
      </div>

      <div className="panel pad setting-card spot rise" style={{ ['--i' as any]: 3 }}>
        <div className="head"><span className="ic"><Icon.cloud /></span><div><h4>云端同步</h4><span>你的 VPS + PHP，端到端加密</span></div></div>
        <Field label="同步服务器地址" hint="填 ghostsync.php 的完整网址（单文件 PHP，无需伪静态）"><input type="text" value={s.syncServer} onChange={e => set({ syncServer: e.target.value })} placeholder="https://你的域名/ghostsync.php" />
          {/^http:\/\//i.test(s.syncServer || '') && !/^http:\/\/(127\.0\.0\.1|localhost)(:|\/|$)/i.test(s.syncServer) && <div className="sub" style={{ color: '#ffcf8a', marginTop: 4 }}>⚠ 正在使用 http：同步包本身是加密的，但访问令牌会明文传输，可能被窃取。建议给域名配置 SSL 后改用 https。</div>}</Field>
        <Field label="访问令牌 (Bearer Token)"><input type="password" value={s.syncToken} onChange={e => set({ syncToken: e.target.value })} /></Field>
        <Field label="同步密码 (端到端加密)" hint="用于派生 AES-256 密钥。只存在本机，绝不上传；忘记将无法解密云端数据。"><input type="password" value={s.syncPassword} onChange={e => set({ syncPassword: e.target.value })} /></Field>
        <Toggle on={!s.autoSyncOff} onChange={v => set({ autoSyncOff: !v })} label="自动同步" hint="启动时对比云端与本地；关闭浏览器后上传环境（含登录状态）；设置、代理列表、指纹库改动后自动上传" />
        {!s.autoSyncOff && <div className="grid2">
          <Field label="云端较新时询问等待（秒）" hint="0 = 不弹窗，直接执行右侧动作"><input type="number" min={0} max={300} value={s.syncPromptSecs ?? 3} onChange={e => set({ syncPromptSecs: Math.max(0, Math.min(300, Math.round(+e.target.value || 0))) })} /></Field>
          <Field label="倒计时结束后"><select value={s.syncPromptAction || 'pull'} onChange={e => set({ syncPromptAction: e.target.value })}><option value="pull">拉取云端覆盖本地</option><option value="keep">保留本地（上传覆盖云端）</option></select></Field>
        </div>}
        {/* ---- session store ---- */}
        <Toggle on={(s as any).syncSessionStorage !== false} onChange={v => set({ syncSessionStorage: v } as any)} label="同步网站登录状态（会话存储）" hint="换设备启动时恢复网站登录。除普通 Cookie 外，还同步分区 Cookie、LocalStorage 和 IndexedDB，适合网页应用（SPA）。关闭后只同步普通 Cookie" />
        <div className={`ss-opts ${(s as any).syncSessionStorage !== false ? '' : 'dim'}`}>
          <SessionScopeChips on={(s as any).syncSessionStorage !== false} />
          {(s as any).syncSessionStorage !== false && <Field label="单个环境快照上限（MB）" hint="超出时保留全部 Cookie 和最近访问网站的 LocalStorage，较早网站的 IndexedDB 先舍弃。只采集 http/https 网站，不含扩展页面">
            <input type="number" min={4} max={256} value={(s as any).sessionStoreCapMB || 32} onChange={e => set({ sessionStoreCapMB: Math.max(4, Math.min(256, Math.round(+e.target.value || 32))) } as any)} /></Field>}
        </div>
        <Field label="本机设备名"><input type="text" value={s.deviceName} onChange={e => set({ deviceName: e.target.value })} /></Field>
        <button className="btn" onClick={test} disabled={testing}><Icon.refresh s={14} /> {testing ? '测试中…' : '测试连接'}</button>
      </div>

      <AppShellSettings s={s} set={set} index={4} />

      <div style={{ display: 'flex', gap: 10 }}>
        <button className="btn primary" onClick={save}><Icon.check s={15} /> 保存设置</button>
        <div className="spacer" style={{ flex: 1 }} />
        <span className="sub num" style={{ alignSelf: 'center', color: 'var(--txt-3)', fontSize: 11.5 }}>DEVICE {s.deviceId}</span>
      </div>
    </div>
  );
};
