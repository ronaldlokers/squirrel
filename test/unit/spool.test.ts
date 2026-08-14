import { mkdtemp, readdir, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { beforeEach, describe, expect, it } from "vitest";

import type { Capture } from "../../src/capture/capture.js";
import { openSpool } from "../../src/capture/spool.js";

let dir: string;

beforeEach(async () => {
  dir = await mkdtemp(join(tmpdir(), "squirrel-spool-"));
});

function capture(overrides: Partial<Capture> = {}): Capture {
  return {
    transport: "campfire",
    externalId: "42",
    conversationId: "7",
    senderId: "1",
    text: "buy milk",
    receivedAt: new Date("2026-08-14T09:31:04.512Z"),
    payload: { message: { id: 42 } },
    ...overrides,
  };
}

describe("spool", () => {
  it("writes a file that survives a round trip unchanged", async () => {
    const spool = await openSpool(dir);
    const name = await spool.write(capture());

    expect(await spool.list()).toEqual([name]);
    const read = await spool.read(name);
    expect(read).toEqual(capture());
    expect(read.receivedAt).toBeInstanceOf(Date);
  });

  it("names files so that they sort chronologically", async () => {
    const spool = await openSpool(dir);
    const later = await spool.write(
      capture({ externalId: "43", receivedAt: new Date("2026-08-14T10:00:00.000Z") }),
    );
    const earlier = await spool.write(capture({ externalId: "42" }));

    expect(await spool.list()).toEqual([earlier, later]);
  });

  it("leaves no .tmp behind on a successful write", async () => {
    const spool = await openSpool(dir);
    await spool.write(capture());
    const entries = await readdir(dir);
    expect(entries.filter((entry) => entry.endsWith(".tmp"))).toEqual([]);
  });

  it("never lists a .tmp, because a partial file must be invisible", async () => {
    const spool = await openSpool(dir);
    await writeFile(join(dir, "000000000000001-campfire-9.json.tmp"), "{ partial");
    expect(await spool.list()).toEqual([]);
  });

  it("sweeps .tmp files left by a crash", async () => {
    const spool = await openSpool(dir);
    await writeFile(join(dir, "000000000000001-campfire-9.json.tmp"), "{ partial");
    expect(await spool.sweep()).toBe(1);
    expect((await readdir(dir)).filter((entry) => entry.endsWith(".tmp"))).toEqual([]);
  });

  // Matrix room ids contain ! and :, Telegram chat ids are negative. None of
  // that may reach a filename.
  it("sanitises an external id that is not filename-safe", async () => {
    const spool = await openSpool(dir);
    const name = await spool.write(capture({ transport: "matrix", externalId: "!a:b/../c" }));
    expect(name).not.toContain("/");
    expect(name).not.toContain(":");
    expect(await spool.read(name)).toMatchObject({ externalId: "!a:b/../c" });
  });

  it("gives an unknown external id a unique name rather than colliding", async () => {
    const spool = await openSpool(dir);
    const first = await spool.write(capture({ externalId: null }));
    const second = await spool.write(capture({ externalId: null }));
    expect(first).not.toBe(second);
    expect(await spool.list()).toHaveLength(2);
  });

  it("removes a drained file", async () => {
    const spool = await openSpool(dir);
    const name = await spool.write(capture());
    await spool.remove(name);
    expect(await spool.list()).toEqual([]);
  });

  it("quarantines rather than deletes, so nothing is ever thrown away", async () => {
    const spool = await openSpool(dir);
    const name = await spool.write(capture());
    await spool.quarantine(name);

    expect(await spool.list()).toEqual([]);
    expect(await readdir(join(dir, "quarantine"))).toEqual([name]);
  });

  it("reports the directory as writable", async () => {
    const spool = await openSpool(dir);
    expect(await spool.writable()).toBe(true);
  });
});
