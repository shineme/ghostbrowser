import React, { useEffect, useState } from 'react';
import { api } from './api';

// ---- session store ----
// What a profile's login snapshot carries (synced inside the encrypted profile).
export type SessionInfo = {
  enabled: boolean; schema: number; savedAt: number; cookies: number; partitioned: number; origins: number;
  lsKeys: number; idbDbs: number; idbRecords: number; idbPartial: boolean; dropped: number; bytes: number; capBytes: number; error?: string;
};

const size = (b: number) => b >= 1048576 ? `${(b / 1048576).toFixed(1)} MB` : `${Math.max(1, Math.round(b / 1024))} KB`;
const n = (v: number) => v.toLocaleString('zh-CN');

/** Chips listed in Settings: what the snapshot syncs. */
export const SessionScopeChips: React.FC<{ on: boolean }> = ({ on }) => (
  <div className="ss-chips">
    <span className="ss-chip on">Cookie</span>
    <span className={`ss-chip ${on ? 'on' : 'off'}`}>分区 Cookie（CHIPS）</span>
    <span className={`ss-chip ${on ? 'on' : 'off'}`}>LocalStorage</span>
    <span className={`ss-chip ${on ? 'part' : 'off'}`} title="大多数网站可完整恢复；无法序列化的值（如 CryptoKey、文件句柄）会跳过">IndexedDB · 尽力恢复</span>
  </div>
);

/** One line in the Cookie dialog: what the current snapshot holds. */
export const SessionSnapshotLine: React.FC<{ profileId: string }> = ({ profileId }) => {
  const [i, setI] = useState<SessionInfo | null>(null);
  useEffect(() => { api.SessionStoreInfo(profileId).then((v: SessionInfo) => setI(v)).catch(() => setI(null)); }, [profileId]);
  if (!i) return null;
  if (i.error) return <div className="ss-line bad"><span className="ss-k">登录快照</span><span className="ss-v">{i.error}</span></div>;
  if (!i.schema) return <div className="ss-line"><span className="ss-k">登录快照</span><span className="ss-v sub">还没有快照，浏览器运行后自动生成</span></div>;
  return (
    <div className="ss-line">
      <span className="ss-k">登录快照</span>
      <span className="ss-v">
        <span className="ss-chip on">Cookie <b className="num">{n(i.cookies)}</b>{i.partitioned > 0 && <> · 分区 <b className="num">{n(i.partitioned)}</b></>}</span>
        {i.schema >= 2 ? <>
          <span className={`ss-chip ${i.origins ? 'on' : 'off'}`}>LocalStorage <b className="num">{n(i.origins)}</b> 站 · <b className="num">{n(i.lsKeys)}</b> 项</span>
          <span className={`ss-chip ${i.idbPartial ? 'part' : i.idbDbs ? 'on' : 'off'}`} title={i.idbPartial ? '部分记录因大小上限或不支持的类型未采集' : ''}>IndexedDB <b className="num">{n(i.idbDbs)}</b> 库 · <b className="num">{n(i.idbRecords)}</b> 条{i.idbPartial ? ' · 部分' : ''}</span>
        </> : <span className="ss-chip off">仅 Cookie（旧格式）</span>}
        <span className="ss-size num" title={i.dropped ? `${i.dropped} 个网站因超出上限未保存` : ''}>{size(i.bytes)} / {size(i.capBytes)}{i.dropped > 0 ? ` · ${i.dropped} 站超限` : ''}</span>
      </span>
    </div>
  );
};
