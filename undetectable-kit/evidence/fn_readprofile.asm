0x0001832fd110  push     r15
0x0001832fd112  push     r14
0x0001832fd114  push     r13
0x0001832fd116  push     r12
0x0001832fd118  push     rsi
0x0001832fd119  push     rdi
0x0001832fd11a  push     rbp
0x0001832fd11b  push     rbx
0x0001832fd11c  sub      rsp, 0x238
0x0001832fd123  movaps   xmmword ptr [rsp + 0x220], xmm6
0x0001832fd12b  mov      rbx, rdx
0x0001832fd12e  mov      r12, rcx
0x0001832fd131  mov      rax, qword ptr [rip + 0xe164f08]   ; 0x191462040
0x0001832fd138  xor      rax, rsp
0x0001832fd13b  mov      qword ptr [rsp + 0x218], rax
0x0001832fd143  movabs   rdi, 0x7fffffffffffffff
0x0001832fd14d  add      rcx, 0x88
0x0001832fd154  xorps    xmm6, xmm6
0x0001832fd157  lea      r14, [rsp + 0x150]
0x0001832fd15f  movaps   xmmword ptr [r14], xmm6
0x0001832fd163  mov      qword ptr [r14 + 0x10], 0
0x0001832fd16b  mov      rdx, r14
0x0001832fd16e  mov      r8, 0xffffffffffffffff
0x0001832fd175  call     0x18158b790
0x0001832fd17a  test     al, al
0x0001832fd17c  je       0x1832ff5b1
0x0001832fd182  movabs   rcx, 0xfffffffc00000000
0x0001832fd18c  movaps   xmmword ptr [rsp + 0x130], xmm6
0x0001832fd194  mov      qword ptr [rsp + 0x140], 0
0x0001832fd1a0  mov      byte ptr [rsp + 0x138], 1
0x0001832fd1a8  mov      rax, r14
0x0001832fd1ab  and      rax, rcx
0x0001832fd1ae  cmp      rax, qword ptr [rip + 0xe1c9653]   ; 0x1914c6808
0x0001832fd1b5  jne      0x1832fd1c4
0x0001832fd1b7  lea      rcx, [rsp + 0x150]
0x0001832fd1bf  call     0x183b07770
0x0001832fd1c4  mov      qword ptr [rsp + 0x140], r14
0x0001832fd1cc  lea      rdx, [r12 + 8]
0x0001832fd1d1  lea      rcx, [rbx + 0x30]
0x0001832fd1d5  cmp      rcx, rdx
0x0001832fd1d8  je       0x1832fd201
0x0001832fd1da  cmp      byte ptr [rbx + 0x47], 0
0x0001832fd1de  movzx    r8d, byte ptr [r12 + 0x1f]
0x0001832fd1e4  js       0x1832ff598
0x0001832fd1ea  test     r8b, r8b
0x0001832fd1ed  js       0x1832ff584
0x0001832fd1f3  mov      rax, qword ptr [rdx + 0x10]
0x0001832fd1f7  mov      qword ptr [rcx + 0x10], rax
0x0001832fd1fb  movups   xmm0, xmmword ptr [rdx]
0x0001832fd1fe  movups   xmmword ptr [rcx], xmm0
0x0001832fd201  mov      byte ptr [rbx + 0x49], 0
0x0001832fd205  lea      rdx, [r12 + 0xa0]
0x0001832fd20d  xorps    xmm6, xmm6
0x0001832fd210  lea      r14, [rsp + 0x1c0]
0x0001832fd218  movaps   xmmword ptr [r14 + 0x10], xmm6
0x0001832fd21d  movaps   xmmword ptr [r14], xmm6
0x0001832fd221  mov      qword ptr [r14 + 0x20], 0
0x0001832fd229  mov      rcx, r14
0x0001832fd22c  mov      r8d, 0x21
0x0001832fd232  call     0x180c1b6d0
0x0001832fd237  mov      rax, qword ptr [r14]
0x0001832fd23a  test     rax, rax
0x0001832fd23d  sete     cl
0x0001832fd240  cmp      eax, -0xc
0x0001832fd243  setae    al
0x0001832fd246  or       al, cl
0x0001832fd248  jne      0x1832fd2a0
0x0001832fd24a  lea      r14, [rsp + 0x1c0]
0x0001832fd252  mov      rcx, r14
0x0001832fd255  call     0x184ee8a50
0x0001832fd25a  lea      r15, [rbx + 0x18]
0x0001832fd25e  mov      rcx, r15
0x0001832fd261  mov      rdx, rax
0x0001832fd264  mov      r8b, 0x30
0x0001832fd267  call     0x18434c750
0x0001832fd26c  movsx    rax, byte ptr [rbx + 0x2f]
0x0001832fd271  test     rax, rax
0x0001832fd274  jns      0x1832fd27e
0x0001832fd276  mov      rax, qword ptr [rbx + 0x20]
0x0001832fd27a  mov      r15, qword ptr [rbx + 0x18]
0x0001832fd27e  lea      r9, [rsp + 0x1a0]
0x0001832fd286  mov      qword ptr [r9], r15
0x0001832fd289  mov      qword ptr [r9 + 8], rax
0x0001832fd28d  lea      rdx, [rsp + 0x180]
0x0001832fd295  mov      rcx, r14
0x0001832fd298  xor      r8d, r8d
0x0001832fd29b  call     0x184b887b0
0x0001832fd2a0  mov      qword ptr [rsp + 0x28], r12
0x0001832fd2a5  lea      rcx, [rsp + 0x1c0]
0x0001832fd2ad  call     0x183d82c20
0x0001832fd2b2  lea      r15, [rsp + 0x130]
0x0001832fd2ba  mov      rcx, r15
0x0001832fd2bd  mov      rdx, rbx
0x0001832fd2c0  call     0x1833030f0
0x0001832fd2c5  xor      r14d, r14d
0x0001832fd2c8  mov      qword ptr [rsp + 0x1b0], r14
0x0001832fd2d0  movaps   xmmword ptr [rsp + 0x1a0], xmm6
0x0001832fd2d8  mov      qword ptr [rsp + 0x190], r14
0x0001832fd2e0  movaps   xmmword ptr [rsp + 0x180], xmm6
0x0001832fd2e8  cmp      byte ptr [r15 + 8], 1
0x0001832fd2ed  mov      ebp, 0
0x0001832fd2f2  jne      0x1832fd33f
0x0001832fd2f4  mov      rcx, qword ptr [rsp + 0x130]
0x0001832fd2fc  mov      rdx, qword ptr [rsp + 0x140]
0x0001832fd304  lea      rax, [rcx + 1]
0x0001832fd308  movsx    r9, byte ptr [rdx + 0x17]
0x0001832fd30d  mov      r8, r9
0x0001832fd310  test     r9, r9
0x0001832fd313  jns      0x1832fd319
0x0001832fd315  mov      r8, qword ptr [rdx + 8]
0x0001832fd319  xor      ebp, ebp
0x0001832fd31b  cmp      rax, r8
0x0001832fd31e  ja       0x1832fd334
0x0001832fd320  test     r9b, r9b
0x0001832fd323  jns      0x1832fd328
0x0001832fd325  mov      rdx, qword ptr [rdx]
0x0001832fd328  movsx    ebp, byte ptr [rdx + rcx]
0x0001832fd32c  mov      qword ptr [rsp + 0x130], rax
0x0001832fd334  cmp      rax, r8
0x0001832fd337  setbe    byte ptr [rsp + 0x138]
0x0001832fd33f  lea      rcx, [rsp + 0x130]
0x0001832fd347  lea      rdx, [rsp + 0x1a0]
0x0001832fd34f  call     0x183303050
0x0001832fd354  cmp      byte ptr [rax + 8], 1
0x0001832fd358  jne      0x1832fd395
0x0001832fd35a  mov      rdx, qword ptr [rax]
0x0001832fd35d  mov      r8, qword ptr [rax + 0x10]
0x0001832fd361  lea      rcx, [rdx + 1]
0x0001832fd365  movsx    r10, byte ptr [r8 + 0x17]
0x0001832fd36a  mov      r9, r10
0x0001832fd36d  test     r10, r10
0x0001832fd370  jns      0x1832fd376
0x0001832fd372  mov      r9, qword ptr [r8 + 8]
0x0001832fd376  xor      r14d, r14d
0x0001832fd379  cmp      rcx, r9
0x0001832fd37c  ja       0x1832fd38e
0x0001832fd37e  test     r10b, r10b
0x0001832fd381  jns      0x1832fd386
0x0001832fd383  mov      r8, qword ptr [r8]
0x0001832fd386  movsx    r14d, byte ptr [r8 + rdx]
0x0001832fd38b  mov      qword ptr [rax], rcx
0x0001832fd38e  cmp      rcx, r9
0x0001832fd391  setbe    byte ptr [rax + 8]
0x0001832fd395  lea      rdx, [rsp + 0x180]
0x0001832fd39d  mov      rcx, rax
0x0001832fd3a0  call     0x183303050
0x0001832fd3a5  mov      sil, byte ptr [rsp + 0x138]
0x0001832fd3ad  cmp      sil, 1
0x0001832fd3b1  jne      0x1832fd44f
0x0001832fd3b7  lea      r15, [rsp + 0x1c0]
0x0001832fd3bf  lea      r8, [rsp + 0x180]
0x0001832fd3c7  mov      rcx, r15
0x0001832fd3ca  mov      edx, r14d
0x0001832fd3cd  call     0x185d44a20
0x0001832fd3d2  mov      eax, dword ptr [r15]
0x0001832fd3d5  mov      dword ptr [rbx + 0x50], eax
0x0001832fd3d8  lea      rdi, [rbx + 0x58]
0x0001832fd3dc  cmp      byte ptr [rbx + 0x6f], 0
0x0001832fd3e0  js       0x183300c10
0x0001832fd3e6  mov      rax, qword ptr [r15 + 0x18]
0x0001832fd3ea  mov      qword ptr [rdi + 0x10], rax
0x0001832fd3ee  movups   xmm0, xmmword ptr [r15 + 8]
0x0001832fd3f3  movups   xmmword ptr [rdi], xmm0
0x0001832fd3f6  lea      r14, [rsp + 0x1c0]
0x0001832fd3fe  lea      r8, [rsp + 0x1a0]
0x0001832fd406  mov      rcx, r14
0x0001832fd409  mov      edx, ebp
0x0001832fd40b  call     0x185d44980
0x0001832fd410  mov      eax, dword ptr [r14]
0x0001832fd413  mov      dword ptr [rbx + 0x70], eax
0x0001832fd416  lea      rdi, [rbx + 0x78]
0x0001832fd41a  cmp      byte ptr [rbx + 0x8f], 0
0x0001832fd421  js       0x183300c2f
0x0001832fd427  mov      rax, qword ptr [r15 + 0x18]
0x0001832fd42b  mov      qword ptr [rdi + 0x10], rax
0x0001832fd42f  movups   xmm0, xmmword ptr [r15 + 8]
0x0001832fd434  movups   xmmword ptr [rdi], xmm0
0x0001832fd437  mov      eax, dword ptr [rbx + 0x50]
0x0001832fd43a  dec      eax
0x0001832fd43c  cmp      eax, 2
0x0001832fd43f  movabs   rdi, 0x7fffffffffffffff
0x0001832fd449  ja       0x1832fd44f
0x0001832fd44b  mov      byte ptr [rbx + 0x48], 1
0x0001832fd44f  cmp      byte ptr [rsp + 0x197], 0
0x0001832fd457  jns      0x1832fd471
0x0001832fd459  mov      rdx, qword ptr [rsp + 0x190]
0x0001832fd461  and      rdx, rdi
0x0001832fd464  mov      rcx, qword ptr [rsp + 0x180]
0x0001832fd46c  call     0x183b135e0
0x0001832fd471  cmp      byte ptr [rsp + 0x1b7], 0
0x0001832fd479  mov      rbp, qword ptr [rsp + 0x28]
0x0001832fd47e  jns      0x1832fd498
0x0001832fd480  mov      rdx, qword ptr [rsp + 0x1b0]
0x0001832fd488  and      rdx, rdi
0x0001832fd48b  mov      rcx, qword ptr [rsp + 0x1a0]
0x0001832fd493  call     0x183b135e0
0x0001832fd498  mov      al, byte ptr [rsp + 0x138]
0x0001832fd49f  test     sil, sil
0x0001832fd4a2  jne      0x1832fd4b8
0x0001832fd4a4  test     al, al
0x0001832fd4a6  je       0x1832fedab
0x0001832fd4ac  mov      byte ptr [rbx + 0x49], 1
0x0001832fd4b0  mov      dil, 1
0x0001832fd4b3  jmp      0x1832ff26e
0x0001832fd4b8  test     al, al
0x0001832fd4ba  je       0x1832fed97
0x0001832fd4c0  mov      rcx, qword ptr [rsp + 0x130]
0x0001832fd4c8  mov      rax, qword ptr [rsp + 0x140]
0x0001832fd4d0  lea      rdx, [rcx + 1]
0x0001832fd4d4  movsx    r8, byte ptr [rax + 0x17]
0x0001832fd4d9  mov      r9, r8
0x0001832fd4dc  test     r8, r8
0x0001832fd4df  jns      0x1832fd4e5
0x0001832fd4e1  mov      r9, qword ptr [rax + 8]
0x0001832fd4e5  mov      r15d, 0x8b
0x0001832fd4eb  cmp      rdx, r9
0x0001832fd4ee  ja       0x1832fed9f
0x0001832fd4f4  mov      r9, rax
0x0001832fd4f7  test     r8b, r8b
0x0001832fd4fa  jns      0x1832fd4ff
0x0001832fd4fc  mov      r9, qword ptr [rax]
0x0001832fd4ff  mov      r8b, byte ptr [r9 + rcx]
0x0001832fd503  mov      byte ptr [rbx + 0x4f8], r8b
0x0001832fd50a  mov      qword ptr [rsp + 0x130], rdx
0x0001832fd512  mov      byte ptr [rsp + 0x138], 1
0x0001832fd51a  lea      rdx, [rcx + 2]
0x0001832fd51e  movsx    r8, byte ptr [rax + 0x17]
0x0001832fd523  mov      r9, r8
0x0001832fd526  test     r8, r8
0x0001832fd529  jns      0x1832fd52f
0x0001832fd52b  mov      r9, qword ptr [rax + 8]
0x0001832fd52f  cmp      rdx, r9
0x0001832fd532  ja       0x1832fed9f
0x0001832fd538  mov      r9, rax
0x0001832fd53b  test     r8b, r8b
0x0001832fd53e  jns      0x1832fd543
0x0001832fd540  mov      r9, qword ptr [rax]
0x0001832fd543  mov      r8b, byte ptr [r9 + rcx + 1]
0x0001832fd548  mov      byte ptr [rbx + 0x4f9], r8b
0x0001832fd54f  mov      qword ptr [rsp + 0x130], rdx
0x0001832fd557  mov      dl, 1
0x0001832fd559  mov      byte ptr [rsp + 0x138], dl
0x0001832fd560  mov      byte ptr [rbx + 0x5b8], dl
0x0001832fd566  lea      rdx, [rcx + 6]
0x0001832fd56a  movsx    r8, byte ptr [rax + 0x17]
0x0001832fd56f  mov      r9, r8
0x0001832fd572  test     r8, r8
0x0001832fd575  jns      0x1832fd57b
0x0001832fd577  mov      r9, qword ptr [rax + 8]
0x0001832fd57b  mov      r15d, 0x92
0x0001832fd581  cmp      rdx, r9
0x0001832fd584  ja       0x1832fed9f
0x0001832fd58a  mov      r9, rax
0x0001832fd58d  test     r8b, r8b
0x0001832fd590  jns      0x1832fd595
0x0001832fd592  mov      r9, qword ptr [rax]
0x0001832fd595  mov      r8d, dword ptr [r9 + rcx + 2]
0x0001832fd59a  mov      dword ptr [rbx + 0x5c0], r8d
0x0001832fd5a1  mov      qword ptr [rsp + 0x130], rdx
0x0001832fd5a9  mov      byte ptr [rsp + 0x138], 1
0x0001832fd5b1  lea      rdx, [rcx + 0xa]
0x0001832fd5b5  movsx    r8, byte ptr [rax + 0x17]
0x0001832fd5ba  mov      r9, r8
0x0001832fd5bd  test     r8, r8
0x0001832fd5c0  jns      0x1832fd5c6
0x0001832fd5c2  mov      r9, qword ptr [rax + 8]
0x0001832fd5c6  cmp      rdx, r9
0x0001832fd5c9  ja       0x1832fed9f
0x0001832fd5cf  mov      r9, rax
0x0001832fd5d2  test     r8b, r8b
0x0001832fd5d5  jns      0x1832fd5da
0x0001832fd5d7  mov      r9, qword ptr [rax]
0x0001832fd5da  mov      r8d, dword ptr [r9 + rcx + 6]
0x0001832fd5df  mov      dword ptr [rbx + 0x5bc], r8d
0x0001832fd5e6  mov      qword ptr [rsp + 0x130], rdx
0x0001832fd5ee  mov      byte ptr [rsp + 0x138], 1
0x0001832fd5f6  lea      rdx, [rcx + 0xb]
0x0001832fd5fa  movsx    r8, byte ptr [rax + 0x17]
0x0001832fd5ff  mov      r9, r8
0x0001832fd602  test     r8, r8
0x0001832fd605  jns      0x1832fd60b
0x0001832fd607  mov      r9, qword ptr [rax + 8]
0x0001832fd60b  mov      r15d, 0x9b
0x0001832fd611  cmp      rdx, r9
0x0001832fd614  ja       0x1832fed9f
0x0001832fd61a  mov      r9, rax
0x0001832fd61d  test     r8b, r8b
0x0001832fd620  jns      0x1832fd625
0x0001832fd622  mov      r9, qword ptr [rax]
0x0001832fd625  mov      r8b, byte ptr [r9 + rcx + 0xa]
0x0001832fd62a  mov      byte ptr [rbx + 0x328], r8b
0x0001832fd631  mov      qword ptr [rsp + 0x130], rdx
0x0001832fd639  mov      byte ptr [rsp + 0x138], 1
0x0001832fd641  lea      rdx, [rcx + 0xf]
0x0001832fd645  movsx    r8, byte ptr [rax + 0x17]
0x0001832fd64a  mov      r9, r8
0x0001832fd64d  test     r8, r8
0x0001832fd650  jns      0x1832fd656
0x0001832fd652  mov      r9, qword ptr [rax + 8]
0x0001832fd656  cmp      rdx, r9
0x0001832fd659  ja       0x1832fed9f
0x0001832fd65f  mov      r9, rax
0x0001832fd662  test     r8b, r8b
0x0001832fd665  jns      0x1832fd66a
0x0001832fd667  mov      r9, qword ptr [rax]
0x0001832fd66a  mov      r8d, dword ptr [r9 + rcx + 0xb]
0x0001832fd66f  mov      dword ptr [rbx + 0x334], r8d
0x0001832fd676  mov      qword ptr [rsp + 0x130], rdx
0x0001832fd67e  mov      byte ptr [rsp + 0x138], 1
0x0001832fd686  lea      rdx, [rcx + 0x13]
0x0001832fd68a  movsx    r8, byte ptr [rax + 0x17]
0x0001832fd68f  mov      r9, r8
0x0001832fd692  test     r8, r8
0x0001832fd695  jns      0x1832fd69b
0x0001832fd697  mov      r9, qword ptr [rax + 8]
0x0001832fd69b  cmp      rdx, r9
0x0001832fd69e  ja       0x1832fed9f
0x0001832fd6a4  mov      r9, rax
0x0001832fd6a7  test     r8b, r8b
0x0001832fd6aa  jns      0x1832fd6af
0x0001832fd6ac  mov      r9, qword ptr [rax]
0x0001832fd6af  mov      r8d, dword ptr [r9 + rcx + 0xf]
0x0001832fd6b4  mov      dword ptr [rbx + 0x338], r8d
0x0001832fd6bb  mov      qword ptr [rsp + 0x130], rdx
0x0001832fd6c3  mov      byte ptr [rsp + 0x138], 1
0x0001832fd6cb  lea      rdx, [rcx + 0x17]
0x0001832fd6cf  movsx    r8, byte ptr [rax + 0x17]
0x0001832fd6d4  mov      r9, r8
0x0001832fd6d7  test     r8, r8
0x0001832fd6da  jns      0x1832fd6e0
0x0001832fd6dc  mov      r9, qword ptr [rax + 8]
0x0001832fd6e0  cmp      rdx, r9
0x0001832fd6e3  ja       0x1832fed9f
0x0001832fd6e9  mov      r9, rax
0x0001832fd6ec  test     r8b, r8b
0x0001832fd6ef  jns      0x1832fd6f4
0x0001832fd6f1  mov      r9, qword ptr [rax]
0x0001832fd6f4  mov      r8d, dword ptr [r9 + rcx + 0x13]
0x0001832fd6f9  mov      dword ptr [rbx + 0x32c], r8d
0x0001832fd700  mov      qword ptr [rsp + 0x130], rdx
0x0001832fd708  mov      byte ptr [rsp + 0x138], 1
0x0001832fd710  lea      r8, [rcx + 0x1b]
0x0001832fd714  movsx    rdx, byte ptr [rax + 0x17]
0x0001832fd719  mov      r9, rdx
0x0001832fd71c  test     rdx, rdx
0x0001832fd71f  jns      0x1832fd725
0x0001832fd721  mov      r9, qword ptr [rax + 8]
0x0001832fd725  cmp      r8, r9
0x0001832fd728  ja       0x1832fed9f
0x0001832fd72e  mov      r9, rax
0x0001832fd731  test     dl, dl
0x0001832fd733  jns      0x1832fd738
0x0001832fd735  mov      r9, qword ptr [rax]
0x0001832fd738  mov      edx, dword ptr [r9 + rcx + 0x17]
0x0001832fd73d  mov      dword ptr [rbx + 0x330], edx
0x0001832fd743  mov      qword ptr [rsp + 0x130], r8
0x0001832fd74b  mov      byte ptr [rsp + 0x138], 1
0x0001832fd753  lea      rdx, [rcx + 0x1c]
0x0001832fd757  movsx    r9, byte ptr [rax + 0x17]
0x0001832fd75c  mov      r10, r9
0x0001832fd75f  test     r9, r9
0x0001832fd762  jns      0x1832fd768
0x0001832fd764  mov      r10, qword ptr [rax + 8]
0x0001832fd768  cmp      rdx, r10
0x0001832fd76b  ja       0x1832fd7e4
0x0001832fd76d  mov      r10, rax
0x0001832fd770  test     r9b, r9b
0x0001832fd773  jns      0x1832fd778
0x0001832fd775  mov      r10, qword ptr [rax]
0x0001832fd778  mov      r8b, byte ptr [r10 + r8]
0x0001832fd77c  mov      byte ptr [rbx + 0x500], r8b
0x0001832fd783  mov      qword ptr [rsp + 0x130], rdx
0x0001832fd78b  mov      byte ptr [rsp + 0x138], 1
0x0001832fd793  test     r8b, 1
0x0001832fd797  je       0x1832fd877
0x0001832fd79d  add      rcx, 0x1d
0x0001832fd7a1  movsx    r10, byte ptr [rax + 0x17]
0x0001832fd7a6  mov      r9, r10
0x0001832fd7a9  test     r10, r10
0x0001832fd7ac  jns      0x1832fd7b2
0x0001832fd7ae  mov      r9, qword ptr [rax + 8]
0x0001832fd7b2  lea      r8, [rbx + 0x508]
0x0001832fd7b9  cmp      rcx, r9
0x0001832fd7bc  ja       0x1832fd7d4
0x0001832fd7be  test     r10b, r10b
0x0001832fd7c1  jns      0x1832fd7c6
0x0001832fd7c3  mov      rax, qword ptr [rax]
0x0001832fd7c6  mov      al, byte ptr [rax + rdx]
0x0001832fd7c9  mov      byte ptr [r8], al
0x0001832fd7cc  mov      qword ptr [rsp + 0x130], rcx
0x0001832fd7d4  cmp      rcx, r9
0x0001832fd7d7  setbe    al
0x0001832fd7da  setbe    byte ptr [rsp + 0x138]
0x0001832fd7e2  jmp      0x1832fd802
0x0001832fd7e4  mov      byte ptr [rsp + 0x138], 0
0x0001832fd7ec  cmp      byte ptr [rbx + 0x500], 1
0x0001832fd7f3  jne      0x1832fed87
0x0001832fd7f9  lea      r8, [rbx + 0x508]
0x0001832fd800  xor      eax, eax
0x0001832fd802  cmp      byte ptr [r8], 1
0x0001832fd806  jne      0x1832fd85f
0x0001832fd808  lea      rdx, [rbx + 0x558]
0x0001832fd80f  lea      r14, [rsp + 0x130]
0x0001832fd817  mov      rcx, r14
0x0001832fd81a  call     0x183303050
0x0001832fd81f  lea      rdx, [rbx + 0x570]
0x0001832fd826  mov      rcx, rax
0x0001832fd829  call     0x183303050
0x0001832fd82e  lea      rdx, [rbx + 0x510]
0x0001832fd835  mov      rcx, rax
0x0001832fd838  call     0x183303050
0x0001832fd83d  lea      rdx, [rbx + 0x528]
0x0001832fd844  mov      rcx, rax
0x0001832fd847  call     0x183303050
0x0001832fd84c  lea      rdx, [rbx + 0x540]
0x0001832fd853  mov      rcx, rax
0x0001832fd856  call     0x183303050
0x0001832fd85b  mov      al, byte ptr [r14 + 8]
0x0001832fd85f  test     al, al
0x0001832fd861  je       0x1832fed87
0x0001832fd867  mov      rdx, qword ptr [rsp + 0x130]
0x0001832fd86f  mov      rax, qword ptr [rsp + 0x140]
0x0001832fd877  lea      rcx, [rdx + 1]
0x0001832fd87b  movsx    r8, byte ptr [rax + 0x17]
0x0001832fd880  mov      r9, r8
0x0001832fd883  test     r8, r8
0x0001832fd886  jns      0x1832fd88c
0x0001832fd888  mov      r9, qword ptr [rax + 8]
0x0001832fd88c  mov      r15d, 0xc3
0x0001832fd892  cmp      rcx, r9
0x0001832fd895  ja       0x1832fed9f
0x0001832fd89b  mov      r9, rax
0x0001832fd89e  test     r8b, r8b
0x0001832fd8a1  jns      0x1832fd8a6
0x0001832fd8a3  mov      r9, qword ptr [rax]
0x0001832fd8a6  mov      r8b, byte ptr [r9 + rdx]
0x0001832fd8aa  mov      byte ptr [rbx + 0x598], r8b
0x0001832fd8b1  mov      qword ptr [rsp + 0x130], rcx
0x0001832fd8b9  mov      byte ptr [rsp + 0x138], 1
0x0001832fd8c1  cmp      r8b, 1
0x0001832fd8c5  mov      qword ptr [rsp + 0x30], rbx
0x0001832fd8ca  jne      0x1832fd93e
0x0001832fd8cc  add      rdx, 5
0x0001832fd8d0  movsx    r8, byte ptr [rax + 0x17]
0x0001832fd8d5  mov      r9, r8
0x0001832fd8d8  test     r8, r8
0x0001832fd8db  jns      0x1832fd8e1
0x0001832fd8dd  mov      r9, qword ptr [rax + 8]
0x0001832fd8e1  cmp      rdx, r9
0x0001832fd8e4  ja       0x1832fed9f
0x0001832fd8ea  test     r8b, r8b
0x0001832fd8ed  jns      0x1832fd8f2
0x0001832fd8ef  mov      rax, qword ptr [rax]
0x0001832fd8f2  mov      r15d, dword ptr [rax + rcx]
0x0001832fd8f6  mov      qword ptr [rsp + 0x130], rdx
0x0001832fd8fe  mov      byte ptr [rsp + 0x138], 1
0x0001832fd906  test     r15d, r15d
0x0001832fd909  jle      0x1832fd93e
0x0001832fd90b  lea      r14, [rsp + 0x1c8]
0x0001832fd913  xorps    xmm6, xmm6
0x0001832fd916  lea      r12, [rsp + 0x130]
0x0001832fd91e  lea      r13, [rsp + 0x1c0]
0x0001832fd926  jmp      0x1832ff45c
0x0001832fd92b  cmp      byte ptr [rsp + 0x138], 1
0x0001832fd933  mov      rbp, qword ptr [rsp + 0x28]
0x0001832fd938  jne      0x183300c05
0x0001832fd93e  mov      byte ptr [rbx + 0x90], 1
0x0001832fd945  lea      rdx, [rbx + 0xf8]
0x0001832fd94c  lea      rcx, [rsp + 0x130]
0x0001832fd954  call     0x183303050
0x0001832fd959  lea      rdx, [rbx + 0x140]
0x0001832fd960  mov      rcx, rax
0x0001832fd963  call     0x183303050
0x0001832fd968  lea      rcx, [rbx + 0x70]
0x0001832fd96c  lea      rdx, [rsp + 0x1c0]
0x0001832fd974  call     0x1832df3e0
0x0001832fd979  lea      rsi, [rbx + 0x110]
0x0001832fd980  cmp      byte ptr [rbx + 0x127], 0
0x0001832fd987  js       0x183300c6f
0x0001832fd98d  mov      rax, qword ptr [rsp + 0x1d0]
0x0001832fd995  mov      qword ptr [rsi + 0x10], rax
0x0001832fd999  movups   xmm0, xmmword ptr [rsp + 0x1c0]
0x0001832fd9a1  movups   xmmword ptr [rsi], xmm0
0x0001832fd9a4  lea      rdx, [rbx + 0x98]
0x0001832fd9ab  lea      rcx, [rsp + 0x130]
0x0001832fd9b3  call     0x183303050
0x0001832fd9b8  lea      rdx, [rbx + 0x128]
0x0001832fd9bf  mov      rcx, rax
0x0001832fd9c2  call     0x183303050
0x0001832fd9c7  lea      rdx, [rbx + 0xb0]
0x0001832fd9ce  mov      rcx, rax
0x0001832fd9d1  call     0x183303050
0x0001832fd9d6  lea      rdx, [rbx + 0xc8]
0x0001832fd9dd  mov      rcx, rax
0x0001832fd9e0  call     0x183303050
0x0001832fd9e5  lea      rdx, [rbx + 0x188]
0x0001832fd9ec  mov      rcx, rax
0x0001832fd9ef  call     0x183303050
0x0001832fd9f4  lea      rdx, [rbx + 0x1a0]
0x0001832fd9fb  mov      rcx, rax
0x0001832fd9fe  call     0x183303050
0x0001832fda03  lea      rdx, [rbx + 0x1b8]
0x0001832fda0a  mov      rcx, rax
0x0001832fda0d  call     0x183303050
0x0001832fda12  lea      rdx, [rbx + 0x1d0]
0x0001832fda19  mov      rcx, rax
0x0001832fda1c  call     0x183303050
0x0001832fda21  lea      rdx, [rbx + 0x1e8]
0x0001832fda28  mov      rcx, rax
0x0001832fda2b  call     0x183303050
0x0001832fda30  cmp      byte ptr [rax + 8], 1
0x0001832fda34  jne      0x1832fda73
0x0001832fda36  mov      rdx, qword ptr [rax]
0x0001832fda39  mov      r8, qword ptr [rax + 0x10]
0x0001832fda3d  lea      rcx, [rdx + 1]
0x0001832fda41  movsx    r10, byte ptr [r8 + 0x17]
0x0001832fda46  mov      r9, r10
0x0001832fda49  test     r10, r10
0x0001832fda4c  jns      0x1832fda52
0x0001832fda4e  mov      r9, qword ptr [r8 + 8]
0x0001832fda52  cmp      rcx, r9
0x0001832fda55  ja       0x1832fda6c
0x0001832fda57  test     r10b, r10b
0x0001832fda5a  jns      0x1832fda5f
0x0001832fda5c  mov      r8, qword ptr [r8]
0x0001832fda5f  mov      dl, byte ptr [r8 + rdx]
0x0001832fda63  mov      byte ptr [rbx + 0x200], dl
0x0001832fda69  inc      qword ptr [rax]
0x0001832fda6c  cmp      rcx, r9
0x0001832fda6f  setbe    byte ptr [rax + 8]
0x0001832fda73  lea      rdx, [rbx + 0x208]
0x0001832fda7a  mov      rcx, rax
0x0001832fda7d  call     0x183303050
0x0001832fda82  cmp      byte ptr [rax + 8], 1
0x0001832fda86  jne      0x1832fdac5
0x0001832fda88  mov      rdx, qword ptr [rax]
0x0001832fda8b  mov      r8, qword ptr [rax + 0x10]
0x0001832fda8f  lea      rcx, [rdx + 1]
0x0001832fda93  movsx    r10, byte ptr [r8 + 0x17]
0x0001832fda98  mov      r9, r10
0x0001832fda9b  test     r10, r10
0x0001832fda9e  jns      0x1832fdaa4
0x0001832fdaa0  mov      r9, qword ptr [r8 + 8]
0x0001832fdaa4  cmp      rcx, r9
0x0001832fdaa7  ja       0x1832fdabe
0x0001832fdaa9  test     r10b, r10b
0x0001832fdaac  jns      0x1832fdab1
0x0001832fdaae  mov      r8, qword ptr [r8]
0x0001832fdab1  mov      dl, byte ptr [r8 + rdx]
0x0001832fdab5  mov      byte ptr [rbx + 0x220], dl
0x0001832fdabb  inc      qword ptr [rax]
0x0001832fdabe  cmp      rcx, r9
0x0001832fdac1  setbe    byte ptr [rax + 8]
0x0001832fdac5  lea      rdx, [rbx + 0xe0]
0x0001832fdacc  mov      rcx, rax
0x0001832fdacf  call     0x183303050
0x0001832fdad4  cmp      byte ptr [rsp + 0x138], 1
0x0001832fdadc  jne      0x1832fed8f
0x0001832fdae2  mov      r9, qword ptr [rsp + 0x130]
0x0001832fdaea  mov      rax, qword ptr [rsp + 0x140]
0x0001832fdaf2  lea      r8, [r9 + 4]
0x0001832fdaf6  movsx    rcx, byte ptr [rax + 0x17]
0x0001832fdafb  mov      rdx, rcx
0x0001832fdafe  test     rcx, rcx
0x0001832fdb01  jns      0x1832fdb07
0x0001832fdb03  mov      rdx, qword ptr [rax + 8]
0x0001832fdb07  mov      r15d, 0xf5
0x0001832fdb0d  cmp      r8, rdx
0x0001832fdb10  ja       0x1832fed9f
0x0001832fdb16  mov      edx, ecx
0x0001832fdb18  mov      r10, rax
0x0001832fdb1b  test     cl, cl
0x0001832fdb1d  jns      0x1832fdb22
0x0001832fdb1f  mov      r10, qword ptr [rax]
0x0001832fdb22  mov      esi, dword ptr [r10 + r9]
0x0001832fdb26  mov      qword ptr [rsp + 0x130], r8
0x0001832fdb2e  mov      byte ptr [rsp + 0x138], 1
0x0001832fdb36  test     esi, esi
0x0001832fdb38  jle      0x1832fdbef
0x0001832fdb3e  lea      r14, [rbx + 0x228]
0x0001832fdb45  xorps    xmm6, xmm6
0x0001832fdb48  lea      r12, [rsp + 0x130]
0x0001832fdb50  lea      r13, [rsp + 0x1c0]
0x0001832fdb58  jmp      0x1832fdb76
0x0001832fdb5a  mov      rdx, qword ptr [rsp + 0x1d0]
0x0001832fdb62  and      rdx, rdi
0x0001832fdb65  mov      rcx, qword ptr [rsp + 0x1c0]
0x0001832fdb6d  call     0x183b135e0
0x0001832fdb72  dec      esi
0x0001832fdb74  je       0x1832fdbcb
0x0001832fdb76  movaps   xmmword ptr [rsp + 0x1c0], xmm6
0x0001832fdb7e  mov      qword ptr [rsp + 0x1d0], 0
0x0001832fdb8a  mov      rcx, r12
0x0001832fdb8d  mov      rdx, r13
0x0001832fdb90  call     0x183303050
0x0001832fdb95  movsx    rax, byte ptr [rsp + 0x1d7]
0x0001832fdb9e  mov      rcx, rax
0x0001832fdba1  test     rax, rax
0x0001832fdba4  jns      0x1832fdbae
0x0001832fdba6  mov      rcx, qword ptr [rsp + 0x1c8]
0x0001832fdbae  test     rcx, rcx
0x0001832fdbb1  je       0x1832fdbc5
0x0001832fdbb3  mov      rcx, r14
0x0001832fdbb6  mov      rdx, r13
0x0001832fdbb9  call     0x1843bd7e0
0x0001832fdbbe  mov      al, byte ptr [rsp + 0x1d7]
0x0001832fdbc5  test     al, al
0x0001832fdbc7  jns      0x1832fdb72
0x0001832fdbc9  jmp      0x1832fdb5a
0x0001832fdbcb  cmp      byte ptr [rsp + 0x138], 1
0x0001832fdbd3  jne      0x1832fed8f
0x0001832fdbd9  mov      r8, qword ptr [rsp + 0x130]
0x0001832fdbe1  mov      rax, qword ptr [rsp + 0x140]
0x0001832fdbe9  mov      dl, byte ptr [rax + 0x17]
0x0001832fdbec  movzx    ecx, dl
0x0001832fdbef  lea      r9, [r8 + 4]
0x0001832fdbf3  mov      r10, rcx
0x0001832fdbf6  test     dl, dl
0x0001832fdbf8  jns      0x1832fdbfe
0x0001832fdbfa  mov      r10, qword ptr [rax + 8]
0x0001832fdbfe  cmp      r9, r10
0x0001832fdc01  ja       0x1832fed9f
0x0001832fdc07  mov      r10, rax
0x0001832fdc0a  test     dl, dl
0x0001832fdc0c  jns      0x1832fdc11
0x0001832fdc0e  mov      r10, qword ptr [rax]
0x0001832fdc11  mov      esi, dword ptr [r10 + r8]
0x0001832fdc15  mov      qword ptr [rsp + 0x130], r9
0x0001832fdc1d  mov      byte ptr [rsp + 0x138], 1
0x0001832fdc25  test     esi, esi
0x0001832fdc27  jle      0x1832fdd3f
0x0001832fdc2d  lea      r12, [rsp + 0x1d8]
0x0001832fdc35  lea      r13, [rbx + 0x170]
0x0001832fdc3c  xorps    xmm6, xmm6
0x0001832fdc3f  lea      rbp, [rsp + 0x130]
0x0001832fdc47  lea      r14, [rsp + 0x1c0]
0x0001832fdc4f  jmp      0x1832fdcad
0x0001832fdc51  movzx    ecx, al
0x0001832fdc54  test     rcx, rcx
0x0001832fdc57  je       0x1832fdc6b
0x0001832fdc59  mov      rcx, r13
0x0001832fdc5c  mov      rdx, r14
0x0001832fdc5f  call     0x1833010e0
0x0001832fdc64  mov      al, byte ptr [rsp + 0x1ef]
0x0001832fdc6b  test     al, al
0x0001832fdc6d  jns      0x1832fdc87
0x0001832fdc6f  mov      rdx, qword ptr [rsp + 0x1e8]
0x0001832fdc77  and      rdx, rdi
0x0001832fdc7a  mov      rcx, qword ptr [rsp + 0x1d8]
0x0001832fdc82  call     0x183b135e0
0x0001832fdc87  cmp      byte ptr [rsp + 0x1d7], 0
0x0001832fdc8f  jns      0x1832fdca9
0x0001832fdc91  mov      rdx, qword ptr [rsp + 0x1d0]
0x0001832fdc99  and      rdx, rdi
0x0001832fdc9c  mov      rcx, qword ptr [rsp + 0x1c0]
0x0001832fdca4  call     0x183b135e0
0x0001832fdca9  dec      esi
0x0001832fdcab  je       0x1832fdd16
0x0001832fdcad  movaps   xmmword ptr [rsp + 0x1e0], xmm6
0x0001832fdcb5  movaps   xmmword ptr [rsp + 0x1d0], xmm6
0x0001832fdcbd  movaps   xmmword ptr [rsp + 0x1c0], xmm6
0x0001832fdcc5  mov      rcx, rbp
0x0001832fdcc8  mov      rdx, r14
0x0001832fdccb  call     0x183303050
0x0001832fdcd0  mov      rcx, rax
0x0001832fdcd3  mov      rdx, r12
0x0001832fdcd6  call     0x183303050
0x0001832fdcdb  movsx    rcx, byte ptr [rsp + 0x1d7]
0x0001832fdce4  test     rcx, rcx
0x0001832fdce7  jns      0x1832fdcf1
0x0001832fdce9  mov      rcx, qword ptr [rsp + 0x1c8]
0x0001832fdcf1  mov      al, byte ptr [rsp + 0x1ef]
0x0001832fdcf8  test     rcx, rcx
0x0001832fdcfb  je       0x1832fdc6b
0x0001832fdd01  test     al, al
0x0001832fdd03  jns      0x1832fdc51
0x0001832fdd09  mov      rcx, qword ptr [rsp + 0x1e0]
0x0001832fdd11  jmp      0x1832fdc54
0x0001832fdd16  cmp      byte ptr [rsp + 0x138], 1
0x0001832fdd1e  jne      0x1832fed7a
0x0001832fdd24  mov      r9, qword ptr [rsp + 0x130]
0x0001832fdd2c  mov      rax, qword ptr [rsp + 0x140]
0x0001832fdd34  mov      dl, byte ptr [rax + 0x17]
0x0001832fdd37  movzx    ecx, dl
0x0001832fdd3a  mov      rbp, qword ptr [rsp + 0x28]
0x0001832fdd3f  lea      r8, [r9 + 4]
0x0001832fdd43  mov      r10, rcx
0x0001832fdd46  test     dl, dl
0x0001832fdd48  jns      0x1832fdd4e
0x0001832fdd4a  mov      r10, qword ptr [rax + 8]
0x0001832fdd4e  cmp      r8, r10
0x0001832fdd51  ja       0x1832fed9f
0x0001832fdd57  mov      r10, rax
0x0001832fdd5a  test     dl, dl
0x0001832fdd5c  jns      0x1832fdd61
0x0001832fdd5e  mov      r10, qword ptr [rax]
0x0001832fdd61  mov      esi, dword ptr [r10 + r9]
0x0001832fdd65  mov      qword ptr [rsp + 0x130], r8
0x0001832fdd6d  mov      byte ptr [rsp + 0x138], 1
0x0001832fdd75  test     esi, esi
0x0001832fdd77  jle      0x1832fde8f
0x0001832fdd7d  lea      r12, [rbx + 0x158]
0x0001832fdd84  lea      r13, [rsp + 0x1d8]
0x0001832fdd8c  xorps    xmm6, xmm6
0x0001832fdd8f  lea      rbp, [rsp + 0x130]
0x0001832fdd97  lea      r14, [rsp + 0x1c0]
0x0001832fdd9f  jmp      0x1832fddfd
0x0001832fdda1  movzx    ecx, al
0x0001832fdda4  test     rcx, rcx
0x0001832fdda7  je       0x1832fddbb
0x0001832fdda9  mov      rcx, r12
0x0001832fddac  mov      rdx, r14
0x0001832fddaf  call     0x1833010e0
0x0001832fddb4  mov      al, byte ptr [rsp + 0x1ef]
0x0001832fddbb  test     al, al
0x0001832fddbd  jns      0x1832fddd7
0x0001832fddbf  mov      rdx, qword ptr [rsp + 0x1e8]
0x0001832fddc7  and      rdx, rdi
0x0001832fddca  mov      rcx, qword ptr [rsp + 0x1d8]
0x0001832fddd2  call     0x183b135e0
0x0001832fddd7  cmp      byte ptr [rsp + 0x1d7], 0
0x0001832fdddf  jns      0x1832fddf9
0x0001832fdde1  mov      rdx, qword ptr [rsp + 0x1d0]
0x0001832fdde9  and      rdx, rdi
0x0001832fddec  mov      rcx, qword ptr [rsp + 0x1c0]
0x0001832fddf4  call     0x183b135e0
0x0001832fddf9  dec      esi
0x0001832fddfb  je       0x1832fde66
0x0001832fddfd  movaps   xmmword ptr [rsp + 0x1e0], xmm6
0x0001832fde05  movaps   xmmword ptr [rsp + 0x1d0], xmm6
0x0001832fde0d  movaps   xmmword ptr [rsp + 0x1c0], xmm6
0x0001832fde15  mov      rcx, rbp
0x0001832fde18  mov      rdx, r14
0x0001832fde1b  call     0x183303050
0x0001832fde20  mov      rcx, rax
0x0001832fde23  mov      rdx, r13
0x0001832fde26  call     0x183303050
0x0001832fde2b  movsx    rcx, byte ptr [rsp + 0x1d7]
0x0001832fde34  test     rcx, rcx
0x0001832fde37  jns      0x1832fde41
0x0001832fde39  mov      rcx, qword ptr [rsp + 0x1c8]
0x0001832fde41  mov      al, byte ptr [rsp + 0x1ef]
0x0001832fde48  test     rcx, rcx
0x0001832fde4b  je       0x1832fddbb
0x0001832fde51  test     al, al
0x0001832fde53  jns      0x1832fdda1
0x0001832fde59  mov      rcx, qword ptr [rsp + 0x1e0]
0x0001832fde61  jmp      0x1832fdda4
0x0001832fde66  cmp      byte ptr [rsp + 0x138], 1
0x0001832fde6e  jne      0x1832fed7a
0x0001832fde74  mov      r8, qword ptr [rsp + 0x130]
0x0001832fde7c  mov      rax, qword ptr [rsp + 0x140]
0x0001832fde84  mov      dl, byte ptr [rax + 0x17]
0x0001832fde87  movzx    ecx, dl
0x0001832fde8a  mov      rbp, qword ptr [rsp + 0x28]
0x0001832fde8f  lea      r9, [r8 + 4]
0x0001832fde93  test     dl, dl
0x0001832fde95  jns      0x1832fde9b
0x0001832fde97  mov      rcx, qword ptr [rax + 8]
0x0001832fde9b  mov      esi, 0xffffffff
0x0001832fdea0  cmp      r9, rcx
0x0001832fdea3  ja       0x1832fdeb8
0x0001832fdea5  test     dl, dl
0x0001832fdea7  jns      0x1832fdeac
0x0001832fdea9  mov      rax, qword ptr [rax]
0x0001832fdeac  mov      esi, dword ptr [rax + r8]
0x0001832fdeb0  mov      qword ptr [rsp + 0x130], r9
0x0001832fdeb8  cmp      r9, rcx
0x0001832fdebb  lea      rcx, [rsp + 0x130]
0x0001832fdec3  setbe    byte ptr [rcx + 8]
0x0001832fdec7  lea      rdx, [rbx + 0x2e0]
0x0001832fdece  call     0x183303050
0x0001832fded3  cmp      byte ptr [rax + 8], 1
0x0001832fded7  jne      0x1832fdf5d
0x0001832fdedd  mov      rcx, qword ptr [rax]
0x0001832fdee0  mov      rdx, qword ptr [rax + 0x10]
0x0001832fdee4  lea      r9, [rcx + 8]
0x0001832fdee8  movsx    r8, byte ptr [rdx + 0x17]
0x0001832fdeed  mov      r10, r8
0x0001832fdef0  test     r8, r8
0x0001832fdef3  jns      0x1832fdef9
0x0001832fdef5  mov      r10, qword ptr [rdx + 8]
0x0001832fdef9  cmp      r9, r10
0x0001832fdefc  ja       0x1832fdf58
0x0001832fdefe  test     r8b, r8b
0x0001832fdf01  jns      0x1832fdf06
0x0001832fdf03  mov      rdx, qword ptr [rdx]
0x0001832fdf06  mov      rcx, qword ptr [rdx + rcx]
0x0001832fdf0a  mov      qword ptr [rbx + 0x2f8], rcx
0x0001832fdf11  mov      r8, qword ptr [rax]
0x0001832fdf14  mov      rcx, qword ptr [rax + 0x10]
0x0001832fdf18  lea      rdx, [r8 + 8]
0x0001832fdf1c  mov      qword ptr [rax], rdx
0x0001832fdf1f  mov      byte ptr [rax + 8], 1
0x0001832fdf23  add      r8, 0x10
0x0001832fdf27  movsx    r9, byte ptr [rcx + 0x17]
0x0001832fdf2c  mov      r10, r9
0x0001832fdf2f  test     r9, r9
0x0001832fdf32  jns      0x1832fdf38
0x0001832fdf34  mov      r10, qword ptr [rcx + 8]
0x0001832fdf38  cmp      r8, r10
0x0001832fdf3b  ja       0x1832fdf58
0x0001832fdf3d  test     r9b, r9b
0x0001832fdf40  jns      0x1832fdf45
0x0001832fdf42  mov      rcx, qword ptr [rcx]
0x0001832fdf45  mov      rcx, qword ptr [rcx + rdx]
0x0001832fdf49  mov      qword ptr [rbx + 0x300], rcx
0x0001832fdf50  add      qword ptr [rax], 8
0x0001832fdf54  mov      cl, 1
0x0001832fdf56  jmp      0x1832fdf5a
0x0001832fdf58  xor      ecx, ecx
0x0001832fdf5a  mov      byte ptr [rax + 8], cl
0x0001832fdf5d  cmp      esi, -1
0x0001832fdf60  je       0x1832fdf85
0x0001832fdf62  cmp      esi, 2
0x0001832fdf65  jne      0x1832fdfa9
0x0001832fdf67  movsx    rax, byte ptr [rbx + 0x2f7]
0x0001832fdf6f  test     rax, rax
0x0001832fdf72  jns      0x1832fdf7b
0x0001832fdf74  mov      rax, qword ptr [rbx + 0x2e8]
0x0001832fdf7b  mov      ecx, 2
0x0001832fdf80  test     rax, rax
0x0001832fdf83  jne      0x1832fdf8a
0x0001832fdf85  mov      ecx, 1
0x0001832fdf8a  mov      dword ptr [rbx + 0x2d8], ecx
0x0001832fdf90  mov      byte ptr [rbx + 0x320], 1
0x0001832fdf97  cmp      dword ptr [rbx + 0x50], 1
0x0001832fdf9b  jne      0x1832fdfbd
0x0001832fdf9d  lea      rax, [rbx + 0x70]
0x0001832fdfa1  cmp      dword ptr [rax], 0
0x0001832fdfa4  sete     al
0x0001832fdfa7  jmp      0x1832fdfbf
0x0001832fdfa9  mov      dword ptr [rbx + 0x2d8], esi
0x0001832fdfaf  dec      esi
0x0001832fdfb1  cmp      esi, 2
0x0001832fdfb4  setb     byte ptr [rbx + 0x320]
0x0001832fdfbb  jb       0x1832fdf97
0x0001832fdfbd  xor      eax, eax
0x0001832fdfbf  mov      byte ptr [rbx + 0x321], al
0x0001832fdfc5  cmp      byte ptr [rbx + 0x320], 1
0x0001832fdfcc  jne      0x1832fe142
0x0001832fdfd2  lea      rax, [rsp + 0x1d4]
0x0001832fdfda  mov      byte ptr [rax + 3], 0x14
0x0001832fdfde  lea      rcx, [rip + 0xc8cf933]   ; "stun:stun.durden.dev"
0x0001832fdfe5  lea      rdx, [rsp + 0x1c0]
0x0001832fdfed  cmp      rdx, rcx
0x0001832fdff0  seta     dl
0x0001832fdff3  cmp      rax, rcx
0x0001832fdff6  setbe    al
0x0001832fdff9  or       al, dl
0x0001832fdffb  je       0x1832ff346
0x0001832fe001  movups   xmm0, xmmword ptr [rip + 0xc8cf910]   ; "stun:stun.durden.dev"
0x0001832fe008  lea      rsi, [rsp + 0x1d8]
0x0001832fe010  movaps   xmmword ptr [rsi - 0x18], xmm0
0x0001832fe014  mov      dword ptr [rsi - 8], 0x7665642e
0x0001832fe01b  mov      byte ptr [rsi - 4], 0
0x0001832fe01f  mov      byte ptr [rsi + 0x17], 0xe
0x0001832fe023  lea      rax, [rsp + 0x1e6]
0x0001832fe02b  lea      rcx, [rip + 0xc8cf8fe]   ; "botchecker.net"
0x0001832fe032  cmp      rsi, rcx
0x0001832fe035  seta     dl
0x0001832fe038  cmp      rax, rcx
0x0001832fe03b  setbe    al
0x0001832fe03e  or       al, dl
0x0001832fe040  je       0x1832ff346
0x0001832fe046  movabs   rax, 0x74656e2e72656b63
0x0001832fe050  mov      qword ptr [rsi + 6], rax
0x0001832fe054  movabs   rax, 0x6b63656863746f62
0x0001832fe05e  mov      qword ptr [rsi], rax
0x0001832fe061  lea      rdi, [rsp + 0x1f0]
0x0001832fe069  mov      byte ptr [rdi - 0xa], 0
0x0001832fe06d  mov      byte ptr [rdi + 0x17], 0x11
0x0001832fe071  lea      rax, [rsp + 0x201]
0x0001832fe079  lea      rcx, [rip + 0xc8cf8c0]   ; "use1-turn.fpjs.io"
0x0001832fe080  cmp      rdi, rcx
0x0001832fe083  seta     dl
0x0001832fe086  cmp      rax, rcx
0x0001832fe089  setbe    al
0x0001832fe08c  or       al, dl
0x0001832fe08e  je       0x1832ff346
0x0001832fe094  movups   xmm0, xmmword ptr [rip + 0xc8cf8a5]   ; "use1-turn.fpjs.io"
0x0001832fe09b  movups   xmmword ptr [rdi], xmm0
0x0001832fe09e  mov      byte ptr [rdi + 0x10], 0x6f
0x0001832fe0a2  mov      byte ptr [rsp + 0x201], 0
0x0001832fe0aa  mov      ecx, 0x48
0x0001832fe0af  call     0x18593bfb8
0x0001832fe0b4  test     rax, rax
0x0001832fe0b7  je       0x1832ff346
0x0001832fe0bd  mov      r14, rax
0x0001832fe0c0  mov      rax, qword ptr [rsp + 0x1d0]
0x0001832fe0c8  mov      qword ptr [r14 + 0x10], rax
0x0001832fe0cc  movaps   xmm0, xmmword ptr [rsp + 0x1c0]
0x0001832fe0d4  movups   xmmword ptr [r14], xmm0
0x0001832fe0d8  mov      rax, r14
0x0001832fe0db  add      rax, 0x18
0x0001832fe0df  je       0x1832ff346
0x0001832fe0e5  mov      rcx, qword ptr [rsi + 0x10]
0x0001832fe0e9  mov      qword ptr [rax + 0x10], rcx
0x0001832fe0ed  movups   xmm0, xmmword ptr [rsi]
0x0001832fe0f0  movups   xmmword ptr [rax], xmm0
0x0001832fe0f3  mov      rax, r14
0x0001832fe0f6  add      rax, 0x30
0x0001832fe0fa  je       0x1832ff346
0x0001832fe100  mov      rcx, qword ptr [rdi + 0x10]
0x0001832fe104  mov      qword ptr [rax + 0x10], rcx
0x0001832fe108  movups   xmm0, xmmword ptr [rdi]
0x0001832fe10b  movups   xmmword ptr [rax], xmm0
0x0001832fe10e  mov      rsi, qword ptr [rbx + 0x308]
0x0001832fe115  test     rsi, rsi
0x0001832fe118  jne      0x183300ea3
0x0001832fe11e  mov      qword ptr [rbx + 0x308], r14
0x0001832fe125  mov      eax, 3
0x0001832fe12a  mov      qword ptr [rbx + 0x310], rax
0x0001832fe131  mov      qword ptr [rbx + 0x318], rax
0x0001832fe138  movabs   rdi, 0x7fffffffffffffff
0x0001832fe142  cmp      byte ptr [rsp + 0x138], 1
0x0001832fe14a  jne      0x1832fed72
0x0001832fe150  mov      rcx, qword ptr [rsp + 0x130]
0x0001832fe158  mov      rdx, qword ptr [rsp + 0x140]
0x0001832fe160  lea      rax, [rcx + 4]
0x0001832fe164  movsx    r9, byte ptr [rdx + 0x17]
0x0001832fe169  mov      r8, r9
0x0001832fe16c  test     r9, r9
0x0001832fe16f  jns      0x1832fe175
0x0001832fe171  mov      r8, qword ptr [rdx + 8]
0x0001832fe175  xor      esi, esi
0x0001832fe177  cmp      rax, r8
0x0001832fe17a  ja       0x1832fe18f
0x0001832fe17c  test     r9b, r9b
0x0001832fe17f  jns      0x1832fe184
0x0001832fe181  mov      rdx, qword ptr [rdx]
0x0001832fe184  mov      esi, dword ptr [rdx + rcx]
0x0001832fe187  mov      qword ptr [rsp + 0x130], rax
0x0001832fe18f  cmp      rax, r8
0x0001832fe192  lea      rcx, [rsp + 0x130]
0x0001832fe19a  setbe    byte ptr [rcx + 8]
0x0001832fe19e  lea      rdx, [rbx + 0x660]
0x0001832fe1a5  call     0x183303050
0x0001832fe1aa  cmp      byte ptr [rax + 8], 1
0x0001832fe1ae  jne      0x1832fe1ee
0x0001832fe1b0  mov      rdx, qword ptr [rax]
0x0001832fe1b3  mov      r8, qword ptr [rax + 0x10]
0x0001832fe1b7  lea      rcx, [rdx + 4]
0x0001832fe1bb  movsx    r10, byte ptr [r8 + 0x17]
0x0001832fe1c0  mov      r9, r10
0x0001832fe1c3  test     r10, r10
0x0001832fe1c6  jns      0x1832fe1cc
0x0001832fe1c8  mov      r9, qword ptr [r8 + 8]
0x0001832fe1cc  cmp      rcx, r9
0x0001832fe1cf  ja       0x1832fe1e7
0x0001832fe1d1  test     r10b, r10b
0x0001832fe1d4  jns      0x1832fe1d9
0x0001832fe1d6  mov      r8, qword ptr [r8]
0x0001832fe1d9  mov      edx, dword ptr [r8 + rdx]
0x0001832fe1dd  mov      dword ptr [rbx + 0x678], edx
0x0001832fe1e3  add      qword ptr [rax], 4
0x0001832fe1e7  cmp      rcx, r9
0x0001832fe1ea  setbe    byte ptr [rax + 8]
0x0001832fe1ee  lea      rdx, [rbx + 0x680]
0x0001832fe1f5  mov      rcx, rax
0x0001832fe1f8  call     0x183303050
0x0001832fe1fd  lea      rdx, [rbx + 0x698]
0x0001832fe204  mov      rcx, rax
0x0001832fe207  call     0x183303050
0x0001832fe20c  mov      dword ptr [rbx + 0x65c], esi
0x0001832fe212  movsx    rax, byte ptr [rbx + 0x697]
0x0001832fe21a  test     rax, rax
0x0001832fe21d  jns      0x1832fe226
0x0001832fe21f  mov      rax, qword ptr [rbx + 0x688]
0x0001832fe226  test     rax, rax
0x0001832fe229  je       0x1832ff57d
0x0001832fe22f  movsx    rax, byte ptr [rbx + 0x6af]
0x0001832fe237  test     rax, rax
0x0001832fe23a  jns      0x1832fe243
0x0001832fe23c  mov      rax, qword ptr [rbx + 0x6a0]
0x0001832fe243  test     rax, rax
0x0001832fe246  setne    al
0x0001832fe249  mov      byte ptr [rbx + 0x67c], al
0x0001832fe24f  cmp      byte ptr [rsp + 0x138], 1
0x0001832fe257  jne      0x1832fed6a
0x0001832fe25d  mov      rax, qword ptr [rsp + 0x130]
0x0001832fe265  mov      rcx, qword ptr [rsp + 0x140]
0x0001832fe26d  lea      rdx, [rax + 1]
0x0001832fe271  movsx    r8, byte ptr [rcx + 0x17]
0x0001832fe276  mov      r9, r8
0x0001832fe279  test     r8, r8
0x0001832fe27c  jns      0x1832fe282
0x0001832fe27e  mov      r9, qword ptr [rcx + 8]
0x0001832fe282  mov      r15d, 0x12d
0x0001832fe288  cmp      rdx, r9
0x0001832fe28b  ja       0x1832fed9f
0x0001832fe291  mov      r9, rcx
0x0001832fe294  test     r8b, r8b
0x0001832fe297  jns      0x1832fe29c
0x0001832fe299  mov      r9, qword ptr [rcx]
0x0001832fe29c  mov      r8b, byte ptr [r9 + rax]
0x0001832fe2a0  mov      byte ptr [rbx + 0x5d8], r8b
0x0001832fe2a7  mov      qword ptr [rsp + 0x130], rdx
0x0001832fe2af  mov      byte ptr [rsp + 0x138], 1
0x0001832fe2b7  lea      rdx, [rax + 9]
0x0001832fe2bb  movsx    r8, byte ptr [rcx + 0x17]
0x0001832fe2c0  mov      r9, r8
0x0001832fe2c3  test     r8, r8
0x0001832fe2c6  jns      0x1832fe2cc
0x0001832fe2c8  mov      r9, qword ptr [rcx + 8]
0x0001832fe2cc  cmp      rdx, r9
0x0001832fe2cf  ja       0x1832fed9f
0x0001832fe2d5  mov      r9, rcx
0x0001832fe2d8  test     r8b, r8b
0x0001832fe2db  jns      0x1832fe2e0
0x0001832fe2dd  mov      r9, qword ptr [rcx]
0x0001832fe2e0  mov      r8, qword ptr [r9 + rax + 1]
0x0001832fe2e5  mov      qword ptr [rbx + 0x5e0], r8
0x0001832fe2ec  mov      qword ptr [rsp + 0x130], rdx
0x0001832fe2f4  mov      byte ptr [rsp + 0x138], 1
0x0001832fe2fc  lea      rdx, [rax + 0xa]
0x0001832fe300  movsx    r8, byte ptr [rcx + 0x17]
0x0001832fe305  mov      r9, r8
0x0001832fe308  test     r8, r8
0x0001832fe30b  jns      0x1832fe311
0x0001832fe30d  mov      r9, qword ptr [rcx + 8]
0x0001832fe311  cmp      rdx, r9
0x0001832fe314  ja       0x1832fe378
0x0001832fe316  mov      r9, rcx
0x0001832fe319  test     r8b, r8b
0x0001832fe31c  jns      0x1832fe321
0x0001832fe31e  mov      r9, qword ptr [rcx]
0x0001832fe321  mov      r8b, byte ptr [r9 + rax + 9]
0x0001832fe326  mov      byte ptr [rbx + 0x380], r8b
0x0001832fe32d  mov      qword ptr [rsp + 0x130], rdx
0x0001832fe335  mov      byte ptr [rsp + 0x138], 1
0x0001832fe33d  add      rax, 0xb
0x0001832fe341  movsx    r9, byte ptr [rcx + 0x17]
0x0001832fe346  mov      r8, r9
0x0001832fe349  test     r9, r9
0x0001832fe34c  jns      0x1832fe352
0x0001832fe34e  mov      r8, qword ptr [rcx + 8]
0x0001832fe352  cmp      rax, r8
0x0001832fe355  ja       0x1832fe370
0x0001832fe357  test     r9b, r9b
0x0001832fe35a  jns      0x1832fe35f
0x0001832fe35c  mov      rcx, qword ptr [rcx]
0x0001832fe35f  mov      cl, byte ptr [rcx + rdx]
0x0001832fe362  mov      byte ptr [rbx + 0x381], cl
0x0001832fe368  mov      qword ptr [rsp + 0x130], rax
0x0001832fe370  cmp      rax, r8
0x0001832fe373  setbe    al
0x0001832fe376  jmp      0x1832fe37a
0x0001832fe378  xor      eax, eax
0x0001832fe37a  lea      rcx, [rsp + 0x130]
0x0001832fe382  mov      byte ptr [rcx + 8], al
0x0001832fe385  lea      rdx, [rbx + 0x3a0]
0x0001832fe38c  call     0x183303050
0x0001832fe391  lea      rdx, [rbx + 0x388]
0x0001832fe398  mov      rcx, rax
0x0001832fe39b  call     0x183303050
0x0001832fe3a0  cmp      byte ptr [rax + 8], 1
0x0001832fe3a4  jne      0x1832fe3e4
0x0001832fe3a6  mov      rdx, qword ptr [rax]
0x0001832fe3a9  mov      r8, qword ptr [rax + 0x10]
0x0001832fe3ad  lea      rcx, [rdx + 4]
0x0001832fe3b1  movsx    r10, byte ptr [r8 + 0x17]
0x0001832fe3b6  mov      r9, r10
0x0001832fe3b9  test     r10, r10
0x0001832fe3bc  jns      0x1832fe3c2
0x0001832fe3be  mov      r9, qword ptr [r8 + 8]
0x0001832fe3c2  cmp      rcx, r9
0x0001832fe3c5  ja       0x1832fe3dd
0x0001832fe3c7  test     r10b, r10b
0x0001832fe3ca  jns      0x1832fe3cf
0x0001832fe3cc  mov      r8, qword ptr [r8]
0x0001832fe3cf  mov      edx, dword ptr [r8 + rdx]
0x0001832fe3d3  mov      dword ptr [rbx + 0x488], edx
0x0001832fe3d9  add      qword ptr [rax], 4
0x0001832fe3dd  cmp      rcx, r9
0x0001832fe3e0  setbe    byte ptr [rax + 8]
0x0001832fe3e4  lea      rdx, [rbx + 0x3c0]
0x0001832fe3eb  mov      rcx, rax
0x0001832fe3ee  call     0x183303050
0x0001832fe3f3  lea      rdx, [rbx + 0x3d8]
0x0001832fe3fa  mov      rcx, rax
0x0001832fe3fd  call     0x183303050
0x0001832fe402  lea      rdx, [rbx + 0x3f0]
0x0001832fe409  mov      rcx, rax
0x0001832fe40c  call     0x183303050
0x0001832fe411  lea      rdx, [rbx + 0x408]
0x0001832fe418  mov      rcx, rax
0x0001832fe41b  call     0x183303050
0x0001832fe420  lea      rdx, [rbx + 0x428]
0x0001832fe427  mov      rcx, rax
0x0001832fe42a  call     0x183303050
0x0001832fe42f  lea      rdx, [rbx + 0x440]
0x0001832fe436  mov      rcx, rax
0x0001832fe439  call     0x183303050
0x0001832fe43e  lea      rdx, [rbx + 0x458]
0x0001832fe445  mov      rcx, rax
0x0001832fe448  call     0x183303050
0x0001832fe44d  lea      rdx, [rbx + 0x470]
0x0001832fe454  mov      rcx, rax
0x0001832fe457  call     0x183303050
0x0001832fe45c  cmp      byte ptr [rsp + 0x138], 1
0x0001832fe464  jne      0x1832fed62
0x0001832fe46a  mov      rax, qword ptr [rsp + 0x130]
0x0001832fe472  mov      rcx, qword ptr [rsp + 0x140]
0x0001832fe47a  lea      rdx, [rax + 1]
0x0001832fe47e  movsx    r8, byte ptr [rcx + 0x17]
0x0001832fe483  mov      r9, r8
0x0001832fe486  test     r8, r8
0x0001832fe489  jns      0x1832fe48f
0x0001832fe48b  mov      r9, qword ptr [rcx + 8]
0x0001832fe48f  mov      r15d, 0x147
0x0001832fe495  cmp      rdx, r9
0x0001832fe498  ja       0x1832fed9f
0x0001832fe49e  mov      r9, rcx
0x0001832fe4a1  test     r8b, r8b
0x0001832fe4a4  jns      0x1832fe4a9
0x0001832fe4a6  mov      r9, qword ptr [rcx]
0x0001832fe4a9  mov      r8b, byte ptr [r9 + rax]
0x0001832fe4ad  mov      byte ptr [rbx + 0x4d0], r8b
0x0001832fe4b4  mov      qword ptr [rsp + 0x130], rdx
0x0001832fe4bc  mov      byte ptr [rsp + 0x138], 1
0x0001832fe4c4  lea      rdx, [rax + 5]
0x0001832fe4c8  movsx    r8, byte ptr [rcx + 0x17]
0x0001832fe4cd  mov      r9, r8
0x0001832fe4d0  test     r8, r8
0x0001832fe4d3  jns      0x1832fe4d9
0x0001832fe4d5  mov      r9, qword ptr [rcx + 8]
0x0001832fe4d9  cmp      rdx, r9
0x0001832fe4dc  ja       0x1832fed9f
0x0001832fe4e2  mov      r9, rcx
0x0001832fe4e5  test     r8b, r8b
0x0001832fe4e8  jns      0x1832fe4ed
0x0001832fe4ea  mov      r9, qword ptr [rcx]
0x0001832fe4ed  mov      r8d, dword ptr [r9 + rax + 1]
0x0001832fe4f2  mov      dword ptr [rbx + 0x4d4], r8d
0x0001832fe4f9  mov      qword ptr [rsp + 0x130], rdx
0x0001832fe501  mov      byte ptr [rsp + 0x138], 1
0x0001832fe509  lea      rdx, [rax + 6]
0x0001832fe50d  movsx    r8, byte ptr [rcx + 0x17]
0x0001832fe512  mov      r9, r8
0x0001832fe515  test     r8, r8
0x0001832fe518  jns      0x1832fe51e
0x0001832fe51a  mov      r9, qword ptr [rcx + 8]
0x0001832fe51e  mov      r15d, 0x154
0x0001832fe524  cmp      rdx, r9
0x0001832fe527  ja       0x1832fed9f
0x0001832fe52d  mov      r9, rcx
0x0001832fe530  test     r8b, r8b
0x0001832fe533  jns      0x1832fe538
0x0001832fe535  mov      r9, qword ptr [rcx]
0x0001832fe538  mov      r8b, byte ptr [r9 + rax + 5]
0x0001832fe53d  mov      byte ptr [rbx + 0x4d8], r8b
0x0001832fe544  mov      qword ptr [rsp + 0x130], rdx
0x0001832fe54c  mov      byte ptr [rsp + 0x138], 1
0x0001832fe554  lea      r8, [rax + 7]
0x0001832fe558  movsx    rdx, byte ptr [rcx + 0x17]
0x0001832fe55d  mov      r9, rdx
0x0001832fe560  test     rdx, rdx
0x0001832fe563  jns      0x1832fe569
0x0001832fe565  mov      r9, qword ptr [rcx + 8]
0x0001832fe569  cmp      r8, r9
0x0001832fe56c  ja       0x1832fed9f
0x0001832fe572  mov      r9, rcx
0x0001832fe575  test     dl, dl
0x0001832fe577  jns      0x1832fe57c
0x0001832fe579  mov      r9, qword ptr [rcx]
0x0001832fe57c  mov      dl, byte ptr [r9 + rax + 6]
0x0001832fe581  mov      byte ptr [rbx + 0x4d9], dl
0x0001832fe587  mov      qword ptr [rsp + 0x130], r8
0x0001832fe58f  mov      byte ptr [rsp + 0x138], 1
0x0001832fe597  add      rax, 0xb
0x0001832fe59b  movsx    rdx, byte ptr [rcx + 0x17]
0x0001832fe5a0  mov      r9, rdx
0x0001832fe5a3  test     rdx, rdx
0x0001832fe5a6  jns      0x1832fe5ac
0x0001832fe5a8  mov      r9, qword ptr [rcx + 8]
0x0001832fe5ac  cmp      rax, r9
0x0001832fe5af  ja       0x1832fed9f
0x0001832fe5b5  mov      r9d, edx
0x0001832fe5b8  mov      r10, rcx
0x0001832fe5bb  test     dl, dl
0x0001832fe5bd  jns      0x1832fe5c2
0x0001832fe5bf  mov      r10, qword ptr [rcx]
0x0001832fe5c2  mov      esi, dword ptr [r10 + r8]
0x0001832fe5c6  mov      qword ptr [rsp + 0x130], rax
0x0001832fe5ce  mov      byte ptr [rsp + 0x138], 1
0x0001832fe5d6  test     esi, esi
0x0001832fe5d8  jle      0x1832fe70f
0x0001832fe5de  lea      r15, [rbx + 0x4e0]
0x0001832fe5e5  xorps    xmm6, xmm6
0x0001832fe5e8  lea      r14, [rsp + 0x130]
0x0001832fe5f0  lea      r12, [rsp + 0x1c0]
0x0001832fe5f8  movabs   rdi, 0xaaaaaaaaaaaaaaab
0x0001832fe602  jmp      0x1832fe66c
0x0001832fe604  mov      rax, qword ptr [rsp + 0x1d0]
0x0001832fe60c  mov      qword ptr [r13 + 0x10], rax
0x0001832fe610  movaps   xmm0, xmmword ptr [rsp + 0x1c0]
0x0001832fe618  movups   xmmword ptr [r13], xmm0
0x0001832fe61d  add      r13, 0x18
0x0001832fe621  mov      rax, qword ptr [rbx + 0x4e0]
0x0001832fe628  mov      rcx, r13
0x0001832fe62b  sub      rcx, rax
0x0001832fe62e  sar      rcx, 3
0x0001832fe632  imul     rcx, rdi
0x0001832fe636  mov      qword ptr [rbx + 0x4e8], rcx
0x0001832fe63d  cmp      r13, rax
0x0001832fe640  je       0x1832ff346
0x0001832fe646  cmp      byte ptr [rsp + 0x1d7], 0
0x0001832fe64e  jns      0x1832fe668
0x0001832fe650  mov      rcx, qword ptr [rsp + 0x1c0]
0x0001832fe658  mov      rdx, qword ptr [rsp + 0x1d0]
0x0001832fe660  add      rdx, rdx
0x0001832fe663  call     0x183b135e0
0x0001832fe668  dec      esi
0x0001832fe66a  je       0x1832fe6df
0x0001832fe66c  movaps   xmmword ptr [rsp + 0x1c0], xmm6
0x0001832fe674  mov      qword ptr [rsp + 0x1d0], 0
0x0001832fe680  mov      rcx, r14
0x0001832fe683  mov      rdx, r12
0x0001832fe686  call     0x1833030f0
0x0001832fe68b  mov      rax, qword ptr [rbx + 0x4e8]
0x0001832fe692  cmp      rax, qword ptr [rbx + 0x4f0]
0x0001832fe699  je       0x183300cf6
0x0001832fe69f  lea      r13, [rax + rax*2]
0x0001832fe6a3  shl      r13, 3
0x0001832fe6a7  add      r13, qword ptr [rbx + 0x4e0]
0x0001832fe6ae  je       0x1832ff346
0x0001832fe6b4  cmp      byte ptr [rsp + 0x1d7], 0
0x0001832fe6bc  jns      0x1832fe604
0x0001832fe6c2  mov      rdx, qword ptr [rsp + 0x1c0]
0x0001832fe6ca  mov      r8, qword ptr [rsp + 0x1c8]
0x0001832fe6d2  mov      rcx, r13
0x0001832fe6d5  call     0x183b47230
0x0001832fe6da  jmp      0x1832fe61d
0x0001832fe6df  cmp      byte ptr [rsp + 0x138], 1
0x0001832fe6e7  jne      0x183300b5e
0x0001832fe6ed  mov      rax, qword ptr [rsp + 0x130]
0x0001832fe6f5  mov      rcx, qword ptr [rsp + 0x140]
0x0001832fe6fd  mov      r9b, byte ptr [rcx + 0x17]
0x0001832fe701  movzx    edx, r9b
0x0001832fe705  movabs   rdi, 0x7fffffffffffffff
0x0001832fe70f  lea      r8, [rax + 1]
0x0001832fe713  test     r9b, r9b
0x0001832fe716  jns      0x1832fe71c
0x0001832fe718  mov      rdx, qword ptr [rcx + 8]
0x0001832fe71c  cmp      r8, rdx
0x0001832fe71f  ja       0x1832fe73a
0x0001832fe721  test     r9b, r9b
0x0001832fe724  jns      0x1832fe729
0x0001832fe726  mov      rcx, qword ptr [rcx]
0x0001832fe729  mov      al, byte ptr [rcx + rax]
0x0001832fe72c  mov      byte ptr [rbx + 0x340], al
0x0001832fe732  mov      qword ptr [rsp + 0x130], r8
0x0001832fe73a  cmp      r8, rdx
0x0001832fe73d  lea      rcx, [rsp + 0x130]
0x0001832fe745  setbe    byte ptr [rcx + 8]
0x0001832fe749  lea      rdx, [rbx + 0x348]
0x0001832fe750  call     0x183303050
0x0001832fe755  movsx    rax, byte ptr [rbx + 0x35f]
0x0001832fe75d  test     rax, rax
0x0001832fe760  jns      0x1832fe769
0x0001832fe762  mov      rax, qword ptr [rbx + 0x350]
0x0001832fe769  test     rax, rax
0x0001832fe76c  jne      0x1832fe775
0x0001832fe76e  mov      byte ptr [rbx + 0x340], 0
0x0001832fe775  cmp      byte ptr [rsp + 0x138], 1
0x0001832fe77d  jne      0x1832fed41
0x0001832fe783  mov      rcx, qword ptr [rsp + 0x130]
0x0001832fe78b  mov      rdx, qword ptr [rsp + 0x140]
0x0001832fe793  lea      rax, [rcx + 1]
0x0001832fe797  movsx    r9, byte ptr [rdx + 0x17]
0x0001832fe79c  mov      r8, r9
0x0001832fe79f  test     r9, r9
0x0001832fe7a2  jns      0x1832fe7a8
0x0001832fe7a4  mov      r8, qword ptr [rdx + 8]
0x0001832fe7a8  cmp      rax, r8
0x0001832fe7ab  ja       0x1832fe7c6
0x0001832fe7ad  test     r9b, r9b
0x0001832fe7b0  jns      0x1832fe7b5
0x0001832fe7b2  mov      rdx, qword ptr [rdx]
0x0001832fe7b5  mov      cl, byte ptr [rdx + rcx]
0x0001832fe7b8  mov      byte ptr [rbx + 0x288], cl
0x0001832fe7be  mov      qword ptr [rsp + 0x130], rax
0x0001832fe7c6  cmp      rax, r8
0x0001832fe7c9  lea      rcx, [rsp + 0x130]
0x0001832fe7d1  setbe    byte ptr [rcx + 8]
0x0001832fe7d5  lea      r14, [rbx + 0x290]
0x0001832fe7dc  mov      rdx, r14
0x0001832fe7df  call     0x183303050
0x0001832fe7e4  lea      r15, [rbx + 0x2a8]
0x0001832fe7eb  mov      rcx, rax
0x0001832fe7ee  mov      rdx, r15
0x0001832fe7f1  call     0x183303050
0x0001832fe7f6  xorps    xmm6, xmm6
0x0001832fe7f9  movaps   xmmword ptr [rsp + 0x1c0], xmm6
0x0001832fe801  mov      qword ptr [rsp + 0x1d0], 0
0x0001832fe80d  movsx    rax, byte ptr [rbx + 0x2a7]
0x0001832fe815  test     rax, rax
0x0001832fe818  js       0x183300a6f
0x0001832fe81e  mov      rcx, qword ptr [r14 + 0x10]
0x0001832fe822  mov      qword ptr [rsp + 0x1d0], rcx
0x0001832fe82a  movups   xmm0, xmmword ptr [r14]
0x0001832fe82e  movaps   xmmword ptr [rsp + 0x1c0], xmm0
0x0001832fe836  movaps   xmmword ptr [rsp + 0x1a0], xmm6
0x0001832fe83e  mov      qword ptr [rsp + 0x1b0], 0
0x0001832fe84a  lea      rsi, [rip + 0xc77d0b7]   ; 0x18fa7b908
0x0001832fe851  lea      r12, [rsp + 0x180]
0x0001832fe859  mov      qword ptr [r12], rsi
0x0001832fe85d  mov      qword ptr [r12 + 8], 1
0x0001832fe866  lea      rdx, [rsp + 0x168]
0x0001832fe86e  mov      qword ptr [rdx], r14
0x0001832fe871  mov      qword ptr [rdx + 8], rax
0x0001832fe875  mov      dword ptr [rsp + 0x20], 1
0x0001832fe87d  lea      r14, [rsp + 0x1a0]
0x0001832fe885  mov      rcx, r14
0x0001832fe888  mov      r8, r12
0x0001832fe88b  mov      r9d, 1
0x0001832fe891  call     0x1846a18a0
0x0001832fe896  cmp      qword ptr [r14 + 8], 2
0x0001832fe89b  jb       0x1832fe908
0x0001832fe89d  mov      r8, qword ptr [rsp + 0x1a0]
0x0001832fe8a5  lea      rdx, [rip + 0xc77d120]   ; 0x18fa7b9cc
0x0001832fe8ac  lea      r14, [rsp + 0x180]
0x0001832fe8b4  mov      rcx, r14
0x0001832fe8b7  call     0x180f61850
0x0001832fe8bc  movsx    r8, byte ptr [r14 + 0x17]
0x0001832fe8c1  test     r8, r8
0x0001832fe8c4  jns      0x1832fe8d6
0x0001832fe8c6  mov      r8, qword ptr [rsp + 0x188]
0x0001832fe8ce  mov      r14, qword ptr [rsp + 0x180]
0x0001832fe8d6  lea      rcx, [rsp + 0x1c0]
0x0001832fe8de  mov      rdx, r14
0x0001832fe8e1  call     0x183b596f0
0x0001832fe8e6  cmp      byte ptr [rsp + 0x197], 0
0x0001832fe8ee  jns      0x1832fe908
0x0001832fe8f0  mov      rdx, qword ptr [rsp + 0x190]
0x0001832fe8f8  and      rdx, rdi
0x0001832fe8fb  mov      rcx, qword ptr [rsp + 0x180]
0x0001832fe903  call     0x183b135e0
0x0001832fe908  movsx    rax, byte ptr [rbx + 0x2bf]
0x0001832fe910  test     rax, rax
0x0001832fe913  jns      0x1832fe91c
0x0001832fe915  mov      rax, qword ptr [rbx + 0x2b0]
0x0001832fe91c  test     rax, rax
0x0001832fe91f  je       0x1832fea7b
0x0001832fe925  lea      rdx, [rip + 0xc77d0a0]   ; 0x18fa7b9cc
0x0001832fe92c  mov      rcx, r12
0x0001832fe92f  mov      r8, r15
0x0001832fe932  call     0x180f61850
0x0001832fe937  movsx    r8, byte ptr [rsp + 0x197]
0x0001832fe940  test     r8, r8
0x0001832fe943  jns      0x1832fe955
0x0001832fe945  mov      r8, qword ptr [rsp + 0x188]
0x0001832fe94d  mov      r12, qword ptr [rsp + 0x180]
0x0001832fe955  lea      rcx, [rsp + 0x1c0]
0x0001832fe95d  mov      rdx, r12
0x0001832fe960  call     0x183b596f0
0x0001832fe965  cmp      byte ptr [rsp + 0x197], 0
0x0001832fe96d  jns      0x1832fe987
0x0001832fe96f  mov      rdx, qword ptr [rsp + 0x190]
0x0001832fe977  and      rdx, rdi
0x0001832fe97a  mov      rcx, qword ptr [rsp + 0x180]
0x0001832fe982  call     0x183b135e0
0x0001832fe987  xorps    xmm0, xmm0
0x0001832fe98a  movaps   xmmword ptr [rsp + 0x180], xmm0
0x0001832fe992  mov      qword ptr [rsp + 0x190], 0
0x0001832fe99e  movsx    rax, byte ptr [rbx + 0x2bf]
0x0001832fe9a6  test     rax, rax
0x0001832fe9a9  jns      0x1832fe9b9
0x0001832fe9ab  mov      r15, qword ptr [rbx + 0x2a8]
0x0001832fe9b2  mov      rax, qword ptr [rbx + 0x2b0]
0x0001832fe9b9  lea      r14, [rsp + 0x168]
0x0001832fe9c1  mov      qword ptr [r14], rsi
0x0001832fe9c4  mov      qword ptr [r14 + 8], 1
0x0001832fe9cc  lea      rdx, [rsp + 0x110]
0x0001832fe9d4  mov      qword ptr [rdx], r15
0x0001832fe9d7  mov      qword ptr [rdx + 8], rax
0x0001832fe9db  mov      dword ptr [rsp + 0x20], 1
0x0001832fe9e3  lea      r15, [rsp + 0x180]
0x0001832fe9eb  mov      rcx, r15
0x0001832fe9ee  mov      r8, r14
0x0001832fe9f1  mov      r9d, 1
0x0001832fe9f7  call     0x1846a18a0
0x0001832fe9fc  cmp      qword ptr [r15 + 8], 2
0x0001832fea01  jb       0x1832fea6a
0x0001832fea03  mov      r8, qword ptr [rsp + 0x180]
0x0001832fea0b  lea      rdx, [rip + 0xc77cfba]   ; 0x18fa7b9cc
0x0001832fea12  mov      rcx, r14
0x0001832fea15  call     0x180f61850
0x0001832fea1a  movsx    r8, byte ptr [rsp + 0x17f]
0x0001832fea23  test     r8, r8
0x0001832fea26  jns      0x1832fea38
0x0001832fea28  mov      r8, qword ptr [rsp + 0x170]
0x0001832fea30  mov      r14, qword ptr [rsp + 0x168]
0x0001832fea38  lea      rcx, [rsp + 0x1c0]
0x0001832fea40  mov      rdx, r14
0x0001832fea43  call     0x183b596f0
0x0001832fea48  cmp      byte ptr [rsp + 0x17f], 0
0x0001832fea50  jns      0x1832fea6a
0x0001832fea52  mov      rdx, qword ptr [rsp + 0x178]
0x0001832fea5a  and      rdx, rdi
0x0001832fea5d  mov      rcx, qword ptr [rsp + 0x168]
0x0001832fea65  call     0x183b135e0
0x0001832fea6a  mov      rsi, qword ptr [rsp + 0x180]
0x0001832fea72  test     rsi, rsi
0x0001832fea75  jne      0x183300b86
0x0001832fea7b  lea      rcx, [rbx + 0x2c0]
0x0001832fea82  cmp      byte ptr [rbx + 0x2d7], 0
0x0001832fea89  movzx    r8d, byte ptr [rsp + 0x1d7]
0x0001832fea92  js       0x183300d20
0x0001832fea98  test     r8b, r8b
0x0001832fea9b  js       0x1832fed0d
0x0001832feaa1  mov      rax, qword ptr [rsp + 0x1d0]
0x0001832feaa9  mov      qword ptr [rcx + 0x10], rax
0x0001832feaad  movaps   xmm0, xmmword ptr [rsp + 0x1c0]
0x0001832feab5  movups   xmmword ptr [rcx], xmm0
0x0001832feab8  mov      rsi, qword ptr [rsp + 0x1a0]
0x0001832feac0  test     rsi, rsi
0x0001832feac3  jne      0x183300adf
0x0001832feac9  cmp      byte ptr [rsp + 0x1d7], 0
0x0001832fead1  jns      0x1832feaeb
0x0001832fead3  mov      rdx, qword ptr [rsp + 0x1d0]
0x0001832feadb  and      rdx, rdi
0x0001832feade  mov      rcx, qword ptr [rsp + 0x1c0]
0x0001832feae6  call     0x183b135e0
0x0001832feaeb  cmp      byte ptr [rsp + 0x138], 1
0x0001832feaf3  jne      0x1832fed39
0x0001832feaf9  mov      r9, qword ptr [rsp + 0x130]
0x0001832feb01  mov      rax, qword ptr [rsp + 0x140]
0x0001832feb09  lea      rcx, [r9 + 1]
0x0001832feb0d  movsx    rdx, byte ptr [rax + 0x17]
0x0001832feb12  mov      r8, rdx
0x0001832feb15  test     rdx, rdx
0x0001832feb18  jns      0x1832feb1e
0x0001832feb1a  mov      r8, qword ptr [rax + 8]
0x0001832feb1e  mov      r15d, 0x182
0x0001832feb24  cmp      rcx, r8
0x0001832feb27  ja       0x1832fed9f
0x0001832feb2d  mov      r8, rax
0x0001832feb30  test     dl, dl
0x0001832feb32  jns      0x1832feb37
0x0001832feb34  mov      r8, qword ptr [rax]
0x0001832feb37  mov      dl, byte ptr [r8 + r9]
0x0001832feb3b  mov      byte ptr [rbx + 0x490], dl
0x0001832feb41  mov      qword ptr [rsp + 0x130], rcx
0x0001832feb49  mov      byte ptr [rsp + 0x138], 1
0x0001832feb51  add      r9, 5
0x0001832feb55  movsx    rdx, byte ptr [rax + 0x17]
0x0001832feb5a  mov      r8, rdx
0x0001832feb5d  test     rdx, rdx
0x0001832feb60  jns      0x1832feb66
0x0001832feb62  mov      r8, qword ptr [rax + 8]
0x0001832feb66  cmp      r9, r8
0x0001832feb69  ja       0x1832fed9f
0x0001832feb6f  mov      r8d, edx
0x0001832feb72  mov      r10, rax
0x0001832feb75  test     dl, dl
0x0001832feb77  jns      0x1832feb7c
0x0001832feb79  mov      r10, qword ptr [rax]
0x0001832feb7c  mov      edi, dword ptr [r10 + rcx]
0x0001832feb80  mov      qword ptr [rsp + 0x130], r9
0x0001832feb88  mov      byte ptr [rsp + 0x138], 1
0x0001832feb90  test     edi, edi
0x0001832feb92  jle      0x1832ffd02
0x0001832feb98  lea      r14, [rsp + 0x1d8]
0x0001832feba0  lea      r15, [rsp + 0x1f0]
0x0001832feba8  xorps    xmm6, xmm6
0x0001832febab  lea      r13, [rsp + 0x130]
0x0001832febb3  lea      rbp, [rsp + 0x1c0]
0x0001832febbb  movabs   rsi, 0xcccccccccccccccd
0x0001832febc5  jmp      0x1832fec36
0x0001832febc7  xor      ecx, ecx
0x0001832febc9  mov      byte ptr [rax + 8], cl
0x0001832febcc  mov      rax, qword ptr [rbx + 0x4a0]
0x0001832febd3  cmp      rax, qword ptr [rbx + 0x4a8]
0x0001832febda  je       0x183300d09
0x0001832febe0  lea      rcx, [rax + rax*4]
0x0001832febe4  shl      rcx, 4
0x0001832febe8  add      rcx, qword ptr [rbx + 0x498]
0x0001832febef  je       0x1832ff346
0x0001832febf5  lea      r12, [rcx + 0x50]
0x0001832febf9  mov      rdx, rbp
0x0001832febfc  call     0x1832e3870
0x0001832fec01  mov      rax, qword ptr [rbx + 0x498]
0x0001832fec08  mov      rcx, r12
0x0001832fec0b  sub      rcx, rax
0x0001832fec0e  sar      rcx, 4
0x0001832fec12  imul     rcx, rsi
0x0001832fec16  mov      qword ptr [rbx + 0x4a0], rcx
0x0001832fec1d  cmp      r12, rax
0x0001832fec20  je       0x1832ff346
0x0001832fec26  mov      rcx, rbp
0x0001832fec29  call     0x1832e3810
0x0001832fec2e  dec      edi
0x0001832fec30  je       0x1832ffcd7
0x0001832fec36  movaps   xmmword ptr [rsp + 0x200], xmm6
0x0001832fec3e  movaps   xmmword ptr [rsp + 0x1f0], xmm6
0x0001832fec46  movaps   xmmword ptr [rsp + 0x1e0], xmm6
0x0001832fec4e  movaps   xmmword ptr [rsp + 0x1d0], xmm6
0x0001832fec56  movaps   xmmword ptr [rsp + 0x1c0], xmm6
0x0001832fec5e  mov      rcx, r13
0x0001832fec61  mov      rdx, rbp
0x0001832fec64  call     0x1833030f0
0x0001832fec69  mov      rcx, rax
0x0001832fec6c  mov      rdx, r14
0x0001832fec6f  call     0x1833030f0
0x0001832fec74  mov      rcx, rax
0x0001832fec77  mov      rdx, r15
0x0001832fec7a  call     0x183303050
0x0001832fec7f  cmp      byte ptr [rax + 8], 1
0x0001832fec83  jne      0x1832febcc
0x0001832fec89  mov      rcx, qword ptr [rax + 0x10]
0x0001832fec8d  movsx    r8, byte ptr [rcx + 0x17]
0x0001832fec92  mov      r9, r8
0x0001832fec95  test     r8, r8
0x0001832fec98  jns      0x1832fec9e
0x0001832fec9a  mov      r9, qword ptr [rcx + 8]
0x0001832fec9e  mov      rdx, qword ptr [rax]
0x0001832feca1  lea      r10, [rdx + 1]
0x0001832feca5  cmp      r10, r9
0x0001832feca8  ja       0x1832febc7
0x0001832fecae  test     r8b, r8b
0x0001832fecb1  jns      0x1832fecb6
0x0001832fecb3  mov      rcx, qword ptr [rcx]
0x0001832fecb6  mov      cl, byte ptr [rcx + rdx]
0x0001832fecb9  mov      byte ptr [rsp + 0x208], cl
0x0001832fecc0  mov      rcx, qword ptr [rax]
0x0001832fecc3  mov      rdx, qword ptr [rax + 0x10]
0x0001832fecc7  lea      r8, [rcx + 1]
0x0001832feccb  mov      qword ptr [rax], r8
0x0001832fecce  mov      byte ptr [rax + 8], 1
0x0001832fecd2  movsx    r10, byte ptr [rdx + 0x17]
0x0001832fecd7  mov      r9, r10
0x0001832fecda  test     r10, r10
0x0001832fecdd  jns      0x1832fece3
0x0001832fecdf  mov      r9, qword ptr [rdx + 8]
0x0001832fece3  add      rcx, 2
0x0001832fece7  cmp      rcx, r9
0x0001832fecea  ja       0x1832fed02
0x0001832fecec  test     r10b, r10b
0x0001832fecef  jns      0x1832fecf4
0x0001832fecf1  mov      rdx, qword ptr [rdx]
0x0001832fecf4  mov      dl, byte ptr [rdx + r8]
0x0001832fecf8  mov      byte ptr [rsp + 0x209], dl
0x0001832fecff  inc      qword ptr [rax]
0x0001832fed02  cmp      rcx, r9
0x0001832fed05  setbe    cl
0x0001832fed08  jmp      0x1832febc9
0x0001832fed0d  mov      rdx, qword ptr [rsp + 0x1c0]
0x0001832fed15  mov      r8, qword ptr [rsp + 0x1c8]
0x0001832fed1d  call     0x183b61e40
0x0001832fed22  jmp      0x1832feab8
0x0001832fed27  mov      r15d, 0x1b4
0x0001832fed2d  movabs   rdi, 0x7fffffffffffffff
0x0001832fed37  jmp      0x1832feda7
0x0001832fed39  mov      r15d, 0x172
0x0001832fed3f  jmp      0x1832feda7
0x0001832fed41  mov      r15d, 0x15d
0x0001832fed47  jmp      0x1832feda7
0x0001832fed49  mov      byte ptr [rsp + 0x138], 0
0x0001832fed51  mov      al, byte ptr [rbp + 0x54]
0x0001832fed54  mov      byte ptr [rbx + 0x6cf], al
0x0001832fed5a  mov      r15d, 0x1e2
0x0001832fed60  jmp      0x1832feda7
0x0001832fed62  mov      r15d, 0x141
0x0001832fed68  jmp      0x1832feda7
0x0001832fed6a  mov      r15d, 0x127
0x0001832fed70  jmp      0x1832feda7
0x0001832fed72  mov      r15d, 0x111
0x0001832fed78  jmp      0x1832feda7
0x0001832fed7a  mov      r15d, 0xf5
0x0001832fed80  mov      rbp, qword ptr [rsp + 0x28]
0x0001832fed85  jmp      0x1832feda7
0x0001832fed87  mov      r15d, 0xab
0x0001832fed8d  jmp      0x1832feda7
0x0001832fed8f  mov      r15d, 0xf5
0x0001832fed95  jmp      0x1832feda7
0x0001832fed97  mov      r15d, 0x85
0x0001832fed9d  jmp      0x1832feda7
0x0001832fed9f  mov      byte ptr [rsp + 0x138], 0
0x0001832feda7  mov      dword ptr [rbp], r15d
0x0001832fedab  lea      r14, [rsp + 0x168]
0x0001832fedb3  mov      rcx, r14
0x0001832fedb6  mov      edx, 0x21f
0x0001832fedbb  call     0x184ec6ea0
0x0001832fedc0  mov      ecx, 0x40
0x0001832fedc5  call     0x18593bfb8
0x0001832fedca  mov      qword ptr [rsp + 0xd0], rax
0x0001832fedd2  mov      qword ptr [rsp + 0xd8], 0x30
0x0001832fedde  lea      rcx, [rdi + 0x41]
0x0001832fede2  mov      qword ptr [rsp + 0xe0], rcx
0x0001832fedea  lea      rcx, [rax + 0x30]
0x0001832fedee  cmp      rcx, rax
0x0001832fedf1  jb       0x1832ff346
0x0001832fedf7  lea      rdx, [rip + 0xc8cea62]   ; "..\..\undetectable\core\stors\profilestorage.cpp"
0x0001832fedfe  cmp      rax, rdx
0x0001832fee01  seta     r8b
0x0001832fee05  cmp      rcx, rdx
0x0001832fee08  setbe    cl
0x0001832fee0b  or       cl, r8b
0x0001832fee0e  je       0x1832ff346
0x0001832fee14  movups   xmm0, xmmword ptr [rip + 0xc8cea65]   ; "ofilestorage.cpp"
0x0001832fee1b  movups   xmmword ptr [rax + 0x20], xmm0
0x0001832fee1f  movups   xmm0, xmmword ptr [rip + 0xc8cea4a]   ; "le\core\stors\profilestorage.cpp"
0x0001832fee26  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832fee2a  movups   xmm0, xmmword ptr [rip + 0xc8cea2f]   ; "..\..\undetectable\core\stors\profilestorage.cpp"
0x0001832fee31  movups   xmmword ptr [rax], xmm0
0x0001832fee34  mov      byte ptr [rax + 0x30], 0
0x0001832fee38  mov      ecx, 0x30
0x0001832fee3d  call     0x18593bfb8
0x0001832fee42  mov      qword ptr [rsp + 0x70], rax
0x0001832fee47  mov      qword ptr [rsp + 0x78], 0x29
0x0001832fee50  lea      rcx, [rdi + 0x31]
0x0001832fee54  mov      qword ptr [rsp + 0x80], rcx
0x0001832fee5c  lea      rcx, [rax + 0x29]
0x0001832fee60  cmp      rcx, rax
0x0001832fee63  jb       0x1832ff346
0x0001832fee69  lea      rdx, [rip + 0xc8ceae8]   ; "undetectable::ProfileStorage::readProfile"
0x0001832fee70  cmp      rax, rdx
0x0001832fee73  seta     r8b
0x0001832fee77  cmp      rcx, rdx
0x0001832fee7a  setbe    cl
0x0001832fee7d  or       cl, r8b
0x0001832fee80  je       0x1832ff346
0x0001832fee86  movups   xmm0, xmmword ptr [rip + 0xc8ceae4]   ; "age::readProfile"
0x0001832fee8d  movups   xmmword ptr [rax + 0x19], xmm0
0x0001832fee91  movups   xmm0, xmmword ptr [rip + 0xc8cead0]   ; "ofileStorage::readProfile"
0x0001832fee98  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832fee9c  movups   xmm0, xmmword ptr [rip + 0xc8ceab5]   ; "undetectable::ProfileStorage::readProfile"
0x0001832feea3  movups   xmmword ptr [rax], xmm0
0x0001832feea6  mov      byte ptr [rax + 0x29], 0
0x0001832feeaa  lea      r15, [rsp + 0x70]
0x0001832feeaf  lea      rbx, [rsp + 0x1c0]
0x0001832feeb7  mov      rcx, r15
0x0001832feeba  mov      rdx, rbx
0x0001832feebd  mov      r8, rax
0x0001832feec0  mov      r9b, 0x5b
0x0001832feec3  call     0x18a58b3f0
0x0001832feec8  movaps   xmm0, xmmword ptr [r15]
0x0001832feecc  lea      rcx, [rsp + 0x90]
0x0001832feed4  movaps   xmmword ptr [rcx], xmm0
0x0001832feed7  mov      rax, qword ptr [r15 + 0x10]
0x0001832feedb  mov      qword ptr [rcx + 0x10], rax
0x0001832feedf  xorps    xmm6, xmm6
0x0001832feee2  movaps   xmmword ptr [r15], xmm6
0x0001832feee6  xor      esi, esi
0x0001832feee8  mov      qword ptr [r15 + 0x10], rsi
0x0001832feeec  lea      rdx, [rip + 0xc8ce549]   ; " ( "
0x0001832feef3  mov      r8d, 3
0x0001832feef9  call     0x183b596f0
0x0001832feefe  movups   xmm0, xmmword ptr [rax]
0x0001832fef01  lea      rcx, [rsp + 0xb0]
0x0001832fef09  movaps   xmmword ptr [rcx], xmm0
0x0001832fef0c  mov      rdx, qword ptr [rax + 0x10]
0x0001832fef10  mov      qword ptr [rcx + 0x10], rdx
0x0001832fef14  mov      qword ptr [rax + 0x10], rsi
0x0001832fef18  movups   xmmword ptr [rax], xmm6
0x0001832fef1b  lea      rdx, [rsp + 0xd0]
0x0001832fef23  movsx    r8, byte ptr [rdx + 0x17]
0x0001832fef28  test     r8, r8
0x0001832fef2b  cmovs    r8, qword ptr [rdx + 8]
0x0001832fef30  cmovs    rdx, qword ptr [rdx]
0x0001832fef34  call     0x183b596f0
0x0001832fef39  movups   xmm0, xmmword ptr [rax]
0x0001832fef3c  lea      rcx, [rsp + 0xf0]
0x0001832fef44  movaps   xmmword ptr [rcx], xmm0
0x0001832fef47  mov      rdx, qword ptr [rax + 0x10]
0x0001832fef4b  mov      qword ptr [rcx + 0x10], rdx
0x0001832fef4f  mov      qword ptr [rax + 0x10], rsi
0x0001832fef53  movups   xmmword ptr [rax], xmm6
0x0001832fef56  lea      rdx, [rip + 0xc77d147]   ; 0x18fa7c0a4
0x0001832fef5d  mov      r8d, 1
0x0001832fef63  call     0x183b596f0
0x0001832fef68  movups   xmm0, xmmword ptr [rax]
0x0001832fef6b  lea      rcx, [rsp + 0x110]
0x0001832fef73  movaps   xmmword ptr [rcx], xmm0
0x0001832fef76  mov      rdx, qword ptr [rax + 0x10]
0x0001832fef7a  mov      qword ptr [rcx + 0x10], rdx
0x0001832fef7e  mov      qword ptr [rax + 0x10], rsi
0x0001832fef82  movups   xmmword ptr [rax], xmm6
0x0001832fef85  movsx    r8, byte ptr [rsp + 0x17f]
0x0001832fef8e  test     r8, r8
0x0001832fef91  cmovs    r8, qword ptr [rsp + 0x170]
0x0001832fef9a  cmovs    r14, qword ptr [rsp + 0x168]
0x0001832fefa3  mov      rdx, r14
0x0001832fefa6  call     0x183b596f0
0x0001832fefab  movups   xmm0, xmmword ptr [rax]
0x0001832fefae  lea      rcx, [rsp + 0x180]
0x0001832fefb6  movaps   xmmword ptr [rcx], xmm0
0x0001832fefb9  mov      rdx, qword ptr [rax + 0x10]
0x0001832fefbd  mov      qword ptr [rcx + 0x10], rdx
0x0001832fefc1  mov      qword ptr [rax + 0x10], rsi
0x0001832fefc5  movups   xmmword ptr [rax], xmm6
0x0001832fefc8  lea      rdx, [rip + 0xc8ce429]   ; " )]"
0x0001832fefcf  mov      r8d, 3
0x0001832fefd5  call     0x183b596f0
0x0001832fefda  movups   xmm0, xmmword ptr [rax]
0x0001832fefdd  lea      r14, [rsp + 0x1a0]
0x0001832fefe5  movaps   xmmword ptr [r14], xmm0
0x0001832fefe9  mov      rcx, qword ptr [rax + 0x10]
0x0001832fefed  mov      qword ptr [r14 + 0x10], rcx
0x0001832feff1  mov      qword ptr [rax + 0x10], rsi
0x0001832feff5  movups   xmmword ptr [rax], xmm6
0x0001832feff8  mov      edx, dword ptr [rbp]
0x0001832feffb  lea      r15, [rsp + 0x38]
0x0001832ff000  mov      rcx, r15
0x0001832ff003  call     0x184ec6ea0
0x0001832ff008  lea      r8, [rip + 0xc8ce999]   ; "Unable to read profile from stream: "
0x0001832ff00f  mov      r9d, 0x24
0x0001832ff015  mov      rcx, r15
0x0001832ff018  xor      edx, edx
0x0001832ff01a  call     0x184d2c080
0x0001832ff01f  movups   xmm0, xmmword ptr [rax]
0x0001832ff022  lea      rdx, [rsp + 0x50]
0x0001832ff027  movaps   xmmword ptr [rdx], xmm0
0x0001832ff02a  mov      rcx, qword ptr [rax + 0x10]
0x0001832ff02e  mov      qword ptr [rdx + 0x10], rcx
0x0001832ff032  mov      qword ptr [rax + 0x10], rsi
0x0001832ff036  movups   xmmword ptr [rax], xmm6
0x0001832ff039  mov      rcx, rbx
0x0001832ff03c  mov      r8, r14
0x0001832ff03f  call     0x181199c30
0x0001832ff044  lea      rcx, [rbp + 0xb8]
0x0001832ff04b  cmp      rcx, rbx
0x0001832ff04e  je       0x1832ff086
0x0001832ff050  cmp      byte ptr [rbp + 0xcf], 0
0x0001832ff057  movzx    r8d, byte ptr [rsp + 0x1d7]
0x0001832ff060  js       0x183300c95
0x0001832ff066  test     r8b, r8b
0x0001832ff069  js       0x1832ff30d
0x0001832ff06f  mov      rax, qword ptr [rsp + 0x1d0]
0x0001832ff077  mov      qword ptr [rcx + 0x10], rax
0x0001832ff07b  movups   xmm0, xmmword ptr [rsp + 0x1c0]
0x0001832ff083  movups   xmmword ptr [rcx], xmm0
0x0001832ff086  lea      rcx, [rbp + 0xd0]
0x0001832ff08d  lea      rdx, [rsp + 0x1d8]
0x0001832ff095  cmp      rcx, rdx
0x0001832ff098  je       0x1832ff0c7
0x0001832ff09a  cmp      byte ptr [rbp + 0xe7], 0
0x0001832ff0a1  movzx    r8d, byte ptr [rsp + 0x1ef]
0x0001832ff0aa  js       0x1832ff327
0x0001832ff0b0  test     r8b, r8b
0x0001832ff0b3  js       0x1832ff2f3
0x0001832ff0b9  mov      rax, qword ptr [rdx + 0x10]
0x0001832ff0bd  mov      qword ptr [rcx + 0x10], rax
0x0001832ff0c1  movups   xmm0, xmmword ptr [rdx]
0x0001832ff0c4  movups   xmmword ptr [rcx], xmm0
0x0001832ff0c7  cmp      byte ptr [rsp + 0x1ef], 0
0x0001832ff0cf  jns      0x1832ff0e9
0x0001832ff0d1  mov      rdx, qword ptr [rsp + 0x1e8]
0x0001832ff0d9  and      rdx, rdi
0x0001832ff0dc  mov      rcx, qword ptr [rsp + 0x1d8]
0x0001832ff0e4  call     0x183b135e0
0x0001832ff0e9  cmp      byte ptr [rsp + 0x1d7], 0
0x0001832ff0f1  jns      0x1832ff10b
0x0001832ff0f3  mov      rdx, qword ptr [rsp + 0x1d0]
0x0001832ff0fb  and      rdx, rdi
0x0001832ff0fe  mov      rcx, qword ptr [rsp + 0x1c0]
0x0001832ff106  call     0x183b135e0
0x0001832ff10b  cmp      byte ptr [rsp + 0x67], 0
0x0001832ff110  jns      0x1832ff124
0x0001832ff112  mov      rdx, qword ptr [rsp + 0x60]
0x0001832ff117  and      rdx, rdi
0x0001832ff11a  mov      rcx, qword ptr [rsp + 0x50]
0x0001832ff11f  call     0x183b135e0
0x0001832ff124  cmp      byte ptr [rsp + 0x4f], 0
0x0001832ff129  jns      0x1832ff13d
0x0001832ff12b  mov      rdx, qword ptr [rsp + 0x48]
0x0001832ff130  and      rdx, rdi
0x0001832ff133  mov      rcx, qword ptr [rsp + 0x38]
0x0001832ff138  call     0x183b135e0
0x0001832ff13d  cmp      byte ptr [rsp + 0x1b7], 0
0x0001832ff145  jns      0x1832ff15f
0x0001832ff147  mov      rdx, qword ptr [rsp + 0x1b0]
0x0001832ff14f  and      rdx, rdi
0x0001832ff152  mov      rcx, qword ptr [rsp + 0x1a0]
0x0001832ff15a  call     0x183b135e0
0x0001832ff15f  cmp      byte ptr [rsp + 0x197], 0
0x0001832ff167  jns      0x1832ff181
0x0001832ff169  mov      rdx, qword ptr [rsp + 0x190]
0x0001832ff171  and      rdx, rdi
0x0001832ff174  mov      rcx, qword ptr [rsp + 0x180]
0x0001832ff17c  call     0x183b135e0
0x0001832ff181  cmp      byte ptr [rsp + 0x127], 0
0x0001832ff189  jns      0x1832ff1a3
0x0001832ff18b  mov      rdx, qword ptr [rsp + 0x120]
0x0001832ff193  and      rdx, rdi
0x0001832ff196  mov      rcx, qword ptr [rsp + 0x110]
0x0001832ff19e  call     0x183b135e0
0x0001832ff1a3  cmp      byte ptr [rsp + 0x107], 0
0x0001832ff1ab  jns      0x1832ff1c5
0x0001832ff1ad  mov      rdx, qword ptr [rsp + 0x100]
0x0001832ff1b5  and      rdx, rdi
0x0001832ff1b8  mov      rcx, qword ptr [rsp + 0xf0]
0x0001832ff1c0  call     0x183b135e0
0x0001832ff1c5  cmp      byte ptr [rsp + 0xc7], 0
0x0001832ff1cd  jns      0x1832ff1e7
0x0001832ff1cf  mov      rdx, qword ptr [rsp + 0xc0]
0x0001832ff1d7  and      rdx, rdi
0x0001832ff1da  mov      rcx, qword ptr [rsp + 0xb0]
0x0001832ff1e2  call     0x183b135e0
0x0001832ff1e7  cmp      byte ptr [rsp + 0xa7], 0
0x0001832ff1ef  jns      0x1832ff209
0x0001832ff1f1  mov      rdx, qword ptr [rsp + 0xa0]
0x0001832ff1f9  and      rdx, rdi
0x0001832ff1fc  mov      rcx, qword ptr [rsp + 0x90]
0x0001832ff204  call     0x183b135e0
0x0001832ff209  cmp      byte ptr [rsp + 0x87], 0
0x0001832ff211  jns      0x1832ff228
0x0001832ff213  mov      rdx, qword ptr [rsp + 0x80]
0x0001832ff21b  and      rdx, rdi
0x0001832ff21e  mov      rcx, qword ptr [rsp + 0x70]
0x0001832ff223  call     0x183b135e0
0x0001832ff228  cmp      byte ptr [rsp + 0xe7], 0
0x0001832ff230  jns      0x1832ff24a
0x0001832ff232  mov      rdx, qword ptr [rsp + 0xe0]
0x0001832ff23a  and      rdx, rdi
0x0001832ff23d  mov      rcx, qword ptr [rsp + 0xd0]
0x0001832ff245  call     0x183b135e0
0x0001832ff24a  cmp      byte ptr [rsp + 0x17f], 0
0x0001832ff252  jns      0x1832ff26c
0x0001832ff254  mov      rdx, qword ptr [rsp + 0x178]
0x0001832ff25c  and      rdx, rdi
0x0001832ff25f  mov      rcx, qword ptr [rsp + 0x168]
0x0001832ff267  call     0x183b135e0
0x0001832ff26c  xor      edi, edi
0x0001832ff26e  mov      rcx, qword ptr [rsp + 0x140]
0x0001832ff276  movabs   rax, 0xfffffffc00000000
0x0001832ff280  and      rax, rcx
0x0001832ff283  cmp      rax, qword ptr [rip + 0xe1c757e]   ; 0x1914c6808
0x0001832ff28a  jne      0x1832ff291
0x0001832ff28c  call     0x183b07950
0x0001832ff291  cmp      byte ptr [rsp + 0x167], 0
0x0001832ff299  jns      0x1832ff2ba
0x0001832ff29b  movabs   rdx, 0x7fffffffffffffff
0x0001832ff2a5  and      rdx, qword ptr [rsp + 0x160]
0x0001832ff2ad  mov      rcx, qword ptr [rsp + 0x150]
0x0001832ff2b5  call     0x183b135e0
0x0001832ff2ba  mov      rax, qword ptr [rsp + 0x218]
0x0001832ff2c2  xor      rax, rsp
0x0001832ff2c5  mov      rcx, qword ptr [rip + 0xe162d74]   ; 0x191462040
0x0001832ff2cc  cmp      rcx, rax
0x0001832ff2cf  jne      0x183300f47
0x0001832ff2d5  mov      eax, edi
0x0001832ff2d7  movaps   xmm6, xmmword ptr [rsp + 0x220]
0x0001832ff2df  add      rsp, 0x238
0x0001832ff2e6  pop      rbx
0x0001832ff2e7  pop      rbp
0x0001832ff2e8  pop      rdi
0x0001832ff2e9  pop      rsi
0x0001832ff2ea  pop      r12
0x0001832ff2ec  pop      r13
0x0001832ff2ee  pop      r14
0x0001832ff2f0  pop      r15
0x0001832ff2f2  ret      
0x0001832ff2f3  mov      rdx, qword ptr [rsp + 0x1d8]
0x0001832ff2fb  mov      r8, qword ptr [rsp + 0x1e0]
0x0001832ff303  call     0x183b61e40
0x0001832ff308  jmp      0x1832ff0c7
0x0001832ff30d  mov      rdx, qword ptr [rsp + 0x1c0]
0x0001832ff315  mov      r8, qword ptr [rsp + 0x1c8]
0x0001832ff31d  call     0x183b61e40
0x0001832ff322  jmp      0x1832ff086
0x0001832ff327  test     r8b, r8b
0x0001832ff32a  jns      0x1832ff33c
0x0001832ff32c  mov      r8, qword ptr [rsp + 0x1e0]
0x0001832ff334  mov      rdx, qword ptr [rsp + 0x1d8]
0x0001832ff33c  call     0x183b61fa0
0x0001832ff341  jmp      0x1832ff0c7
0x0001832ff346  ud2      
0x0001832ff348  mov      r11, rdx
0x0001832ff34b  test     r10b, r10b
0x0001832ff34e  movabs   rdi, 0x7fffffffffffffff
0x0001832ff358  jns      0x1832ff35d
0x0001832ff35a  mov      r11, qword ptr [rdx]
0x0001832ff35d  movsx    ebp, byte ptr [r11 + r8 + 1]
0x0001832ff363  mov      qword ptr [rax], r9
0x0001832ff366  mov      byte ptr [rax + 8], 1
0x0001832ff36a  movsx    r14, byte ptr [rdx + 0x17]
0x0001832ff36f  mov      r11, r14
0x0001832ff372  test     r14, r14
0x0001832ff375  jns      0x1832ff37b
0x0001832ff377  mov      r11, qword ptr [rdx + 8]
0x0001832ff37b  add      r8, 3
0x0001832ff37f  xor      r10d, r10d
0x0001832ff382  cmp      r8, r11
0x0001832ff385  ja       0x1832ff397
0x0001832ff387  test     r14b, r14b
0x0001832ff38a  jns      0x1832ff38f
0x0001832ff38c  mov      rdx, qword ptr [rdx]
0x0001832ff38f  movsx    r10d, byte ptr [rdx + r9]
0x0001832ff394  mov      qword ptr [rax], r8
0x0001832ff397  cmp      r8, r11
0x0001832ff39a  setbe    r11b
0x0001832ff39e  mov      byte ptr [rax + 8], r11b
0x0001832ff3a2  mov      r14, r12
0x0001832ff3a5  mov      r12, r13
0x0001832ff3a8  mov      r13, rbx
0x0001832ff3ab  mov      rbx, qword ptr [rsp + 0x30]
0x0001832ff3b0  mov      dword ptr [rsp + 0x1c0], esi
0x0001832ff3b7  mov      dword ptr [rsp + 0x1e4], ebp
0x0001832ff3be  mov      dword ptr [rsp + 0x1e8], r10d
0x0001832ff3c6  mov      dword ptr [rsp + 0x1e0], ecx
0x0001832ff3cd  mov      rax, qword ptr [rbx + 0x5a8]
0x0001832ff3d4  cmp      rax, qword ptr [rbx + 0x5b0]
0x0001832ff3db  je       0x183300cdf
0x0001832ff3e1  lea      rcx, [rax + rax*2]
0x0001832ff3e5  shl      rcx, 4
0x0001832ff3e9  add      rcx, qword ptr [rbx + 0x5a0]
0x0001832ff3f0  je       0x1832ff346
0x0001832ff3f6  lea      rbp, [rcx + 0x30]
0x0001832ff3fa  mov      rdx, r13
0x0001832ff3fd  call     0x1832df600
0x0001832ff402  mov      rax, qword ptr [rbx + 0x5a0]
0x0001832ff409  mov      rcx, rbp
0x0001832ff40c  sub      rcx, rax
0x0001832ff40f  sar      rcx, 4
0x0001832ff413  movabs   rdx, 0xaaaaaaaaaaaaaaab
0x0001832ff41d  imul     rcx, rdx
0x0001832ff421  mov      qword ptr [rbx + 0x5a8], rcx
0x0001832ff428  cmp      rbp, rax
0x0001832ff42b  je       0x1832ff346
0x0001832ff431  cmp      byte ptr [rsp + 0x1df], 0
0x0001832ff439  jns      0x1832ff453
0x0001832ff43b  mov      rdx, qword ptr [rsp + 0x1d8]
0x0001832ff443  and      rdx, rdi
0x0001832ff446  mov      rcx, qword ptr [rsp + 0x1c8]
0x0001832ff44e  call     0x183b135e0
0x0001832ff453  dec      r15d
0x0001832ff456  je       0x1832fd92b
0x0001832ff45c  movaps   xmmword ptr [rsp + 0x1e0], xmm6
0x0001832ff464  movaps   xmmword ptr [rsp + 0x1d0], xmm6
0x0001832ff46c  movaps   xmmword ptr [rsp + 0x1c0], xmm6
0x0001832ff474  xor      ebp, ebp
0x0001832ff476  cmp      byte ptr [rsp + 0x138], 1
0x0001832ff47e  mov      esi, 0
0x0001832ff483  jne      0x1832ff4d1
0x0001832ff485  mov      rcx, qword ptr [rsp + 0x140]
0x0001832ff48d  movsx    r9, byte ptr [rcx + 0x17]
0x0001832ff492  mov      rax, r9
0x0001832ff495  test     r9, r9
0x0001832ff498  jns      0x1832ff49e
0x0001832ff49a  mov      rax, qword ptr [rcx + 8]
0x0001832ff49e  mov      r8, qword ptr [rsp + 0x130]
0x0001832ff4a6  lea      rdx, [r8 + 1]
0x0001832ff4aa  xor      esi, esi
0x0001832ff4ac  cmp      rdx, rax
0x0001832ff4af  ja       0x1832ff4c6
0x0001832ff4b1  test     r9b, r9b
0x0001832ff4b4  jns      0x1832ff4b9
0x0001832ff4b6  mov      rcx, qword ptr [rcx]
0x0001832ff4b9  movsx    esi, byte ptr [rcx + r8]
0x0001832ff4be  mov      qword ptr [rsp + 0x130], rdx
0x0001832ff4c6  cmp      rdx, rax
0x0001832ff4c9  setbe    byte ptr [rsp + 0x138]
0x0001832ff4d1  mov      rcx, r12
0x0001832ff4d4  mov      rdx, r14
0x0001832ff4d7  call     0x183303050
0x0001832ff4dc  cmp      byte ptr [rax + 8], 1
0x0001832ff4e0  mov      ecx, 0
0x0001832ff4e5  mov      r10d, 0
0x0001832ff4eb  jne      0x1832ff3b0
0x0001832ff4f1  mov      rbx, r13
0x0001832ff4f4  mov      r13, r12
0x0001832ff4f7  mov      r12, r14
0x0001832ff4fa  mov      rdx, qword ptr [rax + 0x10]
0x0001832ff4fe  movsx    r9, byte ptr [rdx + 0x17]
0x0001832ff503  mov      rdi, r9
0x0001832ff506  test     r9, r9
0x0001832ff509  jns      0x1832ff50f
0x0001832ff50b  mov      rdi, qword ptr [rdx + 8]
0x0001832ff50f  mov      r8, qword ptr [rax]
0x0001832ff512  lea      r14, [r8 + 1]
0x0001832ff516  xor      r11d, r11d
0x0001832ff519  mov      ebp, 0
0x0001832ff51e  cmp      r14, rdi
0x0001832ff521  movabs   rdi, 0x7fffffffffffffff
0x0001832ff52b  ja       0x1832ff39e
0x0001832ff531  mov      rcx, rdx
0x0001832ff534  test     r9b, r9b
0x0001832ff537  jns      0x1832ff53c
0x0001832ff539  mov      rcx, qword ptr [rdx]
0x0001832ff53c  movsx    ecx, byte ptr [rcx + r8]
0x0001832ff541  mov      qword ptr [rax], r14
0x0001832ff544  mov      byte ptr [rax + 8], 1
0x0001832ff548  movsx    r10, byte ptr [rdx + 0x17]
0x0001832ff54d  mov      rdi, r10
0x0001832ff550  test     r10, r10
0x0001832ff553  jns      0x1832ff559
0x0001832ff555  mov      rdi, qword ptr [rdx + 8]
0x0001832ff559  lea      r9, [r8 + 2]
0x0001832ff55d  xor      r11d, r11d
0x0001832ff560  cmp      r9, rdi
0x0001832ff563  jbe      0x1832ff348
0x0001832ff569  xor      ebp, ebp
0x0001832ff56b  xor      r10d, r10d
0x0001832ff56e  movabs   rdi, 0x7fffffffffffffff
0x0001832ff578  jmp      0x1832ff39e
0x0001832ff57d  xor      eax, eax
0x0001832ff57f  jmp      0x1832fe249
0x0001832ff584  mov      rdx, qword ptr [r12 + 8]
0x0001832ff589  mov      r8, qword ptr [r12 + 0x10]
0x0001832ff58e  call     0x1853c29f0
0x0001832ff593  jmp      0x1832fd201
0x0001832ff598  test     r8b, r8b
0x0001832ff59b  jns      0x1832ff5a7
0x0001832ff59d  mov      r8, qword ptr [r12 + 0x10]
0x0001832ff5a2  mov      rdx, qword ptr [r12 + 8]
0x0001832ff5a7  call     0x181248280
0x0001832ff5ac  jmp      0x1832fd201
0x0001832ff5b1  lea      r14, [rsp + 0x168]
0x0001832ff5b9  mov      rcx, r14
0x0001832ff5bc  mov      edx, 0x212
0x0001832ff5c1  call     0x184ec6ea0
0x0001832ff5c6  mov      ecx, 0x40
0x0001832ff5cb  call     0x18593bfb8
0x0001832ff5d0  mov      qword ptr [rsp + 0xf0], rax
0x0001832ff5d8  mov      qword ptr [rsp + 0xf8], 0x30
0x0001832ff5e4  lea      rcx, [rdi + 0x41]
0x0001832ff5e8  mov      qword ptr [rsp + 0x100], rcx
0x0001832ff5f0  lea      rcx, [rax + 0x30]
0x0001832ff5f4  cmp      rcx, rax
0x0001832ff5f7  jb       0x1832ff346
0x0001832ff5fd  lea      rdx, [rip + 0xc8ce25c]   ; "..\..\undetectable\core\stors\profilestorage.cpp"
0x0001832ff604  cmp      rax, rdx
0x0001832ff607  seta     r8b
0x0001832ff60b  cmp      rcx, rdx
0x0001832ff60e  setbe    cl
0x0001832ff611  or       cl, r8b
0x0001832ff614  je       0x1832ff346
0x0001832ff61a  movups   xmm0, xmmword ptr [rip + 0xc8ce25f]   ; "ofilestorage.cpp"
0x0001832ff621  movups   xmmword ptr [rax + 0x20], xmm0
0x0001832ff625  movups   xmm0, xmmword ptr [rip + 0xc8ce244]   ; "le\core\stors\profilestorage.cpp"
0x0001832ff62c  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832ff630  movups   xmm0, xmmword ptr [rip + 0xc8ce229]   ; "..\..\undetectable\core\stors\profilestorage.cpp"
0x0001832ff637  movups   xmmword ptr [rax], xmm0
0x0001832ff63a  mov      byte ptr [rax + 0x30], 0
0x0001832ff63e  mov      ecx, 0x30
0x0001832ff643  call     0x18593bfb8
0x0001832ff648  mov      qword ptr [rsp + 0x90], rax
0x0001832ff650  mov      qword ptr [rsp + 0x98], 0x29
0x0001832ff65c  lea      rcx, [rdi + 0x31]
0x0001832ff660  mov      qword ptr [rsp + 0xa0], rcx
0x0001832ff668  lea      rcx, [rax + 0x29]
0x0001832ff66c  cmp      rcx, rax
0x0001832ff66f  jb       0x1832ff346
0x0001832ff675  lea      rdx, [rip + 0xc8ce2dc]   ; "undetectable::ProfileStorage::readProfile"
0x0001832ff67c  cmp      rax, rdx
0x0001832ff67f  seta     r8b
0x0001832ff683  cmp      rcx, rdx
0x0001832ff686  setbe    cl
0x0001832ff689  or       cl, r8b
0x0001832ff68c  je       0x1832ff346
0x0001832ff692  movups   xmm0, xmmword ptr [rip + 0xc8ce2d8]   ; "age::readProfile"
0x0001832ff699  movups   xmmword ptr [rax + 0x19], xmm0
0x0001832ff69d  movups   xmm0, xmmword ptr [rip + 0xc8ce2c4]   ; "ofileStorage::readProfile"
0x0001832ff6a4  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832ff6a8  movups   xmm0, xmmword ptr [rip + 0xc8ce2a9]   ; "undetectable::ProfileStorage::readProfile"
0x0001832ff6af  movups   xmmword ptr [rax], xmm0
0x0001832ff6b2  mov      byte ptr [rax + 0x29], 0
0x0001832ff6b6  lea      r15, [rsp + 0x90]
0x0001832ff6be  lea      rbx, [rsp + 0x1c0]
0x0001832ff6c6  mov      rcx, r15
0x0001832ff6c9  mov      rdx, rbx
0x0001832ff6cc  mov      r8, rax
0x0001832ff6cf  mov      r9b, 0x5b
0x0001832ff6d2  call     0x18a58b3f0
0x0001832ff6d7  movaps   xmm0, xmmword ptr [r15]
0x0001832ff6db  lea      rcx, [rsp + 0xb0]
0x0001832ff6e3  movaps   xmmword ptr [rcx], xmm0
0x0001832ff6e6  mov      rax, qword ptr [r15 + 0x10]
0x0001832ff6ea  mov      qword ptr [rcx + 0x10], rax
0x0001832ff6ee  xorps    xmm6, xmm6
0x0001832ff6f1  movaps   xmmword ptr [r15], xmm6
0x0001832ff6f5  xor      esi, esi
0x0001832ff6f7  mov      qword ptr [r15 + 0x10], rsi
0x0001832ff6fb  lea      rdx, [rip + 0xc8cdd3a]   ; " ( "
0x0001832ff702  mov      r8d, 3
0x0001832ff708  call     0x183b596f0
0x0001832ff70d  movups   xmm0, xmmword ptr [rax]
0x0001832ff710  lea      rcx, [rsp + 0xd0]
0x0001832ff718  movaps   xmmword ptr [rcx], xmm0
0x0001832ff71b  mov      rdx, qword ptr [rax + 0x10]
0x0001832ff71f  mov      qword ptr [rcx + 0x10], rdx
0x0001832ff723  mov      qword ptr [rax + 0x10], rsi
0x0001832ff727  movups   xmmword ptr [rax], xmm6
0x0001832ff72a  lea      rdx, [rsp + 0xf0]
0x0001832ff732  movsx    r8, byte ptr [rdx + 0x17]
0x0001832ff737  test     r8, r8
0x0001832ff73a  cmovs    r8, qword ptr [rdx + 8]
0x0001832ff73f  cmovs    rdx, qword ptr [rdx]
0x0001832ff743  call     0x183b596f0
0x0001832ff748  movups   xmm0, xmmword ptr [rax]
0x0001832ff74b  lea      rcx, [rsp + 0x110]
0x0001832ff753  movaps   xmmword ptr [rcx], xmm0
0x0001832ff756  mov      rdx, qword ptr [rax + 0x10]
0x0001832ff75a  mov      qword ptr [rcx + 0x10], rdx
0x0001832ff75e  mov      qword ptr [rax + 0x10], rsi
0x0001832ff762  movups   xmmword ptr [rax], xmm6
0x0001832ff765  lea      rdx, [rip + 0xc77c938]   ; 0x18fa7c0a4
0x0001832ff76c  mov      r8d, 1
0x0001832ff772  call     0x183b596f0
0x0001832ff777  movups   xmm0, xmmword ptr [rax]
0x0001832ff77a  lea      rcx, [rsp + 0x130]
0x0001832ff782  movaps   xmmword ptr [rcx], xmm0
0x0001832ff785  mov      rdx, qword ptr [rax + 0x10]
0x0001832ff789  mov      qword ptr [rcx + 0x10], rdx
0x0001832ff78d  mov      qword ptr [rax + 0x10], rsi
0x0001832ff791  movups   xmmword ptr [rax], xmm6
0x0001832ff794  movsx    r8, byte ptr [rsp + 0x17f]
0x0001832ff79d  test     r8, r8
0x0001832ff7a0  cmovs    r8, qword ptr [rsp + 0x170]
0x0001832ff7a9  cmovs    r14, qword ptr [rsp + 0x168]
0x0001832ff7b2  mov      rdx, r14
0x0001832ff7b5  call     0x183b596f0
0x0001832ff7ba  movups   xmm0, xmmword ptr [rax]
0x0001832ff7bd  lea      rcx, [rsp + 0x180]
0x0001832ff7c5  movaps   xmmword ptr [rcx], xmm0
0x0001832ff7c8  mov      rdx, qword ptr [rax + 0x10]
0x0001832ff7cc  mov      qword ptr [rcx + 0x10], rdx
0x0001832ff7d0  mov      qword ptr [rax + 0x10], rsi
0x0001832ff7d4  movups   xmmword ptr [rax], xmm6
0x0001832ff7d7  lea      rdx, [rip + 0xc8cdc1a]   ; " )]"
0x0001832ff7de  mov      r8d, 3
0x0001832ff7e4  call     0x183b596f0
0x0001832ff7e9  movups   xmm0, xmmword ptr [rax]
0x0001832ff7ec  movaps   xmmword ptr [rsp + 0x1a0], xmm0
0x0001832ff7f4  mov      rcx, qword ptr [rax + 0x10]
0x0001832ff7f8  mov      qword ptr [rsp + 0x1b0], rcx
0x0001832ff800  mov      qword ptr [rax + 0x10], rsi
0x0001832ff804  movups   xmmword ptr [rax], xmm6
0x0001832ff807  mov      ecx, 0x20
0x0001832ff80c  call     0x18593bfb8
0x0001832ff811  mov      qword ptr [rsp + 0x70], rax
0x0001832ff816  mov      qword ptr [rsp + 0x78], 0x18
0x0001832ff81f  lea      rcx, [rdi + 0x21]
0x0001832ff823  mov      qword ptr [rsp + 0x80], rcx
0x0001832ff82b  lea      rcx, [rax + 0x18]
0x0001832ff82f  cmp      rcx, rax
0x0001832ff832  jb       0x1832ff346
0x0001832ff838  lea      rdx, [rip + 0xc8ce149]   ; "Unable to read info file"
0x0001832ff83f  cmp      rax, rdx
0x0001832ff842  seta     r8b
0x0001832ff846  cmp      rcx, rdx
0x0001832ff849  setbe    cl
0x0001832ff84c  or       cl, r8b
0x0001832ff84f  je       0x1832ff346
0x0001832ff855  movups   xmm0, xmmword ptr [rip + 0xc8ce12c]   ; "Unable to read info file"
0x0001832ff85c  movups   xmmword ptr [rax], xmm0
0x0001832ff85f  movabs   rcx, 0x656c6966206f666e
0x0001832ff869  mov      qword ptr [rax + 0x10], rcx
0x0001832ff86d  mov      byte ptr [rax + 0x18], 0
0x0001832ff871  lea      rdx, [rsp + 0x70]
0x0001832ff876  lea      r8, [rsp + 0x1a0]
0x0001832ff87e  mov      rcx, rbx
0x0001832ff881  call     0x181199c30
0x0001832ff886  lea      rcx, [r12 + 0xb8]
0x0001832ff88e  cmp      rcx, rbx
0x0001832ff891  je       0x1832ff8cb
0x0001832ff893  cmp      byte ptr [r12 + 0xcf], 0
0x0001832ff89c  movzx    r8d, byte ptr [rsp + 0x1d7]
0x0001832ff8a5  js       0x183300d47
0x0001832ff8ab  test     r8b, r8b
0x0001832ff8ae  js       0x1832ffac5
0x0001832ff8b4  mov      rax, qword ptr [rsp + 0x1d0]
0x0001832ff8bc  mov      qword ptr [rcx + 0x10], rax
0x0001832ff8c0  movups   xmm0, xmmword ptr [rsp + 0x1c0]
0x0001832ff8c8  movups   xmmword ptr [rcx], xmm0
0x0001832ff8cb  lea      rcx, [r12 + 0xd0]
0x0001832ff8d3  lea      rdx, [rsp + 0x1d8]
0x0001832ff8db  cmp      rcx, rdx
0x0001832ff8de  je       0x1832ff90f
0x0001832ff8e0  cmp      byte ptr [r12 + 0xe7], 0
0x0001832ff8e9  movzx    r8d, byte ptr [rsp + 0x1ef]
0x0001832ff8f2  js       0x1832ffadf
0x0001832ff8f8  test     r8b, r8b
0x0001832ff8fb  js       0x1832ffaab
0x0001832ff901  mov      rax, qword ptr [rdx + 0x10]
0x0001832ff905  mov      qword ptr [rcx + 0x10], rax
0x0001832ff909  movups   xmm0, xmmword ptr [rdx]
0x0001832ff90c  movups   xmmword ptr [rcx], xmm0
0x0001832ff90f  cmp      byte ptr [rsp + 0x1ef], 0
0x0001832ff917  jns      0x1832ff931
0x0001832ff919  mov      rdx, qword ptr [rsp + 0x1e8]
0x0001832ff921  and      rdx, rdi
0x0001832ff924  mov      rcx, qword ptr [rsp + 0x1d8]
0x0001832ff92c  call     0x183b135e0
0x0001832ff931  cmp      byte ptr [rsp + 0x1d7], 0
0x0001832ff939  jns      0x1832ff953
0x0001832ff93b  mov      rdx, qword ptr [rsp + 0x1d0]
0x0001832ff943  and      rdx, rdi
0x0001832ff946  mov      rcx, qword ptr [rsp + 0x1c0]
0x0001832ff94e  call     0x183b135e0
0x0001832ff953  cmp      byte ptr [rsp + 0x87], 0
0x0001832ff95b  jns      0x1832ff972
0x0001832ff95d  mov      rdx, qword ptr [rsp + 0x80]
0x0001832ff965  and      rdx, rdi
0x0001832ff968  mov      rcx, qword ptr [rsp + 0x70]
0x0001832ff96d  call     0x183b135e0
0x0001832ff972  cmp      byte ptr [rsp + 0x1b7], 0
0x0001832ff97a  jns      0x1832ff994
0x0001832ff97c  mov      rdx, qword ptr [rsp + 0x1b0]
0x0001832ff984  and      rdx, rdi
0x0001832ff987  mov      rcx, qword ptr [rsp + 0x1a0]
0x0001832ff98f  call     0x183b135e0
0x0001832ff994  cmp      byte ptr [rsp + 0x197], 0
0x0001832ff99c  jns      0x1832ff9b6
0x0001832ff99e  mov      rdx, qword ptr [rsp + 0x190]
0x0001832ff9a6  and      rdx, rdi
0x0001832ff9a9  mov      rcx, qword ptr [rsp + 0x180]
0x0001832ff9b1  call     0x183b135e0
0x0001832ff9b6  cmp      byte ptr [rsp + 0x147], 0
0x0001832ff9be  jns      0x1832ff9d8
0x0001832ff9c0  mov      rdx, qword ptr [rsp + 0x140]
0x0001832ff9c8  and      rdx, rdi
0x0001832ff9cb  mov      rcx, qword ptr [rsp + 0x130]
0x0001832ff9d3  call     0x183b135e0
0x0001832ff9d8  cmp      byte ptr [rsp + 0x127], 0
0x0001832ff9e0  jns      0x1832ff9fa
0x0001832ff9e2  mov      rdx, qword ptr [rsp + 0x120]
0x0001832ff9ea  and      rdx, rdi
0x0001832ff9ed  mov      rcx, qword ptr [rsp + 0x110]
0x0001832ff9f5  call     0x183b135e0
0x0001832ff9fa  cmp      byte ptr [rsp + 0xe7], 0
0x0001832ffa02  jns      0x1832ffa1c
0x0001832ffa04  mov      rdx, qword ptr [rsp + 0xe0]
0x0001832ffa0c  and      rdx, rdi
0x0001832ffa0f  mov      rcx, qword ptr [rsp + 0xd0]
0x0001832ffa17  call     0x183b135e0
0x0001832ffa1c  cmp      byte ptr [rsp + 0xc7], 0
0x0001832ffa24  jns      0x1832ffa3e
0x0001832ffa26  mov      rdx, qword ptr [rsp + 0xc0]
0x0001832ffa2e  and      rdx, rdi
0x0001832ffa31  mov      rcx, qword ptr [rsp + 0xb0]
0x0001832ffa39  call     0x183b135e0
0x0001832ffa3e  cmp      byte ptr [rsp + 0xa7], 0
0x0001832ffa46  jns      0x1832ffa60
0x0001832ffa48  mov      rdx, qword ptr [rsp + 0xa0]
0x0001832ffa50  and      rdx, rdi
0x0001832ffa53  mov      rcx, qword ptr [rsp + 0x90]
0x0001832ffa5b  call     0x183b135e0
0x0001832ffa60  cmp      byte ptr [rsp + 0x107], 0
0x0001832ffa68  jns      0x1832ffa82
0x0001832ffa6a  mov      rdx, qword ptr [rsp + 0x100]
0x0001832ffa72  and      rdx, rdi
0x0001832ffa75  mov      rcx, qword ptr [rsp + 0xf0]
0x0001832ffa7d  call     0x183b135e0
0x0001832ffa82  cmp      byte ptr [rsp + 0x17f], 0
0x0001832ffa8a  jns      0x1832ffaa4
0x0001832ffa8c  mov      rdx, qword ptr [rsp + 0x178]
0x0001832ffa94  and      rdx, rdi
0x0001832ffa97  mov      rcx, qword ptr [rsp + 0x168]
0x0001832ffa9f  call     0x183b135e0
0x0001832ffaa4  xor      edi, edi
0x0001832ffaa6  jmp      0x1832ff291
0x0001832ffaab  mov      rdx, qword ptr [rsp + 0x1d8]
0x0001832ffab3  mov      r8, qword ptr [rsp + 0x1e0]
0x0001832ffabb  call     0x183b61e40
0x0001832ffac0  jmp      0x1832ff90f
0x0001832ffac5  mov      rdx, qword ptr [rsp + 0x1c0]
0x0001832ffacd  mov      r8, qword ptr [rsp + 0x1c8]
0x0001832ffad5  call     0x183b61e40
0x0001832ffada  jmp      0x1832ff8cb
0x0001832ffadf  test     r8b, r8b
0x0001832ffae2  jns      0x1832ffaf4
0x0001832ffae4  mov      r8, qword ptr [rsp + 0x1e0]
0x0001832ffaec  mov      rdx, qword ptr [rsp + 0x1d8]
0x0001832ffaf4  call     0x183b61fa0
0x0001832ffaf9  jmp      0x1832ff90f
0x0001832ffafe  mov      rsi, rax
0x0001832ffb01  test     dil, dil
0x0001832ffb04  jns      0x1832ffb09
0x0001832ffb06  mov      rsi, qword ptr [rax]
0x0001832ffb09  mov      esi, dword ptr [rsi + r9 + 0x15]
0x0001832ffb0e  mov      dword ptr [r15], esi
0x0001832ffb11  mov      qword ptr [rsp + 0x130], rdx
0x0001832ffb19  mov      byte ptr [rsp + 0x138], 1
0x0001832ffb21  lea      rsi, [r9 + 0x21]
0x0001832ffb25  movsx    rdi, byte ptr [rax + 0x17]
0x0001832ffb2a  mov      r12, rdi
0x0001832ffb2d  test     rdi, rdi
0x0001832ffb30  jns      0x1832ffb36
0x0001832ffb32  mov      r12, qword ptr [rax + 8]
0x0001832ffb36  cmp      rsi, r12
0x0001832ffb39  jbe      0x183300d69
0x0001832ffb3f  mov      byte ptr [rsp + 0x138], 0
0x0001832ffb47  jmp      0x1832ffc13
0x0001832ffb4c  mov      rdx, rax
0x0001832ffb4f  test     dil, dil
0x0001832ffb52  jns      0x1832ffb57
0x0001832ffb54  mov      rdx, qword ptr [rax]
0x0001832ffb57  mov      edx, dword ptr [rdx + r9 + 0x11]
0x0001832ffb5c  mov      dword ptr [rbx + 0x25c], edx
0x0001832ffb62  mov      qword ptr [rsp + 0x130], rsi
0x0001832ffb6a  mov      byte ptr [rsp + 0x138], 1
0x0001832ffb72  lea      rdx, [r9 + 0x19]
0x0001832ffb76  movsx    rdi, byte ptr [rax + 0x17]
0x0001832ffb7b  mov      r12, rdi
0x0001832ffb7e  test     rdi, rdi
0x0001832ffb81  jns      0x1832ffb87
0x0001832ffb83  mov      r12, qword ptr [rax + 8]
0x0001832ffb87  lea      r15, [rbx + 0x260]
0x0001832ffb8e  cmp      rdx, r12
0x0001832ffb91  jbe      0x1832ffafe
0x0001832ffb97  mov      byte ptr [rsp + 0x138], 0
0x0001832ffb9f  jmp      0x1832ffdf9
0x0001832ffba4  mov      r15d, 0x182
0x0001832ffbaa  movabs   rdi, 0x7fffffffffffffff
0x0001832ffbb4  jmp      0x1832fed80
0x0001832ffbb9  mov      rsi, rax
0x0001832ffbbc  test     dil, dil
0x0001832ffbbf  jns      0x1832ffbc4
0x0001832ffbc1  mov      rsi, qword ptr [rax]
0x0001832ffbc4  mov      esi, dword ptr [rsi + r9 + 0xd]
0x0001832ffbc9  mov      dword ptr [rbx + 0x248], esi
0x0001832ffbcf  mov      qword ptr [rsp + 0x130], rdx
0x0001832ffbd7  mov      byte ptr [rsp + 0x138], 1
0x0001832ffbdf  lea      rsi, [r9 + 0x15]
0x0001832ffbe3  movsx    rdi, byte ptr [rax + 0x17]
0x0001832ffbe8  mov      r15, rdi
0x0001832ffbeb  test     rdi, rdi
0x0001832ffbee  jns      0x1832ffbf4
0x0001832ffbf0  mov      r15, qword ptr [rax + 8]
0x0001832ffbf4  lea      r14, [rbx + 0x25c]
0x0001832ffbfb  cmp      rsi, r15
0x0001832ffbfe  jbe      0x1832ffb4c
0x0001832ffc04  mov      byte ptr [rsp + 0x138], 0
0x0001832ffc0c  lea      r15, [rbx + 0x260]
0x0001832ffc13  xor      edi, edi
0x0001832ffc15  jmp      0x1832ffdfe
0x0001832ffc1a  mov      rdx, rax
0x0001832ffc1d  test     r11b, r11b
0x0001832ffc20  jns      0x1832ffc25
0x0001832ffc22  mov      rdx, qword ptr [rax]
0x0001832ffc25  mov      edx, dword ptr [rdx + r9 + 9]
0x0001832ffc2a  mov      dword ptr [rbx + 0x244], edx
0x0001832ffc30  mov      qword ptr [rsp + 0x130], rsi
0x0001832ffc38  mov      byte ptr [rsp + 0x138], 1
0x0001832ffc40  lea      rdx, [r9 + 0x11]
0x0001832ffc44  movsx    rdi, byte ptr [rax + 0x17]
0x0001832ffc49  mov      r14, rdi
0x0001832ffc4c  test     rdi, rdi
0x0001832ffc4f  jns      0x1832ffc55
0x0001832ffc51  mov      r14, qword ptr [rax + 8]
0x0001832ffc55  lea      r11, [rbx + 0x248]
0x0001832ffc5c  cmp      rdx, r14
0x0001832ffc5f  jbe      0x1832ffbb9
0x0001832ffc65  mov      byte ptr [rsp + 0x138], 0
0x0001832ffc6d  jmp      0x1832ffdeb
0x0001832ffc72  mov      r11, rax
0x0001832ffc75  test     sil, sil
0x0001832ffc78  jns      0x1832ffc7d
0x0001832ffc7a  mov      r11, qword ptr [rax]
0x0001832ffc7d  mov      r11d, dword ptr [r11 + r9 + 5]
0x0001832ffc82  mov      dword ptr [rbx + 0x250], r11d
0x0001832ffc89  mov      qword ptr [rsp + 0x130], rdx
0x0001832ffc91  mov      byte ptr [rsp + 0x138], 1
0x0001832ffc99  lea      rsi, [r9 + 0xd]
0x0001832ffc9d  movsx    r11, byte ptr [rax + 0x17]
0x0001832ffca2  mov      rdi, r11
0x0001832ffca5  test     r11, r11
0x0001832ffca8  jns      0x1832ffcae
0x0001832ffcaa  mov      rdi, qword ptr [rax + 8]
0x0001832ffcae  cmp      rsi, rdi
0x0001832ffcb1  jbe      0x1832ffc1a
0x0001832ffcb7  mov      byte ptr [rsp + 0x138], 0
0x0001832ffcbf  jmp      0x1832ffde1
0x0001832ffcc4  mov      rcx, qword ptr [rbx + 0x244]
0x0001832ffccb  mov      qword ptr [rbx + 0x254], rcx
0x0001832ffcd2  jmp      0x1832ffeb2
0x0001832ffcd7  cmp      byte ptr [rsp + 0x138], 1
0x0001832ffcdf  jne      0x1832ffba4
0x0001832ffce5  mov      r9, qword ptr [rsp + 0x130]
0x0001832ffced  mov      rax, qword ptr [rsp + 0x140]
0x0001832ffcf5  mov      r8b, byte ptr [rax + 0x17]
0x0001832ffcf9  movzx    edx, r8b
0x0001832ffcfd  mov      rbp, qword ptr [rsp + 0x28]
0x0001832ffd02  lea      r10, [r9 + 1]
0x0001832ffd06  test     r8b, r8b
0x0001832ffd09  jns      0x1832ffd0f
0x0001832ffd0b  mov      rdx, qword ptr [rax + 8]
0x0001832ffd0f  cmp      r10, rdx
0x0001832ffd12  ja       0x1832ffdc2
0x0001832ffd18  mov      rcx, rax
0x0001832ffd1b  test     r8b, r8b
0x0001832ffd1e  jns      0x1832ffd23
0x0001832ffd20  mov      rcx, qword ptr [rax]
0x0001832ffd23  mov      cl, byte ptr [rcx + r9]
0x0001832ffd27  mov      byte ptr [rbx + 0x240], cl
0x0001832ffd2d  mov      qword ptr [rsp + 0x130], r10
0x0001832ffd35  mov      byte ptr [rsp + 0x138], 1
0x0001832ffd3d  lea      r11, [r9 + 5]
0x0001832ffd41  movsx    rdx, byte ptr [rax + 0x17]
0x0001832ffd46  mov      rsi, rdx
0x0001832ffd49  test     rdx, rdx
0x0001832ffd4c  jns      0x1832ffd52
0x0001832ffd4e  mov      rsi, qword ptr [rax + 8]
0x0001832ffd52  lea      r8, [rbx + 0x24c]
0x0001832ffd59  cmp      r11, rsi
0x0001832ffd5c  ja       0x1832ffdb5
0x0001832ffd5e  mov      r10, rax
0x0001832ffd61  test     dl, dl
0x0001832ffd63  jns      0x1832ffd68
0x0001832ffd65  mov      r10, qword ptr [rax]
0x0001832ffd68  mov      edx, dword ptr [r10 + r9 + 1]
0x0001832ffd6d  mov      dword ptr [rbx + 0x24c], edx
0x0001832ffd73  mov      qword ptr [rsp + 0x130], r11
0x0001832ffd7b  mov      byte ptr [rsp + 0x138], 1
0x0001832ffd83  lea      rdx, [r9 + 9]
0x0001832ffd87  movsx    rsi, byte ptr [rax + 0x17]
0x0001832ffd8c  mov      rdi, rsi
0x0001832ffd8f  test     rsi, rsi
0x0001832ffd92  jns      0x1832ffd98
0x0001832ffd94  mov      rdi, qword ptr [rax + 8]
0x0001832ffd98  lea      r10, [rbx + 0x250]
0x0001832ffd9f  cmp      rdx, rdi
0x0001832ffda2  jbe      0x1832ffc72
0x0001832ffda8  mov      byte ptr [rsp + 0x138], 0
0x0001832ffdb0  mov      rdx, r11
0x0001832ffdb3  jmp      0x1832ffde1
0x0001832ffdb5  mov      byte ptr [rsp + 0x138], 0
0x0001832ffdbd  mov      r9, r10
0x0001832ffdc0  jmp      0x1832ffdd7
0x0001832ffdc2  mov      byte ptr [rsp + 0x138], 0
0x0001832ffdca  lea      r8, [rbx + 0x24c]
0x0001832ffdd1  mov      cl, byte ptr [rbx + 0x240]
0x0001832ffdd7  lea      r10, [rbx + 0x250]
0x0001832ffdde  mov      rdx, r9
0x0001832ffde1  lea      r11, [rbx + 0x248]
0x0001832ffde8  mov      rsi, rdx
0x0001832ffdeb  lea      r14, [rbx + 0x25c]
0x0001832ffdf2  lea      r15, [rbx + 0x260]
0x0001832ffdf9  xor      edi, edi
0x0001832ffdfb  mov      rdx, rsi
0x0001832ffdfe  cmp      cl, 1
0x0001832ffe01  jne      0x1832ffea7
0x0001832ffe07  cmp      byte ptr [rbx + 0x48], 0
0x0001832ffe0b  jne      0x1832ffe68
0x0001832ffe0d  mov      ecx, dword ptr [r10]
0x0001832ffe10  test     ecx, ecx
0x0001832ffe12  je       0x1832ffcc4
0x0001832ffe18  mov      r8d, dword ptr [r8]
0x0001832ffe1b  test     r8d, r8d
0x0001832ffe1e  je       0x1832ffcc4
0x0001832ffe24  mov      dword ptr [rbx + 0x254], r8d
0x0001832ffe2b  mov      dword ptr [rbx + 0x258], ecx
0x0001832ffe31  mov      r9d, dword ptr [rbx + 0x244]
0x0001832ffe38  cmp      r8d, r9d
0x0001832ffe3b  jne      0x1832ffe45
0x0001832ffe3d  mov      r9d, r8d
0x0001832ffe40  cmp      ecx, dword ptr [r11]
0x0001832ffe43  je       0x1832ffeb2
0x0001832ffe45  mov      esi, dword ptr [r14]
0x0001832ffe48  test     esi, esi
0x0001832ffe4a  je       0x1832ffeb2
0x0001832ffe4c  mov      r10d, dword ptr [r15]
0x0001832ffe4f  test     r10d, r10d
0x0001832ffe52  je       0x1832ffeb2
0x0001832ffe54  sub      r8d, r9d
0x0001832ffe57  add      r8d, esi
0x0001832ffe5a  mov      dword ptr [r14], r8d
0x0001832ffe5d  add      r10d, ecx
0x0001832ffe60  sub      r10d, dword ptr [r11]
0x0001832ffe63  mov      dword ptr [r15], r10d
0x0001832ffe66  jmp      0x1832ffeb2
0x0001832ffe68  mov      r8d, dword ptr [rbx + 0x244]
0x0001832ffe6f  mov      r9d, dword ptr [rbx + 0x248]
0x0001832ffe76  mov      dword ptr [rbx + 0x254], r8d
0x0001832ffe7d  mov      dword ptr [rbx + 0x258], r9d
0x0001832ffe84  mov      cl, 1
0x0001832ffe86  test     r8d, r8d
0x0001832ffe89  je       0x1832ffeb4
0x0001832ffe8b  test     r9d, r9d
0x0001832ffe8e  je       0x1832ffeb4
0x0001832ffe90  add      r9d, 0xffffff7d
0x0001832ffe97  mov      dword ptr [rbx + 0x264], r9d
0x0001832ffe9e  mov      dword ptr [rbx + 0x268], r8d
0x0001832ffea5  jmp      0x1832ffeb4
0x0001832ffea7  mov      qword ptr [rbx + 0x254], 0
0x0001832ffeb2  xor      ecx, ecx
0x0001832ffeb4  mov      byte ptr [rbx + 0x241], cl
0x0001832ffeba  test     dil, dil
0x0001832ffebd  je       0x1832fed27
0x0001832ffec3  lea      rcx, [rdx + 1]
0x0001832ffec7  movsx    r8, byte ptr [rax + 0x17]
0x0001832ffecc  mov      r9, r8
0x0001832ffecf  test     r8, r8
0x0001832ffed2  movabs   rdi, 0x7fffffffffffffff
0x0001832ffedc  jns      0x1832ffee2
0x0001832ffede  mov      r9, qword ptr [rax + 8]
0x0001832ffee2  mov      r15d, 0x1bd
0x0001832ffee8  cmp      rcx, r9
0x0001832ffeeb  ja       0x1832fed9f
0x0001832ffef1  mov      r9, rax
0x0001832ffef4  test     r8b, r8b
0x0001832ffef7  jns      0x1832ffefc
0x0001832ffef9  mov      r9, qword ptr [rax]
0x0001832ffefc  mov      r8b, byte ptr [r9 + rdx]
0x0001832fff00  mov      byte ptr [rbx + 0x360], r8b
0x0001832fff07  mov      qword ptr [rsp + 0x130], rcx
0x0001832fff0f  mov      byte ptr [rsp + 0x138], 1
0x0001832fff17  lea      rcx, [rdx + 2]
0x0001832fff1b  movsx    r8, byte ptr [rax + 0x17]
0x0001832fff20  mov      r9, r8
0x0001832fff23  test     r8, r8
0x0001832fff26  jns      0x1832fff2c
0x0001832fff28  mov      r9, qword ptr [rax + 8]
0x0001832fff2c  cmp      rcx, r9
0x0001832fff2f  ja       0x1832fed9f
0x0001832fff35  mov      r9, rax
0x0001832fff38  test     r8b, r8b
0x0001832fff3b  jns      0x1832fff40
0x0001832fff3d  mov      r9, qword ptr [rax]
0x0001832fff40  mov      r8b, byte ptr [r9 + rdx + 1]
0x0001832fff45  mov      byte ptr [rbx + 0x361], r8b
0x0001832fff4c  mov      qword ptr [rsp + 0x130], rcx
0x0001832fff54  mov      byte ptr [rsp + 0x138], 1
0x0001832fff5c  lea      rcx, [rdx + 0xa]
0x0001832fff60  movsx    r8, byte ptr [rax + 0x17]
0x0001832fff65  mov      r9, r8
0x0001832fff68  test     r8, r8
0x0001832fff6b  jns      0x1832fff71
0x0001832fff6d  mov      r9, qword ptr [rax + 8]
0x0001832fff71  cmp      rcx, r9
0x0001832fff74  ja       0x1832fed9f
0x0001832fff7a  mov      r9, rax
0x0001832fff7d  test     r8b, r8b
0x0001832fff80  jns      0x1832fff85
0x0001832fff82  mov      r9, qword ptr [rax]
0x0001832fff85  mov      r8, qword ptr [r9 + rdx + 2]
0x0001832fff8a  mov      qword ptr [rbx + 0x368], r8
0x0001832fff91  mov      qword ptr [rsp + 0x130], rcx
0x0001832fff99  mov      byte ptr [rsp + 0x138], 1
0x0001832fffa1  lea      rcx, [rdx + 0x12]
0x0001832fffa5  movsx    r8, byte ptr [rax + 0x17]
0x0001832fffaa  mov      r9, r8
0x0001832fffad  test     r8, r8
0x0001832fffb0  jns      0x1832fffb6
0x0001832fffb2  mov      r9, qword ptr [rax + 8]
0x0001832fffb6  cmp      rcx, r9
0x0001832fffb9  ja       0x1832fed9f
0x0001832fffbf  mov      r9, rax
0x0001832fffc2  test     r8b, r8b
0x0001832fffc5  jns      0x1832fffca
0x0001832fffc7  mov      r9, qword ptr [rax]
0x0001832fffca  mov      r8, qword ptr [r9 + rdx + 0xa]
0x0001832fffcf  mov      qword ptr [rbx + 0x370], r8
0x0001832fffd6  mov      qword ptr [rsp + 0x130], rcx
0x0001832fffde  mov      byte ptr [rsp + 0x138], 1
0x0001832fffe6  lea      rcx, [rdx + 0x1a]
0x0001832fffea  movsx    r8, byte ptr [rax + 0x17]
0x0001832fffef  mov      r9, r8
0x0001832ffff2  test     r8, r8
0x0001832ffff5  jns      0x1832ffffb
0x0001832ffff7  mov      r9, qword ptr [rax + 8]
0x0001832ffffb  cmp      rcx, r9
0x0001832ffffe  ja       0x1832fed9f
0x000183300004  mov      r9, rax
0x000183300007  test     r8b, r8b
0x00018330000a  jns      0x18330000f
0x00018330000c  mov      r9, qword ptr [rax]
0x00018330000f  mov      r8, qword ptr [r9 + rdx + 0x12]
0x000183300014  mov      qword ptr [rbx + 0x378], r8
0x00018330001b  mov      qword ptr [rsp + 0x130], rcx
0x000183300023  mov      byte ptr [rsp + 0x138], 1
0x00018330002b  xorps    xmm0, xmm0
0x00018330002e  movhps   xmm0, qword ptr [rip + 0xc2031eb]   ; 0x18f503220
0x000183300035  mov      word ptr [rbx + 0x4b0], 0x101
0x00018330003e  movups   xmmword ptr [rbx + 0x4b8], xmm0
0x000183300045  movabs   rcx, 0x3ff0000000000000
0x00018330004f  mov      qword ptr [rbx + 0x4c8], rcx
0x000183300056  lea      rcx, [rdx + 0x1b]
0x00018330005a  movsx    r8, byte ptr [rax + 0x17]
0x00018330005f  mov      r9, r8
0x000183300062  test     r8, r8
0x000183300065  jns      0x18330006b
0x000183300067  mov      r9, qword ptr [rax + 8]
0x00018330006b  mov      r15d, 0x1cc
0x000183300071  cmp      rcx, r9
0x000183300074  ja       0x1832fed9f
0x00018330007a  mov      r9, rax
0x00018330007d  test     r8b, r8b
0x000183300080  jns      0x183300085
0x000183300082  mov      r9, qword ptr [rax]
0x000183300085  mov      r8b, byte ptr [r9 + rdx + 0x1a]
0x00018330008a  mov      byte ptr [rbx + 0x5c8], r8b
0x000183300091  mov      qword ptr [rsp + 0x130], rcx
0x000183300099  mov      byte ptr [rsp + 0x138], 1
0x0001833000a1  lea      rcx, [rdx + 0x23]
0x0001833000a5  movsx    r8, byte ptr [rax + 0x17]
0x0001833000aa  mov      r9, r8
0x0001833000ad  test     r8, r8
0x0001833000b0  jns      0x1833000b6
0x0001833000b2  mov      r9, qword ptr [rax + 8]
0x0001833000b6  cmp      rcx, r9
0x0001833000b9  ja       0x1832fed9f
0x0001833000bf  mov      r9, rax
0x0001833000c2  test     r8b, r8b
0x0001833000c5  jns      0x1833000ca
0x0001833000c7  mov      r9, qword ptr [rax]
0x0001833000ca  mov      r8, qword ptr [r9 + rdx + 0x1b]
0x0001833000cf  mov      qword ptr [rbx + 0x5d0], r8
0x0001833000d6  mov      qword ptr [rsp + 0x130], rcx
0x0001833000de  mov      byte ptr [rsp + 0x138], 1
0x0001833000e6  lea      rcx, [rdx + 0x24]
0x0001833000ea  movsx    r8, byte ptr [rax + 0x17]
0x0001833000ef  mov      r9, r8
0x0001833000f2  test     r8, r8
0x0001833000f5  jns      0x1833000fb
0x0001833000f7  mov      r9, qword ptr [rax + 8]
0x0001833000fb  cmp      rcx, r9
0x0001833000fe  ja       0x183300c51
0x000183300104  mov      r9, rax
0x000183300107  test     r8b, r8b
0x00018330010a  jns      0x18330010f
0x00018330010c  mov      r9, qword ptr [rax]
0x00018330010f  mov      r8b, byte ptr [r9 + rdx + 0x23]
0x000183300114  mov      byte ptr [rbx + 0x588], r8b
0x00018330011b  mov      qword ptr [rsp + 0x130], rcx
0x000183300123  mov      byte ptr [rsp + 0x138], 1
0x00018330012b  lea      rcx, [rdx + 0x2c]
0x00018330012f  movsx    r8, byte ptr [rax + 0x17]
0x000183300134  mov      r9, r8
0x000183300137  test     r8, r8
0x00018330013a  jns      0x183300140
0x00018330013c  mov      r9, qword ptr [rax + 8]
0x000183300140  cmp      rcx, r9
0x000183300143  ja       0x183300c51
0x000183300149  mov      r9, rax
0x00018330014c  test     r8b, r8b
0x00018330014f  jns      0x183300154
0x000183300151  mov      r9, qword ptr [rax]
0x000183300154  mov      r8, qword ptr [r9 + rdx + 0x24]
0x000183300159  mov      qword ptr [rsp + 0x130], rcx
0x000183300161  mov      byte ptr [rsp + 0x138], 1
0x000183300169  mov      qword ptr [rbx + 0x590], r8
0x000183300170  lea      rcx, [rdx + 0x2d]
0x000183300174  movsx    r8, byte ptr [rax + 0x17]
0x000183300179  mov      r9, r8
0x00018330017c  test     r8, r8
0x00018330017f  jns      0x183300185
0x000183300181  mov      r9, qword ptr [rax + 8]
0x000183300185  cmp      rcx, r9
0x000183300188  ja       0x1832fed49
0x00018330018e  mov      r9, rax
0x000183300191  test     r8b, r8b
0x000183300194  jns      0x183300199
0x000183300196  mov      r9, qword ptr [rax]
0x000183300199  mov      r8b, byte ptr [r9 + rdx + 0x2c]
0x00018330019e  mov      byte ptr [rbx + 0x6c9], r8b
0x0001833001a5  mov      qword ptr [rsp + 0x130], rcx
0x0001833001ad  mov      byte ptr [rsp + 0x138], 1
0x0001833001b5  lea      rcx, [rdx + 0x2e]
0x0001833001b9  movsx    r8, byte ptr [rax + 0x17]
0x0001833001be  mov      r9, r8
0x0001833001c1  test     r8, r8
0x0001833001c4  jns      0x1833001ca
0x0001833001c6  mov      r9, qword ptr [rax + 8]
0x0001833001ca  cmp      rcx, r9
0x0001833001cd  ja       0x1832fed49
0x0001833001d3  mov      r9, rax
0x0001833001d6  test     r8b, r8b
0x0001833001d9  jns      0x1833001de
0x0001833001db  mov      r9, qword ptr [rax]
0x0001833001de  mov      r8b, byte ptr [r9 + rdx + 0x2d]
0x0001833001e3  mov      byte ptr [rbx + 0x6ca], r8b
0x0001833001ea  mov      qword ptr [rsp + 0x130], rcx
0x0001833001f2  mov      byte ptr [rsp + 0x138], 1
0x0001833001fa  lea      rcx, [rdx + 0x2f]
0x0001833001fe  movsx    r8, byte ptr [rax + 0x17]
0x000183300203  mov      r9, r8
0x000183300206  test     r8, r8
0x000183300209  jns      0x18330020f
0x00018330020b  mov      r9, qword ptr [rax + 8]
0x00018330020f  cmp      rcx, r9
0x000183300212  ja       0x1832fed49
0x000183300218  mov      r9, rax
0x00018330021b  test     r8b, r8b
0x00018330021e  jns      0x183300223
0x000183300220  mov      r9, qword ptr [rax]
0x000183300223  mov      r8b, byte ptr [r9 + rdx + 0x2e]
0x000183300228  mov      byte ptr [rbx + 0x6cb], r8b
0x00018330022f  mov      qword ptr [rsp + 0x130], rcx
0x000183300237  mov      byte ptr [rsp + 0x138], 1
0x00018330023f  lea      rcx, [rdx + 0x30]
0x000183300243  movsx    r8, byte ptr [rax + 0x17]
0x000183300248  mov      r9, r8
0x00018330024b  test     r8, r8
0x00018330024e  jns      0x183300254
0x000183300250  mov      r9, qword ptr [rax + 8]
0x000183300254  cmp      rcx, r9
0x000183300257  ja       0x1832fed49
0x00018330025d  mov      r9, rax
0x000183300260  test     r8b, r8b
0x000183300263  jns      0x183300268
0x000183300265  mov      r9, qword ptr [rax]
0x000183300268  mov      r8b, byte ptr [r9 + rdx + 0x2f]
0x00018330026d  mov      byte ptr [rbx + 0x6cc], r8b
0x000183300274  mov      qword ptr [rsp + 0x130], rcx
0x00018330027c  mov      byte ptr [rsp + 0x138], 1
0x000183300284  lea      rcx, [rdx + 0x31]
0x000183300288  movsx    r8, byte ptr [rax + 0x17]
0x00018330028d  mov      r9, r8
0x000183300290  test     r8, r8
0x000183300293  jns      0x183300299
0x000183300295  mov      r9, qword ptr [rax + 8]
0x000183300299  cmp      rcx, r9
0x00018330029c  ja       0x1832fed49
0x0001833002a2  mov      r9, rax
0x0001833002a5  test     r8b, r8b
0x0001833002a8  jns      0x1833002ad
0x0001833002aa  mov      r9, qword ptr [rax]
0x0001833002ad  mov      r8b, byte ptr [r9 + rdx + 0x30]
0x0001833002b2  mov      byte ptr [rbx + 0x6c8], r8b
0x0001833002b9  mov      qword ptr [rsp + 0x130], rcx
0x0001833002c1  mov      byte ptr [rsp + 0x138], 1
0x0001833002c9  lea      rcx, [rdx + 0x32]
0x0001833002cd  movsx    r8, byte ptr [rax + 0x17]
0x0001833002d2  mov      r9, r8
0x0001833002d5  test     r8, r8
0x0001833002d8  jns      0x1833002de
0x0001833002da  mov      r9, qword ptr [rax + 8]
0x0001833002de  cmp      rcx, r9
0x0001833002e1  ja       0x1832fed49
0x0001833002e7  mov      r9, rax
0x0001833002ea  test     r8b, r8b
0x0001833002ed  jns      0x1833002f2
0x0001833002ef  mov      r9, qword ptr [rax]
0x0001833002f2  mov      r8b, byte ptr [r9 + rdx + 0x31]
0x0001833002f7  mov      byte ptr [rbx + 0x6d1], r8b
0x0001833002fe  mov      qword ptr [rsp + 0x130], rcx
0x000183300306  mov      byte ptr [rsp + 0x138], 1
0x00018330030e  lea      rcx, [rdx + 0x33]
0x000183300312  movsx    r8, byte ptr [rax + 0x17]
0x000183300317  mov      r9, r8
0x00018330031a  test     r8, r8
0x00018330031d  jns      0x183300323
0x00018330031f  mov      r9, qword ptr [rax + 8]
0x000183300323  cmp      rcx, r9
0x000183300326  ja       0x1832fed49
0x00018330032c  mov      r9, rax
0x00018330032f  test     r8b, r8b
0x000183300332  jns      0x183300337
0x000183300334  mov      r9, qword ptr [rax]
0x000183300337  mov      r8b, byte ptr [r9 + rdx + 0x32]
0x00018330033c  mov      byte ptr [rbx + 0x6cd], r8b
0x000183300343  mov      qword ptr [rsp + 0x130], rcx
0x00018330034b  mov      byte ptr [rsp + 0x138], 1
0x000183300353  lea      rcx, [rdx + 0x34]
0x000183300357  movsx    r8, byte ptr [rax + 0x17]
0x00018330035c  mov      r9, r8
0x00018330035f  test     r8, r8
0x000183300362  jns      0x183300368
0x000183300364  mov      r9, qword ptr [rax + 8]
0x000183300368  cmp      rcx, r9
0x00018330036b  ja       0x1832fed49
0x000183300371  mov      r9, rax
0x000183300374  test     r8b, r8b
0x000183300377  jns      0x18330037c
0x000183300379  mov      r9, qword ptr [rax]
0x00018330037c  mov      r8b, byte ptr [r9 + rdx + 0x33]
0x000183300381  mov      byte ptr [rbx + 0x6ce], r8b
0x000183300388  mov      qword ptr [rsp + 0x130], rcx
0x000183300390  mov      byte ptr [rsp + 0x138], 1
0x000183300398  add      rdx, 0x35
0x00018330039c  movsx    r8, byte ptr [rax + 0x17]
0x0001833003a1  mov      r9, r8
0x0001833003a4  test     r8, r8
0x0001833003a7  jns      0x1833003ad
0x0001833003a9  mov      r9, qword ptr [rax + 8]
0x0001833003ad  cmp      rdx, r9
0x0001833003b0  ja       0x1832fed49
0x0001833003b6  test     r8b, r8b
0x0001833003b9  jns      0x1833003be
0x0001833003bb  mov      rax, qword ptr [rax]
0x0001833003be  mov      al, byte ptr [rax + rcx]
0x0001833003c1  mov      byte ptr [rbx + 0x6d0], al
0x0001833003c7  lea      rcx, [rsp + 0x130]
0x0001833003cf  mov      qword ptr [rcx], rdx
0x0001833003d2  mov      byte ptr [rcx + 8], 1
0x0001833003d6  mov      al, byte ptr [rbp + 0x54]
0x0001833003d9  mov      byte ptr [rbx + 0x6cf], al
0x0001833003df  lea      rdx, [rbx + 0x638]
0x0001833003e6  call     0x183303050
0x0001833003eb  lea      rdx, [rbx + 0x5f0]
0x0001833003f2  mov      rcx, rax
0x0001833003f5  call     0x183303050
0x0001833003fa  lea      rdx, [rbx + 0x608]
0x000183300401  mov      rcx, rax
0x000183300404  call     0x183303050
0x000183300409  lea      rdx, [rbx + 0x620]
0x000183300410  mov      rcx, rax
0x000183300413  call     0x183303050
0x000183300418  cmp      byte ptr [rax + 8], 1
0x00018330041c  jne      0x18330045b
0x00018330041e  mov      rdx, qword ptr [rax]
0x000183300421  mov      r8, qword ptr [rax + 0x10]
0x000183300425  lea      rcx, [rdx + 1]
0x000183300429  movsx    r10, byte ptr [r8 + 0x17]
0x00018330042e  mov      r9, r10
0x000183300431  test     r10, r10
0x000183300434  jns      0x18330043a
0x000183300436  mov      r9, qword ptr [r8 + 8]
0x00018330043a  cmp      rcx, r9
0x00018330043d  ja       0x183300454
0x00018330043f  test     r10b, r10b
0x000183300442  jns      0x183300447
0x000183300444  mov      r8, qword ptr [r8]
0x000183300447  mov      dl, byte ptr [r8 + rdx]
0x00018330044b  mov      byte ptr [rbx + 0x5e8], dl
0x000183300451  inc      qword ptr [rax]
0x000183300454  cmp      rcx, r9
0x000183300457  setbe    byte ptr [rax + 8]
0x00018330045b  cmp      byte ptr [rsp + 0x138], 1
0x000183300463  jne      0x183300c8a
0x000183300469  mov      rax, qword ptr [rsp + 0x130]
0x000183300471  mov      rdx, qword ptr [rsp + 0x140]
0x000183300479  lea      rcx, [rax + 1]
0x00018330047d  movsx    r8, byte ptr [rdx + 0x17]
0x000183300482  mov      r9, r8
0x000183300485  test     r8, r8
0x000183300488  jns      0x18330048e
0x00018330048a  mov      r9, qword ptr [rdx + 8]
0x00018330048e  cmp      rcx, r9
0x000183300491  ja       0x183300cc2
0x000183300497  test     r8b, r8b
0x00018330049a  jns      0x18330049f
0x00018330049c  mov      rdx, qword ptr [rdx]
0x00018330049f  movzx    eax, byte ptr [rdx + rax]
0x0001833004a3  lea      r14, [rsp + 0x130]
0x0001833004ab  mov      qword ptr [r14], rcx
0x0001833004ae  mov      byte ptr [r14 + 8], 1
0x0001833004b3  mov      dword ptr [rbx + 0x650], eax
0x0001833004b9  lea      rdx, [rbx + 0x6b0]
0x0001833004c0  mov      rcx, r14
0x0001833004c3  call     0x183303050
0x0001833004c8  cmp      byte ptr [r14 + 8], 1
0x0001833004cd  jne      0x183300cb7
0x0001833004d3  cmp      dword ptr [rbx + 0x65c], 4
0x0001833004da  jne      0x183300522
0x0001833004dc  mov      eax, dword ptr [rbx + 0x2d8]
0x0001833004e2  test     eax, eax
0x0001833004e4  setne    cl
0x0001833004e7  cmp      eax, 3
0x0001833004ea  setne    dl
0x0001833004ed  and      dl, cl
0x0001833004ef  mov      byte ptr [rbx + 0x322], dl
0x0001833004f5  cmp      eax, 3
0x0001833004f8  je       0x183300529
0x0001833004fa  test     eax, eax
0x0001833004fc  je       0x183300529
0x0001833004fe  mov      byte ptr [rbx + 0x320], 0
0x000183300505  mov      al, byte ptr [rsp + 0x138]
0x00018330050c  mov      byte ptr [rbx + 0x658], 1
0x000183300513  cmp      al, 1
0x000183300515  je       0x183300530
0x000183300517  mov      r15d, 0x68
0x00018330051d  jmp      0x1832feda7
0x000183300522  mov      byte ptr [rbx + 0x322], 0
0x000183300529  mov      byte ptr [rbx + 0x658], 0
0x000183300530  mov      rax, qword ptr [rsp + 0x130]
0x000183300538  mov      rcx, qword ptr [rsp + 0x140]
0x000183300540  lea      r9, [rax + 1]
0x000183300544  movsx    rdx, byte ptr [rcx + 0x17]
0x000183300549  mov      r8, rdx
0x00018330054c  test     rdx, rdx
0x00018330054f  jns      0x183300555
0x000183300551  mov      r8, qword ptr [rcx + 8]
0x000183300555  mov      r15d, 0x68
0x00018330055b  cmp      r9, r8
0x00018330055e  ja       0x1832fed9f
0x000183300564  mov      r8, rcx
0x000183300567  test     dl, dl
0x000183300569  jns      0x18330056e
0x00018330056b  mov      r8, qword ptr [rcx]
0x00018330056e  movzx    r10d, byte ptr [r8 + rax]
0x000183300573  mov      qword ptr [rsp + 0x130], r9
0x00018330057b  mov      byte ptr [rsp + 0x138], 1
0x000183300583  cmp      r10d, 1
0x000183300587  je       0x183300598
0x000183300589  mov      r8d, edx
0x00018330058c  cmp      r10d, 0xff
0x000183300593  jne      0x1833005a7
0x000183300595  xor      r10d, r10d
0x000183300598  mov      byte ptr [rbx + 0x90], r10b
0x00018330059f  mov      r8b, byte ptr [rcx + 0x17]
0x0001833005a3  movzx    edx, r8b
0x0001833005a7  lea      r10, [rax + 2]
0x0001833005ab  mov      r11, rdx
0x0001833005ae  test     r8b, r8b
0x0001833005b1  jns      0x1833005b7
0x0001833005b3  mov      r11, qword ptr [rcx + 8]
0x0001833005b7  cmp      r10, r11
0x0001833005ba  ja       0x1832fed9f
0x0001833005c0  mov      r11, rcx
0x0001833005c3  test     r8b, r8b
0x0001833005c6  jns      0x1833005cb
0x0001833005c8  mov      r11, qword ptr [rcx]
0x0001833005cb  movzx    r9d, byte ptr [r11 + r9]
0x0001833005d0  mov      qword ptr [rsp + 0x130], r10
0x0001833005d8  mov      byte ptr [rsp + 0x138], 1
0x0001833005e0  cmp      r9d, 1
0x0001833005e4  je       0x1833005f2
0x0001833005e6  cmp      r9d, 0xff
0x0001833005ed  jne      0x183300601
0x0001833005ef  xor      r9d, r9d
0x0001833005f2  mov      byte ptr [rbx + 0x288], r9b
0x0001833005f9  mov      r8b, byte ptr [rcx + 0x17]
0x0001833005fd  movzx    edx, r8b
0x000183300601  lea      r9, [rax + 3]
0x000183300605  mov      r11, rdx
0x000183300608  test     r8b, r8b
0x00018330060b  jns      0x183300611
0x00018330060d  mov      r11, qword ptr [rcx + 8]
0x000183300611  cmp      r9, r11
0x000183300614  ja       0x1832fed9f
0x00018330061a  mov      r11, rcx
0x00018330061d  test     r8b, r8b
0x000183300620  jns      0x183300625
0x000183300622  mov      r11, qword ptr [rcx]
0x000183300625  movzx    r10d, byte ptr [r11 + r10]
0x00018330062a  mov      qword ptr [rsp + 0x130], r9
0x000183300632  mov      byte ptr [rsp + 0x138], 1
0x00018330063a  cmp      r10d, 1
0x00018330063e  je       0x18330064c
0x000183300640  cmp      r10d, 0xff
0x000183300647  jne      0x18330065b
0x000183300649  xor      r10d, r10d
0x00018330064c  mov      byte ptr [rbx + 0x4b0], r10b
0x000183300653  mov      r8b, byte ptr [rcx + 0x17]
0x000183300657  movzx    edx, r8b
0x00018330065b  lea      r10, [rax + 4]
0x00018330065f  mov      r11, rdx
0x000183300662  test     r8b, r8b
0x000183300665  jns      0x18330066b
0x000183300667  mov      r11, qword ptr [rcx + 8]
0x00018330066b  cmp      r10, r11
0x00018330066e  ja       0x1832fed9f
0x000183300674  mov      r11, rcx
0x000183300677  test     r8b, r8b
0x00018330067a  jns      0x18330067f
0x00018330067c  mov      r11, qword ptr [rcx]
0x00018330067f  movzx    r9d, byte ptr [r11 + r9]
0x000183300684  mov      qword ptr [rsp + 0x130], r10
0x00018330068c  mov      byte ptr [rsp + 0x138], 1
0x000183300694  cmp      r9d, 1
0x000183300698  je       0x1833006a6
0x00018330069a  cmp      r9d, 0xff
0x0001833006a1  jne      0x1833006b5
0x0001833006a3  xor      r9d, r9d
0x0001833006a6  mov      byte ptr [rbx + 0x4f8], r9b
0x0001833006ad  mov      r8b, byte ptr [rcx + 0x17]
0x0001833006b1  movzx    edx, r8b
0x0001833006b5  lea      r9, [rax + 5]
0x0001833006b9  mov      r11, rdx
0x0001833006bc  test     r8b, r8b
0x0001833006bf  jns      0x1833006c5
0x0001833006c1  mov      r11, qword ptr [rcx + 8]
0x0001833006c5  cmp      r9, r11
0x0001833006c8  ja       0x1832fed9f
0x0001833006ce  mov      r11, rcx
0x0001833006d1  test     r8b, r8b
0x0001833006d4  jns      0x1833006d9
0x0001833006d6  mov      r11, qword ptr [rcx]
0x0001833006d9  movzx    r10d, byte ptr [r11 + r10]
0x0001833006de  mov      qword ptr [rsp + 0x130], r9
0x0001833006e6  mov      byte ptr [rsp + 0x138], 1
0x0001833006ee  cmp      r10d, 1
0x0001833006f2  je       0x183300700
0x0001833006f4  cmp      r10d, 0xff
0x0001833006fb  jne      0x18330070f
0x0001833006fd  xor      r10d, r10d
0x000183300700  mov      byte ptr [rbx + 0x588], r10b
0x000183300707  mov      r8b, byte ptr [rcx + 0x17]
0x00018330070b  movzx    edx, r8b
0x00018330070f  lea      r10, [rax + 6]
0x000183300713  mov      r11, rdx
0x000183300716  test     r8b, r8b
0x000183300719  jns      0x18330071f
0x00018330071b  mov      r11, qword ptr [rcx + 8]
0x00018330071f  cmp      r10, r11
0x000183300722  ja       0x1832fed9f
0x000183300728  mov      r11, rcx
0x00018330072b  test     r8b, r8b
0x00018330072e  jns      0x183300733
0x000183300730  mov      r11, qword ptr [rcx]
0x000183300733  movzx    r9d, byte ptr [r11 + r9]
0x000183300738  mov      qword ptr [rsp + 0x130], r10
0x000183300740  mov      byte ptr [rsp + 0x138], 1
0x000183300748  cmp      r9d, 1
0x00018330074c  je       0x18330075a
0x00018330074e  cmp      r9d, 0xff
0x000183300755  jne      0x183300769
0x000183300757  xor      r9d, r9d
0x00018330075a  mov      byte ptr [rbx + 0x5c8], r9b
0x000183300761  mov      r8b, byte ptr [rcx + 0x17]
0x000183300765  movzx    edx, r8b
0x000183300769  lea      r11, [rax + 7]
0x00018330076d  mov      r9, rdx
0x000183300770  test     r8b, r8b
0x000183300773  jns      0x183300779
0x000183300775  mov      r9, qword ptr [rcx + 8]
0x000183300779  cmp      r11, r9
0x00018330077c  ja       0x1832fed9f
0x000183300782  mov      r9, rcx
0x000183300785  test     r8b, r8b
0x000183300788  jns      0x18330078d
0x00018330078a  mov      r9, qword ptr [rcx]
0x00018330078d  movzx    r9d, byte ptr [r9 + r10]
0x000183300792  mov      qword ptr [rsp + 0x130], r11
0x00018330079a  mov      byte ptr [rsp + 0x138], 1
0x0001833007a2  cmp      r9d, 1
0x0001833007a6  je       0x1833007b4
0x0001833007a8  cmp      r9d, 0xff
0x0001833007af  jne      0x1833007c3
0x0001833007b1  xor      r9d, r9d
0x0001833007b4  mov      byte ptr [rbx + 0x6c8], r9b
0x0001833007bb  mov      r8b, byte ptr [rcx + 0x17]
0x0001833007bf  movzx    edx, r8b
0x0001833007c3  lea      r9, [rax + 8]
0x0001833007c7  mov      r10, rdx
0x0001833007ca  test     r8b, r8b
0x0001833007cd  jns      0x1833007d3
0x0001833007cf  mov      r10, qword ptr [rcx + 8]
0x0001833007d3  cmp      r9, r10
0x0001833007d6  ja       0x1832fed9f
0x0001833007dc  mov      r10, rcx
0x0001833007df  test     r8b, r8b
0x0001833007e2  jns      0x1833007e7
0x0001833007e4  mov      r10, qword ptr [rcx]
0x0001833007e7  movzx    r10d, byte ptr [r10 + r11]
0x0001833007ec  mov      qword ptr [rsp + 0x130], r9
0x0001833007f4  mov      byte ptr [rsp + 0x138], 1
0x0001833007fc  cmp      r10d, 1
0x000183300800  je       0x18330080e
0x000183300802  cmp      r10d, 0xff
0x000183300809  jne      0x18330081d
0x00018330080b  xor      r10d, r10d
0x00018330080e  mov      byte ptr [rbx + 0x6c9], r10b
0x000183300815  mov      r8b, byte ptr [rcx + 0x17]
0x000183300819  movzx    edx, r8b
0x00018330081d  add      rax, 9
0x000183300821  mov      r10, rdx
0x000183300824  test     r8b, r8b
0x000183300827  jns      0x18330082d
0x000183300829  mov      r10, qword ptr [rcx + 8]
0x00018330082d  cmp      rax, r10
0x000183300830  ja       0x1832fed9f
0x000183300836  mov      r10, rcx
0x000183300839  test     r8b, r8b
0x00018330083c  jns      0x183300841
0x00018330083e  mov      r10, qword ptr [rcx]
0x000183300841  movzx    r9d, byte ptr [r10 + r9]
0x000183300846  mov      qword ptr [rsp + 0x130], rax
0x00018330084e  mov      byte ptr [rsp + 0x138], 1
0x000183300856  cmp      r9d, 1
0x00018330085a  je       0x183300868
0x00018330085c  cmp      r9d, 0xff
0x000183300863  jne      0x183300895
0x000183300865  xor      r9d, r9d
0x000183300868  mov      byte ptr [rbx + 0x6ca], r9b
0x00018330086f  cmp      byte ptr [rsp + 0x138], 1
0x000183300877  jne      0x1832feda7
0x00018330087d  mov      rax, qword ptr [rsp + 0x130]
0x000183300885  mov      rcx, qword ptr [rsp + 0x140]
0x00018330088d  mov      r8b, byte ptr [rcx + 0x17]
0x000183300891  movzx    edx, r8b
0x000183300895  lea      r9, [rax + 1]
0x000183300899  mov      r10, rdx
0x00018330089c  test     r8b, r8b
0x00018330089f  jns      0x1833008a5
0x0001833008a1  mov      r10, qword ptr [rcx + 8]
0x0001833008a5  cmp      r9, r10
0x0001833008a8  ja       0x1832fed9f
0x0001833008ae  mov      r10, rcx
0x0001833008b1  test     r8b, r8b
0x0001833008b4  jns      0x1833008b9
0x0001833008b6  mov      r10, qword ptr [rcx]
0x0001833008b9  movzx    r10d, byte ptr [r10 + rax]
0x0001833008be  mov      qword ptr [rsp + 0x130], r9
0x0001833008c6  mov      byte ptr [rsp + 0x138], 1
0x0001833008ce  cmp      r10d, 1
0x0001833008d2  je       0x1833008e0
0x0001833008d4  cmp      r10d, 0xff
0x0001833008db  jne      0x1833008ef
0x0001833008dd  xor      r10d, r10d
0x0001833008e0  mov      byte ptr [rbx + 0x6cb], r10b
0x0001833008e7  mov      r8b, byte ptr [rcx + 0x17]
0x0001833008eb  movzx    edx, r8b
0x0001833008ef  lea      r10, [rax + 2]
0x0001833008f3  mov      r11, rdx
0x0001833008f6  test     r8b, r8b
0x0001833008f9  jns      0x1833008ff
0x0001833008fb  mov      r11, qword ptr [rcx + 8]
0x0001833008ff  cmp      r10, r11
0x000183300902  ja       0x1832fed9f
0x000183300908  mov      r11, rcx
0x00018330090b  test     r8b, r8b
0x00018330090e  jns      0x183300913
0x000183300910  mov      r11, qword ptr [rcx]
0x000183300913  movzx    r9d, byte ptr [r11 + r9]
0x000183300918  mov      qword ptr [rsp + 0x130], r10
0x000183300920  mov      byte ptr [rsp + 0x138], 1
0x000183300928  cmp      r9d, 1
0x00018330092c  je       0x18330093a
0x00018330092e  cmp      r9d, 0xff
0x000183300935  jne      0x183300949
0x000183300937  xor      r9d, r9d
0x00018330093a  mov      byte ptr [rbx + 0x6cc], r9b
0x000183300941  mov      r8b, byte ptr [rcx + 0x17]
0x000183300945  movzx    edx, r8b
0x000183300949  lea      r9, [rax + 3]
0x00018330094d  mov      r11, rdx
0x000183300950  test     r8b, r8b
0x000183300953  jns      0x183300959
0x000183300955  mov      r11, qword ptr [rcx + 8]
0x000183300959  cmp      r9, r11
0x00018330095c  ja       0x1832fed9f
0x000183300962  mov      r11, rcx
0x000183300965  test     r8b, r8b
0x000183300968  jns      0x18330096d
0x00018330096a  mov      r11, qword ptr [rcx]
0x00018330096d  movzx    r10d, byte ptr [r11 + r10]
0x000183300972  mov      qword ptr [rsp + 0x130], r9
0x00018330097a  mov      byte ptr [rsp + 0x138], 1
0x000183300982  cmp      r10d, 1
0x000183300986  je       0x183300994
0x000183300988  cmp      r10d, 0xff
0x00018330098f  jne      0x1833009a3
0x000183300991  xor      r10d, r10d
0x000183300994  mov      byte ptr [rbx + 0x320], r10b
0x00018330099b  mov      r8b, byte ptr [rcx + 0x17]
0x00018330099f  movzx    edx, r8b
0x0001833009a3  add      rax, 7
0x0001833009a7  test     r8b, r8b
0x0001833009aa  jns      0x1833009b0
0x0001833009ac  mov      rdx, qword ptr [rcx + 8]
0x0001833009b0  cmp      rax, rdx
0x0001833009b3  ja       0x183300a64
0x0001833009b9  test     r8b, r8b
0x0001833009bc  jns      0x1833009c1
0x0001833009be  mov      rcx, qword ptr [rcx]
0x0001833009c1  mov      esi, dword ptr [rcx + r9]
0x0001833009c5  mov      qword ptr [rsp + 0x130], rax
0x0001833009cd  mov      byte ptr [rsp + 0x138], 1
0x0001833009d5  test     esi, esi
0x0001833009d7  jle      0x1832fd4ac
0x0001833009dd  lea      r14, [rbx + 0x308]
0x0001833009e4  xorps    xmm6, xmm6
0x0001833009e7  lea      r15, [rsp + 0x130]
0x0001833009ef  lea      r12, [rsp + 0x1c0]
0x0001833009f7  jmp      0x183300a15
0x0001833009f9  mov      rdx, qword ptr [rsp + 0x1d0]
0x000183300a01  and      rdx, rdi
0x000183300a04  mov      rcx, qword ptr [rsp + 0x1c0]
0x000183300a0c  call     0x183b135e0
0x000183300a11  dec      esi
0x000183300a13  je       0x183300a4b
0x000183300a15  movaps   xmmword ptr [rsp + 0x1c0], xmm6
0x000183300a1d  mov      qword ptr [rsp + 0x1d0], 0
0x000183300a29  mov      rcx, r15
0x000183300a2c  mov      rdx, r12
0x000183300a2f  call     0x183303050
0x000183300a34  mov      rcx, r14
0x000183300a37  mov      rdx, r12
0x000183300a3a  call     0x1843bd7e0
0x000183300a3f  cmp      byte ptr [rsp + 0x1d7], 0
0x000183300a47  jns      0x183300a11
0x000183300a49  jmp      0x1833009f9
0x000183300a4b  cmp      byte ptr [rsp + 0x138], 0
0x000183300a53  jne      0x1832fd4ac
0x000183300a59  mov      r15d, 0x71
0x000183300a5f  jmp      0x1832feda7
0x000183300a64  mov      r15d, 0x71
0x000183300a6a  jmp      0x1832fed9f
0x000183300a6f  mov      rdx, qword ptr [rbx + 0x290]
0x000183300a76  mov      r8, qword ptr [rbx + 0x298]
0x000183300a7d  lea      rcx, [rsp + 0x1c0]
0x000183300a85  call     0x183b13af0
0x000183300a8a  movsx    rax, byte ptr [rbx + 0x2a7]
0x000183300a92  movaps   xmmword ptr [rsp + 0x1a0], xmm6
0x000183300a9a  mov      qword ptr [rsp + 0x1b0], 0
0x000183300aa6  test     rax, rax
0x000183300aa9  jns      0x1832fe84a
0x000183300aaf  mov      r14, qword ptr [rbx + 0x290]
0x000183300ab6  mov      rax, qword ptr [rbx + 0x298]
0x000183300abd  jmp      0x1832fe84a
0x000183300ac2  mov      rdx, qword ptr [r14 + rdi + 0x10]
0x000183300ac7  movabs   rax, 0x7fffffffffffffff
0x000183300ad1  and      rdx, rax
0x000183300ad4  mov      rcx, qword ptr [r14 + rdi]
0x000183300ad8  call     0x183b135e0
0x000183300add  jmp      0x183300b0f
0x000183300adf  mov      rax, qword ptr [rsp + 0x1a8]
0x000183300ae7  shl      rax, 3
0x000183300aeb  lea      rdi, [rax + rax*2]
0x000183300aef  mov      rcx, rsi
0x000183300af2  test     rdi, rdi
0x000183300af5  je       0x183300b1d
0x000183300af7  lea      r14, [rsi - 0x18]
0x000183300afb  mov      rax, r14
0x000183300afe  add      rax, rdi
0x000183300b01  je       0x1832ff346
0x000183300b07  cmp      byte ptr [r14 + rdi + 0x17], 0
0x000183300b0d  js       0x183300ac2
0x000183300b0f  add      rdi, -0x18
0x000183300b13  jne      0x183300afb
0x000183300b15  mov      rcx, qword ptr [rsp + 0x1a0]
0x000183300b1d  sub      rsi, rcx
0x000183300b20  sar      rsi, 3
0x000183300b24  movabs   rax, 0xaaaaaaaaaaaaaaab
0x000183300b2e  imul     rax, rsi
0x000183300b32  mov      qword ptr [rsp + 0x1a8], rax
0x000183300b3a  mov      rax, qword ptr [rsp + 0x1b0]
0x000183300b42  shl      rax, 3
0x000183300b46  lea      rdx, [rax + rax*2]
0x000183300b4a  call     0x183b135e0
0x000183300b4f  movabs   rdi, 0x7fffffffffffffff
0x000183300b59  jmp      0x1832feac9
0x000183300b5e  mov      r15d, 0x154
0x000183300b64  jmp      0x1832fed2d
0x000183300b69  mov      rdx, qword ptr [r14 + rdi + 0x10]
0x000183300b6e  movabs   rax, 0x7fffffffffffffff
0x000183300b78  and      rdx, rax
0x000183300b7b  mov      rcx, qword ptr [r14 + rdi]
0x000183300b7f  call     0x183b135e0
0x000183300b84  jmp      0x183300bb6
0x000183300b86  mov      rax, qword ptr [rsp + 0x188]
0x000183300b8e  shl      rax, 3
0x000183300b92  lea      rdi, [rax + rax*2]
0x000183300b96  mov      rcx, rsi
0x000183300b99  test     rdi, rdi
0x000183300b9c  je       0x183300bc4
0x000183300b9e  lea      r14, [rsi - 0x18]
0x000183300ba2  mov      rax, r14
0x000183300ba5  add      rax, rdi
0x000183300ba8  je       0x1832ff346
0x000183300bae  cmp      byte ptr [r14 + rdi + 0x17], 0
0x000183300bb4  js       0x183300b69
0x000183300bb6  add      rdi, -0x18
0x000183300bba  jne      0x183300ba2
0x000183300bbc  mov      rcx, qword ptr [rsp + 0x180]
0x000183300bc4  sub      rsi, rcx
0x000183300bc7  sar      rsi, 3
0x000183300bcb  movabs   rax, 0xaaaaaaaaaaaaaaab
0x000183300bd5  imul     rax, rsi
0x000183300bd9  mov      qword ptr [rsp + 0x188], rax
0x000183300be1  mov      rax, qword ptr [rsp + 0x190]
0x000183300be9  shl      rax, 3
0x000183300bed  lea      rdx, [rax + rax*2]
0x000183300bf1  call     0x183b135e0
0x000183300bf6  movabs   rdi, 0x7fffffffffffffff
0x000183300c00  jmp      0x1832fea7b
0x000183300c05  mov      r15d, 0xc3
0x000183300c0b  jmp      0x1832feda7
0x000183300c10  mov      rdx, qword ptr [rbx + 0x68]
0x000183300c14  movabs   rax, 0x7fffffffffffffff
0x000183300c1e  and      rdx, rax
0x000183300c21  mov      rcx, qword ptr [rbx + 0x58]
0x000183300c25  call     0x183b135e0
0x000183300c2a  jmp      0x1832fd3e6
0x000183300c2f  mov      rdx, qword ptr [rbx + 0x88]
0x000183300c36  movabs   rax, 0x7fffffffffffffff
0x000183300c40  and      rdx, rax
0x000183300c43  mov      rcx, qword ptr [rbx + 0x78]
0x000183300c47  call     0x183b135e0
0x000183300c4c  jmp      0x1832fd427
0x000183300c51  mov      byte ptr [rsp + 0x138], 0
0x000183300c59  mov      qword ptr [rbx + 0x590], 0
0x000183300c64  mov      r15d, 0x1d4
0x000183300c6a  jmp      0x1832feda7
0x000183300c6f  mov      rdx, qword ptr [rbx + 0x120]
0x000183300c76  and      rdx, rdi
0x000183300c79  mov      rcx, qword ptr [rbx + 0x110]
0x000183300c80  call     0x183b135e0
0x000183300c85  jmp      0x1832fd98d
0x000183300c8a  mov      r15d, 0x1eb
0x000183300c90  jmp      0x1832feda7
0x000183300c95  test     r8b, r8b
0x000183300c98  cmovs    r8, qword ptr [rsp + 0x1c8]
0x000183300ca1  cmovs    rbx, qword ptr [rsp + 0x1c0]
0x000183300caa  mov      rdx, rbx
0x000183300cad  call     0x183b61fa0
0x000183300cb2  jmp      0x1832ff086
0x000183300cb7  mov      r15d, 0x1f7
0x000183300cbd  jmp      0x1832feda7
0x000183300cc2  mov      byte ptr [rsp + 0x138], 0
0x000183300cca  mov      dword ptr [rbx + 0x650], 5
0x000183300cd4  mov      r15d, 0x1f2
0x000183300cda  jmp      0x1832feda7
0x000183300cdf  lea      rcx, [rbx + 0x5a0]
0x000183300ce6  mov      rdx, r13
0x000183300ce9  call     0x183300f60
0x000183300cee  mov      rbp, rax
0x000183300cf1  jmp      0x1832ff402
0x000183300cf6  mov      rcx, r15
0x000183300cf9  mov      rdx, r12
0x000183300cfc  call     0x1833013d0
0x000183300d01  mov      r13, rax
0x000183300d04  jmp      0x1832fe621
0x000183300d09  lea      rcx, [rbx + 0x498]
0x000183300d10  mov      rdx, rbp
0x000183300d13  call     0x183301570
0x000183300d18  mov      r12, rax
0x000183300d1b  jmp      0x1832fec01
0x000183300d20  test     r8b, r8b
0x000183300d23  cmovs    r8, qword ptr [rsp + 0x1c8]
0x000183300d2c  lea      rdx, [rsp + 0x1c0]
0x000183300d34  cmovs    rdx, qword ptr [rsp + 0x1c0]
0x000183300d3d  call     0x183b61fa0
0x000183300d42  jmp      0x1832feab8
0x000183300d47  test     r8b, r8b
0x000183300d4a  cmovs    r8, qword ptr [rsp + 0x1c8]
0x000183300d53  cmovs    rbx, qword ptr [rsp + 0x1c0]
0x000183300d5c  mov      rdx, rbx
0x000183300d5f  call     0x183b61fa0
0x000183300d64  jmp      0x1832ff8cb
0x000183300d69  mov      rdx, rax
0x000183300d6c  test     dil, dil
0x000183300d6f  jns      0x183300d74
0x000183300d71  mov      rdx, qword ptr [rax]
0x000183300d74  mov      rdx, qword ptr [rdx + r9 + 0x19]
0x000183300d79  mov      qword ptr [rbx + 0x270], rdx
0x000183300d80  mov      qword ptr [rsp + 0x130], rsi
0x000183300d88  mov      byte ptr [rsp + 0x138], 1
0x000183300d90  lea      rdx, [r9 + 0x25]
0x000183300d94  movsx    rdi, byte ptr [rax + 0x17]
0x000183300d99  mov      r12, rdi
0x000183300d9c  test     rdi, rdi
0x000183300d9f  jns      0x183300da5
0x000183300da1  mov      r12, qword ptr [rax + 8]
0x000183300da5  cmp      rdx, r12
0x000183300da8  ja       0x1832ffb97
0x000183300dae  mov      rsi, rax
0x000183300db1  test     dil, dil
0x000183300db4  jns      0x183300db9
0x000183300db6  mov      rsi, qword ptr [rax]
0x000183300db9  mov      esi, dword ptr [rsi + r9 + 0x21]
0x000183300dbe  mov      dword ptr [rbx + 0x280], esi
0x000183300dc4  mov      qword ptr [rsp + 0x130], rdx
0x000183300dcc  mov      byte ptr [rsp + 0x138], 1
0x000183300dd4  lea      rsi, [r9 + 0x29]
0x000183300dd8  movsx    rdi, byte ptr [rax + 0x17]
0x000183300ddd  mov      r12, rdi
0x000183300de0  test     rdi, rdi
0x000183300de3  jns      0x183300de9
0x000183300de5  mov      r12, qword ptr [rax + 8]
0x000183300de9  cmp      rsi, r12
0x000183300dec  jbe      0x183300dfa
0x000183300dee  mov      byte ptr [rsp + 0x138], 0
0x000183300df6  xor      edi, edi
0x000183300df8  jmp      0x183300e77
0x000183300dfa  mov      rdx, rax
0x000183300dfd  test     dil, dil
0x000183300e00  jns      0x183300e05
0x000183300e02  mov      rdx, qword ptr [rax]
0x000183300e05  mov      edx, dword ptr [rdx + r9 + 0x25]
0x000183300e0a  mov      rdi, qword ptr [rsp + 0x30]
0x000183300e0f  mov      dword ptr [rdi + 0x27c], edx
0x000183300e15  mov      qword ptr [rsp + 0x130], rsi
0x000183300e1d  mov      byte ptr [rsp + 0x138], 1
0x000183300e25  add      r9, 0x2d
0x000183300e29  movsx    r12, byte ptr [rax + 0x17]
0x000183300e2e  mov      rdx, r12
0x000183300e31  test     r12, r12
0x000183300e34  jns      0x183300e3a
0x000183300e36  mov      rdx, qword ptr [rax + 8]
0x000183300e3a  cmp      r9, rdx
0x000183300e3d  setbe    dil
0x000183300e41  ja       0x183300e69
0x000183300e43  mov      r13, rax
0x000183300e46  test     r12b, r12b
0x000183300e49  jns      0x183300e4e
0x000183300e4b  mov      r13, qword ptr [rax]
0x000183300e4e  mov      esi, dword ptr [r13 + rsi]
0x000183300e53  mov      rbx, qword ptr [rsp + 0x30]
0x000183300e58  mov      dword ptr [rbx + 0x278], esi
0x000183300e5e  mov      qword ptr [rsp + 0x130], r9
0x000183300e66  mov      rsi, r9
0x000183300e69  cmp      r9, rdx
0x000183300e6c  setbe    byte ptr [rsp + 0x138]
0x000183300e74  mov      rdx, rsi
0x000183300e77  mov      rbp, qword ptr [rsp + 0x28]
0x000183300e7c  mov      rbx, qword ptr [rsp + 0x30]
0x000183300e81  jmp      0x1832ffdfe
0x000183300e86  mov      rdx, qword ptr [r15 + rdi + 0x10]
0x000183300e8b  movabs   rax, 0x7fffffffffffffff
0x000183300e95  and      rdx, rax
0x000183300e98  mov      rcx, qword ptr [r15 + rdi]
0x000183300e9c  call     0x183b135e0
0x000183300ea1  jmp      0x183300ed2
0x000183300ea3  mov      rax, qword ptr [rbx + 0x310]
0x000183300eaa  shl      rax, 3
0x000183300eae  lea      rdi, [rax + rax*2]
0x000183300eb2  mov      rcx, rsi
0x000183300eb5  test     rdi, rdi
0x000183300eb8  je       0x183300ee4
0x000183300eba  lea      r15, [rsi - 0x18]
0x000183300ebe  mov      rax, r15
0x000183300ec1  add      rax, rdi
0x000183300ec4  je       0x1832ff346
0x000183300eca  cmp      byte ptr [r15 + rdi + 0x17], 0
0x000183300ed0  js       0x183300e86
0x000183300ed2  add      rdi, -0x18
0x000183300ed6  jne      0x183300ebe
0x000183300ed8  mov      rax, qword ptr [rsp + 0x30]
0x000183300edd  mov      rcx, qword ptr [rax + 0x308]
0x000183300ee4  sub      rsi, rcx
0x000183300ee7  sar      rsi, 3
0x000183300eeb  movabs   rax, 0xaaaaaaaaaaaaaaab
0x000183300ef5  imul     rax, rsi
0x000183300ef9  mov      rbx, qword ptr [rsp + 0x30]
0x000183300efe  mov      qword ptr [rbx + 0x310], rax
0x000183300f05  mov      rax, qword ptr [rbx + 0x318]
0x000183300f0c  shl      rax, 3
0x000183300f10  lea      rdx, [rax + rax*2]
0x000183300f14  call     0x183b135e0
0x000183300f19  mov      qword ptr [rbx + 0x308], r14
0x000183300f20  mov      eax, 3
0x000183300f25  mov      qword ptr [rbx + 0x310], rax
0x000183300f2c  mov      qword ptr [rbx + 0x318], rax
0x000183300f33  movabs   rdi, 0x7fffffffffffffff
0x000183300f3d  mov      rbp, qword ptr [rsp + 0x28]
0x000183300f42  jmp      0x1832fe142
0x000183300f47  mov      rcx, qword ptr [rsp + 0x218]
0x000183300f4f  xor      rcx, rsp
0x000183300f52  call     0x185941420
0x000183300f57  int3     
