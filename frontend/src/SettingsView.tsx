import React, { useEffect, useState } from 'react';
import { api, Settings } from './api';
import { Icon, Field, Toggle, useToast } from './ui';

export const SettingsView: React.FC<{ settings: Settings; onSaved: (s: Settings) => void }> = ({ settings, onSaved }) => {
  const toast = useToast();
  const [s, setS] = useState<Settings>({ ...settings });
  const [testing, setTesting] = useState(false);
  const set = (patch: Partial<Settings>) => setS(v => ({ ...v, ...patch }));

  const [defaultKernel, setDefaultKernel] = useState('');
  const refreshDefault = () => api.DefaultKernelPath().then(setDefaultKernel).catch(() => setDefaultKernel(''));
  useEffect(() => { refreshDefault(); }, []);
  const save = async () => { const out = await api.SaveSettings(s); onSaved(out); refreshDefault(); };
  const resetKernel = async () => { const f = await api.DefaultKernelPath(); if (f) set({ kernelPath: f }); else toast('没有检测到默认浏览器，请点击“浏览”手动指定', 'bad'); };
  const useKit = async () => { try { const out = await api.UseUndetectableKit(); setS(out); onSaved(out); refreshDefault(); toast('已启用定制 Chromium'); } catch (e) { toast(String(e), 'bad'); } };
  const pickKernel = async () => { const f = await api.PickKernel(); if (f) set({ kernelPath: f }); };
  const pickFolder = async () => { const f = await api.PickFolder(); if (f) set({ profilesDir: f }); };
  const test = async () => { setTesting(true); try { await api.TestSync(); toast('同步服务器连接正常', 'ok'); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { setTesting(false); } };

  return (
    <div style={{ maxWidth: 780 }}>
      <div className="panel pad setting-card spot rise" style={{ ['--i' as any]: 0 }}>
        <div className="head"><span className="ic"><Icon.cpu /></span><div><h4>浏览器内核</h4><span>选择指纹在哪一层生效</span></div></div>
        <div className="choice three">
          <button className={s.kernelMode === 'stock' || !s.kernelMode ? 'on' : ''} onClick={() => set({ kernelMode: 'stock' })}><b><Icon.bolt s={15} />标准 Chrome<span className="radio" /></b><small>MV3 扩展在页面主环境覆盖指纹，即装即用</small></button>
          <button className={s.kernelMode === 'undetectable' ? 'on' : ''} onClick={() => set({ kernelMode: 'undetectable' })}><b><Icon.layers s={15} />定制 Chromium<span className="radio" /></b><small>本地指纹库 + 原生模板，需兼容内核</small></button>
          <button className={s.kernelMode === 'custom' ? 'on' : ''} onClick={() => set({ kernelMode: 'custom' })}><b><Icon.shield s={15} />自研补丁内核<span className="radio" /></b><small>fingerprint.json 直接交给 C++ 层</small></button>
        </div>
        <Field label="浏览器 EXE 路径" hint={`默认路径：${defaultKernel || '启动时自动检测，也可以手动选择'}`}>
          <div className="row"><input type="text" value={s.kernelPath} onChange={e => set({ kernelPath: e.target.value })} placeholder="选择 chrome.exe 或 msedge.exe" /><button className="btn" style={{ flex: 'none' }} onClick={pickKernel}><Icon.folder s={15} /> 浏览</button><button className="btn ghost" style={{ flex: 'none' }} onClick={resetKernel}><Icon.refresh s={14} /> 恢复默认</button></div>
        </Field>
        {s.kernelMode === 'undetectable' && <div className="rise">
          <Field label="本地指纹包目录" hint="包含 runtime/chrome 和 fingerprints/library.json"><input type="text" value={s.undetectableKit || ''} onChange={e => set({ undetectableKit: e.target.value })} placeholder="项目目录\\undetectable-kit" /></Field>
          <button className="btn" onClick={useKit}><Icon.layers s={14} /> 使用项目自带 Chromium 与指纹库</button>
          <div className="sub" style={{ marginTop: 10 }}>普通 Chrome / Edge 不能运行原生模板。关闭 GhostBrowser 会停止它启动的环境，当前内置模板已验证 Windows。</div>
        </div>}
      </div>

      <div className="panel pad setting-card spot rise" style={{ ['--i' as any]: 1 }}>
        <div className="head"><span className="ic"><Icon.terminal /></span><div><h4>数据与本地 API</h4><span>供 Puppeteer / Playwright / Selenium 接管</span></div></div>
        <Field label="环境数据目录">
          <div className="row"><input type="text" value={s.profilesDir} onChange={e => set({ profilesDir: e.target.value })} /><button className="btn" style={{ flex: 'none' }} onClick={pickFolder}><Icon.folder s={15} /> 浏览</button></div>
        </Field>
        <Toggle on={s.apiEnabled} onChange={v => set({ apiEnabled: v })} label="启用本地自动化 API" hint="供 Puppeteer / Playwright / Selenium 接管 CDP 端口" />
        <Field label="API 端口"><input type="number" value={s.apiPort} onChange={e => set({ apiPort: +e.target.value })} /></Field>
        <div className="kv" style={{ alignItems: 'center', borderBottom: 0 }}><span className="k">默认操作系统</span><span className="v"><select style={{ width: 160 }} value={s.defaultOS} onChange={e => set({ defaultOS: e.target.value })}><option value="windows">Windows</option><option value="macos">macOS</option><option value="linux">Linux</option></select></span></div>
      </div>

      <div className="panel pad setting-card spot rise" style={{ ['--i' as any]: 2 }}>
        <div className="head"><span className="ic"><Icon.cloud /></span><div><h4>云端同步</h4><span>你的 VPS + PHP，端到端加密</span></div></div>
        <Field label="同步服务器地址" hint="填 ghostsync.php 的完整网址（单文件 PHP，无需伪静态）"><input type="text" value={s.syncServer} onChange={e => set({ syncServer: e.target.value })} placeholder="https://你的域名/ghostsync.php" /></Field>
        <Field label="访问令牌 (Bearer Token)"><input type="password" value={s.syncToken} onChange={e => set({ syncToken: e.target.value })} /></Field>
        <Field label="同步密码 (端到端加密)" hint="用于派生 AES-256 密钥。只存在本机，绝不上传；忘记将无法解密云端数据。"><input type="password" value={s.syncPassword} onChange={e => set({ syncPassword: e.target.value })} /></Field>
        <Toggle on={!s.autoSyncOff} onChange={v => set({ autoSyncOff: !v })} label="自动同步" hint="启动时对比云端与本地；关闭浏览器后上传环境（含登录状态）；设置、代理列表、指纹库改动后自动上传" />
        {!s.autoSyncOff && <div className="grid2">
          <Field label="云端较新时询问等待（秒）" hint="0 = 不弹窗，直接执行右侧动作"><input type="number" min={0} max={300} value={s.syncPromptSecs ?? 3} onChange={e => set({ syncPromptSecs: Math.max(0, Math.min(300, Math.round(+e.target.value || 0))) })} /></Field>
          <Field label="倒计时结束后"><select value={s.syncPromptAction || 'pull'} onChange={e => set({ syncPromptAction: e.target.value })}><option value="pull">拉取云端覆盖本地</option><option value="keep">保留本地（上传覆盖云端）</option></select></Field>
        </div>}
        <Field label="本机设备名"><input type="text" value={s.deviceName} onChange={e => set({ deviceName: e.target.value })} /></Field>
        <button className="btn" onClick={test} disabled={testing}><Icon.refresh s={14} /> {testing ? '测试中…' : '测试连接'}</button>
      </div>

      <div style={{ display: 'flex', gap: 10 }}>
        <button className="btn primary" onClick={save}><Icon.check s={15} /> 保存设置</button>
        <div className="spacer" style={{ flex: 1 }} />
        <span className="sub num" style={{ alignSelf: 'center', color: 'var(--txt-3)', fontSize: 11.5 }}>DEVICE {s.deviceId}</span>
      </div>
    </div>
  );
};
