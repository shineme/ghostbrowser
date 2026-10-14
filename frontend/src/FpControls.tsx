// Fingerprint protection + launch controls (open-source Chromium kernel).
// Used by ProfileDrawer's 「防护」 and 「启动」 tabs.
import React from 'react';
import { Fingerprint } from './api';
import { Icon, Segmented } from './ui';

type FP = Fingerprint & Record<string, any>;
type SetFP = (patch: Partial<Fingerprint> & Record<string, any>) => void;

/** Compact −/+ integer stepper. */
export const NumberStepper: React.FC<{ value: number; onChange: (v: number) => void; min?: number; max?: number; label?: string; suffix?: string }> = ({ value, onChange, min = 0, max = 99, label, suffix }) => {
  const set = (v: number) => onChange(Math.max(min, Math.min(max, Math.round(v || 0))));
  return (
    <div className="stepper">
      {label && <span className="stepper-l">{label}</span>}
      <div className="stepper-c">
        <button type="button" disabled={value <= min} onClick={() => set(value - 1)} aria-label="减少">−</button>
        <input type="text" inputMode="numeric" value={value} onChange={e => /^\d*$/.test(e.target.value) && set(+e.target.value)} />
        <button type="button" disabled={value >= max} onClick={() => set(value + 1)} aria-label="增加">+</button>
      </div>
      {suffix && <span className="stepper-s">{suffix}</span>}
    </div>
  );
};

/** One setting row: title + hint on the left, a control on the right, optional body below. */
const Row: React.FC<{ title: string; hint?: string; tag?: boolean; dim?: boolean; control: React.ReactNode; onClick?: () => void; children?: React.ReactNode }> = ({ title, hint, tag, dim, control, onClick, children }) => (
  <div className={`fprow ${dim ? 'dim' : ''} ${onClick ? 'click' : ''}`}>
    <div className="fprow-h" onClick={onClick}>
      <div className="fprow-t">
        <b>{title}{tag && <span className="ktag" title="定制内核不读取这一项，切换到开源内核后生效">仅开源内核</span>}</b>
        {hint && <small>{hint}</small>}
      </div>
      <div className="fprow-c" onClick={e => onClick && e.stopPropagation()}>{control}</div>
    </div>
    {children && <div className="fprow-b">{children}</div>}
  </div>
);
const Sw: React.FC<{ on: boolean; onChange: (v: boolean) => void }> = ({ on, onChange }) => <div className={`sw ${on ? 'on' : ''}`} onClick={() => onChange(!on)} />;

// Tri-state noise mode; "" follows the old boolean (same rule as fpMode in chromium_fp.go).
const mode = (m: string | undefined, old: boolean) => (m === 'off' || m === 'noise' || m === 'real') ? m : old ? 'noise' : 'real';
const NOISE: [string, string][] = [['noise', '噪声'], ['real', '真实'], ['off', '关闭']];

const KernelNotice: React.FC<{ open: boolean; what: string }> = ({ open, what }) => open
  ? <div className="notice fpk-ok"><Icon.cpu /><div className="grow">当前为开源内核（Chromium），{what}全部生效。</div></div>
  : <div className="notice warn"><Icon.alert /><div className="grow">当前环境使用定制内核：标有「仅开源内核」的项目不会生效，其指纹由内核模板决定。可在「指纹配置」里切换到开源内核。</div></div>;

/** Short seed label (first 8 hex), same as seedShortID in chromium_fp_extras.go. */
export const seedShort = (seed: number) => Math.max(0, Math.floor(Number(seed) || 0)).toString(16).padStart(8, '0').slice(0, 8);

