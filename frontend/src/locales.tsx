import React, { useMemo, useRef, useState } from 'react';
import { SelectField, Floating } from './ui';

// Country presets: code, name, primary timezone, Accept-Language list, capital coordinates.
export type Country = { cc: string; name: string; tz: string; langs: string[]; lat: number; lon: number };
export const COUNTRIES: Country[] = [
  { cc: 'US', name: '美国', tz: 'America/New_York', langs: ['en-US', 'en'], lat: 40.7128, lon: -74.006 },
  { cc: 'CA', name: '加拿大', tz: 'America/Toronto', langs: ['en-CA', 'en', 'fr-CA'], lat: 43.6532, lon: -79.3832 },
  { cc: 'MX', name: '墨西哥', tz: 'America/Mexico_City', langs: ['es-MX', 'es', 'en'], lat: 19.4326, lon: -99.1332 },
  { cc: 'BR', name: '巴西', tz: 'America/Sao_Paulo', langs: ['pt-BR', 'pt', 'en'], lat: -23.5505, lon: -46.6333 },
  { cc: 'AR', name: '阿根廷', tz: 'America/Argentina/Buenos_Aires', langs: ['es-AR', 'es', 'en'], lat: -34.6037, lon: -58.3816 },
  { cc: 'CL', name: '智利', tz: 'America/Santiago', langs: ['es-CL', 'es', 'en'], lat: -33.4489, lon: -70.6693 },
  { cc: 'CO', name: '哥伦比亚', tz: 'America/Bogota', langs: ['es-CO', 'es', 'en'], lat: 4.711, lon: -74.0721 },
  { cc: 'PE', name: '秘鲁', tz: 'America/Lima', langs: ['es-PE', 'es', 'en'], lat: -12.0464, lon: -77.0428 },
  { cc: 'GB', name: '英国', tz: 'Europe/London', langs: ['en-GB', 'en'], lat: 51.5074, lon: -0.1278 },
  { cc: 'IE', name: '爱尔兰', tz: 'Europe/Dublin', langs: ['en-IE', 'en'], lat: 53.3498, lon: -6.2603 },
  { cc: 'DE', name: '德国', tz: 'Europe/Berlin', langs: ['de-DE', 'de', 'en'], lat: 52.52, lon: 13.405 },
  { cc: 'FR', name: '法国', tz: 'Europe/Paris', langs: ['fr-FR', 'fr', 'en'], lat: 48.8566, lon: 2.3522 },
  { cc: 'ES', name: '西班牙', tz: 'Europe/Madrid', langs: ['es-ES', 'es', 'en'], lat: 40.4168, lon: -3.7038 },
  { cc: 'PT', name: '葡萄牙', tz: 'Europe/Lisbon', langs: ['pt-PT', 'pt', 'en'], lat: 38.7223, lon: -9.1393 },
  { cc: 'IT', name: '意大利', tz: 'Europe/Rome', langs: ['it-IT', 'it', 'en'], lat: 41.9028, lon: 12.4964 },
  { cc: 'NL', name: '荷兰', tz: 'Europe/Amsterdam', langs: ['nl-NL', 'nl', 'en'], lat: 52.3676, lon: 4.9041 },
  { cc: 'BE', name: '比利时', tz: 'Europe/Brussels', langs: ['nl-BE', 'fr-BE', 'en'], lat: 50.8503, lon: 4.3517 },
  { cc: 'CH', name: '瑞士', tz: 'Europe/Zurich', langs: ['de-CH', 'fr-CH', 'en'], lat: 47.3769, lon: 8.5417 },
  { cc: 'AT', name: '奥地利', tz: 'Europe/Vienna', langs: ['de-AT', 'de', 'en'], lat: 48.2082, lon: 16.3738 },
  { cc: 'SE', name: '瑞典', tz: 'Europe/Stockholm', langs: ['sv-SE', 'sv', 'en'], lat: 59.3293, lon: 18.0686 },
  { cc: 'NO', name: '挪威', tz: 'Europe/Oslo', langs: ['nb-NO', 'no', 'en'], lat: 59.9139, lon: 10.7522 },
  { cc: 'DK', name: '丹麦', tz: 'Europe/Copenhagen', langs: ['da-DK', 'da', 'en'], lat: 55.6761, lon: 12.5683 },
  { cc: 'FI', name: '芬兰', tz: 'Europe/Helsinki', langs: ['fi-FI', 'fi', 'en'], lat: 60.1699, lon: 24.9384 },
  { cc: 'PL', name: '波兰', tz: 'Europe/Warsaw', langs: ['pl-PL', 'pl', 'en'], lat: 52.2297, lon: 21.0122 },
  { cc: 'CZ', name: '捷克', tz: 'Europe/Prague', langs: ['cs-CZ', 'cs', 'en'], lat: 50.0755, lon: 14.4378 },
  { cc: 'HU', name: '匈牙利', tz: 'Europe/Budapest', langs: ['hu-HU', 'hu', 'en'], lat: 47.4979, lon: 19.0402 },
  { cc: 'RO', name: '罗马尼亚', tz: 'Europe/Bucharest', langs: ['ro-RO', 'ro', 'en'], lat: 44.4268, lon: 26.1025 },
  { cc: 'GR', name: '希腊', tz: 'Europe/Athens', langs: ['el-GR', 'el', 'en'], lat: 37.9838, lon: 23.7275 },
  { cc: 'UA', name: '乌克兰', tz: 'Europe/Kyiv', langs: ['uk-UA', 'uk', 'ru', 'en'], lat: 50.4501, lon: 30.5234 },
  { cc: 'RU', name: '俄罗斯', tz: 'Europe/Moscow', langs: ['ru-RU', 'ru', 'en'], lat: 55.7558, lon: 37.6173 },
  { cc: 'TR', name: '土耳其', tz: 'Europe/Istanbul', langs: ['tr-TR', 'tr', 'en'], lat: 41.0082, lon: 28.9784 },
  { cc: 'IL', name: '以色列', tz: 'Asia/Jerusalem', langs: ['he-IL', 'he', 'en'], lat: 32.0853, lon: 34.7818 },
  { cc: 'AE', name: '阿联酋', tz: 'Asia/Dubai', langs: ['ar-AE', 'ar', 'en'], lat: 25.2048, lon: 55.2708 },
  { cc: 'SA', name: '沙特阿拉伯', tz: 'Asia/Riyadh', langs: ['ar-SA', 'ar', 'en'], lat: 24.7136, lon: 46.6753 },
  { cc: 'EG', name: '埃及', tz: 'Africa/Cairo', langs: ['ar-EG', 'ar', 'en'], lat: 30.0444, lon: 31.2357 },
  { cc: 'ZA', name: '南非', tz: 'Africa/Johannesburg', langs: ['en-ZA', 'en', 'af'], lat: -26.2041, lon: 28.0473 },
  { cc: 'NG', name: '尼日利亚', tz: 'Africa/Lagos', langs: ['en-NG', 'en'], lat: 6.5244, lon: 3.3792 },
  { cc: 'KE', name: '肯尼亚', tz: 'Africa/Nairobi', langs: ['en-KE', 'sw', 'en'], lat: -1.2921, lon: 36.8219 },
  { cc: 'IN', name: '印度', tz: 'Asia/Kolkata', langs: ['en-IN', 'hi', 'en'], lat: 19.076, lon: 72.8777 },
  { cc: 'PK', name: '巴基斯坦', tz: 'Asia/Karachi', langs: ['en-PK', 'ur', 'en'], lat: 24.8607, lon: 67.0011 },
  { cc: 'BD', name: '孟加拉国', tz: 'Asia/Dhaka', langs: ['bn-BD', 'bn', 'en'], lat: 23.8103, lon: 90.4125 },
  { cc: 'TH', name: '泰国', tz: 'Asia/Bangkok', langs: ['th-TH', 'th', 'en'], lat: 13.7563, lon: 100.5018 },
  { cc: 'VN', name: '越南', tz: 'Asia/Ho_Chi_Minh', langs: ['vi-VN', 'vi', 'en'], lat: 10.8231, lon: 106.6297 },
  { cc: 'MY', name: '马来西亚', tz: 'Asia/Kuala_Lumpur', langs: ['ms-MY', 'en-MY', 'en'], lat: 3.139, lon: 101.6869 },
  { cc: 'SG', name: '新加坡', tz: 'Asia/Singapore', langs: ['en-SG', 'en', 'zh-SG'], lat: 1.3521, lon: 103.8198 },
  { cc: 'ID', name: '印度尼西亚', tz: 'Asia/Jakarta', langs: ['id-ID', 'id', 'en'], lat: -6.2088, lon: 106.8456 },
  { cc: 'PH', name: '菲律宾', tz: 'Asia/Manila', langs: ['en-PH', 'fil', 'en'], lat: 14.5995, lon: 120.9842 },
  { cc: 'CN', name: '中国', tz: 'Asia/Shanghai', langs: ['zh-CN', 'zh', 'en'], lat: 31.2304, lon: 121.4737 },
  { cc: 'HK', name: '中国香港', tz: 'Asia/Hong_Kong', langs: ['zh-HK', 'zh', 'en'], lat: 22.3193, lon: 114.1694 },
  { cc: 'TW', name: '中国台湾', tz: 'Asia/Taipei', langs: ['zh-TW', 'zh', 'en'], lat: 25.033, lon: 121.5654 },
  { cc: 'JP', name: '日本', tz: 'Asia/Tokyo', langs: ['ja-JP', 'ja', 'en'], lat: 35.6762, lon: 139.6503 },
  { cc: 'KR', name: '韩国', tz: 'Asia/Seoul', langs: ['ko-KR', 'ko', 'en'], lat: 37.5665, lon: 126.978 },
  { cc: 'AU', name: '澳大利亚', tz: 'Australia/Sydney', langs: ['en-AU', 'en'], lat: -33.8688, lon: 151.2093 },
  { cc: 'NZ', name: '新西兰', tz: 'Pacific/Auckland', langs: ['en-NZ', 'en'], lat: -36.8485, lon: 174.7633 },
];

