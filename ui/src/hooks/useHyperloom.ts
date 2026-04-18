import { useRef, useState, useCallback, useEffect } from 'react';
import { generateMockEvent } from '../lib/mockData';

export interface DebugEvent {
  type: 'staged' | 'committed' | 'reverted';
  agent_id: string;
  tx_id: string;
  path: string;
  op: string;
  value?: string;
  hash: string;
  timestamp: number;
}

const MOCK_MODE = true; // Toggle false when connecting to live Go backend

export function useHyperloom() {
  const eventsRef = useRef<DebugEvent[]>([]);
  const [eventCount, setEventCount] = useState(0);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [isLive, setIsLive] = useState(true);
  const wsRef = useRef<WebSocket | null>(null);

  // Batch UI updates via requestAnimationFrame to maintain 60fps
  const pendingUpdate = useRef(false);
  const flushUpdate = useCallback(() => {
    if (!pendingUpdate.current) {
      pendingUpdate.current = true;
      requestAnimationFrame(() => {
        const len = eventsRef.current.length;
        setEventCount(len);
        setCurrentIndex((prev) => (isLive ? len : prev));
        pendingUpdate.current = false;
      });
    }
  }, [isLive]);

  const addEvent = useCallback(
    (evt: DebugEvent) => {
      eventsRef.current.push(evt);
      flushUpdate();
    },
    [flushUpdate]
  );

  // Connect to live backend OR run mock simulation
  useEffect(() => {
    if (MOCK_MODE) {
      const interval = setInterval(() => {
        addEvent(generateMockEvent());
      }, 400 + Math.random() * 600); // 1-3 events/sec in mock
      return () => clearInterval(interval);
    }

    // Live WebSocket mode
    const ws = new WebSocket('ws://localhost:8080/events');
    wsRef.current = ws;

    ws.onmessage = (msg) => {
      try {
        const evt: DebugEvent = JSON.parse(msg.data);
        addEvent(evt);
      } catch {
        // Skip malformed messages
      }
    };

    ws.onclose = () => {
      // Attempt reconnect after 2s
      setTimeout(() => {
        wsRef.current = new WebSocket('ws://localhost:8080/events');
      }, 2000);
    };

    return () => ws.close();
  }, [addEvent]);

  const scrubTo = useCallback((index: number) => {
    setIsLive(false);
    setCurrentIndex(index);
  }, []);

  const goLive = useCallback(() => {
    setIsLive(true);
    setCurrentIndex(eventsRef.current.length);
  }, []);

  // Derive visible events from timeline cursor
  const visibleEvents = eventsRef.current.slice(0, currentIndex);

  return {
    events: eventsRef.current,
    visibleEvents,
    eventCount,
    currentIndex,
    isLive,
    scrubTo,
    goLive,
  };
}
