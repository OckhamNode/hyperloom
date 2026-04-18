import { useCallback } from 'react';

interface TimeTravelSliderProps {
  max: number;
  current: number;
  isLive: boolean;
  onScrub: (index: number) => void;
  onGoLive: () => void;
  events: { type: string; timestamp: number; agent_id: string }[];
}

export default function TimeTravelSlider({ max, current, isLive, onScrub, onGoLive, events }: TimeTravelSliderProps) {
  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      onScrub(parseInt(e.target.value, 10));
    },
    [onScrub]
  );

  // Count events by type for the status bar
  const committed = events.filter((e) => e.type === 'committed').length;
  const reverted = events.filter((e) => e.type === 'reverted').length;
  const staged = events.filter((e) => e.type === 'staged').length;

  return (
    <div
      style={{
        position: 'fixed',
        bottom: 0,
        left: 0,
        right: 0,
        background: 'var(--bg-surface)',
        borderTop: '1px solid var(--border)',
        padding: '12px 24px',
        display: 'flex',
        alignItems: 'center',
        gap: '16px',
        zIndex: 100,
        backdropFilter: 'blur(12px)',
      }}
    >
      {/* Time label */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', minWidth: '160px' }}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2" strokeLinecap="round">
          <circle cx="12" cy="12" r="10" />
          <polyline points="12 6 12 12 16 14" />
        </svg>
        <span style={{ fontSize: '11px', color: 'var(--text-secondary)', fontWeight: 500 }}>
          T-{current} / {max}
        </span>
      </div>

      {/* The slider */}
      <input
        type="range"
        min={0}
        max={max}
        value={current}
        onChange={handleChange}
        style={{
          flex: 1,
          height: '4px',
          appearance: 'none',
          background: `linear-gradient(to right, var(--accent) 0%, var(--accent) ${max > 0 ? (current / max) * 100 : 0}%, var(--border) ${max > 0 ? (current / max) * 100 : 0}%, var(--border) 100%)`,
          borderRadius: '4px',
          outline: 'none',
          cursor: 'pointer',
        }}
      />

      {/* Live button */}
      <button
        onClick={onGoLive}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '6px',
          padding: '6px 14px',
          fontSize: '11px',
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.06em',
          color: isLive ? '#0A0A0B' : 'var(--text-secondary)',
          background: isLive ? 'var(--success)' : 'var(--bg-elevated)',
          border: `1px solid ${isLive ? 'var(--success)' : 'var(--border)'}`,
          borderRadius: '6px',
          cursor: 'pointer',
          transition: 'all 0.2s',
        }}
      >
        <div
          style={{
            width: 6,
            height: 6,
            borderRadius: '50%',
            background: isLive ? '#0A0A0B' : 'var(--text-muted)',
            animation: isLive ? 'commit-pulse 1.5s ease-in-out infinite' : undefined,
          }}
        />
        LIVE
      </button>

      {/* Stats */}
      <div style={{ display: 'flex', gap: '12px', minWidth: '200px', justifyContent: 'flex-end' }}>
        <span style={{ fontSize: '10px', color: 'var(--success)', fontWeight: 600 }}>
          ● {committed} committed
        </span>
        <span style={{ fontSize: '10px', color: 'var(--accent)', fontWeight: 600 }}>
          ● {staged} staged
        </span>
        <span style={{ fontSize: '10px', color: 'var(--danger)', fontWeight: 600 }}>
          ● {reverted} reverted
        </span>
      </div>
    </div>
  );
}
