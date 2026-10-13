"""Produce a read-only file integrity inventory of this assembled bundle."""
from pathlib import Path
import json, hashlib
kit=Path(__file__).resolve().parents[1]
records=[]
for p in sorted(kit.rglob('*')):
    if not p.is_file() or p.name=='bundle-manifest.json':
        continue
    if any(x in p.relative_to(kit).parts for x in ['profiles','backup-source']):
        continue
    with p.open('rb') as f:
        sha=hashlib.file_digest(f,'sha256').hexdigest()
    records.append({'path':p.relative_to(kit).as_posix(),'bytes':p.stat().st_size,'sha256':sha})
(kit/'bundle-manifest.json').write_text(json.dumps({'schemaVersion':1,'files':records},ensure_ascii=False,indent=2),encoding='utf8')
print('bundle files',len(records),'bytes',sum(x['bytes'] for x in records))
for p in (kit/'bin').glob('*.exe'):
    print(p.name,p.stat().st_size)
