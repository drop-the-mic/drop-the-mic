import type { CSSProperties, ReactNode } from 'react';

interface CardProps {
  children: ReactNode;
  style?: CSSProperties;
  className?: string;
  onClick?: () => void;
  hoverable?: boolean;
}

/** A styled container card. Uses CSS hover class instead of DOM event handlers. */
export function Card({ children, style, className = '', onClick, hoverable }: CardProps) {
  const isInteractive = onClick || hoverable;
  return (
    <div
      className={`${isInteractive ? 'card-hoverable' : ''} ${className}`}
      onClick={onClick}
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 'var(--radius-lg)',
        padding: 20,
        cursor: isInteractive ? 'pointer' : undefined,
        ...style,
      }}
    >
      {children}
    </div>
  );
}

/** A compact stat display card with large number and label. */
export function StatCard({ value, label, color }: { value: string | number; label: string; color?: string }) {
  return (
    <Card style={{ textAlign: 'center', padding: '24px 16px' }}>
      <div style={{ fontSize: 32, fontWeight: 700, color: color || 'var(--text-primary)', lineHeight: 1 }}>{value}</div>
      <div style={{ fontSize: 12, color: 'var(--text-tertiary)', marginTop: 8, textTransform: 'uppercase', letterSpacing: '0.05em', fontWeight: 500 }}>{label}</div>
    </Card>
  );
}
