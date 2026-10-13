import React, { useEffect, useState } from 'react';
import { api, onEvent } from './api';
import { Icon } from './ui';

export type EnvStatus = {
  ready: boolean; kitDir: string; chromeExe: string;
  phase: 'idle' | 'checking' | 'downloading' | 'verifying' | 'extracting' | 'ready' | 'failed';
  kind?: string; pct: number; done: number; total: number; speedBps: number;
  message?: string; error?: string; updatedAt: number; manifestFromServer: boolean;
};

const mb = (b: number) => b ? `${(b / 1048576).toFixed(b > 1e8 ? 0 : 1)} MB` : '';
const label = (k?: string) => k === 'chrome' ? '定制 Chromium' : k === 'fpkit' ? '指纹包' : '';

// First-run bootstrap: the EXE is shipped alone, so the custom Chromium and the
// fingerprint kit are fetched here. Progress comes from the Go side over
// "env:setup"; a finished or failed pass can be dismissed.
export const EnvSetup: React.FC<{ onReady: () => void }> = ({ onReady }) => {
  const [s, setS] = useState<EnvStatus | null>(null);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    api.EnvStatus().then((x: any) => x && x.phase !== 'idle' && setS(x)).catch(() => {});
    return onEvent('env:setup', (x: EnvStatus) => { setS(x); if (x.ready) onReady(); });
  }, []);

  if (!s || s.ready || dismissed) return null;
  const busy = s.phase === 'checking' || s.phase === 'downloading' || s.phase === 'verifying' || s.phase === 'extracting';
  const failed = s.phase === 'failed';
  const step = s.kind === 'chrome' ? 1 : 2;

  return (
    <div className="overlay center" style={{ zIndex: 90 }}>
      <div className="modal" style={{ width: 'min(520px, 94vw)' }}>
        <div className="dhead">
          <div className="glyph">{failed ? <Icon.alert s={18} /> : <Icon.download s={18} />}</div>
          <div>
            <h3>{failed ? '环境安装失败' : '正在准备浏览器内核'}</h3>
            <div className="sub">{failed ? '定制 Chromium 与指纹包没有装好，主界面仍可正常使用。' : `首次运行需要下载${label(s.kind) || '运行文件'}，完成后即可启动环境。`}</div>
          </div>
        </div>
        <div className="dbody">
          {!failed && <div className="env-steps">
            {['chrome', 'fpkit'].map((k, i) => <div key={k} className={`env-step ${s.kind === k ? 'now' : ''}`}>
              <span className="n">{i + 1}</span><span>{k === 'chrome' ? '定制 Chromium（约 213 MB）' : '指纹包'}</span>
              {s.kind === k && !failed && <em className="sub">{s.phase === 'extracting' ? '解压中' : s.phase === 'verifying' ? '校验中' : '下载中'}</em>}
            </div>)}
          </div>}

          {!failed && s.phase === 'downloading' && <>
            <div className="env-bar"><i style={{ width: `${Math.max(s.pct, 1)}%` }} /></div>
            <div className="env-meta num">
              <span>{mb(s.done)}{s.total ? ` / ${mb(s.total)}` : ''}</span>
              <span>{s.pct ? `${s.pct.toFixed(0)}%` : ''}</span>
              <span>{s.speedBps ? `${(s.speedBps / 1048576).toFixed(1)} MB/s` : ''}</span>
            </div>
          </>}
          {!failed && s.phase !== 'downloading' && <div className="env-meta"><span className="sub">{s.message || '检测中…'}</span></div>}

          {failed && <div className="panel pad" style={{ background: 'rgba(255,107,122,.06)', borderColor: 'rgba(255,107,122,.22)' }}>
            <div className="sub" style={{ color: '#ffb3bc', wordBreak: 'break-all' }}>{s.error}</div>
            <div className="sub" style={{ marginTop: 8 }}>目标目录：<code className="mono">{s.kitDir}</code></div>
            <div className="sub" style={{ marginTop: 4 }}>校验来源：{s.manifestFromServer ? '服务器 manifest.json' : 'EXE 内置校验值'}</div>
          </div>}

          <div className="sub" style={{ marginTop: 10 }}>下载目录：<code className="mono">{s.chromeExe || '尚未安装'}</code></div>
        </div>
        <div className="dfoot">
          {!failed && busy && s.phase === 'downloading' && <button className="btn ghost" onClick={() => api.EnvCancel()}>取消下载</button>}
          <div className="spacer" style={{ flex: 1 }} />
          {failed && <button className="btn primary" onClick={() => api.EnvRecheck(false).then((x: any) => setS(x))}><Icon.refresh s={14} /> 重试安装</button>}
          {failed && <button className="btn" onClick={() => setDismissed(true)}>稍后再说</button>}
          {!failed && <span className="sub num" style={{ alignSelf: 'center' }}>{step} / 2</span>}
        </div>
      </div>
    </div>
  );
};
