package main

// Page-side scripts of the session store (sessionstore.go). The value codec is
// GhostBrowser's own: JSON with "$t"-tagged objects for what JSON cannot hold.
// A plain object that has its own "$t" key is wrapped as {$t:"o", v:{...}}.

// jsDumpOrigin(budgetBytes, withIDB) -> {local, lsErr, idb, partial, size}
const jsDumpOrigin = `async (budget, wantIDB) => {
  const B64 = u8 => { let s = ''; for (let i = 0; i < u8.length; i += 0x8000) s += String.fromCharCode.apply(null, u8.subarray(i, i + 0x8000)); return btoa(s); };
  const blobs = [];
  const enc = (v, d) => {
    if (d > 64) throw new Error('depth');
    if (v === undefined) return { $t: 'u' };
    if (v === null || typeof v === 'string' || typeof v === 'boolean') return v;
    if (typeof v === 'number') return Number.isFinite(v) ? v : { $t: 'n', v: String(v) };
    if (typeof v === 'bigint') return { $t: 'bi', v: String(v) };
    if (typeof v !== 'object') throw new Error('type ' + typeof v);
    const tag = Object.prototype.toString.call(v).slice(8, -1);
    switch (tag) {
      case 'Date': return { $t: 'd', v: v.getTime() };
      case 'RegExp': return { $t: 're', s: v.source, f: v.flags };
      case 'ArrayBuffer': return { $t: 'ab', v: B64(new Uint8Array(v)) };
      case 'Map': return { $t: 'map', v: Array.from(v, ([k, x]) => [enc(k, d + 1), enc(x, d + 1)]) };
      case 'Set': return { $t: 'set', v: Array.from(v, x => enc(x, d + 1)) };
      case 'Blob': case 'File': { const o = { $t: 'blob', type: v.type }; if (tag === 'File') { o.name = v.name; o.lm = v.lastModified; } blobs.push([o, v]); return o; }
      case 'Error': return { $t: 'err', n: v.name, m: v.message };
      case 'Array': return v.map(x => enc(x, d + 1));
      case 'Object': { const o = {}; for (const k of Object.keys(v)) o[k] = enc(v[k], d + 1); return Object.prototype.hasOwnProperty.call(v, '$t') ? { $t: 'o', v: o } : o; }
    }
    if (ArrayBuffer.isView(v)) return { $t: 'ta', c: tag, v: B64(new Uint8Array(v.buffer, v.byteOffset, v.byteLength)) };
    throw new Error('unsupported ' + tag);
  };
  const msg = e => String(e && e.message || e);
  const out = { local: null, idb: [], partial: false, size: 0 };
  try {
    const l = {};
    for (let i = 0; i < localStorage.length; i++) { const k = localStorage.key(i); const v = localStorage.getItem(k); if (v === null) continue; l[k] = v; out.size += k.length + v.length + 6; }
    out.local = l;
  } catch (e) { out.lsErr = msg(e); }
  if (!wantIDB) return out;
  if (!self.indexedDB || !indexedDB.databases) { out.partial = true; return out; }
  const req = r => new Promise((ok, no) => { r.onsuccess = () => ok(r.result); r.onerror = () => no(r.error); });
  let infos = [];
  try { infos = await indexedDB.databases(); } catch (e) { out.partial = true; return out; }
  for (const info of infos) {
    if (!info.name) continue;
    let db;
    try {
      db = await new Promise((ok, no) => {
        const r = indexedDB.open(info.name);
        r.onsuccess = () => ok(r.result); r.onerror = () => no(r.error);
        r.onupgradeneeded = () => r.transaction.abort(); // vanished meanwhile: do not create it
        r.onblocked = () => no(new Error('blocked'));
      });
    } catch (e) { out.partial = true; continue; }
    const D = { name: db.name, version: db.version, stores: [] };
    try {
      for (const sn of Array.from(db.objectStoreNames)) {
        const s0 = db.transaction(sn, 'readonly').objectStore(sn);
        const S = { name: sn, keyPath: s0.keyPath, autoIncrement: s0.autoIncrement, indexes: [], records: [] };
        for (const xn of Array.from(s0.indexNames)) { const x = s0.index(xn); S.indexes.push({ name: xn, keyPath: x.keyPath, unique: x.unique, multiEntry: x.multiEntry }); }
        D.stores.push(S);
        let lower = null, more = true;
        while (more) {
          if (out.size > budget) { S.truncated = true; out.partial = true; break; }
          const st = db.transaction(sn, 'readonly').objectStore(sn);
          const range = lower === null ? null : IDBKeyRange.lowerBound(lower, true);
          const [keys, vals] = await Promise.all([req(st.getAllKeys(range, 200)), req(st.getAll(range, 200))]);
          if (keys.length < 200) more = false;
          if (keys.length) lower = keys[keys.length - 1];
          for (let i = 0; i < keys.length; i++) {
            blobs.length = 0;
            let rec;
            try {
              rec = { k: enc(keys[i], 0), v: enc(vals[i], 0) };
              for (const [o, b] of blobs) o.v = B64(new Uint8Array(await b.arrayBuffer()));
            } catch (e) { out.partial = true; S.skipped = (S.skipped || 0) + 1; continue; }
            const n = JSON.stringify(rec).length;
            if (out.size + n > budget) { S.truncated = true; out.partial = true; more = false; break; }
            out.size += n;
            S.records.push(rec);
          }
        }
      }
    } catch (e) { out.partial = true; }
    db.close();
    out.idb.push(D);
  }
  return out;
}`

