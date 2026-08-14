/**
 * What every transport must produce. Nothing here is Campfire-shaped.
 *
 * The nullable fields are nullable for one reason: a transport that cannot
 * parse an envelope must still be able to hand the message over. See the
 * fail-open rule in `policy.ts`.
 */
export interface Capture {
  readonly transport: string;
  readonly externalId: string | null;
  readonly conversationId: string | null;
  readonly senderId: string | null;
  /** Verbatim. Never trimmed, lowercased, or otherwise interpreted. */
  readonly text: string;
  /** Our clock. Campfire sends no timestamp and the next transport may not either. */
  readonly receivedAt: Date;
  /** The original message, untouched, for anything not worth a column. */
  readonly payload: unknown;
}

export type Outcome = "stored" | "ignored" | "failed";

export interface CaptureSink {
  /** Resolves only once the capture is durable. */
  accept(capture: Capture): Promise<Outcome>;
}
