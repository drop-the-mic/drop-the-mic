import type { ReactNode } from 'react';

export function EmptyState({ icon, title, description, action }: {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div style={{
      display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
      padding: '48px 24px', gap: 12, color: 'var(--text-tertiary)',
    }}>
      {icon && <div style={{ opacity: 0.5, marginBottom: 4 }}>{icon}</div>}
      <div style={{ fontSize: 15, fontWeight: 500, color: 'var(--text-secondary)' }}>{title}</div>
      {description && <div style={{ fontSize: 13, maxWidth: 360, textAlign: 'center', lineHeight: 1.5 }}>{description}</div>}
      {action && <div style={{ marginTop: 8 }}>{action}</div>}
    </div>
  );
}

export function Spinner({ size = 20 }: { size?: number }) {
  return (
    <div style={{
      width: size, height: size,
      border: `2px solid var(--border)`,
      borderTopColor: 'var(--accent)',
      borderRadius: '50%',
      animation: 'spin 0.6s linear infinite',
    }} />
  );
}

export function LoadingState() {
  return (
    <div style={{ display: 'flex', justifyContent: 'center', padding: 64 }}>
      <Spinner size={28} />
    </div>
  );
}