// jsRestoreLocal(items) -> number of keys written
const jsRestoreLocal = `(items) => { let n = 0; for (const k of Object.keys(items)) { try { localStorage.setItem(k, items[k]); n++; } catch (e) {} } return n; }`

// jsRestoreIDB(db) -> {Put, Err}. Creates the database at the captured version
// (so the site's own upgrade code sees an up-to-date database), adds missing
// stores and indexes when it may, and puts every record.
const jsRestoreIDB = `async (D) => {
  const U8 = s => { const b = atob(s || ''); const u = new Uint8Array(b.length); for (let i = 0; i < b.length; i++) u[i] = b.charCodeAt(i); return u; };
  const obj = src => { const o = {}; for (const k of Object.keys(src)) o[k] = dec(src[k]); return o; };
  const dec = v => {
    if (v === null || typeof v !== 'object') return v;
    if (Array.isArray(v)) return v.map(dec);
    switch (v.$t) {
      case undefined: return obj(v);
      case 'o': return obj(v.v);
      case 'u': return undefined;
      case 'n': return Number(v.v);
      case 'bi': return BigInt(v.v);
      case 'd': return new Date(v.v);
      case 're': return new RegExp(v.s, v.f);
      case 'ab': return U8(v.v).buffer;
      case 'ta': { const u = U8(v.v); if (v.c === 'DataView') return new DataView(u.buffer); const C = self[v.c]; return C ? new C(u.buffer, 0, u.byteLength / C.BYTES_PER_ELEMENT) : u; }
      case 'map': return new Map(v.v.map(([k, x]) => [dec(k), dec(x)]));
      case 'set': return new Set(v.v.map(dec));
      case 'blob': { const u = U8(v.v); return v.name !== undefined ? new File([u], v.name, { type: v.type, lastModified: v.lm }) : new Blob([u], { type: v.type }); }
      case 'err': { const e = new Error(v.m); e.name = v.n; return e; }
    }
    return obj(v);
  };
  const res = { Put: 0, Err: 0 };
  const open = ver => new Promise((ok, no) => {
    const r = ver ? indexedDB.open(D.name, ver) : indexedDB.open(D.name);
    let t = 0;
    r.onupgradeneeded = () => {
      const db = r.result;
      for (const S of D.stores) {
        const st = db.objectStoreNames.contains(S.name) ? r.transaction.objectStore(S.name)
          : db.createObjectStore(S.name, { keyPath: S.keyPath === null ? undefined : S.keyPath, autoIncrement: S.autoIncrement });
        for (const x of S.indexes || []) if (!st.indexNames.contains(x.name)) st.createIndex(x.name, x.keyPath, { unique: x.unique, multiEntry: x.multiEntry });
      }
    };
    r.onsuccess = () => { clearTimeout(t); ok(r.result); };
    r.onerror = e => { clearTimeout(t); e.preventDefault(); no(r.error); };
    r.onblocked = () => { t = setTimeout(() => no(new Error('blocked')), 4000); };
  });
  let db;
  try { db = await open(D.version > 0 ? D.version : undefined); }
  catch (e) { if (e && e.name === 'VersionError') db = await open(); else throw e; }
  try {
    for (const S of D.stores) {
      if (!db.objectStoreNames.contains(S.name)) { res.Err += S.records.length; continue; }
      for (let i = 0; i < S.records.length; i += 500) {
        await new Promise(ok => {
          const tx = db.transaction(S.name, 'readwrite');
          const st = tx.objectStore(S.name);
          for (const rec of S.records.slice(i, i + 500)) {
            try {
              const q = st.keyPath === null ? st.put(dec(rec.v), dec(rec.k)) : st.put(dec(rec.v));
              q.onsuccess = () => res.Put++;
              q.onerror = e => { e.preventDefault(); res.Err++; };
            } catch (e) { res.Err++; }
          }
          tx.oncomplete = tx.onabort = () => ok();
        });
      }
    }
  } finally { db.close(); }
  return res;
}`
