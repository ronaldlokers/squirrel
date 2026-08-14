import { mkdtemp } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Capture } from "../../src/capture/capture.js";
import type { Allow } from "../../src/capture/policy.js";
import { createSink } from "../../src/capture/sink.js";
import { type Spool, openSpool } from "../../src/capture/spool.js";

const allows: readonly Allow[] = [{ transport: "campfire", conversationId: "7", senderId: "1" }];

let spool: Spool;

beforeEach(async () => {
  spool = await openSpool(await mkdtemp(join(tmpdir(), "squirrel-sink-")));
});

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

describe("createSink", () => {
  it("stores an accepted capture", async () => {
    const sink = createSink(spool, allows);
    expect(await sink.accept(capture())).toBe("stored");
    expect(await spool.list()).toHaveLength(1);
  });

  it("ignores a capture from elsewhere and writes nothing", async () => {
    const sink = createSink(spool, allows);
    expect(await sink.accept(capture({ conversationId: "8" }))).toBe("ignored");
    expect(await spool.list()).toEqual([]);
  });

  // Silence in the room, but not silence in the logs. Being added to a room
  // Squirrel does not belong in should be visible somewhere.
  it("reports an ignored conversation so it is not silent everywhere", async () => {
    const onIgnored = vi.fn();
    const sink = createSink(spool, allows, { onIgnored });

    await sink.accept(capture({ conversationId: "8" }));

    expect(onIgnored).toHaveBeenCalledOnce();
    expect(onIgnored.mock.calls[0]?.[0]).toMatchObject({ conversationId: "8" });
  });

  // A hook is observability. It must never be able to turn a real outcome
  // into a thrown rejection that a transport then misreports to the user.
  it("still ignores and writes nothing when onIgnored throws", async () => {
    const onIgnored = vi.fn(() => {
      throw new Error("logging backend unreachable");
    });
    const sink = createSink(spool, allows, { onIgnored });

    expect(await sink.accept(capture({ conversationId: "8" }))).toBe("ignored");
    expect(await spool.list()).toEqual([]);
  });

  it("stores a fail-open capture", async () => {
    const sink = createSink(spool, allows);
    expect(await sink.accept(capture({ conversationId: null, senderId: null }))).toBe("stored");
    expect(await spool.list()).toHaveLength(1);
  });

  // The one path where the bot admits it could not do its job. Inventing a
  // cheerful answer here is exactly the false trust the design rules out.
  it("reports failure rather than pretending, when the spool cannot write", async () => {
    const broken: Spool = {
      ...spool,
      write: () => Promise.reject(new Error("ENOSPC: no space left on device")),
    };
    const onError = vi.fn();
    const sink = createSink(broken, allows, { onError });

    expect(await sink.accept(capture())).toBe("failed");
    expect(onError).toHaveBeenCalledOnce();
  });

  // Same guarantee on the failure path: a throwing onError must not mask the
  // fact that the capture was not saved.
  it("still reports failure when onError itself throws", async () => {
    const broken: Spool = {
      ...spool,
      write: () => Promise.reject(new Error("ENOSPC: no space left on device")),
    };
    const onError = vi.fn(() => {
      throw new Error("logging backend unreachable");
    });
    const sink = createSink(broken, allows, { onError });

    expect(await sink.accept(capture())).toBe("failed");
  });
});