// Locales offered in the language picker.
export const LANGS: [string, string][] = [
  ['en-US', 'English (United States)'], ['en-GB', 'English (United Kingdom)'], ['en-CA', 'English (Canada)'], ['en-AU', 'English (Australia)'], ['en-IN', 'English (India)'], ['en-SG', 'English (Singapore)'], ['en', 'English'],
  ['zh-CN', '中文（简体）'], ['zh-TW', '中文（繁體·台灣）'], ['zh-HK', '中文（繁體·香港）'], ['zh', '中文'],
  ['ja-JP', '日本語'], ['ko-KR', '한국어'], ['de-DE', 'Deutsch (Deutschland)'], ['de-AT', 'Deutsch (Österreich)'], ['de-CH', 'Deutsch (Schweiz)'], ['de', 'Deutsch'],
  ['fr-FR', 'Français (France)'], ['fr-CA', 'Français (Canada)'], ['fr-BE', 'Français (Belgique)'], ['fr', 'Français'],
  ['es-ES', 'Español (España)'], ['es-MX', 'Español (México)'], ['es-AR', 'Español (Argentina)'], ['es-CO', 'Español (Colombia)'], ['es-CL', 'Español (Chile)'], ['es', 'Español'],
  ['pt-BR', 'Português (Brasil)'], ['pt-PT', 'Português (Portugal)'], ['pt', 'Português'], ['it-IT', 'Italiano'], ['it', 'Italiano (generic)'],
  ['nl-NL', 'Nederlands'], ['nl-BE', 'Nederlands (België)'], ['sv-SE', 'Svenska'], ['nb-NO', 'Norsk bokmål'], ['da-DK', 'Dansk'], ['fi-FI', 'Suomi'],
  ['pl-PL', 'Polski'], ['cs-CZ', 'Čeština'], ['hu-HU', 'Magyar'], ['ro-RO', 'Română'], ['el-GR', 'Ελληνικά'], ['uk-UA', 'Українська'], ['ru-RU', 'Русский'], ['ru', 'Русский (generic)'],
  ['tr-TR', 'Türkçe'], ['he-IL', 'עברית'], ['ar-SA', 'العربية (السعودية)'], ['ar-AE', 'العربية (الإمارات)'], ['ar-EG', 'العربية (مصر)'], ['ar', 'العربية'],
  ['hi-IN', 'हिन्दी'], ['bn-BD', 'বাংলা'], ['ur-PK', 'اردو'], ['th-TH', 'ไทย'], ['vi-VN', 'Tiếng Việt'], ['id-ID', 'Bahasa Indonesia'], ['ms-MY', 'Bahasa Melayu'], ['fil', 'Filipino'], ['sw', 'Kiswahili'], ['af', 'Afrikaans'],
];
const LANG_NAME = Object.fromEntries(LANGS);

