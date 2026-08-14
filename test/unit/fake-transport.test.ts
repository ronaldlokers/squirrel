import { mkdtemp } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

import type { CaptureSink } from "../../src/capture/capture.js";
import { createSink } from "../../src/capture/sink.js";
import { openSpool } from "../../src/capture/spool.js";
import type { HttpMount, Stop, Transport } from "../../src/transports/transport.js";

/**
 * A transport built entirely out of the published interface, with no HTTP and
 * no Campfire anywhere. Nothing in the core may need more than this.
 */
function fakeTransport(): Transport & { deliver: (text: string) => Promise<string> } {
  let sink: CaptureSink | undefined;
  return {
    name: "fake",
    start(given: CaptureSink, _http: HttpMount): Promise<Stop> {
      sink = given;
      return Promise.resolve(() => Promise.resolve());
    },
    send: null,
    async deliver(text: string) {
      if (!sink) throw new Error("not started");
      return sink.accept({
        transport: "fake",
        externalId: `${text.length}`,
        conversationId: "room",
        senderId: "me",
        text,
        receivedAt: new Date(),
        payload: { text },
      });
    },
  };
}

describe("the core, driven by a transport that is not campfire", () => {
  it("stores a capture end to end", async () => {
    const spool = await openSpool(await mkdtemp(join(tmpdir(), "squirrel-fake-")));
    const sink = createSink(spool, [{ transport: "fake", conversationId: "room", senderId: "me" }]);

    const transport = fakeTransport();
    await transport.start(sink, { post: () => {} });

    expect(await transport.deliver("water the plants")).toBe("stored");
    expect(await spool.list()).toHaveLength(1);
  });

  it("cannot send, and says so", () => {
    expect(fakeTransport().send).toBeNull();
  });
});
