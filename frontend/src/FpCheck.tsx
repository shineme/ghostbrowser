import React, { useEffect, useState } from 'react';
import { api, Settings, Profile } from './api';
import { Icon, Field, Toggle, Segmented, SelectField, ago } from './ui';
import { Row } from './FpControls';

// 启动时打开指纹检测页: global card (Settings) + per-profile override (drawer 启动 tab).

export type FpSite = { key: string; name: string; url: string; note: string };

let sitesCache: FpSite[] | null = null;
export const useFpSites = (): FpSite[] => {
  const [l, setL] = useState<FpSite[]>(sitesCache || []);
  useEffect(() => { if (!sitesCache) (api as any).FpCheckSites().then((x: FpSite[]) => { sitesCache = x || []; setL(sitesCache); }).catch(() => {}); }, []);
  return l;
};

const siteName = (sites: FpSite[], key: string | undefined, url?: string) => {
  if (key === 'custom') return url ? `自定义：${url}` : '自定义网址（未填写）';
  return sites.find(s => s.key === (key || 'browserscan'))?.name || 'BrowserScan';
};
const siteNote = (sites: FpSite[], key: string | undefined) =>
  key === 'custom' ? '启动后打开你填写的网址（http / https）' : sites.find(s => s.key === (key || 'browserscan'))?.note || '';
const badURL = (u?: string) => { const t = (u || '').trim(); if (!t) return true; try { const x = new URL(t.includes('://') ? t : 'https://' + t); return !/^https?:$/.test(x.protocol) || !x.host; } catch { return true; } };

/** Settings → 指纹检测页 card. */
export const FpCheckSettingsCard: React.FC<{ s: Settings; set: (p: Partial<Settings>) => void; index: number }> = ({ s, set, index }) => {
  const sites = useFpSites();
  const x = s as any;
  const key = x.fpCheckSite || 'browserscan';
  return (
    <div className="panel pad setting-card spot rise" style={{ ['--i' as any]: index }} id="fpcheck-card">
      <div className="head"><span className="ic"><Icon.fingerprint /></span><div><h4>指纹检测页</h4><span>启动环境后自动打开一个检测网站，检查指纹和 IP 是否一致</span></div></div>
      <Toggle on={!!x.fpCheckOn} onChange={v => set({ fpCheckOn: v } as any)} label="启动时打开指纹检测页" hint="在环境自己的启动页（或恢复的会话）之后多开一个标签页并切到它，不会替换原有页面。本地 API、MCP、RPA 和录制启动的环境不会打开" />
      {x.fpCheckOn && <div className="fpc-opts">
        <Field label="检测网站" hint={siteNote(sites, key)}>
          <SelectField value={key} onChange={v => set({ fpCheckSite: v } as any)} options={[...sites.map(t => [t.key, t.url ? `${t.name} · ${t.url.replace(/^https:\/\//, '').replace(/\/$/, '')}` : t.name] as [string, string]), ['custom', '自定义网址']]} />
        </Field>
        {key === 'custom' && <Field label="自定义网址" hint={badURL(x.fpCheckUrl) ? '请填写 http / https 网址，未填写时不会打开' : undefined}>
          <input type="text" value={x.fpCheckUrl || ''} onChange={e => set({ fpCheckUrl: e.target.value } as any)} placeholder="https://example.com/fingerprint" />
        </Field>}
        <Field label="打开时机" hint={x.fpCheckOnce ? '每个环境只打开一次：新环境第一次启动时，或开启本设置后的第一次启动。可在环境的「启动」里重置' : '每次从界面启动环境都会打开'}>
          <Segmented className="compact" value={x.fpCheckOnce ? 'once' : 'every'} onChange={v => set({ fpCheckOnce: v === 'once' } as any)} items={[['every', '每次启动'], ['once', '仅首次启动']]} />
        </Field>
      </div>}
      <div className="sub">设置随「云端同步」在设备间同步。每个环境可以在编辑窗口的「启动」里单独开关或换成别的检测网站。</div>
    </div>
  );
};

/** Profile drawer → 启动 tab: per-profile override. */
export const FpCheckProfilePane: React.FC<{ p: Profile; setP: (f: (p: Profile) => Profile) => void }> = ({ p, setP }) => {
  const sites = useFpSites();
  const [g, setG] = useState<any>(null);
  useEffect(() => { api.GetSettings().then(setG).catch(() => {}); }, []);
  const x = p as any;
  const upd = (patch: any) => setP(v => ({ ...v, ...patch }));
  const mode: 'follow' | 'on' | 'off' = x.fpCheck === 'on' ? 'on' : x.fpCheck === 'off' ? 'off' : 'follow';
  const gOn = !!g?.fpCheckOn, gOnce = !!g?.fpCheckOnce;
  const on = mode === 'on' || (mode === 'follow' && gOn);
  const once = x.fpCheckFreq === 'once' || (x.fpCheckFreq !== 'every' && gOnce);
  const key = x.fpCheckSite || g?.fpCheckSite || 'browserscan';
  const url = x.fpCheckSite ? x.fpCheckUrl : g?.fpCheckUrl;
  const gName = siteName(sites, g?.fpCheckSite, g?.fpCheckUrl);
  const effect = !on ? (mode === 'off' ? '这个环境不打开检测页' : '全局未开启，这个环境不打开检测页')
    : key === 'custom' && badURL(url) ? '自定义网址未填写或无效，不会打开'
    : once && x.fpCheckAt ? `仅首次启动：已打开过（${ago(x.fpCheckAt)}），之后的启动不再打开`
    : `从界面启动后打开「${siteName(sites, key, url)}」${once ? '（仅这一次）' : '（每次启动）'}`;
  return <>
    <div className="section-title">指纹检测页</div>
    <div className="fprows">
      <Row title="启动时打开指纹检测页" hint={effect}
        control={<Segmented className="compact" value={mode} onChange={v => upd({ fpCheck: v === 'follow' ? '' : v })} items={[['follow', `跟随全局 · ${gOn ? '开' : '关'}`], ['on', '开'], ['off', '关']]} />}>
        {mode !== 'off' && <div className="grid2 fpc-grid">
          <Field label="检测网站">
            <SelectField value={x.fpCheckSite || ''} onChange={v => upd({ fpCheckSite: v })} options={[['', `跟随全局（${gName}）`], ...sites.map(t => [t.key, t.name] as [string, string]), ['custom', '自定义网址']]} />
          </Field>
          <Field label="打开时机">
            <SelectField value={x.fpCheckFreq || ''} onChange={v => upd({ fpCheckFreq: v })} options={[['', `跟随全局（${gOnce ? '仅首次启动' : '每次启动'}）`], ['every', '每次启动'], ['once', '仅首次启动']]} />
          </Field>
        </div>}
        {mode !== 'off' && x.fpCheckSite === 'custom' && <Field label="自定义网址">
          <input type="text" value={x.fpCheckUrl || ''} onChange={e => upd({ fpCheckUrl: e.target.value })} placeholder="https://example.com/fingerprint" />
        </Field>}
        {mode !== 'off' && once && !!x.fpCheckAt && <div className="fpc-reset"><span className="sub">已打开过检测页</span><button type="button" className="btn sm ghost" onClick={() => upd({ fpCheckAt: 0 })}><Icon.refresh s={13} /> 重置，下次启动再打开</button></div>}
      </Row>
    </div>
    <div className="sub fpc-note">检测页在启动页之后多开一个标签页并切到它；本地 API、MCP、RPA 启动时不打开。</div>
  </>;
};
