import { type Server, createServer } from "node:http";
import type { AddressInfo } from "node:net";
import { afterEach, describe, expect, it } from "vitest";

import type { CampfireConfig } from "../../src/config.js";
import { createCampfireTransport } from "../../src/transports/campfire.js";

interface Received {
  readonly url: string;
  readonly body: string;
}

let server: Server | undefined;

afterEach(() => {
  server?.close();
  server = undefined;
});

async function stubCampfire(status = 201): Promise<{ base: string; received: Received[] }> {
  const received: Received[] = [];
  server = createServer((request, response) => {
    const chunks: Buffer[] = [];
    request.on("data", (chunk: Buffer) => chunks.push(chunk));
    request.on("end", () => {
      received.push({ url: request.url ?? "", body: Buffer.concat(chunks).toString("utf8") });
      response.writeHead(status);
      response.end();
    });
  });
  await new Promise<void>((listening) => server?.listen(0, listening));
  const { port } = server?.address() as AddressInfo;
  return { base: `http://127.0.0.1:${port}`, received };
}

function config(overrides: Partial<CampfireConfig> = {}): CampfireConfig {
  return {
    path: "/transports/campfire",
    conversationId: "7",
    senderId: "1",
    baseUrl: null,
    botKey: null,
    ...overrides,
  };
}

describe("campfire send", () => {
  it("is null without a bot key", () => {
    expect(createCampfireTransport(config()).send).toBeNull();
  });

  it("posts the raw text to the bot messages endpoint", async () => {
    const { base, received } = await stubCampfire();
    const transport = createCampfireTransport(config({ baseUrl: base, botKey: "3-abc" }));

    await transport.send?.("7", "time to vacuum");

    expect(received).toHaveLength(1);
    expect(received[0]?.url).toBe("/rooms/7/3-abc/messages");
    expect(received[0]?.body).toBe("time to vacuum");
  });

  it("throws on a rejected send rather than reporting success", async () => {
    const { base } = await stubCampfire(500);
    const transport = createCampfireTransport(config({ baseUrl: base, botKey: "3-abc" }));

    await expect(transport.send?.("7", "hello")).rejects.toThrow(/500/);
  });

  it("tolerates a base url with a trailing slash", async () => {
    const { base, received } = await stubCampfire();
    const transport = createCampfireTransport(config({ baseUrl: `${base}/`, botKey: "3-abc" }));

    await transport.send?.("7", "hello");
    expect(received[0]?.url).toBe("/rooms/7/3-abc/messages");
  });
});
