import type { CaptureSink } from "../capture/capture.js";

export type WebHandler = (request: Request) => Promise<Response>;

/** What a transport may register on the shared server. A polling transport ignores it. */
export interface HttpMount {
  post(path: string, handler: WebHandler): void;
}

export type Stop = () => Promise<void>;

export interface Transport {
  readonly name: string;
  /** Begin receiving. `http` may be ignored by a transport that polls. */
  start(sink: CaptureSink, http: HttpMount): Promise<Stop>;
  /**
   * Send a message the system initiated, rather than one it is answering.
   * Null when this transport cannot initiate — a bot that can only reply has
   * to be able to say so rather than fail at the moment it matters.
   */
  readonly send: ((conversationId: string, text: string) => Promise<void>) | null;
}
