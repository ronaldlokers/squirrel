import type { Capture, CaptureSink, Outcome } from "../capture/capture.js";
import type { CampfireConfig } from "../config.js";
import type { HttpMount, Stop, Transport } from "./transport.js";

export const NAME = "campfire";

/**
 * The Campfire adapter, and every quirk below stays inside this file.
 *
 * Campfire treats the HTTP **response body** as the bot's reply. A 200 with a
 * Content-Type is posted into the room; a non-200 that still carries one is
 * uploaded as an attachment; no Content-Type at all is the only silence. There
 * is a hard seven-second deadline after which Campfire posts its own failure
 * notice over the top. None of that is ours to choose.
 *
 * There is also no signature, no shared secret, and no timestamp. Identity of
 * the caller is the callback URL, guarded by NetworkPolicy; the clock is ours.
 */
interface CampfirePayload {
  readonly user?: { readonly id?: number | string };
  readonly room?: { readonly id?: number | string };
  readonly message?: {
    readonly id?: number | string;
    readonly body?: { readonly plain?: string };
  };
}

function identifier(value: number | string | undefined): string | null {
  return value === undefined || value === null ? null : String(value);
}

/**
 * Fails open. An envelope we cannot read still becomes a capture with null
 * ids, because the alternative — dropping it — is how a payload change
 * upstream silently empties the inbox for a week.
 */
export function captureFrom(body: string, receivedAt: Date): Capture {
  let payload: CampfirePayload;
  try {
    const parsed: unknown = JSON.parse(body);
    // JSON.parse("null") succeeds and yields JS null. typeof null ===
    // "object", so a naive cast would sail through and blow up on
    // payload.message below — guard it into an empty payload instead.
    payload = (typeof parsed === "object" && parsed !== null ? parsed : {}) as CampfirePayload;
  } catch {
    return {
      transport: NAME,
      externalId: null,
      conversationId: null,
      senderId: null,
      text: body,
      receivedAt,
      payload: { unparseable: body },
    };
  }

  return {
    transport: NAME,
    externalId: identifier(payload.message?.id),
    conversationId: identifier(payload.room?.id),
    senderId: identifier(payload.user?.id),
    text: payload.message?.body?.plain ?? "",
    receivedAt,
    payload,
  };
}

export function respond(outcome: Outcome): Response {
  const text = { "Content-Type": "text/plain; charset=utf-8" };
  switch (outcome) {
    case "stored":
      return new Response("🐿️", { status: 200, headers: text });
    case "ignored":
      // No body and no Content-Type: the one path that says nothing at all.
      return new Response(null, { status: 200 });
    case "failed":
      // Still a 200. A non-200 carrying a Content-Type becomes an attachment.
      return new Response("⚠️ couldn't save that — please resend", { status: 200, headers: text });
  }
}

/**
 * Outbound, used when the system initiates rather than answers.
 *
 * Reusing `room.path` from a stored payload would need no credential at all,
 * since that path already embeds a bot key. It is rejected on purpose: outbound
 * would then only reach rooms Squirrel had recently heard from, and a morning
 * nudge would depend on the capture history. That works in testing and fails on
 * a quiet Monday.
 */
function sendVia(baseUrl: string, botKey: string) {
  const base = baseUrl.replace(/\/+$/, "");
  return async (conversationId: string, text: string): Promise<void> => {
    const response = await fetch(`${base}/rooms/${conversationId}/${botKey}/messages`, {
      method: "POST",
      headers: { "Content-Type": "text/plain; charset=utf-8" },
      body: text,
    });
    if (!response.ok) {
      throw new Error(`campfire: send failed with ${response.status}`);
    }
  };
}

export function createCampfireTransport(config: CampfireConfig): Transport {
  return {
    name: NAME,

    start(sink: CaptureSink, http: HttpMount): Promise<Stop> {
      http.post(config.path, async (request: Request) => {
        let outcome: Outcome;
        try {
          outcome = await sink.accept(captureFrom(await request.text(), new Date()));
        } catch (error) {
          // Nothing may escape. An unhandled throw is a 500, and a 500 with a
          // Content-Type uploads an error page into the room.
          process.stderr.write(`campfire: handler failed: ${String(error)}\n`);
          outcome = "failed";
        }
        return respond(outcome);
      });
      return Promise.resolve(() => Promise.resolve());
    },

    // Null unless a bot key is configured. Half-working outbound would fail at
    // exactly the moment it is needed; absent outbound is at least honest.
    send:
      config.baseUrl !== null && config.botKey !== null
        ? sendVia(config.baseUrl, config.botKey)
        : null,
  };
}
