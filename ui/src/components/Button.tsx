import type { ButtonHTMLAttributes, ReactNode } from 'react';

type Variant = 'primary' | 'secondary' | 'danger' | 'ghost';
type Size = 'sm' | 'md' | 'lg';

const variantStyles: Record<Variant, React.CSSProperties> = {
  primary:   { background: 'var(--accent)', color: '#fff', border: 'none' },
  secondary: { background: 'transparent', color: 'var(--text-secondary)', border: '1px solid var(--border)' },
  danger:    { background: 'var(--danger-muted)', color: 'var(--danger)', border: '1px solid rgba(239,68,68,0.2)' },
  ghost:     { background: 'transparent', color: 'var(--text-secondary)', border: 'none' },
};

const sizeStyles: Record<Size, React.CSSProperties> = {
  sm: { padding: '5px 12px', fontSize: 12, borderRadius: 'var(--radius-sm)' },
  md: { padding: '8px 16px', fontSize: 13, borderRadius: 'var(--radius-md)' },
  lg: { padding: '10px 20px', fontSize: 14, borderRadius: 'var(--radius-md)' },
};

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
  icon?: ReactNode;
}

export function Button({ variant = 'primary', size = 'md', icon, children, style, disabled, ...props }: ButtonProps) {
  return (
    <button
      disabled={disabled}
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 6,
        fontWeight: 500,
        cursor: disabled ? 'not-allowed' : 'pointer',
        opacity: disabled ? 0.5 : 1,
        transition: 'var(--transition-fast)',
        ...variantStyles[variant],
        ...sizeStyles[size],
        ...style,
      }}
      {...props}
    >
      {icon}
      {children}
    </button>
  );
}
