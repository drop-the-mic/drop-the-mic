/**
 * HelpTooltip — 동그라미 안에 물음표(?) 아이콘과 호버/클릭 시 설명을 보여주는 컴포넌트.
 *
 * @example 기본 사용 (인라인 텍스트 옆에 배치)
 * ```tsx
 * <span>Verdict <HelpTooltip text="The verification result determined by the LLM." /></span>
 * ```
 *
 * @example Table header
 * ```tsx
 * <th>Severity <HelpTooltip text="Importance level defined by the user." /></th>
 * ```
 *
 * @example Multi-line description
 * ```tsx
 * <HelpTooltip text={"PASS: No issues\nWARN: Warning\nFAIL: Failure"} />
 * ```
 *
 * Props:
 * - `text` (string, required) — Tooltip description. Supports \n for line breaks.
 * - `size` (number, optional, default: 14) — Icon size in px.
 *
 * Behavior:
 * - Shows tooltip on mouse hover
 * - Toggles on click (for mobile)
 * - Closes on outside click
 *
 * Styling:
 * - Uses existing CSS variables (--bg-card, --border, --text-secondary, --shadow-lg, etc.)
 * - No external dependencies
 */

import { useState, useRef, useEffect } from 'react';
import { IconHelp } from './Icons';

export function HelpTooltip({ text, size = 14 }: { text: string; size?: number }) {
  const [show, setShow] = useState(false);
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null);
  const ref = useRef<HTMLSpanElement>(null);

  // Close on outside click
  useEffect(() => {
    if (!show) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setShow(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [show]);

  const updatePos = () => {
    if (!ref.current) return;
    const rect = ref.current.getBoundingClientRect();
    const spaceAbove = rect.top;
    const tooltipHeight = 80; // approximate
    // Show below if not enough space above
    if (spaceAbove < tooltipHeight) {
      setPos({ top: rect.bottom + 6, left: rect.left + rect.width / 2 });
    } else {
      setPos({ top: rect.top - 6, left: rect.left + rect.width / 2 });
    }
  };

  const handleShow = () => { updatePos(); setShow(true); };

  return (
    <span
      ref={ref}
      style={{ display: 'inline-flex', alignItems: 'center', cursor: 'help', verticalAlign: 'middle', marginLeft: 4 }}
      onMouseEnter={handleShow}
      onMouseLeave={() => setShow(false)}
      onClick={(e) => { e.stopPropagation(); if (show) { setShow(false); } else { handleShow(); } }}
    >
      <IconHelp size={size} />
      {show && pos && (
        <span style={{
          position: 'fixed',
          top: pos.top < (ref.current?.getBoundingClientRect().top ?? 0) ? pos.top : undefined,
          bottom: pos.top < (ref.current?.getBoundingClientRect().top ?? 0) ? undefined : `${window.innerHeight - pos.top}px`,
          left: Math.min(Math.max(pos.left, 170), window.innerWidth - 170),
          transform: 'translateX(-50%)',
          padding: '8px 12px',
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 'var(--radius-md)',
          boxShadow: 'var(--shadow-lg)',
          fontSize: 12,
          lineHeight: 1.5,
          color: 'var(--text-secondary)',
          whiteSpace: 'pre-line',
          minWidth: 200,
          maxWidth: 320,
          zIndex: 10000,
          pointerEvents: 'none',
        }}>
          {text}
        </span>
      )}
    </span>
  );
}
