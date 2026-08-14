import type { Capture, CaptureSink, Outcome } from "./capture.js";
import { type Allow, decide } from "./policy.js";
import type { Spool } from "./spool.js";

export interface SinkHooks {
  readonly onError?: (error: unknown) => void;
  /** An ignored capture is silent in the room. It should not be silent in the logs. */
  readonly onIgnored?: (capture: Capture) => void;
}

/**
 * The only place a capture becomes durable.
 *
 * Note what is absent: Postgres. The request path reaches this function and no
 * further, which is what lets a database outage pass without a lost thought.
 */
export function createSink(
  spool: Spool,
  allows: readonly Allow[],
  hooks: SinkHooks = {},
): CaptureSink {
  return {
    async accept(capture: Capture): Promise<Outcome> {
      if (decide(capture, allows) === "ignore") {
        hooks.onIgnored?.(capture);
        return "ignored";
      }
      try {
        await spool.write(capture);
        return "stored";
      } catch (error) {
        hooks.onError?.(error);
        return "failed";
      }
    },
  };
}
