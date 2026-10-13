import React, { useEffect, useState } from 'react';
import { api, Settings } from './api';
import { Icon, Field, Toggle, useToast } from './ui';

export const SettingsView: React.FC<{ settings: Settings; onSaved: (s: Settings) => void }> = ({ settings, onSaved }) => {
  const toast = useToast();
  const [s, setS] = useState<Settings>({ ...settings });
  const [testing, setTesting] = useState(false);
  const [defaultKernel, setDefaultKernel] = useState('');
  const set = (patch: Partial<Settings>) => setS(v => ({ ...v, ...patch }));

  useEffect(() => { api.DefaultKernelPath().then(setDefaultKernel).catch(() => setDefaultKernel('')); }, []);
  const save = async () => { const out = await api.SaveSettings(s); onSaved(out); api.DefaultKernelPath().then(setDefaultKernel).catch(() => setDefaultKernel('')); };
  const pickKernel = async () => { const f = await api.PickKernel(); if (f) set({ kernelPath: f }); };
  const resetKernel = async () => { const f = await api.DefaultKernelPath(); if (f) set({ kernelPath: f }); else toast('没有检测到默认浏览器，请点击“选择 EXE”手动指定', 'bad'); };
  const pickFolder = async () => { const f = await api.PickFolder(); if (f) set({ profilesDir: f }); };
  const test = async () => { setTesting(true); try { await api.TestSync(); toast('同步服务器连接正常', 'ok'); } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { setTesting(false); } };
  const useKit = async () => { try { const out=await api.UseUndetectableKit(); setS(out); onSaved(out); api.DefaultKernelPath().then(setDefaultKernel).catch(() => setDefaultKernel('')); toast('已启用定制 Chromium'); } catch(e) { toast(String(e),'bad'); } };

  return (
    <div style={{ maxWidth: 760 }}>
      <div className="section-title">浏览器内核</div>
      <div className="panel" style={{ padding: 18, marginBottom: 20 }}>
        <Field label="浏览器 EXE 路径" hint={`默认路径：${defaultKernel || '启动时自动检测；也可以手动选择'}`}>
          <div className="row"><input type="text" value={s.kernelPath} onChange={e => set({ kernelPath: e.target.value })} placeholder="选择 chrome.exe 或 msedge.exe" /><button className="btn" style={{ flex: 'none' }} onClick={pickKernel}><Icon.folder s={15} /> 选择 EXE</button><button className="btn" style={{ flex: 'none' }} onClick={resetKernel}>恢复默认</button></div>
        </Field>
        <Field label="指纹注入方式">
          <select value={s.kernelMode} onChange={e => set({ kernelMode: e.target.value })}>
            <option value="stock">标准 Chrome + 注入扩展（即装即用）</option>
            <option value="undetectable">Undetectable 定制 Chromium（本地指纹库）</option>
            <option value="custom">自研补丁内核（需实现 JSON 参数读取）</option>
          </select>
        </Field>
        <Field label="本地指纹包目录" hint="包含 runtime/chrome 和 fingerprints/library.json"><input value={s.undetectableKit || ''} onChange={e=>set({undetectableKit:e.target.value})} placeholder="项目目录\\undetectable-kit" /></Field>
        <button className="btn" onClick={useKit}>使用项目自带 Chromium 与指纹库</button>
        <div className="kv"><span className="k">说明</span><span className="v" style={{ fontFamily: 'inherit', color: 'var(--txt-dim)', textAlign: 'left' }}>“选择 EXE”支持用户指定浏览器程序，路径会随设置保存；定制 Chromium 模式需要选择兼容 Undetectable 指纹模板的定制内核，普通 Chrome / Edge 不能运行原生模板。关闭 GhostBrowser 会停止它启动的环境。当前内置原生模板已验证 Windows。</span></div>
      </div>

      <div className="section-title">数据与本地 API</div>
      <div className="panel" style={{ padding: 18, marginBottom: 20 }}>
        <Field label="环境数据目录">
          <div className="row"><input type="text" value={s.profilesDir} onChange={e => set({ profilesDir: e.target.value })} /><button className="btn" style={{ flex: 'none' }} onClick={pickFolder}><Icon.folder s={15} /> 浏览</button></div>
        </Field>
        <Toggle on={s.apiEnabled} onChange={v => set({ apiEnabled: v })} label="启用本地自动化 API" hint="供 Puppeteer / Playwright / Selenium 接管 CDP 端口" />
        <Field label="API 端口"><input type="number" value={s.apiPort} onChange={e => set({ apiPort: +e.target.value })} /></Field>
        <div className="kv"><span className="k">默认操作系统</span><span className="v"><select style={{ width: 160 }} value={s.defaultOS} onChange={e => set({ defaultOS: e.target.value })}><option value="windows">Windows</option><option value="macos">macOS</option><option value="linux">Linux</option></select></span></div>
      </div>

      <div className="section-title">云端同步 (你的 VPS + PHP)</div>
      <div className="panel" style={{ padding: 18, marginBottom: 20 }}>
        <Field label="同步服务器地址" hint="指向你 VPS 上部署的 PHP 同步端，例如 https://sync.yourdomain.com"><input type="text" value={s.syncServer} onChange={e => set({ syncServer: e.target.value })} placeholder="https://sync.yourdomain.com" /></Field>
        <Field label="访问令牌 (Bearer Token)"><input type="password" value={s.syncToken} onChange={e => set({ syncToken: e.target.value })} /></Field>
        <Field label="同步密码 (端到端加密)" hint="用于派生 AES-256 密钥。只存在本机，绝不上传；忘记将无法解密云端数据。"><input type="password" value={s.syncPassword} onChange={e => set({ syncPassword: e.target.value })} /></Field>
        <Field label="本机设备名"><input type="text" value={s.deviceName} onChange={e => set({ deviceName: e.target.value })} /></Field>
        <button className="btn" onClick={test} disabled={testing}><Icon.refresh s={14} /> {testing ? '测试中…' : '测试连接'}</button>
      </div>

      <div style={{ display: 'flex', gap: 10 }}>
        <button className="btn primary" onClick={save}><Icon.check s={15} /> 保存设置</button>
        <div className="spacer" style={{ flex: 1 }} />
        <span className="sub" style={{ alignSelf: 'center' }}>设备 ID {s.deviceId}</span>
      </div>
    </div>
  );
};
