import { useState, useRef, useEffect } from 'react';
import type { CSSProperties } from 'react';

interface ComboboxProps {
  /** Available options. */
  options: string[];
  /** Currently selected value(s). */
  value: string[];
  /** Called when selection changes. */
  onChange: (value: string[]) => void;
  /** Allow selecting multiple options. */
  multi?: boolean;
  /** Placeholder when nothing is selected. */
  placeholder?: string;
  /** Whether the control is disabled. */
  disabled?: boolean;
  /** Whether the options are still loading. */
  loading?: boolean;
}

export function Combobox({ options, value, onChange, multi, placeholder, disabled, loading }: ComboboxProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const ref = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Close on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
        setSearch('');
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const filtered = options.filter(o =>
    o.toLowerCase().includes(search.toLowerCase())
  );

  const toggle = (item: string) => {
    if (multi) {
      if (value.includes(item)) {
        onChange(value.filter(v => v !== item));
      } else {
        onChange([...value, item]);
      }
    } else {
      onChange([item]);
      setOpen(false);
      setSearch('');
    }
  };

  const remove = (item: string) => {
    onChange(value.filter(v => v !== item));
  };

  const displayText = multi
    ? ''
    : (value[0] || '');

  return (
    <div ref={ref} style={wrapStyle}>
      <div
        className="dtm-input"
        style={triggerStyle(disabled)}
        onClick={() => {
          if (!disabled) {
            setOpen(true);
            setTimeout(() => inputRef.current?.focus(), 0);
          }
        }}
      >
        {multi && value.length > 0 && (
          <div style={tagsStyle}>
            {value.map(v => (
              <span key={v} style={tagStyle}>
                {v}
                <span style={tagRemoveStyle} onClick={e => { e.stopPropagation(); remove(v); }}>&times;</span>
              </span>
            ))}
          </div>
        )}
        <input
          ref={inputRef}
          style={inputStyle}
          value={open ? search : displayText}
          placeholder={value.length === 0 ? (placeholder || 'Select...') : (multi ? 'Search...' : '')}
          disabled={disabled}
          onChange={e => {
            setSearch(e.target.value);
            if (!open) setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={e => {
            if (e.key === 'Escape') {
              setOpen(false);
              setSearch('');
            }
            if (e.key === 'Backspace' && !search && multi && value.length > 0) {
              remove(value[value.length - 1]);
            }
          }}
        />
        <svg width="12" height="12" viewBox="0 0 12 12" style={{ flexShrink: 0, opacity: 0.4 }}>
          <path d="M3 4.5L6 7.5L9 4.5" stroke="currentColor" strokeWidth="1.5" fill="none" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </div>

      {open && (
        <div style={dropdownStyle}>
          {loading ? (
            <div style={emptyStyle}>Loading...</div>
          ) : filtered.length === 0 ? (
            <div style={emptyStyle}>{search ? 'No matches' : 'No options'}</div>
          ) : (
            filtered.map(opt => {
              const selected = value.includes(opt);
              return (
                <div
                  key={opt}
                  style={optionStyle(selected)}
                  onClick={() => toggle(opt)}
                  onMouseEnter={e => (e.currentTarget.style.background = 'var(--bg-hover)')}
                  onMouseLeave={e => (e.currentTarget.style.background = selected ? 'var(--accent-muted, rgba(99,102,241,0.1))' : 'transparent')}
                >
                  {multi && (
                    <span style={checkboxStyle(selected)}>
                      {selected && <span style={{ fontSize: 10 }}>&#10003;</span>}
                    </span>
                  )}
                  <span style={{ fontSize: 13 }}>{opt}</span>
                </div>
              );
            })
          )}
        </div>
      )}
    </div>
  );
}

const wrapStyle: CSSProperties = {
  position: 'relative',
};

const triggerStyle = (disabled?: boolean): CSSProperties => ({
  display: 'flex',
  alignItems: 'center',
  gap: 4,
  cursor: disabled ? 'not-allowed' : 'pointer',
  flexWrap: 'wrap',
  minHeight: 38,
  padding: '4px 12px',
  opacity: disabled ? 0.5 : 1,
});

const inputStyle: CSSProperties = {
  flex: 1,
  minWidth: 60,
  background: 'transparent',
  border: 'none',
  outline: 'none',
  color: 'var(--text-primary)',
  fontSize: 13,
  fontFamily: 'inherit',
  padding: '5px 0',
};

const tagsStyle: CSSProperties = {
  display: 'flex',
  flexWrap: 'wrap',
  gap: 4,
};

const tagStyle: CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 4,
  padding: '2px 8px',
  fontSize: 12,
  background: 'var(--accent-muted, rgba(99,102,241,0.15))',
  color: 'var(--accent)',
  borderRadius: 'var(--radius-sm)',
  lineHeight: '18px',
};

const tagRemoveStyle: CSSProperties = {
  cursor: 'pointer',
  fontSize: 14,
  lineHeight: 1,
  opacity: 0.7,
};

const dropdownStyle: CSSProperties = {
  position: 'absolute',
  top: '100%',
  left: 0,
  right: 0,
  marginTop: 4,
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 'var(--radius-md)',
  maxHeight: 200,
  overflowY: 'auto',
  zIndex: 1000,
  boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
};

const emptyStyle: CSSProperties = {
  padding: '12px 16px',
  fontSize: 13,
  color: 'var(--text-tertiary)',
  textAlign: 'center',
};

const optionStyle = (selected: boolean): CSSProperties => ({
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  padding: '8px 12px',
  cursor: 'pointer',
  background: selected ? 'var(--accent-muted, rgba(99,102,241,0.1))' : 'transparent',
  transition: 'background 0.1s',
});

const checkboxStyle = (checked: boolean): CSSProperties => ({
  width: 16,
  height: 16,
  borderRadius: 3,
  border: checked ? '1px solid var(--accent)' : '1px solid var(--border)',
  background: checked ? 'var(--accent)' : 'transparent',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  color: '#fff',
  flexShrink: 0,
  fontSize: 10,
});
