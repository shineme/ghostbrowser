// Upload / 修复云端 errors that found a browser still running on the profile
// directory. When GhostBrowser can end the leftover processes itself the
// error says so and we offer a one-click cleanup (confirm first).
import { api } from './api';

type ToastFn = (msg: string, kind?: string, action?: { label: string; onClick: () => void }) => void;
type AskFn = (o: { title: string; body?: any; ok?: string; danger?: boolean }) => Promise<boolean>;

export const LEFTOVER_HINT = '结束残留进程';

export function syncErrorToast(id: string, e: any, toast: ToastFn, ask: AskFn, after?: () => void) {
  const msg = String(e?.message || e);
  if (!msg.includes(LEFTOVER_HINT)) { toast(msg, 'bad'); return; }
  toast(msg, 'bad', {
    label: LEFTOVER_HINT,
    onClick: async () => {
      if (!await ask({ title: '结束这个环境的残留浏览器进程？', body: '只会结束 GhostBrowser 为这个环境启动、窗口关闭后仍在后台运行的浏览器进程，不影响其他环境。未保存的页面数据会丢失。', ok: LEFTOVER_HINT, danger: true })) return;
      try { const n = await api.KillLeftoverProcesses(id); toast(`已结束 ${n} 个残留进程，请重新上传 / 修复`, 'ok'); after?.(); }
      catch (e2: any) { toast(String(e2?.message || e2), 'bad'); }
    },
  });
}
