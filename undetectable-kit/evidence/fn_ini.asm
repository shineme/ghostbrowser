0x0001832fb3f0  push     r15
0x0001832fb3f2  push     r14
0x0001832fb3f4  push     r13
0x0001832fb3f6  push     r12
0x0001832fb3f8  push     rsi
0x0001832fb3f9  push     rdi
0x0001832fb3fa  push     rbp
0x0001832fb3fb  push     rbx
0x0001832fb3fc  sub      rsp, 0x278
0x0001832fb403  movaps   xmmword ptr [rsp + 0x260], xmm6
0x0001832fb40b  mov      rdi, rdx
0x0001832fb40e  mov      rsi, rcx
0x0001832fb411  mov      rax, qword ptr [rip + 0xe166c28]   ; 0x191462040
0x0001832fb418  xor      rax, rsp
0x0001832fb41b  mov      qword ptr [rsp + 0x258], rax
0x0001832fb423  xor      r15d, r15d
0x0001832fb426  lea      rbx, [rsp + 0x240]
0x0001832fb42e  mov      qword ptr [rbx + 0x10], r15
0x0001832fb432  xorps    xmm6, xmm6
0x0001832fb435  movaps   xmmword ptr [rbx], xmm6
0x0001832fb438  lea      r14, [rcx + 0x30]
0x0001832fb43c  lea      rax, [rip + 0xc8d22ed]   ; "settings.json"
0x0001832fb443  lea      r8, [rsp + 0x230]
0x0001832fb44b  mov      qword ptr [r8], rax
0x0001832fb44e  mov      qword ptr [r8 + 8], 0xd
0x0001832fb456  mov      rcx, r14
0x0001832fb459  mov      rdx, rbx
0x0001832fb45c  call     0x180b97200
0x0001832fb461  lea      rcx, [rsp + 0x208]
0x0001832fb469  mov      qword ptr [rcx + 0x20], r15
0x0001832fb46d  movups   xmmword ptr [rcx + 0x10], xmm6
0x0001832fb471  movups   xmmword ptr [rcx], xmm6
0x0001832fb474  lea      r13, [rip + 0xc562f55]   ; 0x18f85e3d0
0x0001832fb47b  mov      qword ptr [rcx - 8], r13
0x0001832fb47f  cmp      byte ptr [rbx + 0x17], 0
0x0001832fb483  js       0x1832fb7c0
0x0001832fb489  mov      rax, qword ptr [rsp + 0x250]
0x0001832fb491  mov      qword ptr [rcx + 0x10], rax
0x0001832fb495  movaps   xmm0, xmmword ptr [rsp + 0x240]
0x0001832fb49d  movups   xmmword ptr [rcx], xmm0
0x0001832fb4a0  lea      rcx, [rsp + 0x200]
0x0001832fb4a8  mov      dword ptr [rcx + 0x20], 0x34
0x0001832fb4af  xor      eax, eax
0x0001832fb4b1  mov      qword ptr [rcx + 0x28], rax
0x0001832fb4b5  lea      r15, [rsp + 0x1fc]
0x0001832fb4bd  mov      dword ptr [r15], 0
0x0001832fb4c4  lea      rbx, [rsp + 0x1e0]
0x0001832fb4cc  mov      qword ptr [rbx + 0x10], rax
0x0001832fb4d0  xorps    xmm0, xmm0
0x0001832fb4d3  movaps   xmmword ptr [rbx], xmm0
0x0001832fb4d6  lea      rdx, [rsp + 0x1d8]
0x0001832fb4de  mov      qword ptr [rdx], rax
0x0001832fb4e1  mov      r8, r15
0x0001832fb4e4  mov      r9, rbx
0x0001832fb4e7  call     0x181cf5e70
0x0001832fb4ec  cmp      dword ptr [r15], 0
0x0001832fb4f0  jne      0x1832fc04d
0x0001832fb4f6  mov      rbx, qword ptr [rsp + 0x1d8]
0x0001832fb4fe  cmp      byte ptr [rbx + 0x18], 6
0x0001832fb502  jne      0x1832fbae5
0x0001832fb508  lea      rax, [rip + 0xc8d22a9]   ; "profilesPath"
0x0001832fb50f  lea      rdx, [rsp + 0x80]
0x0001832fb517  mov      qword ptr [rdx], rax
0x0001832fb51a  mov      qword ptr [rdx + 8], 0xc
0x0001832fb522  mov      rcx, rbx
0x0001832fb525  call     0x180449bc0
0x0001832fb52a  test     rax, rax
0x0001832fb52d  je       0x1832fb808
0x0001832fb533  movsx    rcx, byte ptr [rax + 0x17]
0x0001832fb538  test     rcx, rcx
0x0001832fb53b  jns      0x1832fb544
0x0001832fb53d  mov      rcx, qword ptr [rax + 8]
0x0001832fb541  mov      rax, qword ptr [rax]
0x0001832fb544  lea      rdx, [rsp + 0x70]
0x0001832fb549  mov      qword ptr [rdx], rax
0x0001832fb54c  mov      qword ptr [rdx + 8], rcx
0x0001832fb550  lea      rcx, [rsp + 0x1a8]
0x0001832fb558  call     0x1846bf0e0
0x0001832fb55d  cmp      byte ptr [rdi + 0x17], 0
0x0001832fb561  js       0x1832fb7da
0x0001832fb567  mov      rax, qword ptr [rsp + 0x1b8]
0x0001832fb56f  mov      qword ptr [rdi + 0x10], rax
0x0001832fb573  movups   xmm0, xmmword ptr [rsp + 0x1a8]
0x0001832fb57b  movups   xmmword ptr [rdi], xmm0
0x0001832fb57e  lea      rcx, [rdi + 0x18]
0x0001832fb582  cmp      rcx, r14
0x0001832fb585  je       0x1832fb5ae
0x0001832fb587  cmp      byte ptr [rdi + 0x2f], 0
0x0001832fb58b  movzx    r8d, byte ptr [rsi + 0x47]
0x0001832fb590  js       0x1832fb7ee
0x0001832fb596  test     r8b, r8b
0x0001832fb599  js       0x1832fb777
0x0001832fb59f  mov      rax, qword ptr [r14 + 0x10]
0x0001832fb5a3  mov      qword ptr [rcx + 0x10], rax
0x0001832fb5a7  movups   xmm0, xmmword ptr [r14]
0x0001832fb5ab  movups   xmmword ptr [rcx], xmm0
0x0001832fb5ae  lea      rax, [rip + 0xc8d224b]   ; "docVisAllVis"
0x0001832fb5b5  lea      r8, [rsp + 0x60]
0x0001832fb5ba  mov      qword ptr [r8], rax
0x0001832fb5bd  mov      qword ptr [r8 + 8], 0xc
0x0001832fb5c5  lea      rsi, [rsp + 0x1a8]
0x0001832fb5cd  mov      rcx, rbx
0x0001832fb5d0  mov      rdx, rsi
0x0001832fb5d3  call     0x184c63f30
0x0001832fb5d8  xor      ebp, ebp
0x0001832fb5da  cmp      byte ptr [rsi + 1], 0
0x0001832fb5de  movzx    eax, byte ptr [rsi]
0x0001832fb5e1  cmove    eax, ebp
0x0001832fb5e4  mov      byte ptr [rdi + 0x33], al
0x0001832fb5e7  lea      rax, [rip + 0xc8d2222]   ; "docHasFocus"
0x0001832fb5ee  lea      r8, [rsp + 0x50]
0x0001832fb5f3  mov      qword ptr [r8], rax
0x0001832fb5f6  mov      qword ptr [r8 + 8], 0xb
0x0001832fb5fe  lea      rsi, [rsp + 0x1a8]
0x0001832fb606  mov      rcx, rbx
0x0001832fb609  mov      rdx, rsi
0x0001832fb60c  call     0x184c63f30
0x0001832fb611  cmp      byte ptr [rsi + 1], 0
0x0001832fb615  movzx    eax, byte ptr [rsi]
0x0001832fb618  cmove    eax, ebp
0x0001832fb61b  mov      byte ptr [rdi + 0x34], al
0x0001832fb61e  lea      rax, [rip + 0xc8d21fb]   ; "hideProfileName"
0x0001832fb625  lea      r8, [rsp + 0x40]
0x0001832fb62a  mov      qword ptr [r8], rax
0x0001832fb62d  mov      qword ptr [r8 + 8], 0xf
0x0001832fb635  lea      rsi, [rsp + 0x1a8]
0x0001832fb63d  mov      rcx, rbx
0x0001832fb640  mov      rdx, rsi
0x0001832fb643  call     0x184c63f30
0x0001832fb648  cmp      byte ptr [rsi + 1], 0
0x0001832fb64c  movzx    eax, byte ptr [rsi]
0x0001832fb64f  cmove    eax, ebp
0x0001832fb652  mov      byte ptr [rdi + 0x30], al
0x0001832fb655  lea      rax, [rip + 0xc8d21d4]   ; "disDevToolsDetect"
0x0001832fb65c  lea      r8, [rsp + 0x30]
0x0001832fb661  mov      qword ptr [r8], rax
0x0001832fb664  mov      qword ptr [r8 + 8], 0x11
0x0001832fb66c  lea      rsi, [rsp + 0x1a8]
0x0001832fb674  mov      rcx, rbx
0x0001832fb677  mov      rdx, rsi
0x0001832fb67a  call     0x184c63f30
0x0001832fb67f  movzx    eax, byte ptr [rsi]
0x0001832fb682  cmp      byte ptr [rsi + 1], 0
0x0001832fb686  mov      ecx, 1
0x0001832fb68b  cmovne   ecx, eax
0x0001832fb68e  mov      byte ptr [rdi + 0x31], cl
0x0001832fb691  lea      rax, [rip + 0xc8d21b0]   ; "autoAllowBrowserAlerts"
0x0001832fb698  lea      r8, [rsp + 0x20]
0x0001832fb69d  mov      qword ptr [r8], rax
0x0001832fb6a0  mov      qword ptr [r8 + 8], 0x16
0x0001832fb6a8  lea      rsi, [rsp + 0x1a8]
0x0001832fb6b0  mov      rcx, rbx
0x0001832fb6b3  mov      rdx, rsi
0x0001832fb6b6  call     0x184c63f30
0x0001832fb6bb  movzx    eax, byte ptr [rsi]
0x0001832fb6be  cmp      byte ptr [rsi + 1], 0
0x0001832fb6c2  cmove    eax, ebp
0x0001832fb6c5  mov      byte ptr [rdi + 0x32], al
0x0001832fb6c8  mov      sil, 1
0x0001832fb6cb  mov      rdi, qword ptr [rsp + 0x1d8]
0x0001832fb6d3  mov      qword ptr [rsp + 0x1d8], 0
0x0001832fb6df  test     rdi, rdi
0x0001832fb6e2  je       0x1832fb6f9
0x0001832fb6e4  mov      rcx, rdi
0x0001832fb6e7  call     0x184081090
0x0001832fb6ec  mov      edx, 0x20
0x0001832fb6f1  mov      rcx, rdi
0x0001832fb6f4  call     0x183b135e0
0x0001832fb6f9  cmp      byte ptr [rsp + 0x1f7], 0
0x0001832fb701  jns      0x1832fb722
0x0001832fb703  movabs   rdx, 0x7fffffffffffffff
0x0001832fb70d  and      rdx, qword ptr [rsp + 0x1f0]
0x0001832fb715  mov      rcx, qword ptr [rsp + 0x1e0]
0x0001832fb71d  call     0x183b135e0
0x0001832fb722  mov      qword ptr [rsp + 0x200], r13
0x0001832fb72a  cmp      byte ptr [rsp + 0x21f], 0
0x0001832fb732  js       0x1832fb7a3
0x0001832fb734  cmp      byte ptr [rsp + 0x257], 0
0x0001832fb73c  js       0x1832fb789
0x0001832fb73e  mov      rax, qword ptr [rsp + 0x258]
0x0001832fb746  xor      rax, rsp
0x0001832fb749  mov      rcx, qword ptr [rip + 0xe1668f0]   ; 0x191462040
0x0001832fb750  cmp      rcx, rax
0x0001832fb753  jne      0x1832fc3f3
0x0001832fb759  mov      eax, esi
0x0001832fb75b  movaps   xmm6, xmmword ptr [rsp + 0x260]
0x0001832fb763  add      rsp, 0x278
0x0001832fb76a  pop      rbx
0x0001832fb76b  pop      rbp
0x0001832fb76c  pop      rdi
0x0001832fb76d  pop      rsi
0x0001832fb76e  pop      r12
0x0001832fb770  pop      r13
0x0001832fb772  pop      r14
0x0001832fb774  pop      r15
0x0001832fb776  ret      
0x0001832fb777  mov      rdx, qword ptr [rsi + 0x30]
0x0001832fb77b  mov      r8, qword ptr [rsi + 0x38]
0x0001832fb77f  call     0x180af59d0
0x0001832fb784  jmp      0x1832fb5ae
0x0001832fb789  mov      rcx, qword ptr [rsp + 0x240]
0x0001832fb791  mov      rdx, qword ptr [rsp + 0x250]
0x0001832fb799  add      rdx, rdx
0x0001832fb79c  call     0x183b135e0
0x0001832fb7a1  jmp      0x1832fb73e
0x0001832fb7a3  mov      rcx, qword ptr [rsp + 0x208]
0x0001832fb7ab  mov      rdx, qword ptr [rsp + 0x218]
0x0001832fb7b3  add      rdx, rdx
0x0001832fb7b6  call     0x183b135e0
0x0001832fb7bb  jmp      0x1832fb734
0x0001832fb7c0  mov      rdx, qword ptr [rsp + 0x240]
0x0001832fb7c8  mov      r8, qword ptr [rsp + 0x248]
0x0001832fb7d0  call     0x183b8e360
0x0001832fb7d5  jmp      0x1832fb4a0
0x0001832fb7da  mov      rcx, qword ptr [rdi]
0x0001832fb7dd  mov      rdx, qword ptr [rdi + 0x10]
0x0001832fb7e1  add      rdx, rdx
0x0001832fb7e4  call     0x183b135e0
0x0001832fb7e9  jmp      0x1832fb567
0x0001832fb7ee  test     r8b, r8b
0x0001832fb7f1  jns      0x1832fb7fb
0x0001832fb7f3  mov      r8, qword ptr [rsi + 0x38]
0x0001832fb7f7  mov      r14, qword ptr [rsi + 0x30]
0x0001832fb7fb  mov      rdx, r14
0x0001832fb7fe  call     0x180af5a60
0x0001832fb803  jmp      0x1832fb5ae
0x0001832fb808  movabs   rdi, 0x7fffffffffffffff
0x0001832fb812  lea      r14, [rsp + 0x158]
0x0001832fb81a  mov      rcx, r14
0x0001832fb81d  mov      edx, 0x42
0x0001832fb822  call     0x184ec6ea0
0x0001832fb827  mov      ecx, 0x30
0x0001832fb82c  call     0x18593bfb8
0x0001832fb831  mov      qword ptr [rsp + 0x108], rax
0x0001832fb839  mov      qword ptr [rsp + 0x110], 0x2e
0x0001832fb845  lea      rbx, [rdi + 0x31]
0x0001832fb849  mov      qword ptr [rsp + 0x118], rbx
0x0001832fb851  lea      rcx, [rax + 0x2e]
0x0001832fb855  cmp      rcx, rax
0x0001832fb858  jb       0x1832fc403
0x0001832fb85e  lea      rdx, [rip + 0xc8d1e1b]   ; "..\..\undetectable\core\stors\iniconnector.cpp"
0x0001832fb865  cmp      rax, rdx
0x0001832fb868  seta     r8b
0x0001832fb86c  cmp      rcx, rdx
0x0001832fb86f  setbe    cl
0x0001832fb872  or       cl, r8b
0x0001832fb875  je       0x1832fc403
0x0001832fb87b  movups   xmm0, xmmword ptr [rip + 0xc8d1e1c]   ; "iniconnector.cpp"
0x0001832fb882  movups   xmmword ptr [rax + 0x1e], xmm0
0x0001832fb886  movups   xmm0, xmmword ptr [rip + 0xc8d1e03]   ; "le\core\stors\iniconnector.cpp"
0x0001832fb88d  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832fb891  movups   xmm0, xmmword ptr [rip + 0xc8d1de8]   ; "..\..\undetectable\core\stors\iniconnector.cpp"
0x0001832fb898  movups   xmmword ptr [rax], xmm0
0x0001832fb89b  mov      byte ptr [rax + 0x2e], 0
0x0001832fb89f  mov      ecx, 0x30
0x0001832fb8a4  call     0x18593bfb8
0x0001832fb8a9  mov      qword ptr [rsp + 0xb0], rax
0x0001832fb8b1  mov      qword ptr [rsp + 0xb8], 0x24
0x0001832fb8bd  mov      qword ptr [rsp + 0xc0], rbx
0x0001832fb8c5  lea      rcx, [rax + 0x24]
0x0001832fb8c9  cmp      rcx, rax
0x0001832fb8cc  jb       0x1832fc403
0x0001832fb8d2  lea      rdx, [rip + 0xc8d1e67]   ; "undetectable::IniConnector::getSWIni"
0x0001832fb8d9  cmp      rax, rdx
0x0001832fb8dc  seta     r8b
0x0001832fb8e0  cmp      rcx, rdx
0x0001832fb8e3  setbe    cl
0x0001832fb8e6  or       cl, r8b
0x0001832fb8e9  je       0x1832fc403
0x0001832fb8ef  movups   xmm0, xmmword ptr [rip + 0xc8d1e5a]   ; "iConnector::getSWIni"
0x0001832fb8f6  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832fb8fa  movups   xmm0, xmmword ptr [rip + 0xc8d1e3f]   ; "undetectable::IniConnector::getSWIni"
0x0001832fb901  movups   xmmword ptr [rax], xmm0
0x0001832fb904  mov      dword ptr [rax + 0x20], 0x696e4957
0x0001832fb90b  mov      byte ptr [rax + 0x24], 0
0x0001832fb90f  lea      r15, [rsp + 0xb0]
0x0001832fb917  lea      rbx, [rsp + 0x1a8]
0x0001832fb91f  mov      rcx, r15
0x0001832fb922  mov      rdx, rbx
0x0001832fb925  mov      r8, rax
0x0001832fb928  mov      r9b, 0x5b
0x0001832fb92b  call     0x18a58b3f0
0x0001832fb930  movaps   xmm0, xmmword ptr [r15]
0x0001832fb934  lea      rcx, [rsp + 0xd0]
0x0001832fb93c  movaps   xmmword ptr [rcx], xmm0
0x0001832fb93f  mov      rax, qword ptr [r15 + 0x10]
0x0001832fb943  mov      qword ptr [rcx + 0x10], rax
0x0001832fb947  xorps    xmm6, xmm6
0x0001832fb94a  movaps   xmmword ptr [r15], xmm6
0x0001832fb94e  xor      r12d, r12d
0x0001832fb951  mov      qword ptr [r15 + 0x10], r12
0x0001832fb955  lea      rdx, [rip + 0xc8d1ae0]   ; " ( "
0x0001832fb95c  mov      r8d, 3
0x0001832fb962  call     0x183b596f0
0x0001832fb967  movups   xmm0, xmmword ptr [rax]
0x0001832fb96a  lea      rcx, [rsp + 0xf0]
0x0001832fb972  movaps   xmmword ptr [rcx], xmm0
0x0001832fb975  mov      rdx, qword ptr [rax + 0x10]
0x0001832fb979  mov      qword ptr [rcx + 0x10], rdx
0x0001832fb97d  mov      qword ptr [rax + 0x10], r12
0x0001832fb981  movups   xmmword ptr [rax], xmm6
0x0001832fb984  lea      rdx, [rsp + 0x108]
0x0001832fb98c  movsx    r8, byte ptr [rdx + 0x17]
0x0001832fb991  test     r8, r8
0x0001832fb994  cmovs    r8, qword ptr [rdx + 8]
0x0001832fb999  cmovs    rdx, qword ptr [rdx]
0x0001832fb99d  call     0x183b596f0
0x0001832fb9a2  movups   xmm0, xmmword ptr [rax]
0x0001832fb9a5  lea      rcx, [rsp + 0x120]
0x0001832fb9ad  movaps   xmmword ptr [rcx], xmm0
0x0001832fb9b0  mov      rdx, qword ptr [rax + 0x10]
0x0001832fb9b4  mov      qword ptr [rcx + 0x10], rdx
0x0001832fb9b8  mov      qword ptr [rax + 0x10], r12
0x0001832fb9bc  movups   xmmword ptr [rax], xmm6
0x0001832fb9bf  lea      rdx, [rip + 0xc7806de]   ; 0x18fa7c0a4
0x0001832fb9c6  mov      r8d, 1
0x0001832fb9cc  call     0x183b596f0
0x0001832fb9d1  movups   xmm0, xmmword ptr [rax]
0x0001832fb9d4  lea      rcx, [rsp + 0x140]
0x0001832fb9dc  movaps   xmmword ptr [rcx], xmm0
0x0001832fb9df  mov      rdx, qword ptr [rax + 0x10]
0x0001832fb9e3  mov      qword ptr [rcx + 0x10], rdx
0x0001832fb9e7  mov      qword ptr [rax + 0x10], r12
0x0001832fb9eb  movups   xmmword ptr [rax], xmm6
0x0001832fb9ee  movsx    r8, byte ptr [rsp + 0x16f]
0x0001832fb9f7  test     r8, r8
0x0001832fb9fa  cmovs    r8, qword ptr [rsp + 0x160]
0x0001832fba03  cmovs    r14, qword ptr [rsp + 0x158]
0x0001832fba0c  mov      rdx, r14
0x0001832fba0f  call     0x183b596f0
0x0001832fba14  movups   xmm0, xmmword ptr [rax]
0x0001832fba17  lea      rcx, [rsp + 0x170]
0x0001832fba1f  movaps   xmmword ptr [rcx], xmm0
0x0001832fba22  mov      rdx, qword ptr [rax + 0x10]
0x0001832fba26  mov      qword ptr [rcx + 0x10], rdx
0x0001832fba2a  mov      qword ptr [rax + 0x10], r12
0x0001832fba2e  movups   xmmword ptr [rax], xmm6
0x0001832fba31  lea      rdx, [rip + 0xc8d19c0]   ; " )]"
0x0001832fba38  mov      r8d, 3
0x0001832fba3e  call     0x183b596f0
0x0001832fba43  movups   xmm0, xmmword ptr [rax]
0x0001832fba46  movaps   xmmword ptr [rsp + 0x190], xmm0
0x0001832fba4e  mov      rcx, qword ptr [rax + 0x10]
0x0001832fba52  mov      qword ptr [rsp + 0x1a0], rcx
0x0001832fba5a  mov      qword ptr [rax + 0x10], r12
0x0001832fba5e  movups   xmmword ptr [rax], xmm6
0x0001832fba61  mov      ecx, 0x40
0x0001832fba66  call     0x18593bfb8
0x0001832fba6b  mov      qword ptr [rsp + 0x90], rax
0x0001832fba73  mov      qword ptr [rsp + 0x98], 0x34
0x0001832fba7f  lea      rcx, [rdi + 0x41]
0x0001832fba83  mov      qword ptr [rsp + 0xa0], rcx
0x0001832fba8b  lea      rcx, [rax + 0x34]
0x0001832fba8f  cmp      rcx, rax
0x0001832fba92  jb       0x1832fc403
0x0001832fba98  lea      rdx, [rip + 0xc8d1d29]   ; "Unable to parse settigs file: profile path is empty!"
0x0001832fba9f  cmp      rax, rdx
0x0001832fbaa2  seta     r8b
0x0001832fbaa6  cmp      rcx, rdx
0x0001832fbaa9  setbe    cl
0x0001832fbaac  or       cl, r8b
0x0001832fbaaf  je       0x1832fc403
0x0001832fbab5  movups   xmm0, xmmword ptr [rip + 0xc8d1d2c]   ; "ofile path is empty!"
0x0001832fbabc  movups   xmmword ptr [rax + 0x20], xmm0
0x0001832fbac0  movups   xmm0, xmmword ptr [rip + 0xc8d1d11]   ; "settigs file: profile path is empty!"
0x0001832fbac7  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832fbacb  movups   xmm0, xmmword ptr [rip + 0xc8d1cf6]   ; "Unable to parse settigs file: profile path is empty!"
0x0001832fbad2  movups   xmmword ptr [rax], xmm0
0x0001832fbad5  mov      dword ptr [rax + 0x30], 0x21797470
0x0001832fbadc  mov      byte ptr [rax + 0x34], 0
0x0001832fbae0  jmp      0x1832fbdb1
0x0001832fbae5  movabs   rdi, 0x7fffffffffffffff
0x0001832fbaef  lea      r14, [rsp + 0x158]
0x0001832fbaf7  mov      rcx, r14
0x0001832fbafa  mov      edx, 0x37
0x0001832fbaff  call     0x184ec6ea0
0x0001832fbb04  mov      ecx, 0x30
0x0001832fbb09  call     0x18593bfb8
0x0001832fbb0e  mov      qword ptr [rsp + 0x108], rax
0x0001832fbb16  mov      qword ptr [rsp + 0x110], 0x2e
0x0001832fbb22  lea      r12, [rdi + 0x31]
0x0001832fbb26  mov      qword ptr [rsp + 0x118], r12
0x0001832fbb2e  lea      rcx, [rax + 0x2e]
0x0001832fbb32  cmp      rcx, rax
0x0001832fbb35  jb       0x1832fc403
0x0001832fbb3b  lea      rdx, [rip + 0xc8d1b3e]   ; "..\..\undetectable\core\stors\iniconnector.cpp"
0x0001832fbb42  cmp      rax, rdx
0x0001832fbb45  seta     r8b
0x0001832fbb49  cmp      rcx, rdx
0x0001832fbb4c  setbe    cl
0x0001832fbb4f  or       cl, r8b
0x0001832fbb52  je       0x1832fc403
0x0001832fbb58  movups   xmm0, xmmword ptr [rip + 0xc8d1b3f]   ; "iniconnector.cpp"
0x0001832fbb5f  movups   xmmword ptr [rax + 0x1e], xmm0
0x0001832fbb63  movups   xmm0, xmmword ptr [rip + 0xc8d1b26]   ; "le\core\stors\iniconnector.cpp"
0x0001832fbb6a  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832fbb6e  movups   xmm0, xmmword ptr [rip + 0xc8d1b0b]   ; "..\..\undetectable\core\stors\iniconnector.cpp"
0x0001832fbb75  movups   xmmword ptr [rax], xmm0
0x0001832fbb78  mov      byte ptr [rax + 0x2e], 0
0x0001832fbb7c  mov      ecx, 0x30
0x0001832fbb81  call     0x18593bfb8
0x0001832fbb86  mov      qword ptr [rsp + 0xb0], rax
0x0001832fbb8e  mov      qword ptr [rsp + 0xb8], 0x24
0x0001832fbb9a  mov      qword ptr [rsp + 0xc0], r12
0x0001832fbba2  lea      rcx, [rax + 0x24]
0x0001832fbba6  cmp      rcx, rax
0x0001832fbba9  jb       0x1832fc403
0x0001832fbbaf  lea      rdx, [rip + 0xc8d1b8a]   ; "undetectable::IniConnector::getSWIni"
0x0001832fbbb6  cmp      rax, rdx
0x0001832fbbb9  seta     r8b
0x0001832fbbbd  cmp      rcx, rdx
0x0001832fbbc0  setbe    cl
0x0001832fbbc3  or       cl, r8b
0x0001832fbbc6  je       0x1832fc403
0x0001832fbbcc  movups   xmm0, xmmword ptr [rip + 0xc8d1b7d]   ; "iConnector::getSWIni"
0x0001832fbbd3  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832fbbd7  movups   xmm0, xmmword ptr [rip + 0xc8d1b62]   ; "undetectable::IniConnector::getSWIni"
0x0001832fbbde  movups   xmmword ptr [rax], xmm0
0x0001832fbbe1  mov      dword ptr [rax + 0x20], 0x696e4957
0x0001832fbbe8  mov      byte ptr [rax + 0x24], 0
0x0001832fbbec  lea      r15, [rsp + 0xb0]
0x0001832fbbf4  lea      rbx, [rsp + 0x1a8]
0x0001832fbbfc  mov      rcx, r15
0x0001832fbbff  mov      rdx, rbx
0x0001832fbc02  mov      r8, rax
0x0001832fbc05  mov      r9b, 0x5b
0x0001832fbc08  call     0x18a58b3f0
0x0001832fbc0d  movaps   xmm0, xmmword ptr [r15]
0x0001832fbc11  lea      rcx, [rsp + 0xd0]
0x0001832fbc19  movaps   xmmword ptr [rcx], xmm0
0x0001832fbc1c  mov      rax, qword ptr [r15 + 0x10]
0x0001832fbc20  mov      qword ptr [rcx + 0x10], rax
0x0001832fbc24  xorps    xmm6, xmm6
0x0001832fbc27  movaps   xmmword ptr [r15], xmm6
0x0001832fbc2b  xor      ebp, ebp
0x0001832fbc2d  mov      qword ptr [r15 + 0x10], rbp
0x0001832fbc31  lea      rdx, [rip + 0xc8d1804]   ; " ( "
0x0001832fbc38  mov      r8d, 3
0x0001832fbc3e  call     0x183b596f0
0x0001832fbc43  movups   xmm0, xmmword ptr [rax]
0x0001832fbc46  lea      rcx, [rsp + 0xf0]
0x0001832fbc4e  movaps   xmmword ptr [rcx], xmm0
0x0001832fbc51  mov      rdx, qword ptr [rax + 0x10]
0x0001832fbc55  mov      qword ptr [rcx + 0x10], rdx
0x0001832fbc59  mov      qword ptr [rax + 0x10], rbp
0x0001832fbc5d  movups   xmmword ptr [rax], xmm6
0x0001832fbc60  lea      rdx, [rsp + 0x108]
0x0001832fbc68  movsx    r8, byte ptr [rdx + 0x17]
0x0001832fbc6d  test     r8, r8
0x0001832fbc70  cmovs    r8, qword ptr [rdx + 8]
0x0001832fbc75  cmovs    rdx, qword ptr [rdx]
0x0001832fbc79  call     0x183b596f0
0x0001832fbc7e  movups   xmm0, xmmword ptr [rax]
0x0001832fbc81  lea      rcx, [rsp + 0x120]
0x0001832fbc89  movaps   xmmword ptr [rcx], xmm0
0x0001832fbc8c  mov      rdx, qword ptr [rax + 0x10]
0x0001832fbc90  mov      qword ptr [rcx + 0x10], rdx
0x0001832fbc94  mov      qword ptr [rax + 0x10], rbp
0x0001832fbc98  movups   xmmword ptr [rax], xmm6
0x0001832fbc9b  lea      rdx, [rip + 0xc780402]   ; 0x18fa7c0a4
0x0001832fbca2  mov      r8d, 1
0x0001832fbca8  call     0x183b596f0
0x0001832fbcad  movups   xmm0, xmmword ptr [rax]
0x0001832fbcb0  lea      rcx, [rsp + 0x140]
0x0001832fbcb8  movaps   xmmword ptr [rcx], xmm0
0x0001832fbcbb  mov      rdx, qword ptr [rax + 0x10]
0x0001832fbcbf  mov      qword ptr [rcx + 0x10], rdx
0x0001832fbcc3  mov      qword ptr [rax + 0x10], rbp
0x0001832fbcc7  movups   xmmword ptr [rax], xmm6
0x0001832fbcca  movsx    r8, byte ptr [rsp + 0x16f]
0x0001832fbcd3  test     r8, r8
0x0001832fbcd6  cmovs    r8, qword ptr [rsp + 0x160]
0x0001832fbcdf  cmovs    r14, qword ptr [rsp + 0x158]
0x0001832fbce8  mov      rdx, r14
0x0001832fbceb  call     0x183b596f0
0x0001832fbcf0  movups   xmm0, xmmword ptr [rax]
0x0001832fbcf3  lea      rcx, [rsp + 0x170]
0x0001832fbcfb  movaps   xmmword ptr [rcx], xmm0
0x0001832fbcfe  mov      rdx, qword ptr [rax + 0x10]
0x0001832fbd02  mov      qword ptr [rcx + 0x10], rdx
0x0001832fbd06  mov      qword ptr [rax + 0x10], rbp
0x0001832fbd0a  movups   xmmword ptr [rax], xmm6
0x0001832fbd0d  lea      rdx, [rip + 0xc8d16e4]   ; " )]"
0x0001832fbd14  mov      r8d, 3
0x0001832fbd1a  call     0x183b596f0
0x0001832fbd1f  movups   xmm0, xmmword ptr [rax]
0x0001832fbd22  movaps   xmmword ptr [rsp + 0x190], xmm0
0x0001832fbd2a  mov      rcx, qword ptr [rax + 0x10]
0x0001832fbd2e  mov      qword ptr [rsp + 0x1a0], rcx
0x0001832fbd36  mov      qword ptr [rax + 0x10], rbp
0x0001832fbd3a  movups   xmmword ptr [rax], xmm6
0x0001832fbd3d  mov      ecx, 0x30
0x0001832fbd42  call     0x18593bfb8
0x0001832fbd47  mov      qword ptr [rsp + 0x90], rax
0x0001832fbd4f  mov      qword ptr [rsp + 0x98], 0x2f
0x0001832fbd5b  mov      qword ptr [rsp + 0xa0], r12
0x0001832fbd63  lea      rcx, [rax + 0x2f]
0x0001832fbd67  cmp      rcx, rax
0x0001832fbd6a  jb       0x1832fc403
0x0001832fbd70  lea      rdx, [rip + 0xc8d1a11]   ; "Unable to parse settigs file: !value->is_dict()"
0x0001832fbd77  cmp      rax, rdx
0x0001832fbd7a  seta     r8b
0x0001832fbd7e  cmp      rcx, rdx
0x0001832fbd81  setbe    cl
0x0001832fbd84  or       cl, r8b
0x0001832fbd87  je       0x1832fc403
0x0001832fbd8d  movups   xmm0, xmmword ptr [rip + 0xc8d1a13]   ; "value->is_dict()"
0x0001832fbd94  movups   xmmword ptr [rax + 0x1f], xmm0
0x0001832fbd98  movups   xmm0, xmmword ptr [rip + 0xc8d19f9]   ; "settigs file: !value->is_dict()"
0x0001832fbd9f  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832fbda3  movups   xmm0, xmmword ptr [rip + 0xc8d19de]   ; "Unable to parse settigs file: !value->is_dict()"
0x0001832fbdaa  movups   xmmword ptr [rax], xmm0
0x0001832fbdad  mov      byte ptr [rax + 0x2f], 0
0x0001832fbdb1  lea      rdx, [rsp + 0x90]
0x0001832fbdb9  lea      r8, [rsp + 0x190]
0x0001832fbdc1  mov      rcx, rbx
0x0001832fbdc4  call     0x181199c30
0x0001832fbdc9  cmp      rsi, rbx
0x0001832fbdcc  je       0x1832fbe33
0x0001832fbdce  cmp      byte ptr [rsi + 0x17], 0
0x0001832fbdd2  movzx    r8d, byte ptr [rsp + 0x1bf]
0x0001832fbddb  js       0x1832fc028
0x0001832fbde1  test     r8b, r8b
0x0001832fbde4  js       0x1832fbfec
0x0001832fbdea  mov      rax, qword ptr [rsp + 0x1b8]
0x0001832fbdf2  mov      qword ptr [rsi + 0x10], rax
0x0001832fbdf6  movups   xmm0, xmmword ptr [rsp + 0x1a8]
0x0001832fbdfe  movups   xmmword ptr [rsi], xmm0
0x0001832fbe01  lea      rcx, [rsi + 0x18]
0x0001832fbe05  lea      rdx, [rsp + 0x1c0]
0x0001832fbe0d  cmp      byte ptr [rsi + 0x2f], 0
0x0001832fbe11  movzx    r8d, byte ptr [rdx + 0x17]
0x0001832fbe16  js       0x1832fc009
0x0001832fbe1c  test     r8b, r8b
0x0001832fbe1f  js       0x1832fbfd2
0x0001832fbe25  mov      rax, qword ptr [rdx + 0x10]
0x0001832fbe29  mov      qword ptr [rcx + 0x10], rax
0x0001832fbe2d  movups   xmm0, xmmword ptr [rdx]
0x0001832fbe30  movups   xmmword ptr [rcx], xmm0
0x0001832fbe33  cmp      byte ptr [rsp + 0x1d7], 0
0x0001832fbe3b  jns      0x1832fbe55
0x0001832fbe3d  mov      rdx, qword ptr [rsp + 0x1d0]
0x0001832fbe45  and      rdx, rdi
0x0001832fbe48  mov      rcx, qword ptr [rsp + 0x1c0]
0x0001832fbe50  call     0x183b135e0
0x0001832fbe55  cmp      byte ptr [rsp + 0x1bf], 0
0x0001832fbe5d  jns      0x1832fbe77
0x0001832fbe5f  mov      rdx, qword ptr [rsp + 0x1b8]
0x0001832fbe67  and      rdx, rdi
0x0001832fbe6a  mov      rcx, qword ptr [rsp + 0x1a8]
0x0001832fbe72  call     0x183b135e0
0x0001832fbe77  cmp      byte ptr [rsp + 0xa7], 0
0x0001832fbe7f  jns      0x1832fbe99
0x0001832fbe81  mov      rdx, qword ptr [rsp + 0xa0]
0x0001832fbe89  and      rdx, rdi
0x0001832fbe8c  mov      rcx, qword ptr [rsp + 0x90]
0x0001832fbe94  call     0x183b135e0
0x0001832fbe99  cmp      byte ptr [rsp + 0x1a7], 0
0x0001832fbea1  jns      0x1832fbebb
0x0001832fbea3  mov      rdx, qword ptr [rsp + 0x1a0]
0x0001832fbeab  and      rdx, rdi
0x0001832fbeae  mov      rcx, qword ptr [rsp + 0x190]
0x0001832fbeb6  call     0x183b135e0
0x0001832fbebb  cmp      byte ptr [rsp + 0x187], 0
0x0001832fbec3  jns      0x1832fbedd
0x0001832fbec5  mov      rdx, qword ptr [rsp + 0x180]
0x0001832fbecd  and      rdx, rdi
0x0001832fbed0  mov      rcx, qword ptr [rsp + 0x170]
0x0001832fbed8  call     0x183b135e0
0x0001832fbedd  cmp      byte ptr [rsp + 0x157], 0
0x0001832fbee5  jns      0x1832fbeff
0x0001832fbee7  mov      rdx, qword ptr [rsp + 0x150]
0x0001832fbeef  and      rdx, rdi
0x0001832fbef2  mov      rcx, qword ptr [rsp + 0x140]
0x0001832fbefa  call     0x183b135e0
0x0001832fbeff  cmp      byte ptr [rsp + 0x137], 0
0x0001832fbf07  jns      0x1832fbf21
0x0001832fbf09  mov      rdx, qword ptr [rsp + 0x130]
0x0001832fbf11  and      rdx, rdi
0x0001832fbf14  mov      rcx, qword ptr [rsp + 0x120]
0x0001832fbf1c  call     0x183b135e0
0x0001832fbf21  cmp      byte ptr [rsp + 0x107], 0
0x0001832fbf29  jns      0x1832fbf43
0x0001832fbf2b  mov      rdx, qword ptr [rsp + 0x100]
0x0001832fbf33  and      rdx, rdi
0x0001832fbf36  mov      rcx, qword ptr [rsp + 0xf0]
0x0001832fbf3e  call     0x183b135e0
0x0001832fbf43  cmp      byte ptr [rsp + 0xe7], 0
0x0001832fbf4b  jns      0x1832fbf65
0x0001832fbf4d  mov      rdx, qword ptr [rsp + 0xe0]
0x0001832fbf55  and      rdx, rdi
0x0001832fbf58  mov      rcx, qword ptr [rsp + 0xd0]
0x0001832fbf60  call     0x183b135e0
0x0001832fbf65  cmp      byte ptr [rsp + 0xc7], 0
0x0001832fbf6d  jns      0x1832fbf87
0x0001832fbf6f  mov      rdx, qword ptr [rsp + 0xc0]
0x0001832fbf77  and      rdx, rdi
0x0001832fbf7a  mov      rcx, qword ptr [rsp + 0xb0]
0x0001832fbf82  call     0x183b135e0
0x0001832fbf87  cmp      byte ptr [rsp + 0x11f], 0
0x0001832fbf8f  jns      0x1832fbfa9
0x0001832fbf91  mov      rdx, qword ptr [rsp + 0x118]
0x0001832fbf99  and      rdx, rdi
0x0001832fbf9c  mov      rcx, qword ptr [rsp + 0x108]
0x0001832fbfa4  call     0x183b135e0
0x0001832fbfa9  cmp      byte ptr [rsp + 0x16f], 0
0x0001832fbfb1  jns      0x1832fbfcb
0x0001832fbfb3  and      rdi, qword ptr [rsp + 0x168]
0x0001832fbfbb  mov      rcx, qword ptr [rsp + 0x158]
0x0001832fbfc3  mov      rdx, rdi
0x0001832fbfc6  call     0x183b135e0
0x0001832fbfcb  xor      esi, esi
0x0001832fbfcd  jmp      0x1832fb6cb
0x0001832fbfd2  mov      rdx, qword ptr [rsp + 0x1c0]
0x0001832fbfda  mov      r8, qword ptr [rsp + 0x1c8]
0x0001832fbfe2  call     0x183b61e40
0x0001832fbfe7  jmp      0x1832fbe33
0x0001832fbfec  mov      rdx, qword ptr [rsp + 0x1a8]
0x0001832fbff4  mov      r8, qword ptr [rsp + 0x1b0]
0x0001832fbffc  mov      rcx, rsi
0x0001832fbfff  call     0x183b61e40
0x0001832fc004  jmp      0x1832fbe01
0x0001832fc009  test     r8b, r8b
0x0001832fc00c  jns      0x1832fc01e
0x0001832fc00e  mov      r8, qword ptr [rsp + 0x1c8]
0x0001832fc016  mov      rdx, qword ptr [rsp + 0x1c0]
0x0001832fc01e  call     0x183b61fa0
0x0001832fc023  jmp      0x1832fbe33
0x0001832fc028  test     r8b, r8b
0x0001832fc02b  cmovs    r8, qword ptr [rsp + 0x1b0]
0x0001832fc034  cmovs    rbx, qword ptr [rsp + 0x1a8]
0x0001832fc03d  mov      rcx, rsi
0x0001832fc040  mov      rdx, rbx
0x0001832fc043  call     0x183b61fa0
0x0001832fc048  jmp      0x1832fbe01
0x0001832fc04d  movabs   rdi, 0x7fffffffffffffff
0x0001832fc057  lea      r15, [rsp + 0x158]
0x0001832fc05f  mov      rcx, r15
0x0001832fc062  mov      edx, 0x33
0x0001832fc067  call     0x184ec6ea0
0x0001832fc06c  mov      ecx, 0x30
0x0001832fc071  call     0x18593bfb8
0x0001832fc076  mov      qword ptr [rsp + 0x108], rax
0x0001832fc07e  mov      qword ptr [rsp + 0x110], 0x2e
0x0001832fc08a  lea      r14, [rdi + 0x31]
0x0001832fc08e  mov      qword ptr [rsp + 0x118], r14
0x0001832fc096  lea      rcx, [rax + 0x2e]
0x0001832fc09a  cmp      rcx, rax
0x0001832fc09d  jb       0x1832fc403
0x0001832fc0a3  lea      rdx, [rip + 0xc8d15d6]   ; "..\..\undetectable\core\stors\iniconnector.cpp"
0x0001832fc0aa  cmp      rax, rdx
0x0001832fc0ad  seta     r8b
0x0001832fc0b1  cmp      rcx, rdx
0x0001832fc0b4  setbe    cl
0x0001832fc0b7  or       cl, r8b
0x0001832fc0ba  je       0x1832fc403
0x0001832fc0c0  movups   xmm0, xmmword ptr [rip + 0xc8d15d7]   ; "iniconnector.cpp"
0x0001832fc0c7  movups   xmmword ptr [rax + 0x1e], xmm0
0x0001832fc0cb  movups   xmm0, xmmword ptr [rip + 0xc8d15be]   ; "le\core\stors\iniconnector.cpp"
0x0001832fc0d2  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832fc0d6  movups   xmm0, xmmword ptr [rip + 0xc8d15a3]   ; "..\..\undetectable\core\stors\iniconnector.cpp"
0x0001832fc0dd  movups   xmmword ptr [rax], xmm0
0x0001832fc0e0  mov      byte ptr [rax + 0x2e], 0
0x0001832fc0e4  mov      ecx, 0x30
0x0001832fc0e9  call     0x18593bfb8
0x0001832fc0ee  mov      qword ptr [rsp + 0xb0], rax
0x0001832fc0f6  mov      qword ptr [rsp + 0xb8], 0x24
0x0001832fc102  mov      qword ptr [rsp + 0xc0], r14
0x0001832fc10a  lea      rcx, [rax + 0x24]
0x0001832fc10e  cmp      rcx, rax
0x0001832fc111  jb       0x1832fc403
0x0001832fc117  lea      rdx, [rip + 0xc8d1622]   ; "undetectable::IniConnector::getSWIni"
0x0001832fc11e  cmp      rax, rdx
0x0001832fc121  seta     r8b
0x0001832fc125  cmp      rcx, rdx
0x0001832fc128  setbe    cl
0x0001832fc12b  or       cl, r8b
0x0001832fc12e  je       0x1832fc403
0x0001832fc134  movups   xmm0, xmmword ptr [rip + 0xc8d1615]   ; "iConnector::getSWIni"
0x0001832fc13b  movups   xmmword ptr [rax + 0x10], xmm0
0x0001832fc13f  movups   xmm0, xmmword ptr [rip + 0xc8d15fa]   ; "undetectable::IniConnector::getSWIni"
0x0001832fc146  movups   xmmword ptr [rax], xmm0
0x0001832fc149  mov      dword ptr [rax + 0x20], 0x696e4957
0x0001832fc150  mov      byte ptr [rax + 0x24], 0
0x0001832fc154  lea      r12, [rsp + 0xb0]
0x0001832fc15c  lea      r14, [rsp + 0x1a8]
0x0001832fc164  mov      rcx, r12
0x0001832fc167  mov      rdx, r14
0x0001832fc16a  mov      r8, rax
0x0001832fc16d  mov      r9b, 0x5b
0x0001832fc170  call     0x18a58b3f0
0x0001832fc175  movaps   xmm0, xmmword ptr [r12]
0x0001832fc17a  lea      rcx, [rsp + 0xd0]
0x0001832fc182  movaps   xmmword ptr [rcx], xmm0
0x0001832fc185  mov      rax, qword ptr [r12 + 0x10]
0x0001832fc18a  mov      qword ptr [rcx + 0x10], rax
0x0001832fc18e  xorps    xmm6, xmm6
0x0001832fc191  movaps   xmmword ptr [r12], xmm6
0x0001832fc196  xor      ebp, ebp
0x0001832fc198  mov      qword ptr [r12 + 0x10], rbp
0x0001832fc19d  lea      rdx, [rip + 0xc8d1298]   ; " ( "
0x0001832fc1a4  mov      r8d, 3
0x0001832fc1aa  call     0x183b596f0
0x0001832fc1af  movups   xmm0, xmmword ptr [rax]
0x0001832fc1b2  lea      rcx, [rsp + 0xf0]
0x0001832fc1ba  movaps   xmmword ptr [rcx], xmm0
0x0001832fc1bd  mov      rdx, qword ptr [rax + 0x10]
0x0001832fc1c1  mov      qword ptr [rcx + 0x10], rdx
0x0001832fc1c5  mov      qword ptr [rax + 0x10], rbp
0x0001832fc1c9  movups   xmmword ptr [rax], xmm6
0x0001832fc1cc  lea      rdx, [rsp + 0x108]
0x0001832fc1d4  movsx    r8, byte ptr [rdx + 0x17]
0x0001832fc1d9  test     r8, r8
0x0001832fc1dc  cmovs    r8, qword ptr [rdx + 8]
0x0001832fc1e1  cmovs    rdx, qword ptr [rdx]
0x0001832fc1e5  call     0x183b596f0
0x0001832fc1ea  movups   xmm0, xmmword ptr [rax]
0x0001832fc1ed  lea      rcx, [rsp + 0x120]
0x0001832fc1f5  movaps   xmmword ptr [rcx], xmm0
0x0001832fc1f8  mov      rdx, qword ptr [rax + 0x10]
0x0001832fc1fc  mov      qword ptr [rcx + 0x10], rdx
0x0001832fc200  mov      qword ptr [rax + 0x10], rbp
0x0001832fc204  movups   xmmword ptr [rax], xmm6
0x0001832fc207  lea      rdx, [rip + 0xc77fe96]   ; 0x18fa7c0a4
0x0001832fc20e  mov      r8d, 1
0x0001832fc214  call     0x183b596f0
0x0001832fc219  movups   xmm0, xmmword ptr [rax]
0x0001832fc21c  lea      rcx, [rsp + 0x140]
0x0001832fc224  movaps   xmmword ptr [rcx], xmm0
0x0001832fc227  mov      rdx, qword ptr [rax + 0x10]
0x0001832fc22b  mov      qword ptr [rcx + 0x10], rdx
0x0001832fc22f  mov      qword ptr [rax + 0x10], rbp
0x0001832fc233  movups   xmmword ptr [rax], xmm6
0x0001832fc236  movsx    r8, byte ptr [rsp + 0x16f]
0x0001832fc23f  test     r8, r8
0x0001832fc242  cmovs    r8, qword ptr [rsp + 0x160]
0x0001832fc24b  cmovs    r15, qword ptr [rsp + 0x158]
0x0001832fc254  mov      rdx, r15
0x0001832fc257  call     0x183b596f0
0x0001832fc25c  movups   xmm0, xmmword ptr [rax]
0x0001832fc25f  lea      rcx, [rsp + 0x170]
0x0001832fc267  movaps   xmmword ptr [rcx], xmm0
0x0001832fc26a  mov      rdx, qword ptr [rax + 0x10]
0x0001832fc26e  mov      qword ptr [rcx + 0x10], rdx
0x0001832fc272  mov      qword ptr [rax + 0x10], rbp
0x0001832fc276  movups   xmmword ptr [rax], xmm6
0x0001832fc279  lea      rdx, [rip + 0xc8d1178]   ; " )]"
0x0001832fc280  mov      r8d, 3
0x0001832fc286  call     0x183b596f0
0x0001832fc28b  movups   xmm0, xmmword ptr [rax]
0x0001832fc28e  movaps   xmmword ptr [rsp + 0x190], xmm0
0x0001832fc296  mov      rcx, qword ptr [rax + 0x10]
0x0001832fc29a  mov      qword ptr [rsp + 0x1a0], rcx
0x0001832fc2a2  mov      qword ptr [rax + 0x10], rbp
0x0001832fc2a6  movups   xmmword ptr [rax], xmm6
0x0001832fc2a9  movsx    r15, byte ptr [rsp + 0x1f7]
0x0001832fc2b2  test     r15, r15
0x0001832fc2b5  cmovs    r15, qword ptr [rsp + 0x1e8]
0x0001832fc2be  cmovs    rbx, qword ptr [rsp + 0x1e0]
0x0001832fc2c7  movaps   xmmword ptr [rsp + 0x90], xmm6
0x0001832fc2cf  mov      qword ptr [rsp + 0xa0], rbp
0x0001832fc2d7  lea      r12, [r15 + 0x1e]
0x0001832fc2db  lea      rax, [rdi - 0x11]
0x0001832fc2df  cmp      r12, rax
0x0001832fc2e2  ja       0x1832fc405
0x0001832fc2e8  cmp      r12, 0x16
0x0001832fc2ec  ja       0x1832fc39e
0x0001832fc2f2  lea      rax, [rsp + 0x90]
0x0001832fc2fa  mov      byte ptr [rax + 0x17], r12b
0x0001832fc2fe  lea      r12, [rax + 0x1e]
0x0001832fc302  cmp      r12, rax
0x0001832fc305  jb       0x1832fc403
0x0001832fc30b  lea      rcx, [rip + 0xc8d1456]   ; "Unable to parse settigs file: "
0x0001832fc312  cmp      rax, rcx
0x0001832fc315  seta     dl
0x0001832fc318  cmp      r12, rcx
0x0001832fc31b  setbe    cl
0x0001832fc31e  or       cl, dl
0x0001832fc320  je       0x1832fc403
0x0001832fc326  movups   xmm0, xmmword ptr [rip + 0xc8d1449]   ; "e settigs file: "
0x0001832fc32d  movups   xmmword ptr [rax + 0xe], xmm0
0x0001832fc331  movups   xmm0, xmmword ptr [rip + 0xc8d1430]   ; "Unable to parse settigs file: "
0x0001832fc338  movups   xmmword ptr [rax], xmm0
0x0001832fc33b  lea      rax, [r12 + r15]
0x0001832fc33f  cmp      rax, r12
0x0001832fc342  jb       0x1832fc403
0x0001832fc348  cmp      rbx, r12
0x0001832fc34b  jb       0x1832fc356
0x0001832fc34d  cmp      rbx, rax
0x0001832fc350  jb       0x1832fc403
0x0001832fc356  mov      rcx, r12
0x0001832fc359  mov      rdx, rbx
0x0001832fc35c  mov      r8, r15
0x0001832fc35f  call     0x185994bf0
0x0001832fc364  mov      byte ptr [r12 + r15], 0
0x0001832fc369  lea      rdx, [rsp + 0x90]
0x0001832fc371  lea      r8, [rsp + 0x190]
0x0001832fc379  mov      rcx, r14
0x0001832fc37c  call     0x181199c30
0x0001832fc381  cmp      rsi, r14
0x0001832fc384  je       0x1832fbe33
0x0001832fc38a  cmp      byte ptr [rsi + 0x17], 0
0x0001832fc38e  movzx    r8d, byte ptr [rsp + 0x1bf]
0x0001832fc397  js       0x1832fc3d3
0x0001832fc399  jmp      0x1832fbde1
0x0001832fc39e  lea      rbp, [rdi - 0xf]
0x0001832fc3a2  and      rbp, r12
0x0001832fc3a5  lea      rcx, [rbp + 0x10]
0x0001832fc3a9  call     0x18593bfb8
0x0001832fc3ae  lea      rcx, [rdi + rbp]
0x0001832fc3b2  add      rcx, 0x11
0x0001832fc3b6  mov      qword ptr [rsp + 0x90], rax
0x0001832fc3be  mov      qword ptr [rsp + 0x98], r12
0x0001832fc3c6  mov      qword ptr [rsp + 0xa0], rcx
0x0001832fc3ce  jmp      0x1832fc2fe
0x0001832fc3d3  test     r8b, r8b
0x0001832fc3d6  cmovs    r8, qword ptr [rsp + 0x1b0]
0x0001832fc3df  cmovs    r14, qword ptr [rsp + 0x1a8]
0x0001832fc3e8  mov      rcx, rsi
0x0001832fc3eb  mov      rdx, r14
0x0001832fc3ee  jmp      0x1832fc043
0x0001832fc3f3  mov      rcx, qword ptr [rsp + 0x258]
0x0001832fc3fb  xor      rcx, rsp
0x0001832fc3fe  call     0x185941420
0x0001832fc403  ud2      
0x0001832fc405  call     0x185a1ffd0
0x0001832fc40a  int3     