const FALLBACK_TZ = Array.from(new Set(COUNTRIES.map(c => c.tz).concat(['UTC', 'America/Chicago', 'America/Denver', 'America/Los_Angeles', 'America/Phoenix', 'America/Anchorage', 'Pacific/Honolulu', 'Australia/Melbourne', 'Australia/Perth', 'Asia/Kathmandu', 'Asia/Tehran'])));
export const TIMEZONES: string[] = (() => {
  try { const all = (Intl as any).supportedValuesOf?.('timeZone') as string[] | undefined; if (all && all.length) return all; } catch { /* old engines */ }
  return FALLBACK_TZ.sort();
})();
const tzOffset = (tz: string) => {
  try {
    const p = new Intl.DateTimeFormat('en-US', { timeZone: tz, timeZoneName: 'shortOffset' }).formatToParts(new Date()).find(x => x.type === 'timeZoneName');
    return p ? p.value.replace('GMT', 'UTC') : '';
  } catch { return ''; }
};

/** Searchable dropdown: a filter box above a styled list. */
export function SearchSelect({ value, options, onChange, placeholder }: { value: string; options: [string, string][]; onChange: (v: string) => void; placeholder?: string }) {
  const [q, setQ] = useState('');
  const list = useMemo(() => { const s = q.trim().toLowerCase(); return s ? options.filter(([v, l]) => v.toLowerCase().includes(s) || l.toLowerCase().includes(s)) : options; }, [q, options]);
  const shown = list.some(([v]) => v === value) || !value ? list : [[value, options.find(o => o[0] === value)?.[1] || value] as [string, string], ...list];
  return <div className="ssel">
    <input type="text" className="ssel-q" value={q} onChange={e => setQ(e.target.value)} placeholder={placeholder || '搜索…'} />
    <SelectField value={value} onChange={v => { if (v) { onChange(v); setQ(''); } }} options={[...(!value ? [['', list.length ? '请选择' : '没有匹配的机型'] as [string, string]] : []), ...shown]} />
  </div>;
}

