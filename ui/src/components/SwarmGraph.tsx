import { useMemo, useCallback } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  type Node,
  type Edge,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import GhostNodeComponent from './GhostNode';
import type { DebugEvent } from '../hooks/useHyperloom';

const nodeTypes = { ghost: GhostNodeComponent };

interface SwarmGraphProps {
  events: DebugEvent[];
  onSelectEvent: (event: DebugEvent) => void;
}

export default function SwarmGraph({ events, onSelectEvent }: SwarmGraphProps) {
  const { nodes, edges } = useMemo(() => {
    const nodeMap = new Map<string, Node>();
    const edgeSet = new Set<string>();
    const edgeList: Edge[] = [];

    // Track latest event per path for node state
    const latestByPath = new Map<string, DebugEvent>();

    for (const evt of events) {
      if (!evt.path) continue;
      latestByPath.set(evt.path, evt);
    }

    // Build nodes from unique paths
    const paths = Array.from(latestByPath.keys());
    const sortedPaths = paths.sort();

    // Layout: tree-like arrangement based on path depth
    const pathIndex = new Map<string, number>();
    const depthCounts = new Map<number, number>();

    for (const path of sortedPaths) {
      const segments = path.split('/').filter(Boolean);
      const depth = segments.length;
      const count = depthCounts.get(depth) || 0;
      depthCounts.set(depth, count + 1);
      pathIndex.set(path, count);
    }

    for (const path of sortedPaths) {
      const evt = latestByPath.get(path)!;
      const segments = path.split('/').filter(Boolean);
      const depth = segments.length;
      const idx = pathIndex.get(path)!;
      const totalAtDepth = depthCounts.get(depth)!;

      // Spread nodes horizontally, offset to center
      const xSpacing = 240;
      const ySpacing = 160;
      const xOffset = -(totalAtDepth - 1) * xSpacing / 2;

      const nodeId = path;
      nodeMap.set(nodeId, {
        id: nodeId,
        type: 'ghost',
        position: {
          x: xOffset + idx * xSpacing + (depth % 2 === 0 ? 0 : 40),
          y: depth * ySpacing,
        },
        data: {
          label: segments[segments.length - 1],
          path,
          agentId: evt.agent_id,
          lastEvent: evt,
          onClick: onSelectEvent,
        },
      });

      // Create edge from parent path to this path
      if (segments.length > 1) {
        const parentPath = '/' + segments.slice(0, -1).join('/');
        const edgeId = `${parentPath}->${nodeId}`;
        if (!edgeSet.has(edgeId)) {
          edgeSet.add(edgeId);
          edgeList.push({
            id: edgeId,
            source: parentPath,
            target: nodeId,
            label: evt.agent_id.split('-')[0],
            style: { stroke: '#3F3F46', strokeWidth: 1.5 },
            labelStyle: { fill: '#71717A', fontSize: 9, fontWeight: 500 },
            labelBgStyle: { fill: '#161618', fillOpacity: 0.9 },
            animated: evt.type === 'staged',
          });
        }
      }
    }

    // Ensure parent nodes exist even if they haven't been directly written to
    for (const path of sortedPaths) {
      const segments = path.split('/').filter(Boolean);
      for (let i = 1; i < segments.length; i++) {
        const parentPath = '/' + segments.slice(0, i).join('/');
        if (!nodeMap.has(parentPath)) {
          const pDepth = i;
          const pCount = depthCounts.get(pDepth) || 0;
          depthCounts.set(pDepth, pCount + 1);

          const xSpacing = 240;
          const ySpacing = 160;
          const xOffset = -(pCount) * xSpacing / 2;

          nodeMap.set(parentPath, {
            id: parentPath,
            type: 'ghost',
            position: { x: xOffset + pCount * xSpacing, y: pDepth * ySpacing },
            data: {
              label: segments[i - 1],
              path: parentPath,
              agentId: 'system',
              lastEvent: { type: 'staged', agent_id: 'system', tx_id: '', path: parentPath, op: '', hash: '', timestamp: Date.now() },
              onClick: onSelectEvent,
            },
          });
        }
      }
    }

    return { nodes: Array.from(nodeMap.values()), edges: edgeList };
  }, [events, onSelectEvent]);

  const miniMapNodeColor = useCallback((node: Node) => {
    const data = node.data as { lastEvent?: DebugEvent };
    if (data?.lastEvent?.type === 'committed') return '#22C55E';
    if (data?.lastEvent?.type === 'reverted') return '#EF4444';
    return '#6366F1';
  }, []);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={nodeTypes}
      fitView
      minZoom={0.3}
      maxZoom={2}
      proOptions={{ hideAttribution: true }}
      style={{ background: 'var(--bg-primary)' }}
    >
      <Background color="#27272A" gap={20} size={1} />
      <Controls showInteractive={false} />
      <MiniMap
        nodeColor={miniMapNodeColor}
        maskColor="rgba(10, 10, 11, 0.8)"
        style={{ border: '1px solid #27272A', borderRadius: 8 }}
      />
    </ReactFlow>
  );
}
