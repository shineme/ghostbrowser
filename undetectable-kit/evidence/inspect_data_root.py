"""Read-only decode of the recovered Chromium profile-data-root function."""
import struct
from pathlib import Path
from capstone import Cs, CS_ARCH_X86, CS_MODE_64

p=Path(__file__).resolve().parents[1]/'runtime/chrome/chrome.dll'
b=p.read_bytes();pe=struct.unpack_from('<I',b,0x3c)[0];opt=pe+24
base=struct.unpack_from('<Q',b,opt+24)[0]
ns=struct.unpack_from('<H',b,pe+6)[0];ss=opt+struct.unpack_from('<H',b,pe+20)[0]
sects=[struct.unpack_from('<IIII',b,ss+i*40+8) for i in range(ns)]
def off(va):
 r=va-base
 for vs,vr,rs,rf in sects:
  if vr<=r<vr+max(vs,rs):return rf+r-vr
def string(va):
 o=off(va)
 if o is None:return ''
 s=b[o:o+120].split(b'\0')[0]
 return s.decode('ascii') if s and all(32<=v<127 for v in s) else ''
md=Cs(CS_ARCH_X86,CS_MODE_64)
for va,n in [(0x1832fad60,0x690)]:
 for i in md.disasm(b[off(va):off(va)+n],va):
  extra=''
  if 'rip + 0x' in i.op_str:
   disp=int(i.op_str.split('rip + 0x')[1].split(']')[0],16)
   extra=string(i.address+i.size+disp)
  print(hex(i.address),i.mnemonic,i.op_str,extra)