export const GuardPane: React.FC<{ fp: FP; setFP: SetFP; open: boolean; onRegenSeed?: () => Promise<void> }> = ({ fp, setFP, open, onRegenSeed }) => {
  const [regen, setRegen] = React.useState(false);
  const [flip, setFlip] = React.useState(0);
  const doRegen = async () => { if (!onRegenSeed || regen) return; setRegen(true); try { await onRegenSeed(); setFlip(x => x + 1); } finally { setRegen(false); } };
  const t = !open; // mark open-source-only rows when on the custom kernel
  const noiseRow = (title: string, hint: string, key: string, old: string) => {
    const v = mode(fp[key], !!fp[old]);
    return <Row key={key} title={title} hint={hint} tag={t} dim={t} control={<Segmented className="compact" value={v} onChange={m => setFP({ [key]: m, [old]: m === 'noise' })} items={NOISE} />} />;
  };
  const media = (fp.mediaDevicesMode as string) || (fp.hideMediaDevices ? 'off' : 'real');
  const setMedia = (m: string) => setFP({
    mediaDevicesMode: m, hideMediaDevices: m === 'off',
    ...(m === 'custom' && !fp.mediaMics && !fp.mediaSpeakers && !fp.mediaCameras ? { mediaMics: 1, mediaSpeakers: 1, mediaCameras: 1 } : {}),
  });
  return <>
    <KernelNotice open={open} what="以下防护" />

    <div className="section-title">噪声指纹</div>
    <div className="fprows">
      {noiseRow('Canvas', '噪声：按环境种子给画布像素加稳定扰动；真实：不处理；关闭：读取返回空白', 'canvasMode', 'canvasNoise')}
      {noiseRow('WebGL 图像', 'WebGL 绘制结果（readPixels / toDataURL）的指纹', 'webglImageMode', 'webglNoise')}
      {noiseRow('AudioContext', '音频处理结果的指纹，关闭会让音频 API 不可用', 'audioMode', 'audioNoise')}
      {noiseRow('ClientRects', '元素尺寸与文字宽度测量（getBoundingClientRect）', 'clientRectsMode', 'clientRectNoise')}
    </div>
    <div className="seedcard">
      <div className="seed-glyph"><Icon.fingerprint s={18} /></div>
      <div className="seed-t">
        <div className="seed-l">指纹种子 <b key={flip} className="mono seed-id">{seedShort(fp.noiseSeed)}</b></div>
        <small>Canvas、WebGL、音频、ClientRects 噪声与媒体设备 ID 都由它派生；同一种子 = 指纹永远一致</small>
      </div>
      {onRegenSeed && <button type="button" className="btn ghost seed-btn" disabled={regen} onClick={doRegen}><Icon.dice s={14} /> {regen ? '生成中…' : '重新生成指纹种子'}</button>}
    </div>
    <div className="grid2" style={{ marginTop: 10 }}>
      <div className="field" style={{ marginBottom: 0 }}><label>噪声种子 (seed)</label><input type="number" value={fp.noiseSeed} onChange={e => setFP({ noiseSeed: +e.target.value })} /><span className="hint">高级：手动指定种子数值，可在多个环境间复用</span></div>
    </div>

    <div className="section-title">媒体与 API</div>
    <div className="fprows">
      <Row title="WebRTC" hint={fp.webrtc === 'real' ? '会暴露本机真实 IP，不推荐' : fp.webrtc === 'disabled' ? '禁止非代理的 UDP，部分音视频网站不可用' : '对外显示代理出口 IP（推荐）'}
        control={<Segmented className="compact" value={fp.webrtc || 'proxy'} onChange={v => setFP({ webrtc: v })} items={[['proxy', '代理 IP'], ['real', '真实'], ['disabled', '禁用']]} />} />
      <Row title="媒体设备" hint={media === 'custom' ? '网站枚举到的麦克风、扬声器、摄像头数量' : media === 'off' ? '不暴露任何麦克风、扬声器、摄像头' : '使用本机真实设备列表'} tag={t} dim={t}
        control={<Segmented className="compact" value={media} onChange={setMedia} items={[['real', '真实'], ['custom', '自定义'], ['off', '关闭']]} />}>
        {media === 'custom' && <div className="steppers">
          <NumberStepper label="麦克风" value={fp.mediaMics || 0} max={8} onChange={v => setFP({ mediaMics: v })} />
          <NumberStepper label="扬声器" value={fp.mediaSpeakers || 0} max={8} onChange={v => setFP({ mediaSpeakers: v })} />
          <NumberStepper label="摄像头" value={fp.mediaCameras || 0} max={8} onChange={v => setFP({ mediaCameras: v })} />
        </div>}
        {media === 'real' && open && <div className="subsw" onClick={() => setFP({ mediaIdAlias: !fp.mediaIdAlias })}>
          <span onClick={e => e.stopPropagation()}><Sw on={!!fp.mediaIdAlias} onChange={v => setFP({ mediaIdAlias: v })} /></span>
          <div className="subsw-t"><b>设备 ID 别名</b><small>{fp.mediaIdAlias ? '网站看到的 deviceId / groupId 是按种子和网站生成的固定别名，摄像头、麦克风仍可正常使用' : '网站看到浏览器给出的真实设备 ID'}</small></div>
        </div>}
      </Row>
      <Row title="语音列表" hint={fp.speechVoices === 'custom' ? '按系统和语言生成一组常见语音（speechSynthesis）' : fp.speechVoices === 'off' ? 'speechSynthesis.getVoices() 返回空列表' : '使用本机真实语音列表'} tag={t} dim={t}
        control={<Segmented className="compact" value={fp.speechVoices || 'real'} onChange={v => setFP({ speechVoices: v })} items={[['real', '真实'], ['custom', '自定义'], ['off', '关闭']]} />} />
      <Row title="WebGPU" hint={fp.webgpu === 'off' ? 'navigator.gpu 不可用，避免暴露真实显卡' : '保留 WebGPU（显卡信息可能与 WebGL 不一致）'} tag={t} dim={t}
        control={<Segmented className="compact" value={fp.webgpu || 'real'} onChange={v => setFP({ webgpu: v })} items={[['real', '真实'], ['off', '关闭']]} />} />
      <Row title="Do Not Track" hint="navigator.doNotTrack = 1，并发送 DNT 请求头" tag={t} dim={t} onClick={() => setFP({ doNotTrack: !fp.doNotTrack })}
        control={<Sw on={!!fp.doNotTrack} onChange={v => setFP({ doNotTrack: v })} />} />
    </div>

    <div className="section-title">安全</div>
    <div className="fprows">
      <Row title="端口扫描保护" hint="阻止网页探测本机端口（127.0.0.1 / 局域网），白名单内的端口放行" tag={t} dim={t} onClick={() => setFP({ portScanProtection: !fp.portScanProtection })}
        control={<Sw on={!!fp.portScanProtection} onChange={v => setFP({ portScanProtection: v })} />}>
        {fp.portScanProtection && <div className="field" style={{ marginBottom: 0 }}>
          <label>白名单端口</label>
          <input type="text" className="mono" value={fp.scanWhitelist || ''} onChange={e => setFP({ scanWhitelist: e.target.value })} placeholder="如 3000, 8080, localhost:5173" />
          <span className="hint">逗号分隔，可写端口或 主机:端口；留空表示全部拦截</span>
        </div>}
      </Row>
      <Row title="每次启动随机指纹" hint="每次启动都换一个噪声种子，Canvas / WebGL / 音频指纹每次不同（不利于长期养号）" tag={t} dim={t} onClick={() => setFP({ randomFingerprintOnLaunch: !fp.randomFingerprintOnLaunch })}
        control={<Sw on={!!fp.randomFingerprintOnLaunch} onChange={v => setFP({ randomFingerprintOnLaunch: v })} />} />
    </div>
  </>;
};

