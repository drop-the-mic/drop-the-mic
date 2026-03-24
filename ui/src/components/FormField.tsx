import type { InputHTMLAttributes, TextareaHTMLAttributes, SelectHTMLAttributes, ReactNode } from 'react';

const labelStyle: React.CSSProperties = {
  display: 'block', fontSize: 12, fontWeight: 500,
  color: 'var(--text-tertiary)', marginBottom: 6,
  textTransform: 'uppercase', letterSpacing: '0.04em',
};

/** A labeled form field wrapper. */
export function FormField({ label, children, style }: { label: ReactNode; children: ReactNode; style?: React.CSSProperties }) {
  return (
    <div style={{ marginBottom: 16, ...style }}>
      <label style={labelStyle}>{label}</label>
      {children}
    </div>
  );
}

/** Styled text input — focus state handled via CSS class. */
export function Input(props: InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className={`dtm-input ${props.className || ''}`} />;
}

/** Styled textarea — focus state handled via CSS class. */
export function TextArea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea {...props} className={`dtm-input dtm-textarea ${props.className || ''}`} />;
}

/** Styled select — focus state handled via CSS class. */
export function Select(props: SelectHTMLAttributes<HTMLSelectElement> & { children: ReactNode }) {
  return <select {...props} className={`dtm-input dtm-select ${props.className || ''}`}>{props.children}</select>;
}
