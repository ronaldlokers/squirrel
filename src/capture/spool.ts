import { randomUUID } from "node:crypto";
import { constants } from "node:fs";
import { access, mkdir, open, readFile, readdir, rename, unlink } from "node:fs/promises";
import { join } from "node:path";

import type { Capture } from "./capture.js";

export interface Spool {
  /** Durable when this resolves. Returns the filename it created. */
  write(capture: Capture): Promise<string>;
  list(): Promise<string[]>;
  read(name: string): Promise<Capture>;
  remove(name: string): Promise<void>;
  quarantine(name: string): Promise<void>;
  /** Deletes `.tmp` files left by a crash. Returns how many. */
  sweep(): Promise<number>;
  writable(): Promise<boolean>;
}

const QUARANTINE = "quarantine";

/** Epoch millis, zero-padded so lexical order is chronological order. */
function stamp(at: Date): string {
  return String(at.getTime()).padStart(15, "0");
}

/**
 * Matrix room ids carry `!` and `:`, Telegram chat ids are negative, and none
 * of it belongs in a path. Sanitised for the filename only — the real id is
 * inside the file.
 */
function safe(value: string): string {
  return value.replace(/[^A-Za-z0-9_-]/g, "_").slice(0, 64);
}

function filename(capture: Capture): string {
  const id = capture.externalId === null ? randomUUID() : safe(capture.externalId);
  return `${stamp(capture.receivedAt)}-${safe(capture.transport)}-${id}.json`;
}

function serialise(capture: Capture): string {
  return JSON.stringify({ ...capture, receivedAt: capture.receivedAt.toISOString() });
}

function deserialise(raw: string): Capture {
  const parsed = JSON.parse(raw) as Omit<Capture, "receivedAt"> & { receivedAt: string };
  return { ...parsed, receivedAt: new Date(parsed.receivedAt) };
}

export async function openSpool(dir: string): Promise<Spool> {
  await mkdir(join(dir, QUARANTINE), { recursive: true });

  return {
    async write(capture) {
      const name = filename(capture);
      const temporary = join(dir, `${name}.tmp`);

      // Write, then fsync, then rename. The rename is atomic, so the drain
      // loop sees either nothing or a whole file — never half of one.
      const handle = await open(temporary, "w");
      try {
        await handle.writeFile(serialise(capture), "utf8");
        await handle.sync();
      } finally {
        await handle.close();
      }
      await rename(temporary, join(dir, name));

      // The file's own contents are already fsynced above, and the rename is
      // atomic, so a process crash loses nothing. But a directory entry lives
      // in the directory's inode, not the file's, and a bare rename does not
      // force that to disk — a host power loss (this runs on Pis with no
      // UPS) can still lose the rename itself, taking the capture with it.
      // Do not remove this as redundant with the fsync above; it is a
      // different inode.
      const dirHandle = await open(dir, "r");
      try {
        await dirHandle.sync();
      } finally {
        await dirHandle.close();
      }
      return name;
    },

    async list() {
      const entries = await readdir(dir, { withFileTypes: true });
      return entries
        .filter((entry) => entry.isFile() && entry.name.endsWith(".json"))
        .map((entry) => entry.name)
        .sort();
    },

    async read(name) {
      return deserialise(await readFile(join(dir, name), "utf8"));
    },

    async remove(name) {
      await unlink(join(dir, name));
    },

    async quarantine(name) {
      await rename(join(dir, name), join(dir, QUARANTINE, name));
    },

    async sweep() {
      const entries = await readdir(dir, { withFileTypes: true });
      const stale = entries.filter((entry) => entry.isFile() && entry.name.endsWith(".tmp"));
      await Promise.all(stale.map((entry) => unlink(join(dir, entry.name))));
      return stale.length;
    },

    async writable() {
      try {
        await access(dir, constants.W_OK);
        return true;
      } catch {
        return false;
      }
    },
  };
}
