import type { CSSProperties, ReactNode } from 'react';

type Variant = 'pass' | 'warn' | 'fail' | 'info' | 'neutral';

const styles: Record<Variant, CSSProperties> = {
  pass:    { background: 'var(--success-muted)', color: 'var(--success)' },
  warn:    { background: 'var(--warning-muted)', color: 'var(--warning)' },
  fail:    { background: 'var(--danger-muted)',  color: 'var(--danger)' },
  info:    { background: 'var(--accent-muted)',  color: 'var(--accent)' },
  neutral: { background: 'var(--border)',        color: 'var(--text-secondary)' },
};

export function Badge({ variant = 'neutral', children }: { variant?: Variant; children: ReactNode }) {
  return (
    <span style={{
      display: 'inline-flex',
      alignItems: 'center',
      gap: 4,
      padding: '2px 8px',
      borderRadius: 'var(--radius-sm)',
      fontSize: 12,
      fontWeight: 600,
      lineHeight: '18px',
      whiteSpace: 'nowrap',
      ...styles[variant],
    }}>
      {children}
    </span>
  );
}

export function VerdictBadge({ verdict }: { verdict: string }) {
  const v = verdict.toUpperCase();
  const variant: Variant = v === 'PASS' ? 'pass' : v === 'WARN' ? 'warn' : v === 'FAIL' ? 'fail' : 'neutral';
  return <Badge variant={variant}>{v}</Badge>;
}

export function SeverityBadge({ severity }: { severity: string }) {
  return <Badge variant="neutral">severity: {severity}</Badge>;
}
