import type { Capture } from "./capture.js";

export interface Allow {
  readonly transport: string;
  readonly conversationId: string;
  readonly senderId: string;
}

export type Verdict = "accept" | "ignore";

/**
 * The guard, and it fails in two directions on purpose.
 *
 * Understood the envelope and it is the wrong room or the wrong person: fail
 * closed, because the system genuinely was not addressed.
 *
 * Could not understand the envelope at all: fail open. A payload shape change
 * upstream would otherwise drop every capture silently, with the bot still
 * answering cheerfully. Junk rows in a table nobody reads yet is the cheaper
 * mistake by a wide margin.
 */
export function decide(capture: Capture, allows: readonly Allow[]): Verdict {
  if (capture.conversationId === null || capture.senderId === null) return "accept";

  const matched = allows.some(
    (allow) =>
      allow.transport === capture.transport &&
      allow.conversationId === capture.conversationId &&
      allow.senderId === capture.senderId,
  );
  return matched ? "accept" : "ignore";
}
