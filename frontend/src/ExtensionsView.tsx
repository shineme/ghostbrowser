import React from 'react';
import { Profile } from './api';

// Placeholder: 扩展中心
export function ExtensionsView(_p: { profiles: Profile[]; reload?: () => void }) {
  return <div className="empty">扩展中心</div>;
}
