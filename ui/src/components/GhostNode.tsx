import { memo, useEffect, useState } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import { motion, AnimatePresence } from 'framer-motion';
import type { DebugEvent } from '../hooks/useHyperloom';

export interface GhostNodeData {
  label: string;
  path: string;
  agentId: string;
  lastEvent: DebugEvent;
  onClick: (event: DebugEvent) => void;
}

const agentColors: Record<string, string> = {
  'claude-3-opus': '#D97706',
  'gpt-4o': '#10B981',
  'gemini-2.5': '#6366F1',
  'rovo-agent': '#EC4899',
  'crew-worker-1': '#14B8A6',
};

function GhostNodeComponent({ data }: NodeProps) {
  const nodeData = data as unknown as GhostNodeData;
  const [status, setStatus] = useState<'staged' | 'committed' | 'reverted'>('staged');
  const [isShattered, setIsShattered] = useState(false);

  const agentColor = agentColors[nodeData.agentId] || '#6366F1';

  useEffect(() => {
    if (!nodeData.lastEvent) return;
    setStatus(nodeData.lastEvent.type);

    if (nodeData.lastEvent.type === 'reverted') {
      setIsShattered(true);
      const timer = setTimeout(() => setIsShattered(false), 900);
      return () => clearTimeout(timer);
    }
  }, [nodeData.lastEvent]);

  const borderColor =
    status === 'committed'
      ? 'var(--success)'
      : status === 'reverted'
        ? 'var(--danger)'
        : agentColor;

  const animationClass =
    status === 'committed'
      ? 'commit-pulse-active'
      : status === 'reverted'
        ? 'ghost-shatter-active'
        : '';

  return (
    <AnimatePresence>
      {!isShattered && (
        <motion.div
          initial={{ scale: 0.8, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          exit={{ scale: 0.6, opacity: 0, filter: 'blur(6px)' }}
          transition={{ duration: 0.35, ease: 'easeOut' }}
          onClick={() => nodeData.onClick?.(nodeData.lastEvent)}
          className={animationClass}
          style={{
            background: 'var(--bg-surface)',
            border: `1.5px solid ${borderColor}`,
            borderRadius: '10px',
            padding: '12px 16px',
            minWidth: '180px',
            cursor: 'pointer',
            transition: 'border-color 0.2s, box-shadow 0.3s',
            boxShadow: status === 'committed'
              ? `0 0 18px var(--success-glow)`
              : status === 'reverted'
                ? `0 0 20px var(--danger-glow)`
                : `0 0 10px ${agentColor}22`,
            animation: status === 'reverted' ? 'ghost-shatter 0.8s ease-out forwards' : undefined,
          }}
        >
          <Handle type="target" position={Position.Top} style={{ background: borderColor, width: 8, height: 8, border: 'none' }} />

          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
            <div style={{
              width: 8, height: 8, borderRadius: '50%', background: agentColor,
              boxShadow: `0 0 6px ${agentColor}`,
            }} />
            <span style={{ fontSize: '10px', color: 'var(--text-muted)', fontWeight: 500, textTransform: 'uppercase', letterSpacing: '0.05em' }}>
              {nodeData.agentId}
            </span>
          </div>

          <div style={{ fontSize: '12px', fontWeight: 600, color: 'var(--text-primary)', marginBottom: '4px', wordBreak: 'break-all' }}>
            {nodeData.label}
          </div>

          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{
              fontSize: '9px',
              fontWeight: 600,
              textTransform: 'uppercase',
              letterSpacing: '0.08em',
              padding: '2px 6px',
              borderRadius: '4px',
              background: status === 'committed' ? 'rgba(34,197,94,0.15)' : status === 'reverted' ? 'rgba(239,68,68,0.15)' : 'rgba(99,102,241,0.15)',
              color: status === 'committed' ? 'var(--success)' : status === 'reverted' ? 'var(--danger)' : 'var(--accent)',
            }}>
              {status}
            </span>
            <span style={{ fontSize: '9px', color: 'var(--text-muted)' }}>
              {nodeData.lastEvent?.op}
            </span>
          </div>

          <Handle type="source" position={Position.Bottom} style={{ background: borderColor, width: 8, height: 8, border: 'none' }} />
        </motion.div>
      )}
    </AnimatePresence>
  );
}

export default memo(GhostNodeComponent);
