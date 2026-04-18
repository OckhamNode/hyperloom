import { useState, useCallback } from 'react';
import { ReactFlowProvider } from '@xyflow/react';
import { useHyperloom, type DebugEvent } from './hooks/useHyperloom';
import SwarmGraph from './components/SwarmGraph';
import TimeTravelSlider from './components/TimeTravelSlider';
import NodeInspector from './components/NodeInspector';
import './index.css';

export default function App() {
  const { visibleEvents, eventCount, currentIndex, isLive, scrubTo, goLive } = useHyperloom();
  const [selectedEvent, setSelectedEvent] = useState<DebugEvent | null>(null);

  const handleSelectEvent = useCallback((event: DebugEvent) => {
    setSelectedEvent(event);
  }, []);

  const handleCloseInspector = useCallback(() => {
    setSelectedEvent(null);
  }, []);

  return (
    <div style={{ width: '100%', height: '100%', position: 'relative' }}>
      {/* Header bar */}
      <div
        style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          height: '48px',
          background: 'var(--bg-surface)',
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 20px',
          zIndex: 100,
          backdropFilter: 'blur(12px)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '16px' }}>🌌</span>
          <span style={{ fontSize: '13px', fontWeight: 700, color: 'var(--text-primary)', letterSpacing: '-0.02em' }}>
            Hyperloom
          </span>
          <span style={{
            fontSize: '10px',
            fontWeight: 500,
            color: 'var(--accent)',
            background: 'rgba(99,102,241,0.12)',
            padding: '2px 8px',
            borderRadius: '4px',
          }}>
            Time-Travel Debugger
          </span>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
          <span style={{ fontSize: '10px', color: 'var(--text-muted)' }}>
            {eventCount} events ingested
          </span>
          <div style={{
            width: 8, height: 8, borderRadius: '50%',
            background: isLive ? 'var(--success)' : 'var(--warning)',
            boxShadow: isLive ? '0 0 8px var(--success-glow)' : '0 0 8px rgba(245,158,11,0.3)',
            animation: isLive ? 'commit-pulse 2s ease-in-out infinite' : undefined,
          }} />
        </div>
      </div>

      {/* Main graph area */}
      <div style={{ position: 'absolute', top: '48px', bottom: '56px', left: 0, right: selectedEvent ? '380px' : 0, transition: 'right 0.3s ease' }}>
        <ReactFlowProvider>
          <SwarmGraph events={visibleEvents} onSelectEvent={handleSelectEvent} />
        </ReactFlowProvider>
      </div>

      {/* Side panel */}
      <NodeInspector event={selectedEvent} onClose={handleCloseInspector} />

      {/* Bottom timeline */}
      <TimeTravelSlider
        max={eventCount}
        current={currentIndex}
        isLive={isLive}
        onScrub={scrubTo}
        onGoLive={goLive}
        events={visibleEvents}
      />
    </div>
  );
}