export const TimezoneSelect: React.FC<{ value: string; onChange: (v: string) => void }> = ({ value, onChange }) => {
  const opts = useMemo(() => TIMEZONES.map(t => [t, `${t.replace(/_/g, ' ')}  ${tzOffset(t)}`] as [string, string]), []);
  return <SearchSelect value={value} options={opts} onChange={onChange} placeholder="搜索时区，如 Tokyo / UTC+8" />;
};

export const CountrySelect: React.FC<{ value: string; onChange: (c: Country) => void }> = ({ value, onChange }) => {
  const opts = useMemo(() => COUNTRIES.map(c => [c.cc, `${c.cc} · ${c.name}`] as [string, string]), []);
  return <SearchSelect value={value} options={opts} onChange={cc => { const c = COUNTRIES.find(x => x.cc === cc); if (c) onChange(c); }} placeholder="搜索国家/地区，如 US / 日本" />;
};

/** Ordered multi-language picker: chips in priority order plus a dropdown to add more. */
export const LanguagePicker: React.FC<{ value: string[]; onChange: (v: string[]) => void }> = ({ value, onChange }) => {
  const opts = useMemo(() => LANGS.filter(([v]) => !value.includes(v)), [value]);
  const move = (i: number, d: number) => { const n = [...value]; const j = i + d; if (j < 0 || j >= n.length) return; [n[i], n[j]] = [n[j], n[i]]; onChange(n); };
  return <div className="langpick">
    <div className="chips">
      {value.length === 0 && <span className="sub">未选择语言</span>}
      {value.map((l, i) => <span key={l} className="chip" title={LANG_NAME[l] || l}>
        {i > 0 && <button type="button" onClick={() => move(i, -1)} title="提高优先级">‹</button>}
        <b>{l}</b><small>{(LANG_NAME[l] || '').replace(/\s*\(.*\)/, '')}</small>
        <button type="button" onClick={() => onChange(value.filter(x => x !== l))} title="移除">×</button>
      </span>)}
    </div>
    <SearchSelect value="" options={opts} onChange={v => v && onChange([...value, v])} placeholder="搜索并添加语言，如 English / 日本語" />
  </div>;
};

