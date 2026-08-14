import { describe, expect, it, vi } from "vitest";

import type { Capture, CaptureSink, Outcome } from "../../src/capture/capture.js";
import type { CampfireConfig } from "../../src/config.js";
import { captureFrom, createCampfireTransport, respond } from "../../src/transports/campfire.js";
import type { HttpMount, WebHandler } from "../../src/transports/transport.js";

const AT = new Date("2026-08-14T09:31:04.512Z");

const PAYLOAD = {
  user: { id: 1, name: "Ronald" },
  room: { id: 7, name: "Squirrel", path: "/rooms/7/3-abc/messages" },
  message: {
    id: 42,
    body: { html: "<div>buy milk</div>", plain: "buy milk" },
    path: "/rooms/7/@42",
  },
};

const config: CampfireConfig = {
  path: "/transports/campfire",
  conversationId: "7",
  senderId: "1",
  baseUrl: null,
  botKey: null,
};

function mountOne(): { mount: HttpMount; handler: () => WebHandler } {
  let registered: WebHandler | undefined;
  return {
    mount: {
      post(_path, handler) {
        registered = handler;
      },
    },
    handler: () => {
      if (!registered) throw new Error("nothing mounted");
      return registered;
    },
  };
}

function sinkReturning(outcome: Outcome): CaptureSink & { seen: Capture[] } {
  const seen: Capture[] = [];
  return {
    seen,
    accept: (capture) => {
      seen.push(capture);
      return Promise.resolve(outcome);
    },
  };
}

describe("captureFrom", () => {
  it("maps every field of the documented payload", () => {
    expect(captureFrom(JSON.stringify(PAYLOAD), AT)).toEqual({
      transport: "campfire",
      externalId: "42",
      conversationId: "7",
      senderId: "1",
      text: "buy milk",
      receivedAt: AT,
      payload: PAYLOAD,
    });
  });

  it("keeps the text verbatim, including case and surrounding whitespace", () => {
    const payload = { ...PAYLOAD, message: { ...PAYLOAD.message, body: { plain: "  DONE  " } } };
    expect(captureFrom(JSON.stringify(payload), AT).text).toBe("  DONE  ");
  });

  it("fails open on an unparseable body, keeping the raw text", () => {
    const capture = captureFrom("not json at all", AT);
    expect(capture.externalId).toBeNull();
    expect(capture.conversationId).toBeNull();
    expect(capture.senderId).toBeNull();
    expect(capture.payload).toEqual({ unparseable: "not json at all" });
  });

  it("fails open on a payload whose shape changed underneath us", () => {
    const capture = captureFrom(JSON.stringify({ message: { id: 42 } }), AT);
    expect(capture.conversationId).toBeNull();
    expect(capture.senderId).toBeNull();
    expect(capture.externalId).toBe("42");
  });

  it("treats a missing plain body as an empty capture rather than an error", () => {
    expect(captureFrom(JSON.stringify({ ...PAYLOAD, message: { id: 42 } }), AT).text).toBe("");
  });
});

describe("respond", () => {
  it("posts a squirrel when the capture is stored", async () => {
    const response = respond("stored");
    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toBe("text/plain; charset=utf-8");
    expect(await response.text()).toBe("🐿️");
  });

  // Campfire turns any response carrying a Content-Type into a room message,
  // so the only way to say nothing is to send no Content-Type at all.
  it("sends no content-type at all when ignoring", async () => {
    const response = respond("ignored");
    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toBeNull();
    expect(await response.text()).toBe("");
  });

  // 200 even on failure: a non-200 that carries a Content-Type is uploaded
  // into the room as an attachment.
  it("says so plainly when the capture failed, still with a 200", async () => {
    const response = respond("failed");
    expect(response.status).toBe(200);
    expect(await response.text()).toContain("resend");
  });
});

describe("createCampfireTransport", () => {
  it("mounts on the configured path and stores a capture", async () => {
    const { mount, handler } = mountOne();
    const sink = sinkReturning("stored");
    const transport = createCampfireTransport(config);
    await transport.start(sink, mount);

    const response = await handler()(
      new Request("http://test/transports/campfire", {
        method: "POST",
        body: JSON.stringify(PAYLOAD),
      }),
    );

    expect(await response.text()).toBe("🐿️");
    expect(sink.seen[0]?.text).toBe("buy milk");
  });

  it("has no send without a bot key, and says so in the type", () => {
    expect(createCampfireTransport(config).send).toBeNull();
  });

  // An unhandled throw would become a 500, and a 500 with a Content-Type
  // uploads an error page into the room.
  it("never throws out of the handler, even when the sink does", async () => {
    const { mount, handler } = mountOne();
    const exploding: CaptureSink = { accept: () => Promise.reject(new Error("boom")) };
    const transport = createCampfireTransport(config);
    await transport.start(exploding, mount);

    const response = await handler()(
      new Request("http://test/transports/campfire", {
        method: "POST",
        body: JSON.stringify(PAYLOAD),
      }),
    );

    expect(response.status).toBe(200);
    expect(await response.text()).toContain("resend");
  });

  it("stops without error", async () => {
    const { mount } = mountOne();
    const stop = await createCampfireTransport(config).start(sinkReturning("stored"), mount);
    await expect(stop()).resolves.toBeUndefined();
  });
});
