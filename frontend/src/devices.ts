// Device categories shared by the profile editor, the bulk creator and the list.
export type DevType = 'desktop' | 'laptop' | 'phone' | 'tablet' | 'foldable' | 'chromebook' | 'tv';
export const DEVICE_TYPES: { id: DevType; name: string; oses: string[]; trust: 'high' | 'mid' | 'low'; tip: string }[] = [
  { id: 'desktop', name: '台式机', oses: ['windows', 'macos', 'linux'], trust: 'high', tip: '独立显卡、大屏、无电池' },
  { id: 'laptop', name: '笔记本', oses: ['windows', 'macos', 'linux'], trust: 'high', tip: '核显/笔记本显卡、有电池，部分带触屏' },
  { id: 'phone', name: '安卓手机', oses: ['android'], trust: 'mid', tip: '三星、Pixel、小米、OPPO 等真实机型，触控 + 移动 UA' },
  { id: 'tablet', name: '安卓平板', oses: ['android'], trust: 'mid', tip: 'Galaxy Tab、小米平板等，UA 不带 Mobile' },
  { id: 'foldable', name: '折叠屏', oses: ['android'], trust: 'mid', tip: 'Galaxy Z Fold / Flip、Pixel Fold，折叠与展开两种尺寸' },
  { id: 'chromebook', name: 'Chromebook', oses: ['chromeos'], trust: 'mid', tip: 'ChromeOS，核显，部分带触屏' },
  { id: 'tv', name: '电视', oses: ['android', 'tizen', 'webos'], trust: 'low', tip: '电视浏览器内核版本老、流量少，容易显眼，仅建议实验' },
];
export const OS_NAMES: Record<string, string> = { windows: 'Windows', macos: 'macOS', linux: 'Linux', android: 'Android', chromeos: 'ChromeOS', tizen: 'Tizen', webos: 'webOS' };
export const TRUST: Record<string, [string, string]> = { high: ['可信度高', 'ok'], mid: ['可信度中', 'warn'], low: ['可信度低', 'fail'] };
export const devType = (t?: string): DevType => (t || 'desktop') as DevType;
export const devName = (t?: string) => DEVICE_TYPES.find(d => d.id === devType(t))?.name || '台式机';
export const osName = (o: string) => OS_NAMES[o] || o;
export const osVerLabel = (os: string, v: string) => os === 'windows' ? `Windows ${v}` : os === 'macos' ? `macOS ${v}` : os === 'android' ? `Android ${v}` : os === 'chromeos' ? `ChromeOS ${v}` : os === 'tizen' ? `Tizen ${v}` : os === 'webos' ? `webOS ${v}` : `Linux ${v}`;
// Only Windows desktops/laptops can run on the custom kernel.
export const needsOpen = (os: string, t?: string) => os !== 'windows' || !['desktop', 'laptop'].includes(devType(t));
export const uaOSOf = (ua: string) => /Windows NT/.test(ua) ? 'windows' : /Macintosh|Mac OS X/.test(ua) ? 'macos' : /Android/.test(ua) ? 'android' : /CrOS/.test(ua) ? 'chromeos' : /Tizen/.test(ua) ? 'tizen' : /Web0S/.test(ua) ? 'webos' : /Linux|X11/.test(ua) ? 'linux' : '';
