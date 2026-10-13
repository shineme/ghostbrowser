0x00014016e100  mov      rax, rsp
0x00014016e103  mov      qword ptr [rax + 0x20], rbx
0x00014016e107  mov      qword ptr [rax + 0x18], r8
0x00014016e10b  mov      qword ptr [rax + 8], rcx
0x00014016e10f  push     rbp
0x00014016e110  push     rsi
0x00014016e111  push     rdi
0x00014016e112  push     r12
0x00014016e114  push     r13
0x00014016e116  push     r14
0x00014016e118  push     r15
0x00014016e11a  lea      rbp, [rax - 0xd18]
0x00014016e121  sub      rsp, 0xde0
0x00014016e128  movaps   xmmword ptr [rax - 0x48], xmm6
0x00014016e12c  mov      r13, r9
0x00014016e12f  mov      rsi, rdx
0x00014016e132  xor      r14d, r14d
0x00014016e135  mov      dword ptr [rbp - 0x7c], r14d
0x00014016e139  mov      rdi, qword ptr [rbp + 0xd40]
0x00014016e140  add      rdi, 0x340
0x00014016e147  mov      rcx, rdi
0x00014016e14a  call     qword ptr [rip + 0x2f6c08]   ; 0x140464d58
0x00014016e150  test     al, al
0x00014016e152  je       0x14016e15d
0x00014016e154  mov      rcx, rsi
0x00014016e157  call     qword ptr [rip + 0x2f6bfb]   ; 0x140464d58
0x00014016e15d  mov      rcx, rdi
0x00014016e160  call     qword ptr [rip + 0x2f6c6a]   ; 0x140464dd0
0x00014016e166  mov      r12, rax
0x00014016e169  mov      rcx, rsi
0x00014016e16c  call     qword ptr [rip + 0x2f6c5e]   ; 0x140464dd0
0x00014016e172  add      rax, 0xa
0x00014016e176  add      r12, rax
0x00014016e179  xor      r8d, r8d
0x00014016e17c  mov      rdx, r12
0x00014016e17f  lea      rcx, [rbp + 0xf8]
0x00014016e186  call     qword ptr [rip + 0x2f6bc4]   ; 0x140464d50
0x00014016e18c  nop      
0x00014016e18d  mov      rax, qword ptr [rbp + 0x100]
0x00014016e194  mov      qword ptr [rbp + 0xd28], rax
0x00014016e19b  mov      qword ptr [rsp + 0x68], rax
0x00014016e1a0  mov      rcx, rdi
0x00014016e1a3  call     qword ptr [rip + 0x2f6c27]   ; 0x140464dd0
0x00014016e1a9  mov      r15, rax
0x00014016e1ac  test     rax, rax
0x00014016e1af  je       0x14016e1ce
0x00014016e1b1  lea      rbx, [rax + rax]
0x00014016e1b5  mov      rcx, rdi
0x00014016e1b8  call     qword ptr [rip + 0x2f6be2]   ; 0x140464da0
0x00014016e1be  mov      rdx, rax
0x00014016e1c1  mov      r8, rbx
0x00014016e1c4  mov      rcx, qword ptr [rsp + 0x68]
0x00014016e1c9  call     0x140431dea
0x00014016e1ce  mov      rcx, qword ptr [rsp + 0x68]
0x00014016e1d3  lea      rax, [rcx + r15*2]
0x00014016e1d7  mov      qword ptr [rsp + 0x68], rax
0x00014016e1dc  mov      rcx, rsi
0x00014016e1df  call     qword ptr [rip + 0x2f6beb]   ; 0x140464dd0
0x00014016e1e5  mov      rdi, rax
0x00014016e1e8  test     rax, rax
0x00014016e1eb  je       0x14016e20a
0x00014016e1ed  lea      rbx, [rax + rax]
0x00014016e1f1  mov      rcx, rsi
0x00014016e1f4  call     qword ptr [rip + 0x2f6ba6]   ; 0x140464da0
0x00014016e1fa  mov      rdx, rax
0x00014016e1fd  mov      r8, rbx
0x00014016e200  mov      rcx, qword ptr [rsp + 0x68]
0x00014016e205  call     0x140431dea
0x00014016e20a  mov      rcx, qword ptr [rsp + 0x68]
0x00014016e20f  lea      rax, [rcx + rdi*2]
0x00014016e213  mov      qword ptr [rsp + 0x68], rax
0x00014016e218  mov      qword ptr [rbp + 0x10], 0xa
0x00014016e220  lea      rcx, [rip + 0x31b181]   ; "/startInfo"
0x00014016e227  call     qword ptr [rip + 0x2f6c9b]   ; 0x140464ec8
0x00014016e22d  mov      qword ptr [rbp + 0x18], rax
0x00014016e231  movaps   xmm0, xmmword ptr [rbp + 0x10]
0x00014016e235  movdqa   xmmword ptr [rbp + 0x730], xmm0
0x00014016e23d  lea      rdx, [rsp + 0x68]
0x00014016e242  lea      rcx, [rbp + 0x730]
0x00014016e249  call     qword ptr [rip + 0x2f6af1]   ; 0x140464d40
0x00014016e24f  mov      rdx, qword ptr [rsp + 0x68]
0x00014016e254  sub      rdx, qword ptr [rbp + 0xd28]
0x00014016e25b  sar      rdx, 1
0x00014016e25e  cmp      r12, rdx
0x00014016e261  je       0x14016e270
0x00014016e263  lea      rcx, [rbp + 0xf8]
0x00014016e26a  call     qword ptr [rip + 0x2f6b58]   ; 0x140464dc8
0x00014016e270  lea      rdx, [rbp + 0xf8]
0x00014016e277  lea      rcx, [rbp + 0x770]
0x00014016e27e  call     qword ptr [rip + 0x2f6b54]   ; 0x140464dd8
0x00014016e284  nop      
0x00014016e285  lea      rcx, [rbp + 0xf8]
0x00014016e28c  call     qword ptr [rip + 0x2f6d16]   ; 0x140464fa8
0x00014016e292  mov      dword ptr [rbp - 0x7c], 6
0x00014016e299  mov      r15, qword ptr [rbp + 0xd20]
0x00014016e2a0  lea      rcx, [r15 + 0x18]
0x00014016e2a4  lea      rdx, [rbp + 0x770]
0x00014016e2ab  call     qword ptr [rip + 0x2f6bff]   ; 0x140464eb0
0x00014016e2b1  lea      rcx, [rbp + 0x770]
0x00014016e2b8  call     qword ptr [rip + 0x2f6cea]   ; 0x140464fa8
0x00014016e2be  lea      rdx, [rip + 0x31b0f3]   ; "additional_info"
0x00014016e2c5  lea      rcx, [rbp + 0x788]
0x00014016e2cc  call     qword ptr [rip + 0x2f6bbe]   ; 0x140464e90
0x00014016e2d2  nop      
0x00014016e2d3  lea      r8, [rbp + 0x788]
0x00014016e2da  lea      rdx, [rbp + 0x7e8]
0x00014016e2e1  mov      rcx, r13
0x00014016e2e4  call     qword ptr [rip + 0x2f655e]   ; 0x140464848
0x00014016e2ea  nop      
0x00014016e2eb  lea      rdx, [rbp + 0x7d0]
0x00014016e2f2  mov      rcx, rax
0x00014016e2f5  call     qword ptr [rip + 0x2f642d]   ; 0x140464728
0x00014016e2fb  nop      
0x00014016e2fc  lea      rdx, [rbp + 0x7b8]
0x00014016e303  mov      rcx, rax
0x00014016e306  call     qword ptr [rip + 0x2f67ec]   ; 0x140464af8
0x00014016e30c  nop      
0x00014016e30d  xor      r8d, r8d
0x00014016e310  mov      rdx, rax
0x00014016e313  lea      rcx, [rbp - 0x50]
0x00014016e317  call     0x1402fc7f0
0x00014016e31c  nop      
0x00014016e31d  lea      rcx, [rbp + 0x7b8]
0x00014016e324  call     qword ptr [rip + 0x2f6af6]   ; 0x140464e20
0x00014016e32a  nop      
0x00014016e32b  lea      rcx, [rbp + 0x7d0]
0x00014016e332  call     qword ptr [rip + 0x2f6c70]   ; 0x140464fa8
0x00014016e338  nop      
0x00014016e339  lea      rcx, [rbp + 0x7e8]
0x00014016e340  call     qword ptr [rip + 0x2f6532]   ; 0x140464878
0x00014016e346  nop      
0x00014016e347  lea      rcx, [rbp + 0x788]
0x00014016e34e  call     qword ptr [rip + 0x2f6c54]   ; 0x140464fa8
0x00014016e354  lea      rdx, [rip + 0x318975]   ; "jsDefineText"
0x00014016e35b  lea      rcx, [rbp + 0x2c0]
0x00014016e362  call     qword ptr [rip + 0x2f6b28]   ; 0x140464e90
0x00014016e368  nop      
0x00014016e369  lea      r8, [rbp + 0x2c0]
0x00014016e370  lea      rdx, [rbp + 0x800]
0x00014016e377  mov      rcx, r13
0x00014016e37a  call     qword ptr [rip + 0x2f64c8]   ; 0x140464848
0x00014016e380  nop      
0x00014016e381  lea      rdx, [rbp - 0x20]
0x00014016e385  mov      rcx, rax
0x00014016e388  call     qword ptr [rip + 0x2f639a]   ; 0x140464728
0x00014016e38e  nop      
0x00014016e38f  lea      rcx, [rbp + 0x800]
0x00014016e396  call     qword ptr [rip + 0x2f64dc]   ; 0x140464878
0x00014016e39c  nop      
0x00014016e39d  lea      rcx, [rbp + 0x2c0]
0x00014016e3a4  call     qword ptr [rip + 0x2f6bfe]   ; 0x140464fa8
0x00014016e3aa  lea      rdx, [rip + 0x3189c7]   ; "randomInt0"
0x00014016e3b1  lea      rcx, [rbp + 0x2d8]
0x00014016e3b8  call     qword ptr [rip + 0x2f6ad2]   ; 0x140464e90
0x00014016e3be  nop      
0x00014016e3bf  lea      r8, [rbp + 0x2d8]
0x00014016e3c6  lea      rdx, [rbp + 0x830]
0x00014016e3cd  mov      rcx, r13
0x00014016e3d0  call     qword ptr [rip + 0x2f6472]   ; 0x140464848
0x00014016e3d6  nop      
0x00014016e3d7  lea      rdx, [rbp + 0x818]
0x00014016e3de  mov      rcx, rax
0x00014016e3e1  call     qword ptr [rip + 0x2f6341]   ; 0x140464728
0x00014016e3e7  nop      
0x00014016e3e8  xor      edx, edx
0x00014016e3ea  lea      r8d, [rdx + 0xa]
0x00014016e3ee  mov      rcx, rax
0x00014016e3f1  call     qword ptr [rip + 0x2f6889]   ; 0x140464c80
0x00014016e3f7  mov      qword ptr [rbp - 8], rax
0x00014016e3fb  lea      rcx, [rbp + 0x818]
0x00014016e402  call     qword ptr [rip + 0x2f6ba0]   ; 0x140464fa8
0x00014016e408  nop      
0x00014016e409  lea      rcx, [rbp + 0x830]
0x00014016e410  call     qword ptr [rip + 0x2f6462]   ; 0x140464878
0x00014016e416  nop      
0x00014016e417  lea      rcx, [rbp + 0x2d8]
0x00014016e41e  call     qword ptr [rip + 0x2f6b84]   ; 0x140464fa8
0x00014016e424  mov      rdx, r15
0x00014016e427  lea      rcx, [rsp + 0x38]
0x00014016e42c  call     0x1402bd1b0
0x00014016e431  lea      rdx, [rsi + 0x18]
0x00014016e435  lea      rcx, [rsp + 0x38]
0x00014016e43a  call     0x1402bd3a0
0x00014016e43f  mov      rbx, rax
0x00014016e442  lea      rcx, [rsi + 0x300]
0x00014016e449  call     0x140260d70
0x00014016e44e  mov      byte ptr [rbp + 0xd20], al
0x00014016e454  lea      rdx, [rbp + 0xd20]
0x00014016e45b  mov      rcx, rbx
0x00014016e45e  call     0x1402bd1c0
0x00014016e463  mov      rbx, rax
0x00014016e466  lea      rdx, [rbp + 0x8c0]
0x00014016e46d  lea      rcx, [rsi + 0x300]
0x00014016e474  call     0x1402278d0
0x00014016e479  nop      
0x00014016e47a  mov      rdx, rax
0x00014016e47d  mov      rcx, rbx
0x00014016e480  call     0x1402bd3a0
0x00014016e485  mov      rbx, rax
0x00014016e488  lea      rcx, [rsi + 0x2e0]
0x00014016e48f  call     0x140260d70
0x00014016e494  mov      byte ptr [rbp + 0xd28], al
0x00014016e49a  lea      rdx, [rbp + 0xd28]
0x00014016e4a1  mov      rcx, rbx
0x00014016e4a4  call     0x1402bd1c0
0x00014016e4a9  mov      rbx, rax
0x00014016e4ac  lea      rdx, [rbp + 0x8a8]
0x00014016e4b3  lea      rcx, [rsi + 0x2e0]
0x00014016e4ba  call     0x1402278d0
0x00014016e4bf  nop      
0x00014016e4c0  mov      rdx, rax
0x00014016e4c3  mov      rcx, rbx
0x00014016e4c6  call     0x1402bd3a0
0x00014016e4cb  lea      rdx, [rsi + 0x4a0]
0x00014016e4d2  mov      rcx, rax
0x00014016e4d5  call     0x1402bd1c0
0x00014016e4da  lea      rdx, [rsi + 0x4a1]
0x00014016e4e1  mov      rcx, rax
0x00014016e4e4  call     0x1402bd1c0
0x00014016e4e9  lea      rdx, [rsi + 0x450]
0x00014016e4f0  mov      rcx, rax
0x00014016e4f3  call     0x1402bd230
0x00014016e4f8  lea      rdx, [rsi + 0x44c]
0x00014016e4ff  mov      rcx, rax
0x00014016e502  call     0x1402bd230
0x00014016e507  lea      rdx, [rsi + 0x440]
0x00014016e50e  mov      rcx, rax
0x00014016e511  call     0x1402bd1c0
0x00014016e516  mov      rbx, rax
0x00014016e519  lea      rdx, [rip + 0x318928]   ; "CanvasWebglRandomParameter"
0x00014016e520  lea      rcx, [rbp + 0x338]
0x00014016e527  call     qword ptr [rip + 0x2f6963]   ; 0x140464e90
0x00014016e52d  nop      
0x00014016e52e  lea      r8, [rbp + 0x338]
0x00014016e535  lea      rdx, [rbp + 0x890]
0x00014016e53c  mov      rcx, r13
0x00014016e53f  call     qword ptr [rip + 0x2f6303]   ; 0x140464848
0x00014016e545  nop      
0x00014016e546  xor      edx, edx
0x00014016e548  mov      rcx, rax
0x00014016e54b  call     qword ptr [rip + 0x2f631f]   ; 0x140464870
0x00014016e551  mov      dword ptr [rbp - 0x48], eax
0x00014016e554  lea      rdx, [rbp - 0x48]
0x00014016e558  mov      rcx, rbx
0x00014016e55b  call     0x1402bd230
0x00014016e560  mov      rbx, rax
0x00014016e563  lea      rdx, [rip + 0x31894e]   ; "CanvasWebglRandomParameter2"
0x00014016e56a  lea      rcx, [rbp + 0x320]
0x00014016e571  call     qword ptr [rip + 0x2f6919]   ; 0x140464e90
0x00014016e577  nop      
0x00014016e578  lea      r8, [rbp + 0x320]
0x00014016e57f  lea      rdx, [rbp + 0x878]
0x00014016e586  mov      rcx, r13
0x00014016e589  call     qword ptr [rip + 0x2f62b9]   ; 0x140464848
0x00014016e58f  nop      
0x00014016e590  xor      edx, edx
0x00014016e592  mov      rcx, rax
0x00014016e595  call     qword ptr [rip + 0x2f62d5]   ; 0x140464870
0x00014016e59b  mov      dword ptr [rbp + 8], eax
0x00014016e59e  lea      rdx, [rbp + 8]
0x00014016e5a2  mov      rcx, rbx
0x00014016e5a5  call     0x1402bd230
0x00014016e5aa  mov      rbx, rax
0x00014016e5ad  lea      rdx, [rip + 0x31895c]   ; "pixel1"
0x00014016e5b4  lea      rcx, [rbp + 0x308]
0x00014016e5bb  call     qword ptr [rip + 0x2f68cf]   ; 0x140464e90
0x00014016e5c1  nop      
0x00014016e5c2  lea      r8, [rbp + 0x308]
0x00014016e5c9  lea      rdx, [rbp + 0x860]
0x00014016e5d0  mov      rcx, r13
0x00014016e5d3  call     qword ptr [rip + 0x2f626f]   ; 0x140464848
0x00014016e5d9  nop      
0x00014016e5da  xor      edx, edx
0x00014016e5dc  mov      rcx, rax
0x00014016e5df  call     qword ptr [rip + 0x2f628b]   ; 0x140464870
0x00014016e5e5  mov      dword ptr [rbp + 0xc], eax
0x00014016e5e8  lea      rdx, [rbp + 0xc]
0x00014016e5ec  mov      rcx, rbx
0x00014016e5ef  call     0x1402bd230
0x00014016e5f4  mov      rbx, rax
0x00014016e5f7  lea      rdx, [rip + 0x31893a]   ; "pixel2"
0x00014016e5fe  lea      rcx, [rbp + 0x2f0]
0x00014016e605  call     qword ptr [rip + 0x2f6885]   ; 0x140464e90
0x00014016e60b  nop      
0x00014016e60c  lea      r8, [rbp + 0x2f0]
0x00014016e613  lea      rdx, [rbp + 0x7a0]
0x00014016e61a  mov      rcx, r13
0x00014016e61d  call     qword ptr [rip + 0x2f6225]   ; 0x140464848
0x00014016e623  nop      
0x00014016e624  xor      edx, edx
0x00014016e626  mov      rcx, rax
0x00014016e629  call     qword ptr [rip + 0x2f6241]   ; 0x140464870
0x00014016e62f  mov      dword ptr [rbp - 0x78], eax
0x00014016e632  lea      rdx, [rbp - 0x78]
0x00014016e636  mov      rcx, rbx
0x00014016e639  call     0x1402bd230
0x00014016e63e  nop      
0x00014016e63f  lea      rcx, [rbp + 0x7a0]
0x00014016e646  call     qword ptr [rip + 0x2f622c]   ; 0x140464878
0x00014016e64c  nop      
0x00014016e64d  lea      rcx, [rbp + 0x2f0]
0x00014016e654  call     qword ptr [rip + 0x2f694e]   ; 0x140464fa8
0x00014016e65a  nop      
0x00014016e65b  lea      rcx, [rbp + 0x860]
0x00014016e662  call     qword ptr [rip + 0x2f6210]   ; 0x140464878
0x00014016e668  nop      
0x00014016e669  lea      rcx, [rbp + 0x308]
0x00014016e670  call     qword ptr [rip + 0x2f6932]   ; 0x140464fa8
0x00014016e676  nop      
0x00014016e677  lea      rcx, [rbp + 0x878]
0x00014016e67e  call     qword ptr [rip + 0x2f61f4]   ; 0x140464878
0x00014016e684  nop      
0x00014016e685  lea      rcx, [rbp + 0x320]
0x00014016e68c  call     qword ptr [rip + 0x2f6916]   ; 0x140464fa8
0x00014016e692  nop      
0x00014016e693  lea      rcx, [rbp + 0x890]
0x00014016e69a  call     qword ptr [rip + 0x2f61d8]   ; 0x140464878
0x00014016e6a0  nop      
0x00014016e6a1  lea      rcx, [rbp + 0x338]
0x00014016e6a8  call     qword ptr [rip + 0x2f68fa]   ; 0x140464fa8
0x00014016e6ae  nop      
0x00014016e6af  lea      rcx, [rbp + 0x8a8]
0x00014016e6b6  call     qword ptr [rip + 0x2f68ec]   ; 0x140464fa8
0x00014016e6bc  nop      
0x00014016e6bd  lea      rcx, [rbp + 0x8c0]
0x00014016e6c4  call     qword ptr [rip + 0x2f68de]   ; 0x140464fa8
0x00014016e6ca  lea      rdx, [rsi + 0x448]
0x00014016e6d1  lea      rcx, [rsp + 0x38]
0x00014016e6d6  call     0x1402bd1c0
0x00014016e6db  cmp      byte ptr [rsi + 0x448], 0
0x00014016e6e2  je       0x14016eb2b
0x00014016e6e8  lea      rdx, [rip + 0x31acd9]   ; "gpuDataObj"
0x00014016e6ef  lea      rcx, [rbp + 0x368]
0x00014016e6f6  call     qword ptr [rip + 0x2f6794]   ; 0x140464e90
0x00014016e6fc  nop      
0x00014016e6fd  lea      r8, [rbp + 0x368]
0x00014016e704  lea      rdx, [rbp + 0xbf8]
0x00014016e70b  lea      rcx, [rbp - 0x50]
0x00014016e70f  call     qword ptr [rip + 0x2f612b]   ; 0x140464840
0x00014016e715  lea      rdx, [rbp]
0x00014016e719  mov      rcx, rax
0x00014016e71c  call     qword ptr [rip + 0x2f612e]   ; 0x140464850
0x00014016e722  mov      rbx, rax
0x00014016e725  lea      rdx, [rip + 0x31acac]   ; "adapterInfo"
0x00014016e72c  lea      rcx, [rbp + 0x350]
0x00014016e733  call     qword ptr [rip + 0x2f6757]   ; 0x140464e90
0x00014016e739  nop      
0x00014016e73a  lea      r8, [rbp + 0x350]
0x00014016e741  lea      rdx, [rbp + 0xbe8]
0x00014016e748  mov      rcx, rbx
0x00014016e74b  call     qword ptr [rip + 0x2f60ef]   ; 0x140464840
0x00014016e751  lea      rdx, [rsp + 0x78]
0x00014016e756  mov      rcx, rax
0x00014016e759  call     qword ptr [rip + 0x2f60f1]   ; 0x140464850
0x00014016e75f  nop      
0x00014016e760  lea      rcx, [rbp + 0x350]
0x00014016e767  call     qword ptr [rip + 0x2f683b]   ; 0x140464fa8
0x00014016e76d  nop      
0x00014016e76e  lea      rcx, [rbp]
0x00014016e772  call     qword ptr [rip + 0x2f6148]   ; 0x1404648c0
0x00014016e778  nop      
0x00014016e779  lea      rcx, [rbp + 0x368]
0x00014016e780  call     qword ptr [rip + 0x2f6822]   ; 0x140464fa8
0x00014016e786  lea      rcx, [rsp + 0x78]
0x00014016e78b  call     qword ptr [rip + 0x2f5d9f]   ; 0x140464530
0x00014016e791  lea      rdx, [rsp + 0x30]
0x00014016e796  lea      rcx, [rsp + 0x38]
0x00014016e79b  test     al, al
0x00014016e79d  jne      0x14016eb15
0x00014016e7a3  mov      byte ptr [rsp + 0x30], 1
0x00014016e7a8  call     0x1402bd1c0
0x00014016e7ad  mov      rdi, rax
0x00014016e7b0  lea      rdx, [rip + 0x31ac31]   ; "architecture"
0x00014016e7b7  lea      rcx, [rbp + 0x458]
0x00014016e7be  call     qword ptr [rip + 0x2f66cc]   ; 0x140464e90
0x00014016e7c4  nop      
0x00014016e7c5  lea      r8, [rbp + 0x458]
0x00014016e7cc  lea      rdx, [rbp + 0xc48]
0x00014016e7d3  lea      rcx, [rsp + 0x78]
0x00014016e7d8  call     qword ptr [rip + 0x2f6062]   ; 0x140464840
0x00014016e7de  mov      rbx, rax
0x00014016e7e1  lea      rcx, [rbp + 0x440]
0x00014016e7e8  call     qword ptr [rip + 0x2f66d2]   ; 0x140464ec0
0x00014016e7ee  nop      
0x00014016e7ef  lea      r8, [rbp + 0x440]
0x00014016e7f6  lea      rdx, [rbp + 0x9b0]
0x00014016e7fd  mov      rcx, rbx
0x00014016e800  call     qword ptr [rip + 0x2f5d52]   ; 0x140464558
0x00014016e806  nop      
0x00014016e807  lea      rdx, [rbp + 0x998]
0x00014016e80e  mov      rcx, rax
0x00014016e811  call     qword ptr [rip + 0x2f62e1]   ; 0x140464af8
0x00014016e817  nop      
0x00014016e818  mov      rdx, rax
0x00014016e81b  mov      rcx, rdi
0x00014016e81e  call     0x1402bd310
0x00014016e823  mov      rdi, rax
0x00014016e826  lea      rdx, [rip + 0x31abcb]   ; "driver"
0x00014016e82d  lea      rcx, [rbp + 0x428]
0x00014016e834  call     qword ptr [rip + 0x2f6656]   ; 0x140464e90
0x00014016e83a  nop      
0x00014016e83b  lea      r8, [rbp + 0x428]
0x00014016e842  lea      rdx, [rbp + 0xbc8]
0x00014016e849  lea      rcx, [rsp + 0x78]
0x00014016e84e  call     qword ptr [rip + 0x2f5fec]   ; 0x140464840
0x00014016e854  mov      rbx, rax
0x00014016e857  lea      rcx, [rbp + 0x410]
0x00014016e85e  call     qword ptr [rip + 0x2f665c]   ; 0x140464ec0
0x00014016e864  nop      
0x00014016e865  lea      r8, [rbp + 0x410]
0x00014016e86c  lea      rdx, [rbp + 0x980]
0x00014016e873  mov      rcx, rbx
0x00014016e876  call     qword ptr [rip + 0x2f5cdc]   ; 0x140464558
0x00014016e87c  nop      
0x00014016e87d  lea      rdx, [rbp + 0x968]
0x00014016e884  mov      rcx, rax
0x00014016e887  call     qword ptr [rip + 0x2f626b]   ; 0x140464af8
0x00014016e88d  nop      
0x00014016e88e  mov      rdx, rax
0x00014016e891  mov      rcx, rdi
0x00014016e894  call     0x1402bd310
0x00014016e899  mov      rdi, rax
0x00014016e89c  lea      rdx, [rip + 0x31ab5d]   ; "vendor"
0x00014016e8a3  lea      rcx, [rbp + 0x3f8]
0x00014016e8aa  call     qword ptr [rip + 0x2f65e0]   ; 0x140464e90
0x00014016e8b0  nop      
0x00014016e8b1  lea      r8, [rbp + 0x3f8]
0x00014016e8b8  lea      rdx, [rbp + 0xca8]
0x00014016e8bf  lea      rcx, [rsp + 0x78]
0x00014016e8c4  call     qword ptr [rip + 0x2f5f76]   ; 0x140464840
0x00014016e8ca  mov      rbx, rax
0x00014016e8cd  lea      rcx, [rbp + 0x3e0]
0x00014016e8d4  call     qword ptr [rip + 0x2f65e6]   ; 0x140464ec0
0x00014016e8da  nop      
0x00014016e8db  lea      r8, [rbp + 0x3e0]
0x00014016e8e2  lea      rdx, [rbp + 0x950]
0x00014016e8e9  mov      rcx, rbx
0x00014016e8ec  call     qword ptr [rip + 0x2f5c66]   ; 0x140464558
0x00014016e8f2  nop      
0x00014016e8f3  lea      rdx, [rbp + 0x938]
0x00014016e8fa  mov      rcx, rax
0x00014016e8fd  call     qword ptr [rip + 0x2f61f5]   ; 0x140464af8
0x00014016e903  nop      
0x00014016e904  mov      rdx, rax
0x00014016e907  mov      rcx, rdi
0x00014016e90a  call     0x1402bd310
0x00014016e90f  mov      rdi, rax
0x00014016e912  lea      rdx, [rip + 0x31aaef]   ; "device"
0x00014016e919  lea      rcx, [rbp + 0x3c8]
0x00014016e920  call     qword ptr [rip + 0x2f656a]   ; 0x140464e90
0x00014016e926  nop      
0x00014016e927  lea      r8, [rbp + 0x3c8]
0x00014016e92e  lea      rdx, [rbp + 0xc98]
0x00014016e935  lea      rcx, [rsp + 0x78]
0x00014016e93a  call     qword ptr [rip + 0x2f5f00]   ; 0x140464840
0x00014016e940  mov      rbx, rax
0x00014016e943  lea      rcx, [rbp + 0x3b0]
0x00014016e94a  call     qword ptr [rip + 0x2f6570]   ; 0x140464ec0
0x00014016e950  nop      
0x00014016e951  lea      r8, [rbp + 0x3b0]
0x00014016e958  lea      rdx, [rbp + 0xb30]
0x00014016e95f  mov      rcx, rbx
0x00014016e962  call     qword ptr [rip + 0x2f5bf0]   ; 0x140464558
0x00014016e968  nop      
0x00014016e969  lea      rdx, [rbp + 0x908]
0x00014016e970  mov      rcx, rax
0x00014016e973  call     qword ptr [rip + 0x2f617f]   ; 0x140464af8
0x00014016e979  nop      
0x00014016e97a  mov      rdx, rax
0x00014016e97d  mov      rcx, rdi
0x00014016e980  call     0x1402bd310
0x00014016e985  mov      rdi, rax
0x00014016e988  lea      rdx, [rip + 0x3150c1]   ; "description"
0x00014016e98f  lea      rcx, [rbp + 0x398]
0x00014016e996  call     qword ptr [rip + 0x2f64f4]   ; 0x140464e90
0x00014016e99c  nop      
0x00014016e99d  lea      r8, [rbp + 0x398]
0x00014016e9a4  lea      rdx, [rbp + 0xc88]
0x00014016e9ab  lea      rcx, [rsp + 0x78]
0x00014016e9b0  call     qword ptr [rip + 0x2f5e8a]   ; 0x140464840
0x00014016e9b6  mov      rbx, rax
0x00014016e9b9  lea      rcx, [rbp + 0x380]
0x00014016e9c0  call     qword ptr [rip + 0x2f64fa]   ; 0x140464ec0
0x00014016e9c6  nop      
0x00014016e9c7  lea      r8, [rbp + 0x380]
0x00014016e9ce  lea      rdx, [rbp + 0x848]
0x00014016e9d5  mov      rcx, rbx
0x00014016e9d8  call     qword ptr [rip + 0x2f5b7a]   ; 0x140464558
0x00014016e9de  nop      
0x00014016e9df  lea      rdx, [rbp + 0x8d8]
0x00014016e9e6  mov      rcx, rax
0x00014016e9e9  call     qword ptr [rip + 0x2f6109]   ; 0x140464af8
0x00014016e9ef  nop      
0x00014016e9f0  mov      rdx, rax
0x00014016e9f3  mov      rcx, rdi
0x00014016e9f6  call     0x1402bd310
0x00014016e9fb  nop      
0x00014016e9fc  lea      rcx, [rbp + 0x8d8]
0x00014016ea03  call     qword ptr [rip + 0x2f6417]   ; 0x140464e20
0x00014016ea09  nop      
0x00014016ea0a  lea      rcx, [rbp + 0x848]
0x00014016ea11  call     qword ptr [rip + 0x2f6591]   ; 0x140464fa8
0x00014016ea17  nop      
0x00014016ea18  lea      rcx, [rbp + 0x380]
0x00014016ea1f  call     qword ptr [rip + 0x2f6583]   ; 0x140464fa8
0x00014016ea25  nop      
0x00014016ea26  lea      rcx, [rbp + 0x398]
0x00014016ea2d  call     qword ptr [rip + 0x2f6575]   ; 0x140464fa8
0x00014016ea33  nop      
0x00014016ea34  lea      rcx, [rbp + 0x908]
0x00014016ea3b  call     qword ptr [rip + 0x2f63df]   ; 0x140464e20
0x00014016ea41  nop      
0x00014016ea42  lea      rcx, [rbp + 0xb30]
0x00014016ea49  call     qword ptr [rip + 0x2f6559]   ; 0x140464fa8
0x00014016ea4f  nop      
0x00014016ea50  lea      rcx, [rbp + 0x3b0]
0x00014016ea57  call     qword ptr [rip + 0x2f654b]   ; 0x140464fa8
0x00014016ea5d  nop      
0x00014016ea5e  lea      rcx, [rbp + 0x3c8]
0x00014016ea65  call     qword ptr [rip + 0x2f653d]   ; 0x140464fa8
0x00014016ea6b  nop      
0x00014016ea6c  lea      rcx, [rbp + 0x938]
0x00014016ea73  call     qword ptr [rip + 0x2f63a7]   ; 0x140464e20
0x00014016ea79  nop      
0x00014016ea7a  lea      rcx, [rbp + 0x950]
0x00014016ea81  call     qword ptr [rip + 0x2f6521]   ; 0x140464fa8
0x00014016ea87  nop      
0x00014016ea88  lea      rcx, [rbp + 0x3e0]
0x00014016ea8f  call     qword ptr [rip + 0x2f6513]   ; 0x140464fa8
0x00014016ea95  nop      
0x00014016ea96  lea      rcx, [rbp + 0x3f8]
0x00014016ea9d  call     qword ptr [rip + 0x2f6505]   ; 0x140464fa8
0x00014016eaa3  nop      
0x00014016eaa4  lea      rcx, [rbp + 0x968]
0x00014016eaab  call     qword ptr [rip + 0x2f636f]   ; 0x140464e20
0x00014016eab1  nop      
0x00014016eab2  lea      rcx, [rbp + 0x980]
0x00014016eab9  call     qword ptr [rip + 0x2f64e9]   ; 0x140464fa8
0x00014016eabf  nop      
0x00014016eac0  lea      rcx, [rbp + 0x410]
0x00014016eac7  call     qword ptr [rip + 0x2f64db]   ; 0x140464fa8
0x00014016eacd  nop      
0x00014016eace  lea      rcx, [rbp + 0x428]
0x00014016ead5  call     qword ptr [rip + 0x2f64cd]   ; 0x140464fa8
0x00014016eadb  nop      
0x00014016eadc  lea      rcx, [rbp + 0x998]
0x00014016eae3  call     qword ptr [rip + 0x2f6337]   ; 0x140464e20
0x00014016eae9  nop      
0x00014016eaea  lea      rcx, [rbp + 0x9b0]
0x00014016eaf1  call     qword ptr [rip + 0x2f64b1]   ; 0x140464fa8
0x00014016eaf7  nop      
0x00014016eaf8  lea      rcx, [rbp + 0x440]
0x00014016eaff  call     qword ptr [rip + 0x2f64a3]   ; 0x140464fa8
0x00014016eb05  nop      
0x00014016eb06  lea      rcx, [rbp + 0x458]
0x00014016eb0d  call     qword ptr [rip + 0x2f6495]   ; 0x140464fa8
0x00014016eb13  jmp      0x14016eb20
0x00014016eb15  mov      byte ptr [rsp + 0x30], 0
0x00014016eb1a  call     0x1402bd1c0
0x00014016eb1f  nop      
0x00014016eb20  lea      rcx, [rsp + 0x78]
0x00014016eb25  call     qword ptr [rip + 0x2f5d95]   ; 0x1404648c0
0x00014016eb2b  lea      r9, [rbp - 0x50]
0x00014016eb2f  mov      r8, rsi
0x00014016eb32  lea      rdx, [rsp + 0x38]
0x00014016eb37  mov      rcx, r15
0x00014016eb3a  call     0x14016b590
0x00014016eb3f  lea      r9, [rbp - 0x20]
0x00014016eb43  mov      r8, rsi
0x00014016eb46  lea      rdx, [rsp + 0x38]
0x00014016eb4b  mov      rcx, r15
0x00014016eb4e  call     0x14016cd90
0x00014016eb53  mov      eax, dword ptr [rsi + 0x388]
0x00014016eb59  mov      dword ptr [rbp - 0x74], eax
0x00014016eb5c  lea      rdx, [rbp - 0x74]
0x00014016eb60  lea      rcx, [rsp + 0x38]
0x00014016eb65  call     0x1402bd230
0x00014016eb6a  mov      rbx, rax
0x00014016eb6d  lea      rcx, [rsi + 0x390]
0x00014016eb74  lea      rdx, [rbp + 0x9f8]
0x00014016eb7b  call     qword ptr [rip + 0x2f5b07]   ; 0x140464688
0x00014016eb81  nop      
0x00014016eb82  mov      rdx, rax
0x00014016eb85  mov      rcx, rbx
0x00014016eb88  call     0x1402bd310
0x00014016eb8d  lea      rdx, [rbp - 8]
0x00014016eb91  mov      rcx, rax
0x00014016eb94  call     0x1402bd2a0
0x00014016eb99  mov      rbx, rax
0x00014016eb9c  lea      rdx, [rip + 0x318215]   ; "randomInt1"
0x00014016eba3  lea      rcx, [rbp + 0x470]
0x00014016ebaa  call     qword ptr [rip + 0x2f62e0]   ; 0x140464e90
0x00014016ebb0  nop      
0x00014016ebb1  lea      r8, [rbp + 0x470]
0x00014016ebb8  lea      rdx, [rbp + 0x9e0]
0x00014016ebbf  mov      rcx, r13
0x00014016ebc2  call     qword ptr [rip + 0x2f5c80]   ; 0x140464848
0x00014016ebc8  nop      
0x00014016ebc9  lea      rdx, [rbp + 0x9c8]
0x00014016ebd0  mov      rcx, rax
0x00014016ebd3  call     qword ptr [rip + 0x2f5b4f]   ; 0x140464728
0x00014016ebd9  nop      
0x00014016ebda  xor      edx, edx
0x00014016ebdc  lea      r8d, [rdx + 0xa]
0x00014016ebe0  mov      rcx, rax
0x00014016ebe3  call     qword ptr [rip + 0x2f6097]   ; 0x140464c80
0x00014016ebe9  mov      qword ptr [rbp + 0x10], rax
0x00014016ebed  lea      rdx, [rbp + 0x10]
0x00014016ebf1  mov      rcx, rbx
0x00014016ebf4  call     0x1402bd2a0
0x00014016ebf9  nop      
0x00014016ebfa  lea      rcx, [rbp + 0x9c8]
0x00014016ec01  call     qword ptr [rip + 0x2f63a1]   ; 0x140464fa8
0x00014016ec07  nop      
0x00014016ec08  lea      rcx, [rbp + 0x9e0]
0x00014016ec0f  call     qword ptr [rip + 0x2f5c63]   ; 0x140464878
0x00014016ec15  nop      
0x00014016ec16  lea      rcx, [rbp + 0x470]
0x00014016ec1d  call     qword ptr [rip + 0x2f6385]   ; 0x140464fa8
0x00014016ec23  nop      
0x00014016ec24  lea      rcx, [rbp + 0x9f8]
0x00014016ec2b  call     qword ptr [rip + 0x2f61ef]   ; 0x140464e20
0x00014016ec31  mov      rdi, qword ptr [rbp + 0xd30]
0x00014016ec38  mov      eax, dword ptr [rdi]
0x00014016ec3a  mov      dword ptr [rbp - 0x70], eax
0x00014016ec3d  lea      rdx, [rbp - 0x70]
0x00014016ec41  lea      rcx, [rsp + 0x38]
0x00014016ec46  call     0x1402bd230
0x00014016ec4b  mov      rbx, rax
0x00014016ec4e  lea      rcx, [rdi + 8]
0x00014016ec52  lea      rdx, [rbp + 0xa40]
0x00014016ec59  call     qword ptr [rip + 0x2f5a29]   ; 0x140464688
0x00014016ec5f  nop      
0x00014016ec60  mov      rdx, rax
0x00014016ec63  mov      rcx, rbx
0x00014016ec66  call     0x1402bd310
0x00014016ec6b  lea      rdx, [rdi + 0x20]
0x00014016ec6f  mov      rcx, rax
0x00014016ec72  call     0x1402bd230
0x00014016ec77  mov      rbx, rax
0x00014016ec7a  lea      rcx, [rdi + 0x28]
0x00014016ec7e  lea      rdx, [rbp + 0xa28]
0x00014016ec85  call     qword ptr [rip + 0x2f59fd]   ; 0x140464688
0x00014016ec8b  nop      
0x00014016ec8c  mov      rdx, rax
0x00014016ec8f  mov      rcx, rbx
0x00014016ec92  call     0x1402bd310
0x00014016ec97  mov      rbx, rax
0x00014016ec9a  lea      rcx, [rdi + 0x40]
0x00014016ec9e  lea      rdx, [rbp + 0xa10]
0x00014016eca5  call     qword ptr [rip + 0x2f59dd]   ; 0x140464688
0x00014016ecab  nop      
0x00014016ecac  mov      rdx, rax
0x00014016ecaf  mov      rcx, rbx
0x00014016ecb2  call     0x1402bd310
0x00014016ecb7  nop      
0x00014016ecb8  lea      rcx, [rbp + 0xa10]
0x00014016ecbf  call     qword ptr [rip + 0x2f615b]   ; 0x140464e20
0x00014016ecc5  nop      
0x00014016ecc6  lea      rcx, [rbp + 0xa28]
0x00014016eccd  call     qword ptr [rip + 0x2f614d]   ; 0x140464e20
0x00014016ecd3  nop      
0x00014016ecd4  lea      rcx, [rbp + 0xa40]
0x00014016ecdb  call     qword ptr [rip + 0x2f613f]   ; 0x140464e20
0x00014016ece1  lea      rdx, [rsi + 0x441]
0x00014016ece8  lea      rcx, [rsp + 0x38]
0x00014016eced  call     0x1402bd1c0
0x00014016ecf2  lea      rdx, [rbp - 8]
0x00014016ecf6  mov      rcx, rax
0x00014016ecf9  call     0x1402bd2a0
0x00014016ecfe  lea      rdx, [rip + 0x31800b]   ; "webglConfig"
0x00014016ed05  lea      rcx, [rbp + 0x488]
0x00014016ed0c  call     qword ptr [rip + 0x2f617e]   ; 0x140464e90
0x00014016ed12  nop      
0x00014016ed13  lea      r8, [rbp + 0x488]
0x00014016ed1a  lea      rdx, [rbp + 0xa88]
0x00014016ed21  mov      rcx, r13
0x00014016ed24  call     qword ptr [rip + 0x2f5b1e]   ; 0x140464848
0x00014016ed2a  nop      
0x00014016ed2b  lea      rdx, [rbp + 0xa70]
0x00014016ed32  mov      rcx, rax
0x00014016ed35  call     qword ptr [rip + 0x2f59ed]   ; 0x140464728
0x00014016ed3b  nop      
0x00014016ed3c  lea      rdx, [rbp + 0xa58]
0x00014016ed43  mov      rcx, rax
0x00014016ed46  call     qword ptr [rip + 0x2f5dac]   ; 0x140464af8
0x00014016ed4c  nop      
0x00014016ed4d  xor      r8d, r8d
0x00014016ed50  mov      rdx, rax
0x00014016ed53  lea      rcx, [rsp + 0x70]
0x00014016ed58  call     0x1402fc7f0
0x00014016ed5d  nop      
0x00014016ed5e  lea      rcx, [rbp + 0xa58]
0x00014016ed65  call     qword ptr [rip + 0x2f60b5]   ; 0x140464e20
0x00014016ed6b  nop      
0x00014016ed6c  lea      rcx, [rbp + 0xa70]
0x00014016ed73  call     qword ptr [rip + 0x2f622f]   ; 0x140464fa8
0x00014016ed79  nop      
0x00014016ed7a  lea      rcx, [rbp + 0xa88]
0x00014016ed81  call     qword ptr [rip + 0x2f5af1]   ; 0x140464878
0x00014016ed87  nop      
0x00014016ed88  lea      rcx, [rbp + 0x488]
0x00014016ed8f  call     qword ptr [rip + 0x2f6213]   ; 0x140464fa8
0x00014016ed95  lea      rdx, [rip + 0x31a674]   ; "webgl"
0x00014016ed9c  lea      rcx, [rbp + 0x4a0]
0x00014016eda3  call     qword ptr [rip + 0x2f60e7]   ; 0x140464e90
0x00014016eda9  nop      
0x00014016edaa  lea      r8, [rbp + 0x4a0]
0x00014016edb1  lea      rdx, [rbp + 0xc78]
0x00014016edb8  lea      rcx, [rsp + 0x70]
0x00014016edbd  call     qword ptr [rip + 0x2f5a7d]   ; 0x140464840
0x00014016edc3  lea      rdx, [rbp - 0x58]
0x00014016edc7  mov      rcx, rax
0x00014016edca  call     qword ptr [rip + 0x2f5a80]   ; 0x140464850
0x00014016edd0  nop      
0x00014016edd1  lea      rcx, [rbp + 0x4a0]
0x00014016edd8  call     qword ptr [rip + 0x2f61ca]   ; 0x140464fa8
0x00014016edde  lea      rdx, [rip + 0x31a633]   ; "webgl2"
0x00014016ede5  lea      rcx, [rbp + 0x4b8]
0x00014016edec  call     qword ptr [rip + 0x2f609e]   ; 0x140464e90
0x00014016edf2  nop      
0x00014016edf3  lea      r8, [rbp + 0x4b8]
0x00014016edfa  lea      rdx, [rbp + 0xc68]
0x00014016ee01  lea      rcx, [rsp + 0x70]
0x00014016ee06  call     qword ptr [rip + 0x2f5a34]   ; 0x140464840
0x00014016ee0c  lea      rdx, [rbp - 0x60]
0x00014016ee10  mov      rcx, rax
0x00014016ee13  call     qword ptr [rip + 0x2f5a37]   ; 0x140464850
0x00014016ee19  nop      
0x00014016ee1a  lea      rcx, [rbp + 0x4b8]
0x00014016ee21  call     qword ptr [rip + 0x2f6181]   ; 0x140464fa8
0x00014016ee27  lea      rdx, [rsi + 0x442]
0x00014016ee2e  lea      rcx, [rsp + 0x38]
0x00014016ee33  call     0x1402bd1c0
0x00014016ee38  lea      rdx, [rsi + 0x443]
0x00014016ee3f  mov      rcx, rax
0x00014016ee42  call     0x1402bd1c0
0x00014016ee47  mov      rdi, rax
0x00014016ee4a  lea      rdx, [rip + 0x31a5cf]   ; "Unmasked Renderer"
0x00014016ee51  lea      rcx, [rbp + 0x80]
0x00014016ee58  call     qword ptr [rip + 0x2f6032]   ; 0x140464e90
0x00014016ee5e  nop      
0x00014016ee5f  lea      r8, [rbp + 0x80]
0x00014016ee66  lea      rdx, [rbp + 0xbd8]
0x00014016ee6d  lea      rcx, [rsp + 0x70]
0x00014016ee72  call     qword ptr [rip + 0x2f59c8]   ; 0x140464840
0x00014016ee78  mov      rbx, rax
0x00014016ee7b  lea      rcx, [rbp + 0x98]
0x00014016ee82  call     qword ptr [rip + 0x2f6038]   ; 0x140464ec0
0x00014016ee88  nop      
0x00014016ee89  lea      r8, [rbp + 0x98]
0x00014016ee90  lea      rdx, [rbp + 0x278]
0x00014016ee97  mov      rcx, rbx
0x00014016ee9a  call     qword ptr [rip + 0x2f56b8]   ; 0x140464558
0x00014016eea0  nop      
0x00014016eea1  lea      rdx, [rbp + 0x260]
0x00014016eea8  mov      rcx, rax
0x00014016eeab  call     qword ptr [rip + 0x2f5c47]   ; 0x140464af8
0x00014016eeb1  nop      
0x00014016eeb2  mov      rdx, rax
0x00014016eeb5  mov      rcx, rdi
0x00014016eeb8  call     0x1402bd310
0x00014016eebd  mov      rdi, rax
0x00014016eec0  lea      rdx, [rip + 0x31a571]   ; "Unmasked Vendor"
0x00014016eec7  lea      rcx, [rbp + 0xb0]
0x00014016eece  call     qword ptr [rip + 0x2f5fbc]   ; 0x140464e90
0x00014016eed4  nop      
0x00014016eed5  lea      r8, [rbp + 0xb0]
0x00014016eedc  lea      rdx, [rbp + 0xbb8]
0x00014016eee3  lea      rcx, [rsp + 0x70]
0x00014016eee8  call     qword ptr [rip + 0x2f5952]   ; 0x140464840
0x00014016eeee  mov      rbx, rax
0x00014016eef1  lea      rcx, [rbp + 0xc8]
0x00014016eef8  call     qword ptr [rip + 0x2f5fc2]   ; 0x140464ec0
0x00014016eefe  nop      
0x00014016eeff  lea      r8, [rbp + 0xc8]
0x00014016ef06  lea      rdx, [rbp + 0x248]
0x00014016ef0d  mov      rcx, rbx
0x00014016ef10  call     qword ptr [rip + 0x2f5642]   ; 0x140464558
0x00014016ef16  nop      
0x00014016ef17  lea      rdx, [rbp + 0x230]
0x00014016ef1e  mov      rcx, rax
0x00014016ef21  call     qword ptr [rip + 0x2f5bd1]   ; 0x140464af8
0x00014016ef27  nop      
0x00014016ef28  mov      rdx, rax
0x00014016ef2b  mov      rcx, rdi
0x00014016ef2e  call     0x1402bd310
0x00014016ef33  mov      rbx, rax
0x00014016ef36  lea      rdx, [rip + 0x317ebb]   ; "randomWebgl"
0x00014016ef3d  lea      rcx, [rbp + 0xe0]
0x00014016ef44  call     qword ptr [rip + 0x2f5f46]   ; 0x140464e90
0x00014016ef4a  nop      
0x00014016ef4b  lea      r8, [rbp + 0xe0]
0x00014016ef52  lea      rdx, [rbp + 0x110]
0x00014016ef59  mov      rcx, r13
0x00014016ef5c  call     qword ptr [rip + 0x2f58e6]   ; 0x140464848
0x00014016ef62  nop      
0x00014016ef63  xor      edx, edx
0x00014016ef65  mov      rcx, rax
0x00014016ef68  call     qword ptr [rip + 0x2f5902]   ; 0x140464870
0x00014016ef6e  mov      dword ptr [rbp - 0x6c], eax
0x00014016ef71  lea      rdx, [rbp - 0x6c]
0x00014016ef75  mov      rcx, rbx
0x00014016ef78  call     0x1402bd230
0x00014016ef7d  mov      rdi, rax
0x00014016ef80  lea      rdx, [rip + 0x31a4c1]   ; "GL Version"
0x00014016ef87  lea      rcx, [rbp + 0x638]
0x00014016ef8e  call     qword ptr [rip + 0x2f5efc]   ; 0x140464e90
0x00014016ef94  nop      
0x00014016ef95  lea      r8, [rbp + 0x638]
0x00014016ef9c  lea      rdx, [rbp + 0xcb8]
0x00014016efa3  lea      rcx, [rbp - 0x58]
0x00014016efa7  call     qword ptr [rip + 0x2f5893]   ; 0x140464840
0x00014016efad  mov      rbx, rax
0x00014016efb0  lea      rcx, [rbp + 0x620]
0x00014016efb7  call     qword ptr [rip + 0x2f5f03]   ; 0x140464ec0
0x00014016efbd  nop      
0x00014016efbe  lea      r8, [rbp + 0x620]
0x00014016efc5  lea      rdx, [rbp + 0x128]
0x00014016efcc  mov      rcx, rbx
0x00014016efcf  call     qword ptr [rip + 0x2f5583]   ; 0x140464558
0x00014016efd5  nop      
0x00014016efd6  lea      rdx, [rbp + 0x140]
0x00014016efdd  mov      rcx, rax
0x00014016efe0  call     qword ptr [rip + 0x2f5b12]   ; 0x140464af8
0x00014016efe6  nop      
0x00014016efe7  mov      rdx, rax
0x00014016efea  mov      rcx, rdi
0x00014016efed  call     0x1402bd310
0x00014016eff2  mov      rdi, rax
0x00014016eff5  lea      rdx, [rip + 0x31a45c]   ; "Shading Language Version"
0x00014016effc  lea      rcx, [rbp + 0x608]
0x00014016f003  call     qword ptr [rip + 0x2f5e87]   ; 0x140464e90
0x00014016f009  nop      
0x00014016f00a  lea      r8, [rbp + 0x608]
0x00014016f011  lea      rdx, [rbp + 0xc38]
0x00014016f018  lea      rcx, [rbp - 0x58]
0x00014016f01c  call     qword ptr [rip + 0x2f581e]   ; 0x140464840
0x00014016f022  mov      rbx, rax
0x00014016f025  lea      rcx, [rbp + 0x5f0]
0x00014016f02c  call     qword ptr [rip + 0x2f5e8e]   ; 0x140464ec0
0x00014016f032  nop      
0x00014016f033  lea      r8, [rbp + 0x5f0]
0x00014016f03a  lea      rdx, [rbp + 0x158]
0x00014016f041  mov      rcx, rbx
0x00014016f044  call     qword ptr [rip + 0x2f550e]   ; 0x140464558
0x00014016f04a  nop      
0x00014016f04b  lea      rdx, [rbp + 0x170]
0x00014016f052  mov      rcx, rax
0x00014016f055  call     qword ptr [rip + 0x2f5a9d]   ; 0x140464af8
0x00014016f05b  nop      
0x00014016f05c  mov      rdx, rax
0x00014016f05f  mov      rcx, rdi
0x00014016f062  call     0x1402bd310
0x00014016f067  mov      rdi, rax
0x00014016f06a  lea      rdx, [rip + 0x31a403]   ; "Vendor"
0x00014016f071  lea      rcx, [rbp + 0x5d8]
0x00014016f078  call     qword ptr [rip + 0x2f5e12]   ; 0x140464e90
0x00014016f07e  nop      
0x00014016f07f  lea      r8, [rbp + 0x5d8]
0x00014016f086  lea      rdx, [rbp + 0xc28]
0x00014016f08d  lea      rcx, [rbp - 0x58]
0x00014016f091  call     qword ptr [rip + 0x2f57a9]   ; 0x140464840
0x00014016f097  mov      rbx, rax
0x00014016f09a  lea      rcx, [rbp + 0x5c0]
0x00014016f0a1  call     qword ptr [rip + 0x2f5e19]   ; 0x140464ec0
0x00014016f0a7  nop      
0x00014016f0a8  lea      r8, [rbp + 0x5c0]
0x00014016f0af  lea      rdx, [rbp + 0x188]
0x00014016f0b6  mov      rcx, rbx
0x00014016f0b9  call     qword ptr [rip + 0x2f5499]   ; 0x140464558
0x00014016f0bf  nop      
0x00014016f0c0  lea      rdx, [rbp + 0x1a0]
0x00014016f0c7  mov      rcx, rax
0x00014016f0ca  call     qword ptr [rip + 0x2f5a28]   ; 0x140464af8
0x00014016f0d0  nop      
0x00014016f0d1  mov      rdx, rax
0x00014016f0d4  mov      rcx, rdi
0x00014016f0d7  call     0x1402bd310
0x00014016f0dc  mov      rdi, rax
0x00014016f0df  lea      rdx, [rip + 0x31a39a]   ; "Renderer"
0x00014016f0e6  lea      rcx, [rbp + 0x5a8]
0x00014016f0ed  call     qword ptr [rip + 0x2f5d9d]   ; 0x140464e90
0x00014016f0f3  nop      
0x00014016f0f4  lea      r8, [rbp + 0x5a8]
0x00014016f0fb  lea      rdx, [rbp + 0xc18]
0x00014016f102  lea      rcx, [rbp - 0x58]
0x00014016f106  call     qword ptr [rip + 0x2f5734]   ; 0x140464840
0x00014016f10c  mov      rbx, rax
0x00014016f10f  lea      rcx, [rbp + 0x590]
0x00014016f116  call     qword ptr [rip + 0x2f5da4]   ; 0x140464ec0
0x00014016f11c  nop      
0x00014016f11d  lea      r8, [rbp + 0x590]
0x00014016f124  lea      rdx, [rbp + 0x1b8]
0x00014016f12b  mov      rcx, rbx
0x00014016f12e  call     qword ptr [rip + 0x2f5424]   ; 0x140464558
0x00014016f134  nop      
0x00014016f135  lea      rdx, [rbp + 0x1d0]
0x00014016f13c  mov      rcx, rax
0x00014016f13f  call     qword ptr [rip + 0x2f59b3]   ; 0x140464af8
0x00014016f145  nop      
0x00014016f146  mov      rdx, rax
0x00014016f149  mov      rcx, rdi
0x00014016f14c  call     0x1402bd310
0x00014016f151  mov      rdi, rax
0x00014016f154  lea      rdx, [rip + 0x31a2ed]   ; "GL Version"
0x00014016f15b  lea      rcx, [rbp + 0x578]
0x00014016f162  call     qword ptr [rip + 0x2f5d28]   ; 0x140464e90
0x00014016f168  nop      
0x00014016f169  lea      r8, [rbp + 0x578]
0x00014016f170  lea      rdx, [rbp + 0xc08]
0x00014016f177  lea      rcx, [rbp - 0x60]
0x00014016f17b  call     qword ptr [rip + 0x2f56bf]   ; 0x140464840
0x00014016f181  mov      rbx, rax
0x00014016f184  lea      rcx, [rbp + 0x560]
0x00014016f18b  call     qword ptr [rip + 0x2f5d2f]   ; 0x140464ec0
0x00014016f191  nop      
0x00014016f192  lea      r8, [rbp + 0x560]
0x00014016f199  lea      rdx, [rbp + 0x1e8]
0x00014016f1a0  mov      rcx, rbx
0x00014016f1a3  call     qword ptr [rip + 0x2f53af]   ; 0x140464558
0x00014016f1a9  nop      
0x00014016f1aa  lea      rdx, [rbp + 0x30]
0x00014016f1ae  mov      rcx, rax
0x00014016f1b1  call     qword ptr [rip + 0x2f5941]   ; 0x140464af8
0x00014016f1b7  nop      
0x00014016f1b8  mov      rdx, rax
0x00014016f1bb  mov      rcx, rdi
0x00014016f1be  call     0x1402bd310
0x00014016f1c3  mov      rdi, rax
0x00014016f1c6  lea      rdx, [rip + 0x31a28b]   ; "Shading Language Version"
0x00014016f1cd  lea      rcx, [rbp + 0x548]
0x00014016f1d4  call     qword ptr [rip + 0x2f5cb6]   ; 0x140464e90
0x00014016f1da  nop      
0x00014016f1db  lea      r8, [rbp + 0x548]
0x00014016f1e2  lea      rdx, [rbp + 0x740]
0x00014016f1e9  lea      rcx, [rbp - 0x60]
0x00014016f1ed  call     qword ptr [rip + 0x2f564d]   ; 0x140464840
0x00014016f1f3  mov      rbx, rax
0x00014016f1f6  lea      rcx, [rbp + 0x530]
0x00014016f1fd  call     qword ptr [rip + 0x2f5cbd]   ; 0x140464ec0
0x00014016f203  nop      
0x00014016f204  lea      r8, [rbp + 0x530]
0x00014016f20b  lea      rdx, [rbp + 0xb18]
0x00014016f212  mov      rcx, rbx
0x00014016f215  call     qword ptr [rip + 0x2f533d]   ; 0x140464558
0x00014016f21b  nop      
0x00014016f21c  lea      rdx, [rbp + 0xb00]
0x00014016f223  mov      rcx, rax
0x00014016f226  call     qword ptr [rip + 0x2f58cc]   ; 0x140464af8
0x00014016f22c  nop      
0x00014016f22d  mov      rdx, rax
0x00014016f230  mov      rcx, rdi
0x00014016f233  call     0x1402bd310
0x00014016f238  mov      rdi, rax
0x00014016f23b  lea      rdx, [rip + 0x31a232]   ; "Vendor"
0x00014016f242  lea      rcx, [rbp + 0x518]
0x00014016f249  call     qword ptr [rip + 0x2f5c41]   ; 0x140464e90
0x00014016f24f  nop      
0x00014016f250  lea      r8, [rbp + 0x518]
0x00014016f257  lea      rdx, [rbp + 0x750]
0x00014016f25e  lea      rcx, [rbp - 0x60]
0x00014016f262  call     qword ptr [rip + 0x2f55d8]   ; 0x140464840
0x00014016f268  mov      rbx, rax
0x00014016f26b  lea      rcx, [rbp + 0x500]
0x00014016f272  call     qword ptr [rip + 0x2f5c48]   ; 0x140464ec0
0x00014016f278  nop      
0x00014016f279  lea      r8, [rbp + 0x500]
0x00014016f280  lea      rdx, [rbp + 0xae8]
0x00014016f287  mov      rcx, rbx
0x00014016f28a  call     qword ptr [rip + 0x2f52c8]   ; 0x140464558
0x00014016f290  nop      
0x00014016f291  lea      rdx, [rbp + 0xad0]
0x00014016f298  mov      rcx, rax
0x00014016f29b  call     qword ptr [rip + 0x2f5857]   ; 0x140464af8
0x00014016f2a1  nop      
0x00014016f2a2  mov      rdx, rax
0x00014016f2a5  mov      rcx, rdi
0x00014016f2a8  call     0x1402bd310
0x00014016f2ad  mov      rdi, rax
0x00014016f2b0  lea      rdx, [rip + 0x31a1c9]   ; "Renderer"
0x00014016f2b7  lea      rcx, [rbp + 0x4e8]
0x00014016f2be  call     qword ptr [rip + 0x2f5bcc]   ; 0x140464e90
0x00014016f2c4  nop      
0x00014016f2c5  lea      r8, [rbp + 0x4e8]
0x00014016f2cc  lea      rdx, [rbp + 0x760]
0x00014016f2d3  lea      rcx, [rbp - 0x60]
0x00014016f2d7  call     qword ptr [rip + 0x2f5563]   ; 0x140464840
0x00014016f2dd  mov      rbx, rax
0x00014016f2e0  lea      rcx, [rbp + 0x4d0]
0x00014016f2e7  call     qword ptr [rip + 0x2f5bd3]   ; 0x140464ec0
0x00014016f2ed  nop      
0x00014016f2ee  lea      r8, [rbp + 0x4d0]
0x00014016f2f5  lea      rdx, [rbp + 0xab8]
0x00014016f2fc  mov      rcx, rbx
0x00014016f2ff  call     qword ptr [rip + 0x2f5253]   ; 0x140464558
0x00014016f305  nop      
0x00014016f306  lea      rdx, [rbp + 0xaa0]
0x00014016f30d  mov      rcx, rax
0x00014016f310  call     qword ptr [rip + 0x2f57e2]   ; 0x140464af8
0x00014016f316  nop      
0x00014016f317  mov      rdx, rax
0x00014016f31a  mov      rcx, rdi
0x00014016f31d  call     0x1402bd310
0x00014016f322  nop      
0x00014016f323  lea      rcx, [rbp + 0xaa0]
0x00014016f32a  call     qword ptr [rip + 0x2f5af0]   ; 0x140464e20
0x00014016f330  nop      
0x00014016f331  lea      rcx, [rbp + 0xab8]
0x00014016f338  call     qword ptr [rip + 0x2f5c6a]   ; 0x140464fa8
0x00014016f33e  nop      
0x00014016f33f  lea      rcx, [rbp + 0x4d0]
0x00014016f346  call     qword ptr [rip + 0x2f5c5c]   ; 0x140464fa8
0x00014016f34c  nop      
0x00014016f34d  lea      rcx, [rbp + 0x4e8]
0x00014016f354  call     qword ptr [rip + 0x2f5c4e]   ; 0x140464fa8
0x00014016f35a  nop      
0x00014016f35b  lea      rcx, [rbp + 0xad0]
0x00014016f362  call     qword ptr [rip + 0x2f5ab8]   ; 0x140464e20
0x00014016f368  nop      
0x00014016f369  lea      rcx, [rbp + 0xae8]
0x00014016f370  call     qword ptr [rip + 0x2f5c32]   ; 0x140464fa8
0x00014016f376  nop      
0x00014016f377  lea      rcx, [rbp + 0x500]
0x00014016f37e  call     qword ptr [rip + 0x2f5c24]   ; 0x140464fa8
0x00014016f384  nop      
0x00014016f385  lea      rcx, [rbp + 0x518]
0x00014016f38c  call     qword ptr [rip + 0x2f5c16]   ; 0x140464fa8
0x00014016f392  nop      
0x00014016f393  lea      rcx, [rbp + 0xb00]
0x00014016f39a  call     qword ptr [rip + 0x2f5a80]   ; 0x140464e20
0x00014016f3a0  nop      
0x00014016f3a1  lea      rcx, [rbp + 0xb18]
0x00014016f3a8  call     qword ptr [rip + 0x2f5bfa]   ; 0x140464fa8
0x00014016f3ae  nop      
0x00014016f3af  lea      rcx, [rbp + 0x530]
0x00014016f3b6  call     qword ptr [rip + 0x2f5bec]   ; 0x140464fa8
0x00014016f3bc  nop      
0x00014016f3bd  lea      rcx, [rbp + 0x548]
0x00014016f3c4  call     qword ptr [rip + 0x2f5bde]   ; 0x140464fa8
0x00014016f3ca  nop      
0x00014016f3cb  lea      rcx, [rbp + 0x30]
0x00014016f3cf  call     qword ptr [rip + 0x2f5a4b]   ; 0x140464e20
0x00014016f3d5  nop      
0x00014016f3d6  lea      rcx, [rbp + 0x1e8]
0x00014016f3dd  call     qword ptr [rip + 0x2f5bc5]   ; 0x140464fa8
0x00014016f3e3  nop      
0x00014016f3e4  lea      rcx, [rbp + 0x560]
0x00014016f3eb  call     qword ptr [rip + 0x2f5bb7]   ; 0x140464fa8
0x00014016f3f1  nop      
0x00014016f3f2  lea      rcx, [rbp + 0x578]
0x00014016f3f9  call     qword ptr [rip + 0x2f5ba9]   ; 0x140464fa8
0x00014016f3ff  nop      
0x00014016f400  lea      rcx, [rbp + 0x1d0]
0x00014016f407  call     qword ptr [rip + 0x2f5a13]   ; 0x140464e20
0x00014016f40d  nop      
0x00014016f40e  lea      rcx, [rbp + 0x1b8]
0x00014016f415  call     qword ptr [rip + 0x2f5b8d]   ; 0x140464fa8
0x00014016f41b  nop      
0x00014016f41c  lea      rcx, [rbp + 0x590]
0x00014016f423  call     qword ptr [rip + 0x2f5b7f]   ; 0x140464fa8
0x00014016f429  nop      
0x00014016f42a  lea      rcx, [rbp + 0x5a8]
0x00014016f431  call     qword ptr [rip + 0x2f5b71]   ; 0x140464fa8
0x00014016f437  nop      
0x00014016f438  lea      rcx, [rbp + 0x1a0]
0x00014016f43f  call     qword ptr [rip + 0x2f59db]   ; 0x140464e20
0x00014016f445  nop      
0x00014016f446  lea      rcx, [rbp + 0x188]
0x00014016f44d  call     qword ptr [rip + 0x2f5b55]   ; 0x140464fa8
0x00014016f453  nop      
0x00014016f454  lea      rcx, [rbp + 0x5c0]
0x00014016f45b  call     qword ptr [rip + 0x2f5b47]   ; 0x140464fa8
0x00014016f461  nop      
0x00014016f462  lea      rcx, [rbp + 0x5d8]
0x00014016f469  call     qword ptr [rip + 0x2f5b39]   ; 0x140464fa8
0x00014016f46f  nop      
0x00014016f470  lea      rcx, [rbp + 0x170]
0x00014016f477  call     qword ptr [rip + 0x2f59a3]   ; 0x140464e20
0x00014016f47d  nop      
0x00014016f47e  lea      rcx, [rbp + 0x158]
0x00014016f485  call     qword ptr [rip + 0x2f5b1d]   ; 0x140464fa8
0x00014016f48b  nop      
0x00014016f48c  lea      rcx, [rbp + 0x5f0]
0x00014016f493  call     qword ptr [rip + 0x2f5b0f]   ; 0x140464fa8
0x00014016f499  nop      
0x00014016f49a  lea      rcx, [rbp + 0x608]
0x00014016f4a1  call     qword ptr [rip + 0x2f5b01]   ; 0x140464fa8
0x00014016f4a7  nop      
0x00014016f4a8  lea      rcx, [rbp + 0x140]
0x00014016f4af  call     qword ptr [rip + 0x2f596b]   ; 0x140464e20
0x00014016f4b5  nop      
0x00014016f4b6  lea      rcx, [rbp + 0x128]
0x00014016f4bd  call     qword ptr [rip + 0x2f5ae5]   ; 0x140464fa8
0x00014016f4c3  nop      
0x00014016f4c4  lea      rcx, [rbp + 0x620]
0x00014016f4cb  call     qword ptr [rip + 0x2f5ad7]   ; 0x140464fa8
0x00014016f4d1  nop      
0x00014016f4d2  lea      rcx, [rbp + 0x638]
0x00014016f4d9  call     qword ptr [rip + 0x2f5ac9]   ; 0x140464fa8
0x00014016f4df  nop      
0x00014016f4e0  lea      rcx, [rbp + 0x110]
0x00014016f4e7  call     qword ptr [rip + 0x2f538b]   ; 0x140464878
0x00014016f4ed  nop      
0x00014016f4ee  lea      rcx, [rbp + 0xe0]
0x00014016f4f5  call     qword ptr [rip + 0x2f5aad]   ; 0x140464fa8
0x00014016f4fb  nop      
0x00014016f4fc  lea      rcx, [rbp + 0x230]
0x00014016f503  call     qword ptr [rip + 0x2f5917]   ; 0x140464e20
0x00014016f509  nop      
0x00014016f50a  lea      rcx, [rbp + 0x248]
0x00014016f511  call     qword ptr [rip + 0x2f5a91]   ; 0x140464fa8
0x00014016f517  nop      
0x00014016f518  lea      rcx, [rbp + 0xc8]
0x00014016f51f  call     qword ptr [rip + 0x2f5a83]   ; 0x140464fa8
0x00014016f525  nop      
0x00014016f526  lea      rcx, [rbp + 0xb0]
0x00014016f52d  call     qword ptr [rip + 0x2f5a75]   ; 0x140464fa8
0x00014016f533  nop      
0x00014016f534  lea      rcx, [rbp + 0x260]
0x00014016f53b  call     qword ptr [rip + 0x2f58df]   ; 0x140464e20
0x00014016f541  nop      
0x00014016f542  lea      rcx, [rbp + 0x278]
0x00014016f549  call     qword ptr [rip + 0x2f5a59]   ; 0x140464fa8
0x00014016f54f  nop      
0x00014016f550  lea      rcx, [rbp + 0x98]
0x00014016f557  call     qword ptr [rip + 0x2f5a4b]   ; 0x140464fa8
0x00014016f55d  nop      
0x00014016f55e  lea      rcx, [rbp + 0x80]
0x00014016f565  call     qword ptr [rip + 0x2f5a3d]   ; 0x140464fa8
0x00014016f56b  nop      
0x00014016f56c  lea      rcx, [rbp - 0x60]
0x00014016f570  call     qword ptr [rip + 0x2f534a]   ; 0x1404648c0
0x00014016f576  nop      
0x00014016f577  lea      rcx, [rbp - 0x58]
0x00014016f57b  call     qword ptr [rip + 0x2f533f]   ; 0x1404648c0
0x00014016f581  nop      
0x00014016f582  lea      rcx, [rsp + 0x70]
0x00014016f587  call     qword ptr [rip + 0x2f5333]   ; 0x1404648c0
0x00014016f58d  lea      rdx, [rsi + 0x447]
0x00014016f594  lea      rcx, [rsp + 0x38]
0x00014016f599  call     0x1402bd1c0
0x00014016f59e  mov      rbx, rax
0x00014016f5a1  lea      rdx, [rip + 0x317850]   ; "randomWebgl"
0x00014016f5a8  lea      rcx, [rbp + 0x68]
0x00014016f5ac  call     qword ptr [rip + 0x2f58de]   ; 0x140464e90
0x00014016f5b2  nop      
0x00014016f5b3  lea      r8, [rbp + 0x68]
0x00014016f5b7  lea      rdx, [rbp + 0x290]
0x00014016f5be  mov      rcx, r13
0x00014016f5c1  call     qword ptr [rip + 0x2f5281]   ; 0x140464848
0x00014016f5c7  nop      
0x00014016f5c8  xor      edx, edx
0x00014016f5ca  mov      rcx, rax
0x00014016f5cd  call     qword ptr [rip + 0x2f529d]   ; 0x140464870
0x00014016f5d3  mov      dword ptr [rbp - 0x68], eax
0x00014016f5d6  lea      rdx, [rbp - 0x68]
0x00014016f5da  mov      rcx, rbx
0x00014016f5dd  call     0x1402bd230
0x00014016f5e2  nop      
0x00014016f5e3  lea      rcx, [rbp + 0x290]
0x00014016f5ea  call     qword ptr [rip + 0x2f5288]   ; 0x140464878
0x00014016f5f0  nop      
0x00014016f5f1  lea      rcx, [rbp + 0x68]
0x00014016f5f5  call     qword ptr [rip + 0x2f59ad]   ; 0x140464fa8
0x00014016f5fb  cmp      dword ptr [rsi + 0x368], 1
0x00014016f602  setne    byte ptr [rbp + 0xd30]
0x00014016f609  lea      rdx, [rbp + 0xd30]
0x00014016f610  lea      rcx, [rsp + 0x38]
0x00014016f615  call     0x1402bd1c0
0x00014016f61a  cmp      dword ptr [rsi + 0x368], 2
0x00014016f621  sete     byte ptr [rsp + 0x31]
0x00014016f626  lea      rdx, [rsp + 0x31]
0x00014016f62b  mov      rcx, rax
0x00014016f62e  call     0x1402bd1c0
0x00014016f633  mov      qword ptr [rsp + 0x40], r14
0x00014016f638  xorps    xmm0, xmm0
0x00014016f63b  movdqu   xmmword ptr [rsp + 0x48], xmm0
0x00014016f641  movabs   r15, 0x2aaaaaaaaaaaaaab
0x00014016f64b  cmp      dword ptr [rsi + 0x368], 2
0x00014016f652  jne      0x14016fa52
0x00014016f658  lea      rdx, [rip + 0x319e31]   ; "randFonts"
0x00014016f65f  lea      rcx, [rbp + 0x50]
0x00014016f663  call     qword ptr [rip + 0x2f5827]   ; 0x140464e90
0x00014016f669  nop      
0x00014016f66a  lea      r8, [rbp + 0x50]
0x00014016f66e  lea      rdx, [rbp + 0x2a8]
0x00014016f675  mov      rcx, r13
0x00014016f678  call     qword ptr [rip + 0x2f51ca]   ; 0x140464848
0x00014016f67e  nop      
0x00014016f67f  lea      rdx, [rbp - 0x30]
0x00014016f683  mov      rcx, rax
0x00014016f686  call     qword ptr [rip + 0x2f51dc]   ; 0x140464868
0x00014016f68c  nop      
0x00014016f68d  lea      rcx, [rbp + 0x2a8]
0x00014016f694  call     qword ptr [rip + 0x2f51de]   ; 0x140464878
0x00014016f69a  nop      
0x00014016f69b  lea      rcx, [rbp + 0x50]
0x00014016f69f  call     qword ptr [rip + 0x2f5903]   ; 0x140464fa8
0x00014016f6a5  mov      r12d, r14d
0x00014016f6a8  lea      rcx, [rbp - 0x30]
0x00014016f6ac  call     qword ptr [rip + 0x2f5176]   ; 0x140464828
0x00014016f6b2  test     rax, rax
0x00014016f6b5  jle      0x14016fa48
0x00014016f6bb  mov      rbx, r14
0x00014016f6be  nop      
0x00014016f6c0  mov      r8, rbx
0x00014016f6c3  lea      rdx, [rbp + 0xba8]
0x00014016f6ca  lea      rcx, [rbp - 0x30]
0x00014016f6ce  call     qword ptr [rip + 0x2f514c]   ; 0x140464820
0x00014016f6d4  mov      rbx, rax
0x00014016f6d7  lea      rcx, [rbp + 0x650]
0x00014016f6de  call     qword ptr [rip + 0x2f57dc]   ; 0x140464ec0
0x00014016f6e4  nop      
0x00014016f6e5  lea      r8, [rbp + 0x650]
0x00014016f6ec  lea      rdx, [rbp + 0xb60]
0x00014016f6f3  mov      rcx, rbx
0x00014016f6f6  call     qword ptr [rip + 0x2f4e5c]   ; 0x140464558
0x00014016f6fc  mov      r11, rax
0x00014016f6ff  mov      rdi, qword ptr [rsp + 0x50]
0x00014016f704  mov      rax, qword ptr [rsp + 0x40]
0x00014016f709  test     rax, rax
0x00014016f70c  je       0x14016f7b3
0x00014016f712  mov      ecx, dword ptr [rax]
0x00014016f714  cmp      ecx, 1
0x00014016f717  jg       0x14016f7b3
0x00014016f71d  mov      r8, qword ptr [rsp + 0x40]
0x00014016f722  lea      r10, [r8 + 0x17]
0x00014016f726  and      r10, 0xfffffffffffffff8
0x00014016f72a  mov      r9, qword ptr [rsp + 0x48]
0x00014016f72f  mov      rcx, r9
0x00014016f732  sub      rcx, r10
0x00014016f735  mov      rax, r15
0x00014016f738  imul     rcx
0x00014016f73b  sar      rdx, 2
0x00014016f73f  mov      rax, rdx
0x00014016f742  shr      rax, 0x3f
0x00014016f746  add      rdx, rax
0x00014016f749  mov      rax, qword ptr [r8 + 8]
0x00014016f74d  sub      rax, rdx
0x00014016f750  mov      rcx, qword ptr [rsp + 0x50]
0x00014016f755  cmp      rax, rcx
0x00014016f758  je       0x14016f775
0x00014016f75a  lea      rax, [rcx + rcx*2]
0x00014016f75e  lea      rcx, [r9 + rax*8]
0x00014016f762  mov      rdx, r11
0x00014016f765  call     qword ptr [rip + 0x2f566d]   ; 0x140464dd8
0x00014016f76b  inc      qword ptr [rsp + 0x50]
0x00014016f770  jmp      0x14016f9f0
0x00014016f775  test     rcx, rcx
0x00014016f778  jne      0x14016f7b3
0x00014016f77a  mov      rcx, r9
0x00014016f77d  sub      rcx, r10
0x00014016f780  mov      rax, r15
0x00014016f783  imul     rcx
0x00014016f786  sar      rdx, 2
0x00014016f78a  mov      rax, rdx
0x00014016f78d  shr      rax, 0x3f
0x00014016f791  add      rdx, rax
0x00014016f794  je       0x14016f7b3
0x00014016f796  lea      rcx, [r9 - 0x18]
0x00014016f79a  mov      rdx, r11
0x00014016f79d  call     qword ptr [rip + 0x2f5635]   ; 0x140464dd8
0x00014016f7a3  sub      qword ptr [rsp + 0x48], 0x18
0x00014016f7a9  inc      qword ptr [rsp + 0x50]
0x00014016f7ae  jmp      0x14016f9f0
0x00014016f7b3  mov      rdx, r11
0x00014016f7b6  lea      rcx, [rbp + 0x200]
0x00014016f7bd  call     qword ptr [rip + 0x2f5615]   ; 0x140464dd8
0x00014016f7c3  nop      
0x00014016f7c4  cmp      qword ptr [rsp + 0x50], 0
0x00014016f7ca  je       0x14016f7d6
0x00014016f7cc  test     rdi, rdi
0x00014016f7cf  jne      0x14016f7d6
0x00014016f7d1  mov      r15b, 1
0x00014016f7d4  jmp      0x14016f7d9
0x00014016f7d6  xor      r15b, r15b
0x00014016f7d9  movzx    ebx, r15b
0x00014016f7dd  mov      rax, qword ptr [rsp + 0x40]
0x00014016f7e2  test     rax, rax
0x00014016f7e5  je       0x14016f96a
0x00014016f7eb  mov      eax, dword ptr [rax]
0x00014016f7ed  cmp      eax, 1
0x00014016f7f0  jg       0x14016f96a
0x00014016f7f6  mov      r9, qword ptr [rsp + 0x48]
0x00014016f7fb  mov      r8, qword ptr [rsp + 0x50]
0x00014016f800  cmp      ebx, 1
0x00014016f803  jne      0x14016f897
0x00014016f809  mov      rdx, qword ptr [rsp + 0x40]
0x00014016f80e  lea      rcx, [rdx + 0x17]
0x00014016f812  and      rcx, 0xfffffffffffffff8
0x00014016f816  mov      rax, r9
0x00014016f819  sub      rax, rcx
0x00014016f81c  cmp      rax, 0x18
0x00014016f820  jge      0x14016f987
0x00014016f826  mov      r11, qword ptr [rdx + 8]
0x00014016f82a  lea      rax, [rdx + 0x17]
0x00014016f82e  and      rax, 0xfffffffffffffff8
0x00014016f832  mov      rcx, r9
0x00014016f835  sub      rcx, rax
0x00014016f838  movabs   rax, 0x2aaaaaaaaaaaaaab
0x00014016f842  imul     rcx
0x00014016f845  mov      r10, rdx
0x00014016f848  sar      r10, 2
0x00014016f84c  mov      rax, r10
0x00014016f84f  shr      rax, 0x3f
0x00014016f853  add      r10, rax
0x00014016f856  mov      rcx, r11
0x00014016f859  sub      rcx, r8
0x00014016f85c  sub      rcx, r10
0x00014016f85f  cmp      rcx, 1
0x00014016f863  jl       0x14016f96a
0x00014016f869  lea      rcx, [r8 + r8*2]
0x00014016f86d  cmp      rcx, r11
0x00014016f870  jge      0x14016f96a
0x00014016f876  sub      r11, r8
0x00014016f879  lea      rax, [r11 - 1]
0x00014016f87d  cqo      
0x00014016f87f  sub      rax, rdx
0x00014016f882  sar      rax, 1
0x00014016f885  mov      rdx, r14
0x00014016f888  test     rax, rax
0x00014016f88b  cmovg    rdx, rax
0x00014016f88f  inc      rdx
0x00014016f892  jmp      0x14016f929
0x00014016f897  mov      r10, qword ptr [rsp + 0x40]
0x00014016f89c  test     r15b, r15b
0x00014016f89f  jne      0x14016f8de
0x00014016f8a1  lea      rax, [r10 + 0x17]
0x00014016f8a5  and      rax, 0xfffffffffffffff8
0x00014016f8a9  mov      rcx, r9
0x00014016f8ac  sub      rcx, rax
0x00014016f8af  movabs   rax, 0x2aaaaaaaaaaaaaab
0x00014016f8b9  imul     rcx
0x00014016f8bc  sar      rdx, 2
0x00014016f8c0  mov      rax, rdx
0x00014016f8c3  shr      rax, 0x3f
0x00014016f8c7  add      rdx, rax
0x00014016f8ca  mov      rax, qword ptr [r10 + 8]
0x00014016f8ce  sub      rax, rdx
0x00014016f8d1  sub      rax, r8
0x00014016f8d4  cmp      rax, 1
0x00014016f8d8  jge      0x14016f987
0x00014016f8de  mov      r11, qword ptr [r10 + 8]
0x00014016f8e2  lea      rax, [r10 + 0x17]
0x00014016f8e6  and      rax, 0xfffffffffffffff8
0x00014016f8ea  mov      rcx, r9
0x00014016f8ed  sub      rcx, rax
0x00014016f8f0  movabs   rax, 0x2aaaaaaaaaaaaaab
0x00014016f8fa  imul     rcx
0x00014016f8fd  mov      r10, rdx
0x00014016f900  sar      r10, 2
0x00014016f904  mov      rax, r10
0x00014016f907  shr      rax, 0x3f
0x00014016f90b  add      r10, rax
0x00014016f90e  mov      rdx, r14
0x00014016f911  test     r15b, r15b
0x00014016f914  jne      0x14016f96a
0x00014016f916  cmp      r10, 1
0x00014016f91a  jl       0x14016f96a
0x00014016f91c  lea      rcx, [r11 + r11]
0x00014016f920  lea      rax, [r8 + r8*2]
0x00014016f924  cmp      rax, rcx
0x00014016f927  jge      0x14016f96a
0x00014016f929  sub      rdx, r10
0x00014016f92c  lea      rax, [rdx + rdx*2]
0x00014016f930  lea      rbx, [r9 + rax*8]
0x00014016f934  test     r8, r8
0x00014016f937  je       0x14016f960
0x00014016f939  cmp      r9, rbx
0x00014016f93c  je       0x14016f960
0x00014016f93e  test     r9, r9
0x00014016f941  je       0x14016f960
0x00014016f943  test     rbx, rbx
0x00014016f946  je       0x14016f960
0x00014016f948  lea      r8, [r8 + r8*2]
0x00014016f94c  shl      r8, 3
0x00014016f950  mov      rdx, r9
0x00014016f953  mov      rcx, rbx
0x00014016f956  call     0x140431df0
0x00014016f95b  mov      r8, qword ptr [rsp + 0x50]
0x00014016f960  mov      r9, rbx
0x00014016f963  mov      qword ptr [rsp + 0x48], rbx
0x00014016f968  jmp      0x14016f987
0x00014016f96a  xor      r9d, r9d
0x00014016f96d  lea      r8d, [r9 + 1]
0x00014016f971  mov      edx, ebx
0x00014016f973  lea      rcx, [rsp + 0x40]
0x00014016f978  call     0x1400aa910
0x00014016f97d  mov      r8, qword ptr [rsp + 0x50]
0x00014016f982  mov      r9, qword ptr [rsp + 0x48]
0x00014016f987  test     r15b, r15b
0x00014016f98a  je       0x14016f9a5
0x00014016f98c  lea      rcx, [r9 - 0x18]
0x00014016f990  lea      rdx, [rbp + 0x200]
0x00014016f997  call     qword ptr [rip + 0x2f543b]   ; 0x140464dd8
0x00014016f99d  sub      qword ptr [rsp + 0x48], 0x18
0x00014016f9a3  jmp      0x14016f9d4
0x00014016f9a5  lea      rax, [rdi + rdi*2]
0x00014016f9a9  lea      rbx, [r9 + rax*8]
0x00014016f9ad  sub      r8, rdi
0x00014016f9b0  lea      r8, [r8 + r8*2]
0x00014016f9b4  shl      r8, 3
0x00014016f9b8  lea      rcx, [rbx + 0x18]
0x00014016f9bc  mov      rdx, rbx
0x00014016f9bf  call     0x140431df0
0x00014016f9c4  lea      rdx, [rbp + 0x200]
0x00014016f9cb  mov      rcx, rbx
0x00014016f9ce  call     qword ptr [rip + 0x2f5404]   ; 0x140464dd8
0x00014016f9d4  inc      qword ptr [rsp + 0x50]
0x00014016f9d9  lea      rcx, [rbp + 0x200]
0x00014016f9e0  call     qword ptr [rip + 0x2f55c2]   ; 0x140464fa8
0x00014016f9e6  movabs   r15, 0x2aaaaaaaaaaaaaab
0x00014016f9f0  mov      rax, qword ptr [rsp + 0x40]
0x00014016f9f5  test     rax, rax
0x00014016f9f8  je       0x14016fa01
0x00014016f9fa  mov      eax, dword ptr [rax]
0x00014016f9fc  cmp      eax, 1
0x00014016f9ff  jle      0x14016fa14
0x00014016fa01  xor      r9d, r9d
0x00014016fa04  xor      r8d, r8d
0x00014016fa07  xor      edx, edx
0x00014016fa09  lea      rcx, [rsp + 0x40]
0x00014016fa0e  call     0x1400aa910
0x00014016fa13  nop      
0x00014016fa14  lea      rcx, [rbp + 0xb60]
0x00014016fa1b  call     qword ptr [rip + 0x2f5587]   ; 0x140464fa8
0x00014016fa21  nop      
0x00014016fa22  lea      rcx, [rbp + 0x650]
0x00014016fa29  call     qword ptr [rip + 0x2f5579]   ; 0x140464fa8
0x00014016fa2f  inc      r12d
0x00014016fa32  movsxd   rbx, r12d
0x00014016fa35  lea      rcx, [rbp - 0x30]
0x00014016fa39  call     qword ptr [rip + 0x2f4de9]   ; 0x140464828
0x00014016fa3f  cmp      rbx, rax
0x00014016fa42  jl       0x14016f6c0
0x00014016fa48  lea      rcx, [rbp - 0x30]
0x00014016fa4c  call     qword ptr [rip + 0x2f4dde]   ; 0x140464830
0x00014016fa52  lea      rdx, [rip + 0x319a47]   ; "fontsList2"
0x00014016fa59  lea      rcx, [rbp + 0x668]
0x00014016fa60  call     qword ptr [rip + 0x2f542a]   ; 0x140464e90
0x00014016fa66  nop      
0x00014016fa67  lea      r8, [rbp + 0x668]
0x00014016fa6e  lea      rdx, [rbp + 0xb78]
0x00014016fa75  mov      rcx, r13
0x00014016fa78  call     qword ptr [rip + 0x2f4dca]   ; 0x140464848
0x00014016fa7e  nop      
0x00014016fa7f  lea      rdx, [rbp - 0x28]
0x00014016fa83  mov      rcx, rax
0x00014016fa86  call     qword ptr [rip + 0x2f4ddc]   ; 0x140464868
0x00014016fa8c  nop      
0x00014016fa8d  lea      rcx, [rbp + 0xb78]
0x00014016fa94  call     qword ptr [rip + 0x2f4dde]   ; 0x140464878
0x00014016fa9a  nop      
0x00014016fa9b  lea      rcx, [rbp + 0x668]
0x00014016faa2  call     qword ptr [rip + 0x2f5500]   ; 0x140464fa8
0x00014016faa8  mov      r12d, r14d
0x00014016faab  lea      rcx, [rbp - 0x28]
0x00014016faaf  call     qword ptr [rip + 0x2f4d73]   ; 0x140464828
0x00014016fab5  test     rax, rax
0x00014016fab8  jle      0x14016fe49
0x00014016fabe  mov      rbx, r14
0x00014016fac1  mov      r8, rbx
0x00014016fac4  lea      rdx, [rbp + 0xc58]
0x00014016facb  lea      rcx, [rbp - 0x28]
0x00014016facf  call     qword ptr [rip + 0x2f4d4b]   ; 0x140464820
0x00014016fad5  mov      rbx, rax
0x00014016fad8  lea      rcx, [rbp + 0x680]
0x00014016fadf  call     qword ptr [rip + 0x2f53db]   ; 0x140464ec0
0x00014016fae5  nop      
0x00014016fae6  lea      r8, [rbp + 0x680]
0x00014016faed  lea      rdx, [rbp + 0xb90]
0x00014016faf4  mov      rcx, rbx
0x00014016faf7  call     qword ptr [rip + 0x2f4a5b]   ; 0x140464558
0x00014016fafd  mov      r11, rax
0x00014016fb00  mov      rdi, qword ptr [rsp + 0x50]
0x00014016fb05  mov      rax, qword ptr [rsp + 0x40]
0x00014016fb0a  test     rax, rax
0x00014016fb0d  je       0x14016fbb4
0x00014016fb13  mov      ecx, dword ptr [rax]
0x00014016fb15  cmp      ecx, 1
0x00014016fb18  jg       0x14016fbb4
0x00014016fb1e  mov      r8, qword ptr [rsp + 0x40]
0x00014016fb23  lea      r10, [r8 + 0x17]
0x00014016fb27  and      r10, 0xfffffffffffffff8
0x00014016fb2b  mov      r9, qword ptr [rsp + 0x48]
0x00014016fb30  mov      rcx, r9
0x00014016fb33  sub      rcx, r10
0x00014016fb36  mov      rax, r15
0x00014016fb39  imul     rcx
0x00014016fb3c  sar      rdx, 2
0x00014016fb40  mov      rax, rdx
0x00014016fb43  shr      rax, 0x3f
0x00014016fb47  add      rdx, rax
0x00014016fb4a  mov      rax, qword ptr [r8 + 8]
0x00014016fb4e  sub      rax, rdx
0x00014016fb51  mov      rcx, qword ptr [rsp + 0x50]
0x00014016fb56  cmp      rax, rcx
0x00014016fb59  je       0x14016fb76
0x00014016fb5b  lea      rax, [rcx + rcx*2]
0x00014016fb5f  lea      rcx, [r9 + rax*8]
0x00014016fb63  mov      rdx, r11
0x00014016fb66  call     qword ptr [rip + 0x2f526c]   ; 0x140464dd8
0x00014016fb6c  inc      qword ptr [rsp + 0x50]
0x00014016fb71  jmp      0x14016fdf1
0x00014016fb76  test     rcx, rcx
0x00014016fb79  jne      0x14016fbb4
0x00014016fb7b  mov      rcx, r9
0x00014016fb7e  sub      rcx, r10
0x00014016fb81  mov      rax, r15
0x00014016fb84  imul     rcx
0x00014016fb87  sar      rdx, 2
0x00014016fb8b  mov      rax, rdx
0x00014016fb8e  shr      rax, 0x3f
0x00014016fb92  add      rdx, rax
0x00014016fb95  je       0x14016fbb4
0x00014016fb97  lea      rcx, [r9 - 0x18]
0x00014016fb9b  mov      rdx, r11
0x00014016fb9e  call     qword ptr [rip + 0x2f5234]   ; 0x140464dd8
0x00014016fba4  sub      qword ptr [rsp + 0x48], 0x18
0x00014016fbaa  inc      qword ptr [rsp + 0x50]
0x00014016fbaf  jmp      0x14016fdf1
0x00014016fbb4  mov      rdx, r11
0x00014016fbb7  lea      rcx, [rbp + 0x218]
0x00014016fbbe  call     qword ptr [rip + 0x2f5214]   ; 0x140464dd8
0x00014016fbc4  nop      
0x00014016fbc5  cmp      qword ptr [rsp + 0x50], 0
0x00014016fbcb  je       0x14016fbd7
0x00014016fbcd  test     rdi, rdi
0x00014016fbd0  jne      0x14016fbd7
0x00014016fbd2  mov      r15b, 1
0x00014016fbd5  jmp      0x14016fbda
0x00014016fbd7  xor      r15b, r15b
0x00014016fbda  movzx    ebx, r15b
0x00014016fbde  mov      rax, qword ptr [rsp + 0x40]
0x00014016fbe3  test     rax, rax
0x00014016fbe6  je       0x14016fd6b
0x00014016fbec  mov      eax, dword ptr [rax]
0x00014016fbee  cmp      eax, 1
0x00014016fbf1  jg       0x14016fd6b
0x00014016fbf7  mov      r9, qword ptr [rsp + 0x48]
0x00014016fbfc  mov      r8, qword ptr [rsp + 0x50]
0x00014016fc01  cmp      ebx, 1
0x00014016fc04  jne      0x14016fc98
0x00014016fc0a  mov      rdx, qword ptr [rsp + 0x40]
0x00014016fc0f  lea      rcx, [rdx + 0x17]
0x00014016fc13  and      rcx, 0xfffffffffffffff8
0x00014016fc17  mov      rax, r9
0x00014016fc1a  sub      rax, rcx
0x00014016fc1d  cmp      rax, 0x18
0x00014016fc21  jge      0x14016fd88
0x00014016fc27  mov      r11, qword ptr [rdx + 8]
0x00014016fc2b  lea      rax, [rdx + 0x17]
0x00014016fc2f  and      rax, 0xfffffffffffffff8
0x00014016fc33  mov      rcx, r9
0x00014016fc36  sub      rcx, rax
0x00014016fc39  movabs   rax, 0x2aaaaaaaaaaaaaab
0x00014016fc43  imul     rcx
0x00014016fc46  mov      r10, rdx
0x00014016fc49  sar      r10, 2
0x00014016fc4d  mov      rax, r10
0x00014016fc50  shr      rax, 0x3f
0x00014016fc54  add      r10, rax
0x00014016fc57  mov      rcx, r11
0x00014016fc5a  sub      rcx, r10
0x00014016fc5d  sub      rcx, r8
0x00014016fc60  cmp      rcx, 1
0x00014016fc64  jl       0x14016fd6b
0x00014016fc6a  lea      rcx, [r8 + r8*2]
0x00014016fc6e  cmp      rcx, r11
0x00014016fc71  jge      0x14016fd6b
0x00014016fc77  sub      r11, r8
0x00014016fc7a  lea      rax, [r11 - 1]
0x00014016fc7e  cqo      
0x00014016fc80  sub      rax, rdx
0x00014016fc83  sar      rax, 1
0x00014016fc86  mov      rdx, r14
0x00014016fc89  test     rax, rax
0x00014016fc8c  cmovg    rdx, rax
0x00014016fc90  inc      rdx
0x00014016fc93  jmp      0x14016fd2a
0x00014016fc98  mov      r10, qword ptr [rsp + 0x40]
0x00014016fc9d  test     r15b, r15b
0x00014016fca0  jne      0x14016fcdf
0x00014016fca2  lea      rax, [r10 + 0x17]
0x00014016fca6  and      rax, 0xfffffffffffffff8
0x00014016fcaa  mov      rcx, r9
0x00014016fcad  sub      rcx, rax
0x00014016fcb0  movabs   rax, 0x2aaaaaaaaaaaaaab
0x00014016fcba  imul     rcx
0x00014016fcbd  sar      rdx, 2
0x00014016fcc1  mov      rax, rdx
0x00014016fcc4  shr      rax, 0x3f
0x00014016fcc8  add      rdx, rax
0x00014016fccb  mov      rax, qword ptr [r10 + 8]
0x00014016fccf  sub      rax, rdx
0x00014016fcd2  sub      rax, r8
0x00014016fcd5  cmp      rax, 1
0x00014016fcd9  jge      0x14016fd88
0x00014016fcdf  mov      r11, qword ptr [r10 + 8]
0x00014016fce3  lea      rax, [r10 + 0x17]
0x00014016fce7  and      rax, 0xfffffffffffffff8
0x00014016fceb  mov      rcx, r9
0x00014016fcee  sub      rcx, rax
0x00014016fcf1  movabs   rax, 0x2aaaaaaaaaaaaaab
0x00014016fcfb  imul     rcx
0x00014016fcfe  mov      r10, rdx
0x00014016fd01  sar      r10, 2
0x00014016fd05  mov      rax, r10
0x00014016fd08  shr      rax, 0x3f
0x00014016fd0c  add      r10, rax
0x00014016fd0f  mov      rdx, r14
0x00014016fd12  test     r15b, r15b
0x00014016fd15  jne      0x14016fd6b
0x00014016fd17  cmp      r10, 1
0x00014016fd1b  jl       0x14016fd6b
0x00014016fd1d  lea      rcx, [r11 + r11]
0x00014016fd21  lea      rax, [r8 + r8*2]
0x00014016fd25  cmp      rax, rcx
0x00014016fd28  jge      0x14016fd6b
0x00014016fd2a  sub      rdx, r10
0x00014016fd2d  lea      rax, [rdx + rdx*2]
0x00014016fd31  lea      rbx, [r9 + rax*8]
0x00014016fd35  test     r8, r8
0x00014016fd38  je       0x14016fd61
0x00014016fd3a  cmp      r9, rbx
0x00014016fd3d  je       0x14016fd61
0x00014016fd3f  test     r9, r9
0x00014016fd42  je       0x14016fd61
0x00014016fd44  test     rbx, rbx
0x00014016fd47  je       0x14016fd61
0x00014016fd49  lea      r8, [r8 + r8*2]
0x00014016fd4d  shl      r8, 3
0x00014016fd51  mov      rdx, r9
0x00014016fd54  mov      rcx, rbx
0x00014016fd57  call     0x140431df0
0x00014016fd5c  mov      r8, qword ptr [rsp + 0x50]
0x00014016fd61  mov      r9, rbx
0x00014016fd64  mov      qword ptr [rsp + 0x48], rbx
0x00014016fd69  jmp      0x14016fd88
0x00014016fd6b  xor      r9d, r9d
0x00014016fd6e  lea      r8d, [r9 + 1]
0x00014016fd72  mov      edx, ebx
0x00014016fd74  lea      rcx, [rsp + 0x40]
0x00014016fd79  call     0x1400aa910
0x00014016fd7e  mov      r8, qword ptr [rsp + 0x50]
0x00014016fd83  mov      r9, qword ptr [rsp + 0x48]
0x00014016fd88  test     r15b, r15b
0x00014016fd8b  je       0x14016fda6
0x00014016fd8d  lea      rcx, [r9 - 0x18]
0x00014016fd91  lea      rdx, [rbp + 0x218]
0x00014016fd98  call     qword ptr [rip + 0x2f503a]   ; 0x140464dd8
0x00014016fd9e  sub      qword ptr [rsp + 0x48], 0x18
0x00014016fda4  jmp      0x14016fdd5
0x00014016fda6  lea      rax, [rdi + rdi*2]
0x00014016fdaa  lea      rbx, [r9 + rax*8]
0x00014016fdae  sub      r8, rdi
0x00014016fdb1  lea      r8, [r8 + r8*2]
0x00014016fdb5  shl      r8, 3
0x00014016fdb9  lea      rcx, [rbx + 0x18]
0x00014016fdbd  mov      rdx, rbx
0x00014016fdc0  call     0x140431df0
0x00014016fdc5  lea      rdx, [rbp + 0x218]
0x00014016fdcc  mov      rcx, rbx
0x00014016fdcf  call     qword ptr [rip + 0x2f5003]   ; 0x140464dd8
0x00014016fdd5  inc      qword ptr [rsp + 0x50]
0x00014016fdda  lea      rcx, [rbp + 0x218]
0x00014016fde1  call     qword ptr [rip + 0x2f51c1]   ; 0x140464fa8
0x00014016fde7  movabs   r15, 0x2aaaaaaaaaaaaaab
0x00014016fdf1  mov      rax, qword ptr [rsp + 0x40]
0x00014016fdf6  test     rax, rax
0x00014016fdf9  je       0x14016fe02
0x00014016fdfb  mov      eax, dword ptr [rax]
0x00014016fdfd  cmp      eax, 1
0x00014016fe00  jle      0x14016fe15
0x00014016fe02  xor      r9d, r9d
0x00014016fe05  xor      r8d, r8d
0x00014016fe08  xor      edx, edx
0x00014016fe0a  lea      rcx, [rsp + 0x40]
0x00014016fe0f  call     0x1400aa910
0x00014016fe14  nop      
0x00014016fe15  lea      rcx, [rbp + 0xb90]
0x00014016fe1c  call     qword ptr [rip + 0x2f5186]   ; 0x140464fa8
0x00014016fe22  nop      
0x00014016fe23  lea      rcx, [rbp + 0x680]
0x00014016fe2a  call     qword ptr [rip + 0x2f5178]   ; 0x140464fa8
0x00014016fe30  inc      r12d
0x00014016fe33  movsxd   rbx, r12d
0x00014016fe36  lea      rcx, [rbp - 0x28]
0x00014016fe3a  call     qword ptr [rip + 0x2f49e8]   ; 0x140464828
0x00014016fe40  cmp      rbx, rax
0x00014016fe43  jl       0x14016fac1
0x00014016fe49  cmp      qword ptr [rsp + 0x50], 0
0x00014016fe4f  jne      0x14016ffca
0x00014016fe55  lea      rdx, [rip + 0x319654]   ; "fontsList"
0x00014016fe5c  lea      rcx, [rbp + 0x698]
0x00014016fe63  call     qword ptr [rip + 0x2f5027]   ; 0x140464e90
0x00014016fe69  nop      
0x00014016fe6a  lea      r8, [rbp + 0x698]
0x00014016fe71  lea      rdx, [rbp + 0x8f0]
0x00014016fe78  mov      rcx, r13
0x00014016fe7b  call     qword ptr [rip + 0x2f49c7]   ; 0x140464848
0x00014016fe81  nop      
0x00014016fe82  lea      rdx, [rbp + 0x6c8]
0x00014016fe89  mov      rcx, rax
0x00014016fe8c  call     qword ptr [rip + 0x2f4896]   ; 0x140464728
0x00014016fe92  nop      
0x00014016fe93  lea      rcx, [rbp + 0x8f0]
0x00014016fe9a  call     qword ptr [rip + 0x2f49d8]   ; 0x140464878
0x00014016fea0  nop      
0x00014016fea1  lea      rcx, [rbp + 0x698]
0x00014016fea8  call     qword ptr [rip + 0x2f50fa]   ; 0x140464fa8
0x00014016feae  lea      rdx, [rip + 0x31960b]   ; "\{[^f]*font-family:[^']*'([^']*)';[^;]*;[^f]*font-style:\s*([^;]*)"
0x00014016feb5  lea      rcx, [rbp + 0x6b0]
0x00014016febc  call     qword ptr [rip + 0x2f4fce]   ; 0x140464e90
0x00014016fec2  nop      
0x00014016fec3  mov      r8d, r14d
0x00014016fec6  lea      rdx, [rbp + 0x6b0]
0x00014016fecd  lea      rcx, [rbp + 0x20]
0x00014016fed1  call     qword ptr [rip + 0x2f4dc9]   ; 0x140464ca0
0x00014016fed7  nop      
0x00014016fed8  lea      rcx, [rbp + 0x6b0]
0x00014016fedf  call     qword ptr [rip + 0x2f50c3]   ; 0x140464fa8
0x00014016fee5  mov      dword ptr [rsp + 0x28], r14d
0x00014016feea  mov      dword ptr [rsp + 0x20], r14d
0x00014016feef  xor      r9d, r9d
0x00014016fef2  lea      r8, [rbp + 0x6c8]
0x00014016fef9  lea      rdx, [rbp - 0x38]
0x00014016fefd  lea      rcx, [rbp + 0x20]
0x00014016ff01  call     qword ptr [rip + 0x2f40d9]   ; 0x140463fe0
0x00014016ff07  nop      
0x00014016ff08  lea      rcx, [rbp - 0x38]
0x00014016ff0c  call     qword ptr [rip + 0x2f40be]   ; 0x140463fd0
0x00014016ff12  test     al, al
0x00014016ff14  je       0x14016ffa7
0x00014016ff1a  nop      word ptr [rax + rax]
0x00014016ff20  lea      rdx, [rbp + 0x28]
0x00014016ff24  lea      rcx, [rbp - 0x38]
0x00014016ff28  call     qword ptr [rip + 0x2f409a]   ; 0x140463fc8
0x00014016ff2e  nop      
0x00014016ff2f  mov      r8d, 1
0x00014016ff35  lea      rdx, [rbp + 0x920]
0x00014016ff3c  lea      rcx, [rbp + 0x28]
0x00014016ff40  call     qword ptr [rip + 0x2f4c32]   ; 0x140464b78
0x00014016ff46  nop      
0x00014016ff47  mov      r8, rax
0x00014016ff4a  mov      rdx, qword ptr [rsp + 0x50]
0x00014016ff4f  lea      rcx, [rsp + 0x40]
0x00014016ff54  call     0x140105800
0x00014016ff59  mov      rax, qword ptr [rsp + 0x40]
0x00014016ff5e  test     rax, rax
0x00014016ff61  je       0x14016ff6a
0x00014016ff63  mov      eax, dword ptr [rax]
0x00014016ff65  cmp      eax, 1
0x00014016ff68  jle      0x14016ff7d
0x00014016ff6a  xor      r9d, r9d
0x00014016ff6d  xor      r8d, r8d
0x00014016ff70  xor      edx, edx
0x00014016ff72  lea      rcx, [rsp + 0x40]
0x00014016ff77  call     0x1400aa910
0x00014016ff7c  nop      
0x00014016ff7d  lea      rcx, [rbp + 0x920]
0x00014016ff84  call     qword ptr [rip + 0x2f501e]   ; 0x140464fa8
0x00014016ff8a  nop      
0x00014016ff8b  lea      rcx, [rbp + 0x28]
0x00014016ff8f  call     qword ptr [rip + 0x2f4bf3]   ; 0x140464b88
0x00014016ff95  lea      rcx, [rbp - 0x38]
0x00014016ff99  call     qword ptr [rip + 0x2f4031]   ; 0x140463fd0
0x00014016ff9f  test     al, al
0x00014016ffa1  jne      0x14016ff20
0x00014016ffa7  lea      rcx, [rbp - 0x38]
0x00014016ffab  call     qword ptr [rip + 0x2f4027]   ; 0x140463fd8
0x00014016ffb1  nop      
0x00014016ffb2  lea      rcx, [rbp + 0x20]
0x00014016ffb6  call     qword ptr [rip + 0x2f4cdc]   ; 0x140464c98
0x00014016ffbc  nop      
0x00014016ffbd  lea      rcx, [rbp + 0x6c8]
0x00014016ffc4  call     qword ptr [rip + 0x2f4fde]   ; 0x140464fa8
0x00014016ffca  lea      rdx, [rip + 0x319533]   ; "Times"
0x00014016ffd1  lea      rcx, [rbp + 0x6f8]
0x00014016ffd8  call     qword ptr [rip + 0x2f4eb2]   ; 0x140464e90
0x00014016ffde  nop      
0x00014016ffdf  lea      r8, [rbp + 0x6f8]
0x00014016ffe6  mov      rdx, qword ptr [rsp + 0x50]
0x00014016ffeb  lea      rcx, [rsp + 0x40]
0x00014016fff0  call     0x140105800
0x00014016fff5  mov      rax, qword ptr [rsp + 0x40]
0x00014016fffa  test     rax, rax
0x00014016fffd  je       0x140170006
0x00014016ffff  mov      eax, dword ptr [rax]
0x000140170001  cmp      eax, 1
0x000140170004  jle      0x140170018
0x000140170006  xor      r9d, r9d
0x000140170009  xor      r8d, r8d
0x00014017000c  xor      edx, edx
0x00014017000e  lea      rcx, [rsp + 0x40]
0x000140170013  call     0x1400aa910
0x000140170018  lea      rdx, [rip + 0x3194f1]   ; "Times New Roman"
0x00014017001f  lea      rcx, [rbp + 0x6e0]
0x000140170026  call     qword ptr [rip + 0x2f4e64]   ; 0x140464e90
0x00014017002c  nop      
0x00014017002d  lea      r8, [rbp + 0x6e0]
0x000140170034  mov      rdx, qword ptr [rsp + 0x50]
0x000140170039  lea      rcx, [rsp + 0x40]
0x00014017003e  call     0x140105800
0x000140170043  mov      rax, qword ptr [rsp + 0x40]
0x000140170048  test     rax, rax
0x00014017004b  je       0x140170054
0x00014017004d  mov      eax, dword ptr [rax]
0x00014017004f  cmp      eax, 1
0x000140170052  jle      0x140170067
0x000140170054  xor      r9d, r9d
0x000140170057  xor      r8d, r8d
0x00014017005a  xor      edx, edx
0x00014017005c  lea      rcx, [rsp + 0x40]
0x000140170061  call     0x1400aa910
0x000140170066  nop      
0x000140170067  lea      rcx, [rbp + 0x6e0]
0x00014017006e  call     qword ptr [rip + 0x2f4f34]   ; 0x140464fa8
0x000140170074  nop      
0x000140170075  lea      rcx, [rbp + 0x6f8]
0x00014017007c  call     qword ptr [rip + 0x2f4f26]   ; 0x140464fa8
0x000140170082  lea      rcx, [rbp + 0xb48]
0x000140170089  call     qword ptr [rip + 0x2f4e31]   ; 0x140464ec0
0x00014017008f  nop      
0x000140170090  mov      rdx, rax
0x000140170093  lea      rcx, [rbp + 0x710]
0x00014017009a  call     qword ptr [rip + 0x2f4e18]   ; 0x140464eb8
0x0001401700a0  nop      
0x0001401700a1  lea      rdx, [rbp + 0x710]
0x0001401700a8  lea      rcx, [rsp + 0x40]
0x0001401700ad  call     0x1400eb090
0x0001401700b2  nop      
0x0001401700b3  lea      rcx, [rbp + 0x710]
0x0001401700ba  call     qword ptr [rip + 0x2f4ee8]   ; 0x140464fa8
0x0001401700c0  nop      
0x0001401700c1  lea      rcx, [rbp + 0xb48]
0x0001401700c8  call     qword ptr [rip + 0x2f4eda]   ; 0x140464fa8
0x0001401700ce  lea      rcx, [rsp + 0x40]
0x0001401700d3  call     qword ptr [rip + 0x2f4417]   ; 0x1404644f0
0x0001401700d9  mov      eax, dword ptr [rsp + 0x50]
0x0001401700dd  mov      dword ptr [rbp - 0x40], eax
0x0001401700e0  lea      rdx, [rbp - 0x40]
0x0001401700e4  lea      rcx, [rsp + 0x38]
0x0001401700e9  call     0x1402bd230
0x0001401700ee  mov      edi, r14d
0x0001401700f1  cmp      qword ptr [rsp + 0x50], rdi
0x0001401700f6  jle      0x140170146
0x0001401700f8  mov      rbx, r14
0x0001401700fb  nop      dword ptr [rax + rax]
0x000140170100  mov      rax, qword ptr [rsp + 0x40]
0x000140170105  test     rax, rax
0x000140170108  je       0x140170111
0x00014017010a  mov      eax, dword ptr [rax]
0x00014017010c  cmp      eax, 1
0x00014017010f  jle      0x140170123
0x000140170111  xor      r9d, r9d
0x000140170114  xor      r8d, r8d
0x000140170117  xor      edx, edx
0x000140170119  lea      rcx, [rsp + 0x40]
0x00014017011e  call     0x1400aa910
0x000140170123  lea      rcx, [rbx + rbx*2]
0x000140170127  mov      rax, qword ptr [rsp + 0x48]
0x00014017012c  lea      rdx, [rax + rcx*8]
0x000140170130  lea      rcx, [rsp + 0x38]
0x000140170135  call     0x1402bd3a0
0x00014017013a  inc      edi
0x00014017013c  movsxd   rbx, edi
0x00014017013f  cmp      rbx, qword ptr [rsp + 0x50]
0x000140170144  jl       0x140170100
0x000140170146  lea      rcx, [rbp - 0x28]
0x00014017014a  call     qword ptr [rip + 0x2f46e0]   ; 0x140464830
0x000140170150  nop      
0x000140170151  mov      rcx, qword ptr [rsp + 0x40]
0x000140170156  test     rcx, rcx
0x000140170159  je       0x1401701a2
0x00014017015b  mov      eax, 0xffffffff
0x000140170160  lock xadd dword ptr [rcx], eax
0x000140170164  cmp      eax, 1
0x000140170167  setne    al
0x00014017016a  test     al, al
0x00014017016c  jne      0x1401701a2
0x00014017016e  mov      rax, qword ptr [rsp + 0x50]
0x000140170173  lea      rcx, [rax + rax*2]
0x000140170177  mov      rbx, qword ptr [rsp + 0x48]
0x00014017017c  lea      rdi, [rbx + rcx*8]
0x000140170180  cmp      rbx, rdi
0x000140170183  je       0x140170197
0x000140170185  mov      rcx, rbx
0x000140170188  call     qword ptr [rip + 0x2f4e1a]   ; 0x140464fa8
0x00014017018e  add      rbx, 0x18
0x000140170192  cmp      rbx, rdi
0x000140170195  jne      0x140170185
0x000140170197  mov      rcx, qword ptr [rsp + 0x40]
0x00014017019c  call     qword ptr [rip + 0x2f8eee]   ; 0x140469090
0x0001401701a2  lea      rdx, [rsi + 0x3e9]
0x0001401701a9  lea      rcx, [rsp + 0x38]
0x0001401701ae  call     0x1402bd1c0
0x0001401701b3  mov      rbx, rax
0x0001401701b6  lea      rcx, [rsi + 0x3f0]
0x0001401701bd  lea      rdx, [rbp + 0x2a8]
0x0001401701c4  call     qword ptr [rip + 0x2f44be]   ; 0x140464688
0x0001401701ca  nop      
0x0001401701cb  mov      rdx, rax
0x0001401701ce  mov      rcx, rbx
0x0001401701d1  call     0x1402bd310
0x0001401701d6  nop      
0x0001401701d7  lea      rcx, [rbp + 0x2a8]
0x0001401701de  call     qword ptr [rip + 0x2f4c3c]   ; 0x140464e20
0x0001401701e4  mov      byte ptr [rbp + 0xd20], 1
0x0001401701eb  lea      rdx, [rbp + 0xd20]
0x0001401701f2  lea      rcx, [rsp + 0x38]
0x0001401701f7  call     0x1402bd1c0
0x0001401701fc  mov      rbx, rax
0x0001401701ff  lea      rcx, [rsi + 0x338]
0x000140170206  lea      rdx, [rbp + 0x278]
0x00014017020d  call     qword ptr [rip + 0x2f4475]   ; 0x140464688
0x000140170213  nop      
0x000140170214  mov      rdx, rax
0x000140170217  mov      rcx, rbx
0x00014017021a  call     0x1402bd310
0x00014017021f  mov      rbx, rax
0x000140170222  lea      rcx, [rsi + 0x350]
0x000140170229  lea      rdx, [rbp + 0x290]
0x000140170230  call     qword ptr [rip + 0x2f4452]   ; 0x140464688
0x000140170236  nop      
0x000140170237  mov      rdx, rax
0x00014017023a  mov      rcx, rbx
0x00014017023d  call     0x1402bd310
0x000140170242  nop      
0x000140170243  lea      rcx, [rbp + 0x290]
0x00014017024a  call     qword ptr [rip + 0x2f4bd0]   ; 0x140464e20
0x000140170250  nop      
0x000140170251  lea      rcx, [rbp + 0x278]
0x000140170258  call     qword ptr [rip + 0x2f4bc2]   ; 0x140464e20
0x00014017025e  lea      rdx, [rsi + 0x449]
0x000140170265  lea      rcx, [rsp + 0x38]
0x00014017026a  call     0x1402bd1c0
0x00014017026f  mov      r8, rsi
0x000140170272  lea      rdx, [rsp + 0x38]
0x000140170277  lea      rcx, [rbp - 0x80]
0x00014017027b  call     0x1402d1ae0
0x000140170280  cmp      byte ptr [rsi + 0x446], 0
0x000140170287  je       0x1401702b5
0x000140170289  lea      r8, [rip + 0x319290]   ; ""devicePixelRatio":(\d*\.?\d*)"
0x000140170290  lea      rdx, [rbp - 0x20]
0x000140170294  lea      rcx, [rbp + 0x260]
0x00014017029b  call     0x14016e040
0x0001401702a0  nop      
0x0001401702a1  mov      dword ptr [rbp - 0x7c], 7
0x0001401702a8  xor      edx, edx
0x0001401702aa  mov      rcx, rax
0x0001401702ad  call     qword ptr [rip + 0x2f4ad5]   ; 0x140464d88
0x0001401702b3  jmp      0x1401702bd
0x0001401702b5  movsd    xmm0, qword ptr [rip + 0x30e31b]   ; 0x14047e5d8
0x0001401702bd  movsd    qword ptr [rbp + 0x10], xmm0
0x0001401702c2  test     byte ptr [rbp - 0x7c], 1
0x0001401702c6  je       0x1401702d5
0x0001401702c8  lea      rcx, [rbp + 0x260]
0x0001401702cf  call     qword ptr [rip + 0x2f4cd3]   ; 0x140464fa8
0x0001401702d5  lea      rdx, [rsi + 0x445]
0x0001401702dc  lea      rcx, [rsp + 0x38]
0x0001401702e1  call     0x1402bd1c0
0x0001401702e6  lea      rdx, [rsi + 0x1b8]
0x0001401702ed  mov      rcx, rax
0x0001401702f0  call     0x1402bd230
0x0001401702f5  lea      rdx, [rsi + 0x1bc]
0x0001401702fc  mov      rcx, rax
0x0001401702ff  call     0x1402bd230
0x000140170304  lea      rdx, [rsi + 0x1b0]
0x00014017030b  mov      rcx, rax
0x00014017030e  call     0x1402bd230
0x000140170313  lea      rdx, [rsi + 0x1b4]
0x00014017031a  mov      rcx, rax
0x00014017031d  call     0x1402bd230
0x000140170322  mov      rbx, rax
0x000140170325  lea      r8, [rip + 0x319214]   ; ""availWidth":(\d*)"
0x00014017032c  lea      rdx, [rbp - 0x20]
0x000140170330  lea      rcx, [rbp + 0x140]
0x000140170337  call     0x14016e040
0x00014017033c  nop      
0x00014017033d  xor      edx, edx
0x00014017033f  lea      r8d, [rdx + 0xa]
0x000140170343  mov      rcx, rax
0x000140170346  call     qword ptr [rip + 0x2f47a4]   ; 0x140464af0
0x00014017034c  mov      dword ptr [rbp - 0x68], eax
0x00014017034f  lea      rdx, [rbp - 0x68]
0x000140170353  mov      rcx, rbx
0x000140170356  call     0x1402bd230
0x00014017035b  mov      rbx, rax
0x00014017035e  lea      r8, [rip + 0x3191f3]   ; ""availHeight":(\d*)"
0x000140170365  lea      rdx, [rbp - 0x20]
0x000140170369  lea      rcx, [rbp + 0x128]
0x000140170370  call     0x14016e040
0x000140170375  nop      
0x000140170376  xor      edx, edx
0x000140170378  lea      r8d, [rdx + 0xa]
0x00014017037c  mov      rcx, rax
0x00014017037f  call     qword ptr [rip + 0x2f476b]   ; 0x140464af0
0x000140170385  mov      dword ptr [rbp - 0x6c], eax
0x000140170388  lea      rdx, [rbp - 0x6c]
0x00014017038c  mov      rcx, rbx
0x00014017038f  call     0x1402bd230
0x000140170394  lea      rdx, [rbp + 0x10]
0x000140170398  mov      rcx, rax
0x00014017039b  call     0x1402bd2a0
0x0001401703a0  mov      rbx, rax
0x0001401703a3  lea      r8, [rip + 0x3191c6]   ; ""maxTouchPoints":(\d*)"
0x0001401703aa  lea      rdx, [rbp - 0x20]
0x0001401703ae  lea      rcx, [rbp + 0x110]
0x0001401703b5  call     0x14016e040
0x0001401703ba  nop      
0x0001401703bb  xor      edx, edx
0x0001401703bd  lea      r8d, [rdx + 0xa]
0x0001401703c1  mov      rcx, rax
0x0001401703c4  call     qword ptr [rip + 0x2f4726]   ; 0x140464af0
0x0001401703ca  mov      dword ptr [rbp - 0x70], eax
0x0001401703cd  lea      rdx, [rbp - 0x70]
0x0001401703d1  mov      rcx, rbx
0x0001401703d4  call     0x1402bd230
0x0001401703d9  mov      rbx, rax
0x0001401703dc  lea      r8, [rip + 0x3191a5]   ; ""colorDepth":(\d*)"
0x0001401703e3  lea      rdx, [rbp - 0x20]
0x0001401703e7  lea      rcx, [rbp + 0x230]
0x0001401703ee  call     0x14016e040
0x0001401703f3  nop      
0x0001401703f4  xor      edx, edx
0x0001401703f6  lea      r8d, [rdx + 0xa]
0x0001401703fa  mov      rcx, rax
0x0001401703fd  call     qword ptr [rip + 0x2f46ed]   ; 0x140464af0
0x000140170403  mov      dword ptr [rbp - 0x74], eax
0x000140170406  lea      rdx, [rbp - 0x74]
0x00014017040a  mov      rcx, rbx
0x00014017040d  call     0x1402bd230
0x000140170412  mov      rbx, rax
0x000140170415  lea      r8, [rip + 0x319184]   ; ""pixelDepth":(\d*)"
0x00014017041c  lea      rdx, [rbp - 0x20]
0x000140170420  lea      rcx, [rbp + 0x248]
0x000140170427  call     0x14016e040
0x00014017042c  nop      
0x00014017042d  xor      edx, edx
0x00014017042f  lea      r8d, [rdx + 0xa]
0x000140170433  mov      rcx, rax
0x000140170436  call     qword ptr [rip + 0x2f46b4]   ; 0x140464af0
0x00014017043c  mov      dword ptr [rbp - 0x78], eax
0x00014017043f  lea      rdx, [rbp - 0x78]
0x000140170443  mov      rcx, rbx
0x000140170446  call     0x1402bd230
0x00014017044b  nop      
0x00014017044c  lea      rcx, [rbp + 0x248]
0x000140170453  call     qword ptr [rip + 0x2f4b4f]   ; 0x140464fa8
0x000140170459  nop      
0x00014017045a  lea      rcx, [rbp + 0x230]
0x000140170461  call     qword ptr [rip + 0x2f4b41]   ; 0x140464fa8
0x000140170467  nop      
0x000140170468  lea      rcx, [rbp + 0x110]
0x00014017046f  call     qword ptr [rip + 0x2f4b33]   ; 0x140464fa8
0x000140170475  nop      
0x000140170476  lea      rcx, [rbp + 0x128]
0x00014017047d  call     qword ptr [rip + 0x2f4b25]   ; 0x140464fa8
0x000140170483  nop      
0x000140170484  lea      rcx, [rbp + 0x140]
0x00014017048b  call     qword ptr [rip + 0x2f4b17]   ; 0x140464fa8
0x000140170491  lea      rdx, [rsi + 0x3e0]
0x000140170498  lea      rcx, [rsp + 0x38]
0x00014017049d  call     0x1402bd1c0
0x0001401704a2  lea      rdx, [rsi + 0x3a9]
0x0001401704a9  mov      rcx, rax
0x0001401704ac  call     0x1402bd1c0
0x0001401704b1  mov      rbx, rax
0x0001401704b4  lea      rcx, [rsi + 0x3b0]
0x0001401704bb  xor      edx, edx
0x0001401704bd  call     qword ptr [rip + 0x2f48c5]   ; 0x140464d88
0x0001401704c3  movsd    qword ptr [rbp], xmm0
0x0001401704c8  lea      rdx, [rbp]
0x0001401704cc  mov      rcx, rbx
0x0001401704cf  call     0x1402bd2a0
0x0001401704d4  mov      rbx, rax
0x0001401704d7  lea      rcx, [rsi + 0x3c8]
0x0001401704de  xor      edx, edx
0x0001401704e0  call     qword ptr [rip + 0x2f48a2]   ; 0x140464d88
0x0001401704e6  movsd    qword ptr [rbp - 0x58], xmm0
0x0001401704eb  lea      rdx, [rbp - 0x58]
0x0001401704ef  mov      rcx, rbx
0x0001401704f2  call     0x1402bd2a0
0x0001401704f7  mov      rbx, rax
0x0001401704fa  lea      rdx, [rip + 0x316847]   ; "acc"
0x000140170501  lea      rcx, [rbp + 0x50]
0x000140170505  call     qword ptr [rip + 0x2f4985]   ; 0x140464e90
0x00014017050b  nop      
0x00014017050c  lea      r8, [rbp + 0x50]
0x000140170510  lea      rdx, [rbp + 0x158]
0x000140170517  mov      rcx, r13
0x00014017051a  call     qword ptr [rip + 0x2f4328]   ; 0x140464848
0x000140170520  nop      
0x000140170521  xorps    xmm1, xmm1
0x000140170524  mov      rcx, rax
0x000140170527  call     qword ptr [rip + 0x2f412b]   ; 0x140464658
0x00014017052d  movsd    qword ptr [rbp - 0x60], xmm0
0x000140170532  lea      rdx, [rbp - 0x60]
0x000140170536  mov      rcx, rbx
0x000140170539  call     0x1402bd2a0
0x00014017053e  nop      
0x00014017053f  lea      rcx, [rbp + 0x158]
0x000140170546  call     qword ptr [rip + 0x2f432c]   ; 0x140464878
0x00014017054c  nop      
0x00014017054d  lea      rcx, [rbp + 0x50]
0x000140170551  call     qword ptr [rip + 0x2f4a51]   ; 0x140464fa8
0x000140170557  mov      byte ptr [rbp + 0xd20], 1
0x00014017055e  lea      rdx, [rbp + 0xd20]
0x000140170565  lea      rcx, [rsp + 0x38]
0x00014017056a  call     0x1402bd1c0
0x00014017056f  mov      rbx, rax
0x000140170572  lea      rdx, [rip + 0x31903f]   ; "storage"
0x000140170579  lea      rcx, [rbp + 0x68]
0x00014017057d  call     qword ptr [rip + 0x2f490d]   ; 0x140464e90
0x000140170583  nop      
0x000140170584  lea      r8, [rbp + 0x68]
0x000140170588  lea      rdx, [rbp + 0x760]
0x00014017058f  lea      rcx, [rbp - 0x50]
0x000140170593  call     qword ptr [rip + 0x2f42a7]   ; 0x140464840
0x000140170599  xorps    xmm1, xmm1
0x00014017059c  mov      rcx, rax
0x00014017059f  call     qword ptr [rip + 0x2f42b3]   ; 0x140464858
0x0001401705a5  xor      ecx, ecx
0x0001401705a7  movabs   rdi, 0x8000000000000000
0x0001401705b1  movsd    xmm6, qword ptr [rip + 0x3192ff]   ; 0x1404898b8
0x0001401705b9  comisd   xmm0, xmm6
0x0001401705bd  jb       0x1401705cc
0x0001401705bf  subsd    xmm0, xmm6
0x0001401705c3  comisd   xmm0, xmm6
0x0001401705c7  jae      0x1401705cc
0x0001401705c9  mov      rcx, rdi
0x0001401705cc  cvttsd2si rax, xmm0
0x0001401705d1  add      rax, rcx
0x0001401705d4  mov      qword ptr [rsp + 0x78], rax
0x0001401705d9  lea      rdx, [rsp + 0x78]
0x0001401705de  mov      rcx, rbx
0x0001401705e1  call     0x1402bd2a0
0x0001401705e6  nop      
0x0001401705e7  lea      rcx, [rbp + 0x68]
0x0001401705eb  call     qword ptr [rip + 0x2f49b7]   ; 0x140464fa8
0x0001401705f1  lea      rdx, [rip + 0x318fc8]   ; "performance"
0x0001401705f8  lea      rcx, [rbp + 0x80]
0x0001401705ff  call     qword ptr [rip + 0x2f488b]   ; 0x140464e90
0x000140170605  nop      
0x000140170606  lea      rdx, [rbp + 0x80]
0x00014017060d  lea      rcx, [rbp - 0x50]
0x000140170611  call     qword ptr [rip + 0x2f4221]   ; 0x140464838
0x000140170617  movzx    ebx, al
0x00014017061a  lea      rcx, [rbp + 0x80]
0x000140170621  call     qword ptr [rip + 0x2f4981]   ; 0x140464fa8
0x000140170627  test     bl, bl
0x000140170629  je       0x140170718
0x00014017062f  lea      rdx, [rip + 0x318f8a]   ; "performance"
0x000140170636  lea      rcx, [rbp + 0x98]
0x00014017063d  call     qword ptr [rip + 0x2f484d]   ; 0x140464e90
0x000140170643  nop      
0x000140170644  lea      r8, [rbp + 0x98]
0x00014017064b  lea      rdx, [rbp + 0x750]
0x000140170652  lea      rcx, [rbp - 0x50]
0x000140170656  call     qword ptr [rip + 0x2f41e4]   ; 0x140464840
0x00014017065c  lea      rdx, [rsp + 0x70]
0x000140170661  mov      rcx, rax
0x000140170664  call     qword ptr [rip + 0x2f41e6]   ; 0x140464850
0x00014017066a  nop      
0x00014017066b  lea      rcx, [rbp + 0x98]
0x000140170672  call     qword ptr [rip + 0x2f4930]   ; 0x140464fa8
0x000140170678  mov      byte ptr [rbp + 0xd20], 1
0x00014017067f  lea      rdx, [rbp + 0xd20]
0x000140170686  lea      rcx, [rsp + 0x38]
0x00014017068b  call     0x1402bd1c0
0x000140170690  mov      rbx, rax
0x000140170693  lea      rdx, [rip + 0x318f36]   ; "jsHeapSizeLimit"
0x00014017069a  lea      rcx, [rbp + 0xb0]
0x0001401706a1  call     qword ptr [rip + 0x2f47e9]   ; 0x140464e90
0x0001401706a7  nop      
0x0001401706a8  lea      r8, [rbp + 0xb0]
0x0001401706af  lea      rdx, [rbp + 0x740]
0x0001401706b6  lea      rcx, [rsp + 0x70]
0x0001401706bb  call     qword ptr [rip + 0x2f417f]   ; 0x140464840
0x0001401706c1  xorps    xmm1, xmm1
0x0001401706c4  mov      rcx, rax
0x0001401706c7  call     qword ptr [rip + 0x2f418b]   ; 0x140464858
0x0001401706cd  xor      ecx, ecx
0x0001401706cf  comisd   xmm0, xmm6
0x0001401706d3  jb       0x1401706e2
0x0001401706d5  subsd    xmm0, xmm6
0x0001401706d9  comisd   xmm0, xmm6
0x0001401706dd  jae      0x1401706e2
0x0001401706df  mov      rcx, rdi
0x0001401706e2  cvttsd2si rax, xmm0
0x0001401706e7  add      rax, rcx
0x0001401706ea  mov      qword ptr [rsp + 0x68], rax
0x0001401706ef  lea      rdx, [rsp + 0x68]
0x0001401706f4  mov      rcx, rbx
0x0001401706f7  call     0x1402bd2a0
0x0001401706fc  nop      
0x0001401706fd  lea      rcx, [rbp + 0xb0]
0x000140170704  call     qword ptr [rip + 0x2f489e]   ; 0x140464fa8
0x00014017070a  nop      
0x00014017070b  lea      rcx, [rsp + 0x70]
0x000140170710  call     qword ptr [rip + 0x2f41aa]   ; 0x1404648c0
0x000140170716  jmp      0x140170742
0x000140170718  mov      byte ptr [rbp + 0xd20], 0
0x00014017071f  lea      rdx, [rbp + 0xd20]
0x000140170726  lea      rcx, [rsp + 0x38]
0x00014017072b  call     0x1402bd1c0
0x000140170730  mov      qword ptr [rsp + 0x70], r14
0x000140170735  lea      rdx, [rsp + 0x70]
0x00014017073a  mov      rcx, rax
0x00014017073d  call     0x1402bd2a0
0x000140170742  mov      byte ptr [rbp + 0xd20], 1
0x000140170749  lea      rdx, [rbp + 0xd20]
0x000140170750  lea      rcx, [rsp + 0x38]
0x000140170755  call     0x1402bd1c0
0x00014017075a  mov      byte ptr [rbp + 0xd30], 1
0x000140170761  lea      rdx, [rbp + 0xd30]
0x000140170768  mov      rcx, rax
0x00014017076b  call     0x1402bd1c0
0x000140170770  mov      byte ptr [rbp + 0xd28], 1
0x000140170777  lea      rdx, [rbp + 0xd28]
0x00014017077e  mov      rcx, rax
0x000140170781  call     0x1402bd1c0
0x000140170786  mov      byte ptr [rsp + 0x31], 1
0x00014017078b  lea      rdx, [rsp + 0x31]
0x000140170790  mov      rcx, rax
0x000140170793  call     0x1402bd1c0
0x000140170798  mov      rbx, rax
0x00014017079b  lea      rdx, [rip + 0x318e3e]   ; ""Chrome PDF Viewer""
0x0001401707a2  lea      rcx, [rbp + 0xc8]
0x0001401707a9  call     qword ptr [rip + 0x2f46e1]   ; 0x140464e90
0x0001401707af  nop      
0x0001401707b0  mov      r9d, 1
0x0001401707b6  xor      r8d, r8d
0x0001401707b9  lea      rdx, [rbp + 0xc8]
0x0001401707c0  lea      rcx, [rbp - 0x20]
0x0001401707c4  call     qword ptr [rip + 0x2f420e]   ; 0x1404649d8
0x0001401707ca  cmp      rax, -1
0x0001401707ce  setne    byte ptr [rsp + 0x30]
0x0001401707d3  lea      rdx, [rsp + 0x30]
0x0001401707d8  mov      rcx, rbx
0x0001401707db  call     0x1402bd1c0
0x0001401707e0  mov      rbx, qword ptr [rbp + 0xd40]
0x0001401707e7  lea      rdx, [rbx + 0x1cc]
0x0001401707ee  mov      rcx, rax
0x0001401707f1  call     0x1402bd1c0
0x0001401707f6  lea      rdx, [rbx + 0x1ce]
0x0001401707fd  mov      rcx, rax
0x000140170800  call     0x1402bd1c0
0x000140170805  lea      rdx, [rbx + 0x1cf]
0x00014017080c  mov      rcx, rax
0x00014017080f  call     0x1402bd1c0
0x000140170814  lea      rdx, [rbx + 0x1d1]
0x00014017081b  mov      rcx, rax
0x00014017081e  call     0x1402bd1c0
0x000140170823  nop      
0x000140170824  lea      rcx, [rbp + 0xc8]
0x00014017082b  call     qword ptr [rip + 0x2f4777]   ; 0x140464fa8
0x000140170831  mov      eax, dword ptr [rsi + 0x300]
0x000140170837  mov      dword ptr [rbp + 0x30], eax
0x00014017083a  lea      rdx, [rsi + 0x308]
0x000140170841  lea      rcx, [rbp + 0x38]
0x000140170845  call     qword ptr [rip + 0x2f466d]   ; 0x140464eb8
0x00014017084b  nop      
0x00014017084c  lea      rcx, [rbp + 0x30]
0x000140170850  call     0x140260d70
0x000140170855  test     eax, eax
0x000140170857  jne      0x1401708f5
0x00014017085d  lea      rdx, [rip + 0x318d94]   ; "uaFullVersion":""
0x000140170864  lea      rcx, [rbp + 0xe0]
0x00014017086b  call     qword ptr [rip + 0x2f461f]   ; 0x140464e90
0x000140170871  nop      
0x000140170872  mov      r9d, 1
0x000140170878  xor      r8d, r8d
0x00014017087b  lea      rdx, [rbp + 0xe0]
0x000140170882  lea      rcx, [rbp - 0x20]
0x000140170886  call     qword ptr [rip + 0x2f414c]   ; 0x1404649d8
0x00014017088c  lea      ebx, [rax + 0x10]
0x00014017088f  lea      rcx, [rbp + 0xe0]
0x000140170896  call     qword ptr [rip + 0x2f470c]   ; 0x140464fa8
0x00014017089c  mov      dl, 0x22
0x00014017089e  lea      rcx, [rbp - 0x48]
0x0001401708a2  call     qword ptr [rip + 0x2f42c8]   ; 0x140464b70
0x0001401708a8  lea      ecx, [rbx + 1]
0x0001401708ab  movsxd   r8, ecx
0x0001401708ae  mov      r9d, 1
0x0001401708b4  movzx    edx, word ptr [rax]
0x0001401708b7  lea      rcx, [rbp - 0x20]
0x0001401708bb  call     qword ptr [rip + 0x2f3a1f]   ; 0x1404642e0
0x0001401708c1  sub      eax, ebx
0x0001401708c3  movsxd   r9, eax
0x0001401708c6  movsxd   r8, ebx
0x0001401708c9  lea      rdx, [rbp + 0x170]
0x0001401708d0  lea      rcx, [rbp - 0x20]
0x0001401708d4  call     qword ptr [rip + 0x2f434e]   ; 0x140464c28
0x0001401708da  nop      
0x0001401708db  mov      rdx, rax
0x0001401708de  lea      rcx, [rbp + 0x30]
0x0001401708e2  call     0x1402af8a0
0x0001401708e7  nop      
0x0001401708e8  lea      rcx, [rbp + 0x170]
0x0001401708ef  call     qword ptr [rip + 0x2f46b3]   ; 0x140464fa8
0x0001401708f5  call     0x1402d9eb0
0x0001401708fa  lea      rcx, [rsi + 0x198]
0x000140170901  mov      qword ptr [rsp + 0x20], rcx
0x000140170906  lea      r9, [rbp + 0x30]
0x00014017090a  lea      r8, [rsi + 0x2e0]
0x000140170911  lea      rdx, [rbp + 0x1d0]
0x000140170918  mov      rcx, rax
0x00014017091b  call     0x1402d7100
0x000140170920  nop      
0x000140170921  mov      rdx, rax
0x000140170924  lea      rcx, [rsp + 0x38]
0x000140170929  call     0x1402bd310
0x00014017092e  mov      rbx, rax
0x000140170931  mov      r8, 0xffffffffffffffff
0x000140170938  lea      rdx, [rip + 0x318ccd]   ; "2026"
0x00014017093f  lea      rcx, [rbp + 0x1b8]
0x000140170946  call     qword ptr [rip + 0x2f3c44]   ; 0x140464590
0x00014017094c  nop      
0x00014017094d  mov      rdx, rax
0x000140170950  mov      rcx, rbx
0x000140170953  call     0x1402bd310
0x000140170958  mov      rbx, rax
0x00014017095b  mov      r8, 0xffffffffffffffff
0x000140170962  lea      rdx, [rip + 0x318caf]   ; "Copyright 2026 Google LLC. All rights reserved."
0x000140170969  lea      rcx, [rbp + 0x1a0]
0x000140170970  call     qword ptr [rip + 0x2f3c1a]   ; 0x140464590
0x000140170976  nop      
0x000140170977  mov      rdx, rax
0x00014017097a  mov      rcx, rbx
0x00014017097d  call     0x1402bd310
0x000140170982  mov      rbx, rax
0x000140170985  mov      r8, 0xffffffffffffffff
0x00014017098c  lea      rdx, [rip + 0x318cb5]   ; "stable"
0x000140170993  lea      rcx, [rbp + 0x188]
0x00014017099a  call     qword ptr [rip + 0x2f3bf0]   ; 0x140464590
0x0001401709a0  nop      
0x0001401709a1  mov      rdx, rax
0x0001401709a4  mov      rcx, rbx
0x0001401709a7  call     0x1402bd310
0x0001401709ac  mov      rbx, rax
0x0001401709af  lea      rcx, [rsi + 0x300]
0x0001401709b6  call     0x140260d70
0x0001401709bb  test     eax, eax
0x0001401709bd  jne      0x1401709e5
0x0001401709bf  lea      rcx, [rsi + 0x2e0]
0x0001401709c6  call     0x140260d70
0x0001401709cb  cmp      eax, 2
0x0001401709ce  je       0x1401709e5
0x0001401709d0  lea      rcx, [rsi + 0x2e0]
0x0001401709d7  call     0x140260d70
0x0001401709dc  cmp      eax, 3
0x0001401709df  je       0x1401709e5
0x0001401709e1  mov      al, 1
0x0001401709e3  jmp      0x1401709e7
0x0001401709e5  xor      al, al
0x0001401709e7  mov      byte ptr [rbp + 0xd40], al
0x0001401709ed  lea      rdx, [rbp + 0xd40]
0x0001401709f4  mov      rcx, rbx
0x0001401709f7  call     0x1402bd1c0
0x0001401709fc  nop      
0x0001401709fd  lea      rcx, [rbp + 0x188]
0x000140170a04  call     qword ptr [rip + 0x2f4416]   ; 0x140464e20
0x000140170a0a  nop      
0x000140170a0b  lea      rcx, [rbp + 0x1a0]
0x000140170a12  call     qword ptr [rip + 0x2f4408]   ; 0x140464e20
0x000140170a18  nop      
0x000140170a19  lea      rcx, [rbp + 0x1b8]
0x000140170a20  call     qword ptr [rip + 0x2f43fa]   ; 0x140464e20
0x000140170a26  nop      
0x000140170a27  lea      rcx, [rbp + 0x1d0]
0x000140170a2e  call     qword ptr [rip + 0x2f43ec]   ; 0x140464e20
0x000140170a34  nop      
0x000140170a35  lea      rcx, [rbp + 0x38]
0x000140170a39  call     qword ptr [rip + 0x2f4569]   ; 0x140464fa8
0x000140170a3f  mov      eax, dword ptr [rsi + 0x4a4]
0x000140170a45  cmp      eax, 0xf
0x000140170a48  jne      0x140170a72
0x000140170a4a  lea      rcx, [rsi + 0x2e0]
0x000140170a51  call     0x140260d70
0x000140170a56  cmp      eax, 1
0x000140170a59  jne      0x140170a6b
0x000140170a5b  lea      rcx, [rsi + 0x300]
0x000140170a62  call     0x140260d70
0x000140170a67  test     eax, eax
0x000140170a69  je       0x140170a77
0x000140170a6b  mov      eax, 5
0x000140170a70  jmp      0x140170a86
0x000140170a72  cmp      eax, 0x10
0x000140170a75  jne      0x140170a86
0x000140170a77  mov      rax, qword ptr [rbp - 8]
0x000140170a7b  add      rax, rax
0x000140170a7e  not      eax
0x000140170a80  and      eax, 2
0x000140170a83  or       eax, 1
0x000140170a86  mov      byte ptr [rbp + 0xd20], al
0x000140170a8c  lea      rdx, [rbp + 0xd20]
0x000140170a93  lea      rcx, [rsp + 0x38]
0x000140170a98  call     0x1402bd1c0
0x000140170a9d  call     0x1402bc6e0
0x000140170aa2  mov      rcx, rax
0x000140170aa5  call     0x1400f7230
0x000140170aaa  mov      rdx, rax
0x000140170aad  lea      rcx, [rsp + 0x38]
0x000140170ab2  call     0x1402bd310
0x000140170ab7  movzx    eax, byte ptr [rsi + 0x4b0]
0x000140170abe  mov      byte ptr [rbp + 0xd20], al
0x000140170ac4  lea      rdx, [rbp + 0xd20]
0x000140170acb  lea      rcx, [rsp + 0x38]
0x000140170ad0  call     0x1402bd1c0
0x000140170ad5  movzx    ecx, byte ptr [rsi + 0x4b4]
0x000140170adc  mov      byte ptr [rsp + 0x58], cl
0x000140170ae0  lea      rdx, [rsp + 0x58]
0x000140170ae5  mov      rcx, rax
0x000140170ae8  call     0x1402bd1c0
0x000140170aed  movzx    ecx, byte ptr [rsi + 0x4b8]
0x000140170af4  mov      byte ptr [rsp + 0x59], cl
0x000140170af8  lea      rdx, [rsp + 0x59]
0x000140170afd  mov      rcx, rax
0x000140170b00  call     0x1402bd1c0
0x000140170b05  movzx    ecx, byte ptr [rsi + 0x4bc]
0x000140170b0c  mov      byte ptr [rsp + 0x5a], cl
0x000140170b10  lea      rdx, [rsp + 0x5a]
0x000140170b15  mov      rcx, rax
0x000140170b18  call     0x1402bd1c0
0x000140170b1d  movzx    ecx, byte ptr [rsi + 0x4c0]
0x000140170b24  mov      byte ptr [rsp + 0x5b], cl
0x000140170b28  lea      rdx, [rsp + 0x5b]
0x000140170b2d  mov      rcx, rax
0x000140170b30  call     0x1402bd1c0
0x000140170b35  movzx    ecx, byte ptr [rsi + 0x4c4]
0x000140170b3c  mov      byte ptr [rsp + 0x5c], cl
0x000140170b40  lea      rdx, [rsp + 0x5c]
0x000140170b45  mov      rcx, rax
0x000140170b48  call     0x1402bd1c0
0x000140170b4d  movzx    ecx, byte ptr [rsi + 0x4c8]
0x000140170b54  mov      byte ptr [rsp + 0x5d], cl
0x000140170b58  lea      rdx, [rsp + 0x5d]
0x000140170b5d  mov      rcx, rax
0x000140170b60  call     0x1402bd1c0
0x000140170b65  movzx    ecx, byte ptr [rsi + 0x4cc]
0x000140170b6c  mov      byte ptr [rsp + 0x5e], cl
0x000140170b70  lea      rdx, [rsp + 0x5e]
0x000140170b75  mov      rcx, rax
0x000140170b78  call     0x1402bd1c0
0x000140170b7d  movzx    ecx, byte ptr [rsi + 0x4d0]
0x000140170b84  mov      byte ptr [rsp + 0x5f], cl
0x000140170b88  lea      rdx, [rsp + 0x5f]
0x000140170b8d  mov      rcx, rax
0x000140170b90  call     0x1402bd1c0
0x000140170b95  movzx    ecx, byte ptr [rsi + 0x4d4]
0x000140170b9c  mov      byte ptr [rsp + 0x60], cl
0x000140170ba0  lea      rdx, [rsp + 0x60]
0x000140170ba5  mov      rcx, rax
0x000140170ba8  call     0x1402bd1c0
0x000140170bad  movzx    ecx, byte ptr [rsi + 0x4d8]
0x000140170bb4  mov      byte ptr [rsp + 0x61], cl
0x000140170bb8  lea      rdx, [rsp + 0x61]
0x000140170bbd  mov      rcx, rax
0x000140170bc0  call     0x1402bd1c0
0x000140170bc5  movzx    ecx, byte ptr [rsi + 0x4dc]
0x000140170bcc  mov      byte ptr [rsp + 0x62], cl
0x000140170bd0  lea      rdx, [rsp + 0x62]
0x000140170bd5  mov      rcx, rax
0x000140170bd8  call     0x1402bd1c0
0x000140170bdd  lea      rdx, [rbp + 0xd20]
0x000140170be4  lea      rcx, [rsp + 0x38]
0x000140170be9  cmp      byte ptr [rsi + 0x4f8], 0
0x000140170bf0  je       0x140170c59
0x000140170bf2  mov      eax, dword ptr [rsi + 0x4f0]
0x000140170bf8  mov      dword ptr [rbp + 0xd20], eax
0x000140170bfe  call     0x1402bd230
0x000140170c03  cmp      qword ptr [rsi + 0x4f0], 0
0x000140170c0b  jle      0x140170c66
0x000140170c0d  mov      rax, r14
0x000140170c10  lea      rcx, [rax + rax*2]
0x000140170c14  mov      rax, qword ptr [rsi + 0x4e8]
0x000140170c1b  lea      rcx, [rax + rcx*8]
0x000140170c1f  lea      rdx, [rbp + 0x1e8]
0x000140170c26  call     qword ptr [rip + 0x2f3a5c]   ; 0x140464688
0x000140170c2c  nop      
0x000140170c2d  mov      rdx, rax
0x000140170c30  lea      rcx, [rsp + 0x38]
0x000140170c35  call     0x1402bd310
0x000140170c3a  nop      
0x000140170c3b  lea      rcx, [rbp + 0x1e8]
0x000140170c42  call     qword ptr [rip + 0x2f41d8]   ; 0x140464e20
0x000140170c48  inc      r14d
0x000140170c4b  movsxd   rax, r14d
0x000140170c4e  cmp      rax, qword ptr [rsi + 0x4f0]
0x000140170c55  jl       0x140170c10
0x000140170c57  jmp      0x140170c66
0x000140170c59  mov      dword ptr [rbp + 0xd20], r14d
0x000140170c60  call     0x1402bd230
0x000140170c65  nop      
0x000140170c66  lea      rcx, [rbp - 0x20]
0x000140170c6a  call     qword ptr [rip + 0x2f4338]   ; 0x140464fa8
0x000140170c70  nop      
0x000140170c71  lea      rcx, [rbp - 0x50]
0x000140170c75  call     qword ptr [rip + 0x2f3c45]   ; 0x1404648c0
0x000140170c7b  mov      rbx, qword ptr [rsp + 0xe38]
0x000140170c83  movaps   xmm6, xmmword ptr [rsp + 0xdd0]
0x000140170c8b  add      rsp, 0xde0
0x000140170c92  pop      r15
0x000140170c94  pop      r14
0x000140170c96  pop      r13
0x000140170c98  pop      r12
0x000140170c9a  pop      rdi
0x000140170c9b  pop      rsi
0x000140170c9c  pop      rbp
0x000140170c9d  ret      