/** Font picker: chips for chosen fonts, dropdown to add any font of this OS, quick all / random. */
export const FontPicker: React.FC<{ value: string[]; all: string[]; onChange: (v: string[]) => void; onSave?: (f: string) => void }> = ({ value, all, onChange, onSave }) => {
  const [custom, setCustom] = useState('');
  const addCustom = (save: boolean) => { const f = custom.trim(); if (!f) return; if (!value.includes(f)) onChange([...value, f]); if (save && onSave) onSave(f); setCustom(''); };
  const opts = useMemo(() => all.filter(f => !value.includes(f)).map(f => [f, f] as [string, string]), [all, value]);
  const random = () => onChange(all.filter((_, i) => i < 12 || Math.random() > 0.18));
  return <div className="langpick">
    <div className="row" style={{ gap: 6, marginBottom: 8 }}>
      <button type="button" className="btn sm" onClick={() => onChange([...all])}>全部</button>
      <button type="button" className="btn sm" onClick={random}>随机子集</button>
      <button type="button" className="btn sm" onClick={() => onChange([])}>清空</button>
    </div>
    <div className="chips" style={{ maxHeight: 150, overflowY: 'auto' }}>
      {value.length === 0 && <span className="sub">未选择字体</span>}
      {value.map(f => <span key={f} className="chip" style={{ fontFamily: `'${f}'` }}><b style={{ fontFamily: 'inherit', fontWeight: 500 }}>{f}</b>
        <button type="button" onClick={() => onChange(value.filter(x => x !== f))} title="移除">×</button></span>)}
    </div>
    <SearchSelect value="" options={opts} onChange={v => v && onChange([...value, v])} placeholder="搜索并添加字体" />
    <div className="combo-row" style={{ marginTop: 6 }}>
      <input type="text" value={custom} onChange={e => setCustom(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') addCustom(false); }} placeholder="手动输入字体名，回车添加" />
      <button type="button" className="btn sm combo-btn" onClick={() => addCustom(false)}>添加</button>
      {onSave && <button type="button" className="btn sm combo-btn" onClick={() => addCustom(true)}>★ 添加并存入库</button>}
    </div>
  </div>;
};

/**
 * Combo: type freely or pick from the list. Values the user typed can be stored
 * in their own fingerprint library (★) and removed again (×).
 */
export const Combo: React.FC<{
  value: string; options: string[]; custom?: string[]; onChange: (v: string) => void;
  onSave?: (v: string) => void; onRemove?: (v: string) => void; placeholder?: string; multiline?: boolean; disabled?: boolean; render?: (v: string) => string;
}> = ({ value, options, custom = [], onChange, onSave, onRemove, placeholder, multiline, disabled, render }) => {
  const [open, setOpen] = useState(false);
  const [typed, setTyped] = useState(false);
  const q = typed ? value.trim().toLowerCase() : '';
  const list = useMemo(() => (q ? options.filter(o => o.toLowerCase().includes(q)) : options).slice(0, 300), [q, options]);
  const known = options.includes(value);
  const pickV = (v: string) => { onChange(v); setOpen(false); setTyped(false); };
  const row = useRef<HTMLDivElement>(null);
  const common = {
    value, disabled, placeholder: placeholder || '手动输入，或点 ▾ 从列表选择',
    onChange: (e: any) => { onChange(e.target.value); setTyped(true); setOpen(true); },
    onFocus: () => setTyped(false), onBlur: () => setTimeout(() => setOpen(false), 120),
    onKeyDown: (e: any) => { if (e.key === 'Escape') setOpen(false); },
  };
  return <div className="combo">
    <div className="combo-row" ref={row}>
      {multiline ? <textarea {...common} style={{ minHeight: 54 }} /> : <input type="text" {...common} />}
      <button type="button" className="btn sm combo-btn" disabled={disabled} onMouseDown={e => e.preventDefault()} onClick={() => { setTyped(false); setOpen(o => !o); }} title="从列表选择">▾</button>
      {onSave && value.trim() && !known && !disabled && <button type="button" className="btn sm combo-btn" onClick={() => onSave(value.trim())} title="存入我的指纹库">★ 存入库</button>}
    </div>
    <Floating open={open} anchor={row} onClose={() => setOpen(false)} matchWidth gap={4} className="combo-list" onMouseDown={e => e.preventDefault()}>
      {list.length === 0 && <div className="combo-empty">没有匹配项，可以直接使用输入的值{onSave ? '，或点「★ 存入库」' : ''}</div>}
      {list.map(o => <div key={o} className={'combo-item' + (o === value ? ' on' : '')} onClick={() => pickV(o)}>
        {custom.includes(o) && <span className="combo-star" title="我的指纹库">★</span>}
        <span className="combo-text">{render ? render(o) : o}</span>
        {custom.includes(o) && onRemove && <button type="button" className="combo-x" title="从我的指纹库删除" onClick={e => { e.stopPropagation(); onRemove(o); }}>×</button>}
      </div>)}
    </Floating>
  </div>;
};

export const countryOfTz = (tz: string) => COUNTRIES.find(c => c.tz === tz)?.cc || '';
