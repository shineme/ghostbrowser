import React, { useEffect, useState } from 'react';
import { api, Settings, Profile } from './api';
import { Field, Segmented, SelectField } from './ui';
import { fmtBytes } from './Transfers';

// 同步内容: global choice (Settings → 云端同步) + per-profile override (drawer).

export type SyncMode = 'lite' | 'standard' | 'full';
export const SYNC_MODES: [SyncMode, string][] = [['lite', '精简'], ['standard', '标准'], ['full', '完整']];
export const normMode = (m?: string): SyncMode => (m === 'standard' || m === 'full' ? m : 'lite');
const modeName = (m?: string) => SYNC_MODES.find(x => x[0] === normMode(m))![1];

const CHIPS: Record<SyncMode, string[]> = {
  lite: ['Cookie / 登录态', 'LocalStorage', 'IndexedDB', '扩展及其数据', '书签', '偏好设置', '已存密码 / 自动填充', '指纹与会话快照'],
  standard: ['+ 浏览历史 / 网站图标 / 常用网站', '+ 上次打开的标签页', '+ Service Worker 与离线缓存（CacheStorage）', '+ 网页推送订阅'],
  full: ['+ 浏览器内部状态（同步数据、隐私沙盒、DIPS、预测数据等）'],
};
const HINT: Record<SyncMode, string> = {
  lite: '只同步登录状态和环境配置，关闭浏览器后通常只上传几百 KB 到几 MB。换设备后历史记录为空，网站离线缓存会重新下载',
  standard: '在精简基础上加上历史记录、标签页和网站的 Service Worker 缓存。视频、社交类网站的离线缓存可能有几十到上百 MB',
  full: '除缓存和可重新下载的组件外全部同步，体积最大',
};

/** Settings → 云端同步: the 同步内容 choice. */
export const SyncContentSetting: React.FC<{ s: Settings; set: (p: Partial<Settings>) => void }> = ({ s, set }) => {
  const m = normMode((s as any).syncContent);
  const shown: SyncMode[] = m === 'lite' ? ['lite'] : m === 'standard' ? ['lite', 'standard'] : ['lite', 'standard', 'full'];
  return (
    <Field label="同步内容">
      <Segmented className="compact" value={m} onChange={v => set({ syncContent: v === 'lite' ? '' : v } as any)} items={SYNC_MODES} />
      <div className="sub sc-note">{HINT[m]}</div>
      <div className="ss-chips sc-chips">
        {shown.flatMap(k => CHIPS[k].map(c => <span key={c} className="ss-chip on">{c}</span>))}
      </div>
      <div className="sub sc-note">所有模式都不同步：网页缓存、代码缓存、GPU 着色器缓存，以及浏览器可自行重新下载的组件（屏幕 AI、Widevine、安全浏览列表、拼写 / 断字数据等）。拉取云端时本机已有的这些数据会保留。每个环境可在编辑窗口单独设置。</div>
    </Field>
  );
};

type Est = { lite: number; standard: number; full: number; never: number; mode: string; exists: boolean };

/** Profile drawer → 基本信息: per-profile override with estimated sizes. */
export const SyncContentProfileField: React.FC<{ p: Profile; setP: (f: (p: Profile) => Profile) => void; existing: boolean }> = ({ p, setP, existing }) => {
  const [g, setG] = useState<SyncMode>('lite');
  const [est, setEst] = useState<Est | null>(null);
  useEffect(() => { api.GetSettings().then((x: any) => setG(normMode(x?.syncContent))).catch(() => {}); }, []);
  useEffect(() => { if (existing && p.id) (api as any).SyncSizeEstimate(p.id).then(setEst).catch(() => setEst(null)); }, [existing, p.id]);
  const own = (p as any).syncContent || '';
  const eff: SyncMode = own ? normMode(own) : g;
  const size = (m: SyncMode) => est?.exists ? ` · 约 ${fmtBytes(est[m])}` : '';
  return (
    <Field label="同步内容" hint={`${own ? '' : '跟随全局：'}${modeName(eff)}。${HINT[eff]}${est?.exists && est.never > 0 ? `。缓存和可重新下载的组件 ${fmtBytes(est.never)} 不会上传` : ''}`}>
      <SelectField value={own} onChange={v => setP(x => ({ ...x, syncContent: v } as any))}
        options={[['', `跟随全局（${modeName(g)}${size(g)}）`], ...SYNC_MODES.map(([k, n]) => [k, `${n}${size(k)}`] as [string, string])]} />
    </Field>
  );
};