const BLOCKED = ['--user-data-dir', '--remote-debugging-*', '--proxy-server', '--user-agent'];

export const LaunchPane: React.FC<{ fp: FP; setFP: SetFP; open: boolean }> = ({ fp, setFP, open }) => {
  const t = !open;
  const win = (fp.windowMode as string) || 'auto';
  const hw = fp.hardwareAcceleration !== false;
  const setWin = (m: string) => setFP({
    windowMode: m,
    ...(m === 'custom' && !fp.windowWidth ? { windowWidth: Math.round((fp.screenWidth || 1920) * 0.8), windowHeight: Math.round((fp.screenHeight || 1080) * 0.8), windowX: fp.windowX || 0, windowY: fp.windowY || 0 } : {}),
  });
  const num = (k: string, label: string, ph: string) => (
    <div className="field" style={{ marginBottom: 0 }}><label>{label}</label>
      <input type="number" min={0} value={fp[k] ?? ''} placeholder={ph} onChange={e => setFP({ [k]: e.target.value === '' ? 0 : Math.max(0, Math.round(+e.target.value)) })} /></div>
  );
  return <>
    <KernelNotice open={open} what="以下启动选项" />

    <div className="section-title">窗口</div>
    <div className="fprows">
      <Row title="窗口大小" hint={win === 'custom' ? '按下方宽高和位置打开窗口（像素）' : win === 'maximized' ? '启动后最大化' : `按指纹分辨率自动计算（${fp.screenWidth}×${fp.screenHeight} 的 80%）`} tag={t} dim={t}
        control={<Segmented className="compact" value={win} onChange={setWin} items={[['auto', '自动'], ['custom', '自定义'], ['maximized', '最大化']]} />}>
        {win === 'custom' && <div className="grid4">
          {num('windowWidth', '宽 W', '1280')}{num('windowHeight', '高 H', '800')}{num('windowX', '左 X', '0')}{num('windowY', '上 Y', '0')}
        </div>}
      </Row>
    </div>

    <div className="section-title">性能与流量</div>
    <div className="fprows">
      <Row title="硬件加速" hint={hw ? '使用显卡渲染（推荐）' : '已关闭：加 --disable-gpu，低配机器或虚拟机更稳定'} tag={t} dim={t} onClick={() => setFP({ hardwareAcceleration: !hw })}
        control={<Sw on={hw} onChange={v => setFP({ hardwareAcceleration: v })} />} />
      <Row title="禁止加载图片" hint="节省流量，页面里的图片不再下载" tag={t} dim={t} onClick={() => setFP({ blockImages: !fp.blockImages })} control={<Sw on={!!fp.blockImages} onChange={v => setFP({ blockImages: v })} />} />
      <Row title="禁止加载视频" hint="拦截视频与流媒体请求" tag={t} dim={t} onClick={() => setFP({ blockVideo: !fp.blockVideo })} control={<Sw on={!!fp.blockVideo} onChange={v => setFP({ blockVideo: v })} />} />
      <Row title="静音" hint="浏览器所有标签页不出声" tag={t} dim={t} onClick={() => setFP({ blockSound: !fp.blockSound })} control={<Sw on={!!fp.blockSound} onChange={v => setFP({ blockSound: v })} />} />
    </div>

    <div className="section-title">自定义启动参数</div>
    <div className={`field ${t ? 'dimf' : ''}`}>
      <textarea className="mono" value={fp.startupParameters || ''} onChange={e => setFP({ startupParameters: e.target.value })} placeholder={'--lang=en-US\n--disable-features=Translate'} spellCheck={false} />
      <span className="hint">每行或空格分隔一个 Chromium 参数，同名参数会覆盖默认值。{t && <span className="ktag">仅开源内核</span>}</span>
      <div className="flagnote"><Icon.lock s={12} /><span>以下参数由程序接管，会被忽略：</span>{BLOCKED.map(b => <code key={b}>{b}</code>)}</div>
    </div>
  </>;
};
