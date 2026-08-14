import { describe, expect, it } from "vitest";

import type { Capture } from "../../src/capture/capture.js";
import { type Allow, decide } from "../../src/capture/policy.js";

const allows: readonly Allow[] = [{ transport: "campfire", conversationId: "7", senderId: "1" }];

function capture(overrides: Partial<Capture> = {}): Capture {
  return {
    transport: "campfire",
    externalId: "42",
    conversationId: "7",
    senderId: "1",
    text: "buy milk",
    receivedAt: new Date("2026-08-14T09:31:04.512Z"),
    payload: {},
    ...overrides,
  };
}

describe("decide", () => {
  it("accepts the configured conversation and sender", () => {
    expect(decide(capture(), allows)).toBe("accept");
  });

  it("ignores another conversation", () => {
    expect(decide(capture({ conversationId: "8" }), allows)).toBe("ignore");
  });

  it("ignores another sender in the right conversation", () => {
    expect(decide(capture({ senderId: "2" }), allows)).toBe("ignore");
  });

  it("ignores a transport that is not configured", () => {
    expect(decide(capture({ transport: "matrix" }), allows)).toBe("ignore");
  });

  // The load-bearing case. If Campfire changes its payload shape and room.id
  // goes undefined, failing closed drops every capture silently for days.
  it("fails open when the conversation is unknown", () => {
    expect(decide(capture({ conversationId: null }), allows)).toBe("accept");
  });

  it("fails open when the sender is unknown", () => {
    expect(decide(capture({ senderId: null }), allows)).toBe("accept");
  });

  it("fails open even for a transport that is not configured", () => {
    expect(decide(capture({ transport: "matrix", conversationId: null }), allows)).toBe("accept");
  });

  it("accepts against any matching entry when several are configured", () => {
    const many: readonly Allow[] = [
      ...allows,
      { transport: "matrix", conversationId: "!room:example", senderId: "@me:example" },
    ];
    const matrix = capture({
      transport: "matrix",
      conversationId: "!room:example",
      senderId: "@me:example",
    });
    expect(decide(matrix, many)).toBe("accept");
  });
});
