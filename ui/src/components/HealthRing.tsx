interface HealthRingProps {
  pass: number;
  warn: number;
  fail: number;
  size?: number;
}

export function HealthRing({ pass, warn, fail, size = 120 }: HealthRingProps) {
  const total = pass + warn + fail;
  if (total === 0) {
    return (
      <div style={{ width: size, height: size, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <svg width={size} height={size} viewBox="0 0 120 120">
          <circle cx="60" cy="60" r="50" fill="none" stroke="var(--border)" strokeWidth="10" />
          <text x="60" y="60" textAnchor="middle" dominantBaseline="central" fill="var(--text-tertiary)" fontSize="14" fontWeight="500">N/A</text>
        </svg>
      </div>
    );
  }

  const radius = 50;
  const circumference = 2 * Math.PI * radius;
  const passLen = (pass / total) * circumference;
  const warnLen = (warn / total) * circumference;
  const failLen = (fail / total) * circumference;
  const pct = Math.round((pass / total) * 100);

  let offset = -circumference / 4; // start from top
  const segments = [
    { len: passLen, color: 'var(--success)', offset },
    { len: warnLen, color: 'var(--warning)', offset: offset + passLen },
    { len: failLen, color: 'var(--danger)', offset: offset + passLen + warnLen },
  ].filter(s => s.len > 0);

  return (
    <div style={{ width: size, height: size, position: 'relative' }}>
      <svg width={size} height={size} viewBox="0 0 120 120">
        <circle cx="60" cy="60" r={radius} fill="none" stroke="var(--border)" strokeWidth="10" />
        {segments.map((seg, i) => (
          <circle
            key={i} cx="60" cy="60" r={radius}
            fill="none" stroke={seg.color} strokeWidth="10"
            strokeDasharray={`${seg.len} ${circumference - seg.len}`}
            strokeDashoffset={-seg.offset}
            strokeLinecap="round"
            style={{ transition: 'all 0.5s ease' }}
          />
        ))}
        <text x="60" y="55" textAnchor="middle" dominantBaseline="central" fill="var(--text-primary)" fontSize="24" fontWeight="700">{pct}%</text>
        <text x="60" y="74" textAnchor="middle" dominantBaseline="central" fill="var(--text-tertiary)" fontSize="10" fontWeight="500">HEALTHY</text>
      </svg>
    </div>
  );
}
