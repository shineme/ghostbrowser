import React, { useEffect, useState } from 'react';
import { api, Profile, NativePreset } from './api';
import { Field, Icon, useToast } from './ui';

export const NativeTools: React.FC<{profiles: Profile[]; onClose:()=>void; onChanged:()=>void}> = ({profiles,onClose,onChanged}) => {
 const toast=useToast();
 const [presets,setPresets]=useState<NativePreset[]>([]);
 const [preset,setPreset]=useState('');
 const [id,setId]=useState(profiles[0]?.id||'');
 const [name,setName]=useState('Native Profile');
 const [raw,setRaw]=useState('');
 const [busy,setBusy]=useState(false);
 useEffect(()=>{api.NativePresets().then(list=>{setPresets(list);setPreset(list[0]?.id||'');}).catch(e=>toast(String(e),'bad'));},[]);
 const action=async(fn:()=>Promise<void>)=>{setBusy(true);try{await fn();}catch(e){toast(String(e),'bad');}finally{setBusy(false);}};
 const create=()=>action(async()=>{await api.CreateNativeProfile(preset,name);onChanged();toast('已从指纹库创建环境');onClose();});
 const importJSON=()=>action(async()=>{await api.ImportNativeJSON(raw);onChanged();toast('已导入为新环境');onClose();});
 const exportJSON=()=>action(async()=>{setRaw(await api.ExportNativeJSON(id));});
 const probe=()=>action(async()=>{setRaw(await api.NativeSnapshot(id));toast('实时指纹已读回，请查看 mismatches');});
 const download=()=>{const url=URL.createObjectURL(new Blob([raw],{type:'application/json'}));const a=document.createElement('a');a.href=url;a.download='native-profile.json';a.click();URL.revokeObjectURL(url);};
 return <div className="overlay" onMouseDown={e=>e.target===e.currentTarget&&onClose()}><div className="drawer" style={{width:850,maxWidth:'95vw'}}>
  <div className="dhead"><h3>本地指纹库 / JSON</h3><div className="spacer"/><button className="btn ghost icon" onClick={onClose}><Icon.x/></button></div>
  <div className="dbody">
   <div className="sub" style={{marginBottom:14}}>本地有 {presets.length} 份环境快照，均来自配置 3285798。JSON 启动需要在设置中启用定制 Chromium。</div>
   <Field label="指纹快照"><select value={preset} onChange={e=>setPreset(e.target.value)}>{presets.map(p=><option key={p.id} value={p.id}>{p.name} · 配置 {p.configId}</option>)}</select></Field>
   <Field label="新环境名称"><input value={name} onChange={e=>setName(e.target.value)}/></Field>
   <button className="btn" disabled={busy||!preset} onClick={create}>使用选中指纹新建环境</button>
   <hr/>
   <Field label="已有环境"><select value={id} onChange={e=>setId(e.target.value)}><option value="">选择环境</option>{profiles.map(p=><option key={p.id} value={p.id}>{p.name}</option>)}</select></Field>
   <div style={{display:'flex',gap:10,marginBottom:14}}><button className="btn" disabled={busy||!id} onClick={exportJSON}>导出启动 JSON</button><button className="btn" disabled={busy||!id} onClick={probe}>验证运行中的指纹</button></div>
   <Field label="配置 JSON / 验证结果" hint="导入使用 schemaVersion=1 的启动 JSON；运行路径由本机设置确定。验证结果不能作为启动配置导入。"><textarea style={{minHeight:310,fontFamily:'monospace'}} value={raw} onChange={e=>setRaw(e.target.value)} placeholder="粘贴 native-profile.json"/></Field>
   <input type="file" accept=".json,application/json" onChange={async e=>{const file=e.target.files?.[0];if(file)setRaw(await file.text());}}/>
   <div className="sub" style={{marginTop:14}}>字体、媒体设备、语音、WebGL 参数与未映射的噪声开关继承所选二进制模板。编辑 JSON 并不意味着所有字段都已支持原生重写，详细支持范围见包内文档。</div>
  </div>
  <div className="dfoot"><button className="btn" disabled={!raw} onClick={download}>保存 JSON 文件</button><div className="spacer"/><button className="btn primary" disabled={busy||!raw} onClick={importJSON}>{busy?'处理中…':'导入为新环境'}</button><button className="btn" onClick={onClose}>关闭</button></div>
 </div></div>;
};
