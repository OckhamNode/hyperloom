import { motion, AnimatePresence } from 'framer-motion';
import type { DebugEvent } from '../hooks/useHyperloom';

interface NodeInspectorProps {
  event: DebugEvent | null;
  onClose: () => void;
}

export default function NodeInspector({ event, onClose }: NodeInspectorProps) {
  if (!event) return null;

  const rows = [
    { label: 'Path', value: event.path },
    { label: 'Agent', value: event.agent_id },
    { label: 'Transaction', value: event.tx_id },
    { label: 'Operation', value: event.op },
    { label: 'Status', value: event.type },
    { label: 'Hash', value: event.hash || '—' },
    { label: 'Timestamp', value: new Date(event.timestamp).toISOString() },
  ];

  const statusColor =
    event.type === 'committed'
      ? 'var(--success)'
      : event.type === 'reverted'
        ? 'var(--danger)'
        : 'var(--accent)';

  return (
    <AnimatePresence>
      <motion.div
        key="inspector"
        initial={{ x: 400, opacity: 0 }}
        animate={{ x: 0, opacity: 1 }}
        exit={{ x: 400, opacity: 0 }}
        transition={{ type: 'spring', stiffness: 300, damping: 30 }}
        style={{
          position: 'fixed',
          top: 0,
          right: 0,
          bottom: 56,
          width: '380px',
          background: 'var(--bg-surface)',
          borderLeft: '1px solid var(--border)',
          display: 'flex',
          flexDirection: 'column',
          zIndex: 200,
          boxShadow: '-8px 0 30px rgba(0,0,0,0.4)',
        }}
      >
        {/* Header */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            padding: '16px 20px',
            borderBottom: '1px solid var(--border)',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <div style={{ width: 10, height: 10, borderRadius: '50%', background: statusColor, boxShadow: `0 0 8px ${statusColor}` }} />
            <span style={{ fontSize: '13px', fontWeight: 600, color: 'var(--text-primary)' }}>
              Node Inspector
            </span>
          </div>
          <button
            onClick={onClose}
            style={{
              background: 'var(--bg-elevated)',
              border: '1px solid var(--border)',
              borderRadius: '6px',
              color: 'var(--text-secondary)',
              cursor: 'pointer',
              padding: '4px 10px',
              fontSize: '11px',
              fontWeight: 500,
            }}
          >
            ✕
          </button>
        </div>

        {/* Metadata rows */}
        <div style={{ padding: '16px 20px', flex: 1, overflowY: 'auto' }}>
          {rows.map(({ label, value }) => (
            <div
              key={label}
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                padding: '10px 0',
                borderBottom: '1px solid var(--border)',
              }}
            >
              <span style={{ fontSize: '11px', color: 'var(--text-muted)', fontWeight: 500, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
                {label}
              </span>
              <span
                style={{
                  fontSize: '11px',
                  color: label === 'Status' ? statusColor : 'var(--text-primary)',
                  fontWeight: label === 'Status' ? 700 : 400,
                  fontFamily: label === 'Hash' || label === 'Transaction' ? 'monospace' : 'inherit',
                  maxWidth: '200px',
                  textAlign: 'right',
                  wordBreak: 'break-all',
                }}
              >
                {value}
              </span>
            </div>
          ))}

          {/* Raw Value */}
          <div style={{ marginTop: '20px' }}>
            <span style={{ fontSize: '11px', color: 'var(--text-muted)', fontWeight: 500, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              Raw Context Diff
            </span>
            <pre
              style={{
                marginTop: '8px',
                background: 'var(--bg-primary)',
                border: '1px solid var(--border)',
                borderRadius: '8px',
                padding: '14px',
                fontSize: '11px',
                color: 'var(--accent)',
                fontFamily: '"JetBrains Mono", "Fira Code", monospace',
                overflowX: 'auto',
                whiteSpace: 'pre-wrap',
                lineHeight: 1.6,
              }}
            >
              {event.value
                ? (() => {
                    try {
                      return JSON.stringify(JSON.parse(event.value), null, 2);
                    } catch {
                      return event.value;
                    }
                  })()
                : 'null (tombstone / reverted)'}
            </pre>
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  );
}
