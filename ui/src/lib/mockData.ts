import type { DebugEvent } from '../hooks/useHyperloom';

let counter = 0;

const agents = ['claude-3-opus', 'gpt-4o', 'gemini-2.5', 'rovo-agent', 'crew-worker-1'];
const paths = [
  '/memory/session_1/summary',
  '/memory/session_1/intent',
  '/context/project_alpha/codebase',
  '/context/project_alpha/security_scan',
  '/agents/researcher/findings',
  '/agents/reviewer/feedback',
  '/pipeline/stage_3/output',
  '/pipeline/stage_4/validation',
];

const values = [
  '"Analyzed 142 API endpoints for vulnerabilities"',
  '"Code review complete — 3 critical issues found"',
  '["security_patch_applied", "regression_test_pass"]',
  '{"status": "processing", "confidence": 0.94}',
  '"Generated architectural diagram for service mesh"',
  '{"tokens_used": 4200, "latency_ms": 340}',
];

function pick<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

export function generateMockEvent(): DebugEvent {
  counter++;
  const agent = pick(agents);
  const txId = `tx_${agent.replace(/[^a-z0-9]/g, '_')}_${Math.floor(counter / 3)}`;

  // Every 8th event is a revert (hallucination)
  if (counter % 8 === 0) {
    return {
      type: 'reverted',
      agent_id: agent,
      tx_id: txId,
      path: pick(paths),
      op: 'SET',
      value: undefined,
      hash: '',
      timestamp: Date.now(),
    };
  }

  // Every 3rd event is a commit
  if (counter % 3 === 0) {
    const hash = Array.from({ length: 8 }, () =>
      Math.floor(Math.random() * 16).toString(16)
    ).join('');

    return {
      type: 'committed',
      agent_id: agent,
      tx_id: txId,
      path: pick(paths),
      op: pick(['SET', 'APPEND']),
      value: pick(values),
      hash,
      timestamp: Date.now(),
    };
  }

  // Default: staged
  return {
    type: 'staged',
    agent_id: agent,
    tx_id: txId,
    path: pick(paths),
    op: pick(['SET', 'APPEND']),
    value: pick(values),
    hash: '',
    timestamp: Date.now(),
  };
}
