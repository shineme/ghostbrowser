import json
from pathlib import Path
kit=Path(__file__).resolve().parents[1]
lib=json.loads((kit/'fingerprints/library.json').read_text(encoding='utf8'))
paths=set()
def walk(prefix,value):
 if isinstance(value,dict):
  for k,v in value.items():walk(prefix+'.'+k,v)
 elif isinstance(value,list):
  if not value:paths.add(prefix+'[]')
  elif all(not isinstance(v,(dict,list)) for v in value):paths.add(prefix+'[]')
  else:
   for v in value:walk(prefix+'[]',v)
 else:paths.add(prefix)
for p in lib['profiles']:
 walk('data',p['data']);walk('configMeta',p['configMeta'])
c=lib['profiles'][0];ai=c['configMeta']['additional_info'];wg=ai['webglParameters']['parameters']
summary={'snapshotCount':len(lib['profiles']),'uniqueConfigIDs':lib['uniqueConfigIDs'],'dataTopKeys':len(c['data']),'configMetaTopKeys':len(c['configMeta']),'additionalInfoTopKeys':len(ai),'UAClientHintTopKeys':len(ai['userAgentDataObj']),'webglParameterTopKeys':len(wg),'canonicalLeafPaths':len(paths),'webglLeafPaths':len([p for p in paths if p.startswith('configMeta.additional_info.webglParameters.')]),'leafPaths':sorted(paths)}
(kit/'fingerprints/field-index.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2),encoding='utf8')
print(json.dumps({k:v for k,v in summary.items() if k!='leafPaths'},ensure_ascii=False,indent=2))
