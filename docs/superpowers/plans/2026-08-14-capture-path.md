# Phase 1 Capture Path Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A Campfire direct message arrives, its raw text is stored durably, and the room gets a 🐿️ back.

**Architecture:** The process splits in one place. A *core* (`capture/`, `people/`, `db/`) knows about captures, durability, people, and a database. A *transport* (`transports/`) knows about exactly one chat system. The request path writes each capture to a spool directory with `fsync` + atomic `rename` and acknowledges from there; a drain loop moves spooled files into Postgres afterwards. Postgres is never on the request path, so a database outage cannot lose a message.

**Tech Stack:** Node 24, TypeScript (ESM, NodeNext), Hono + `@hono/node-server`, Drizzle ORM + `pg`, drizzle-kit migrations, Vitest, Biome.

**Spec:** `docs/superpowers/specs/2026-08-14-capture-path-design.md`

## Global Constraints

- Node 24 LTS. `package.json` declares `"engines": { "node": ">=24" }` and `"type": "module"`.
- TypeScript `moduleResolution: "NodeNext"`. **All relative imports carry a `.js` extension**, including from `.ts` files.
- Runtime dependencies are exactly: `hono`, `@hono/node-server`, `drizzle-orm`, `pg`. Do not add others. Dev dependencies are exactly: `typescript`, `@types/node`, `@types/pg`, `vitest`, `@biomejs/biome`, `drizzle-kit`.
- **Nothing under `src/capture/`, `src/people/` or `src/db/` may import from `src/transports/`.** Enforced by a test in Task 12, not by convention.
- The HTTP handler must never throw. A 500 carrying a `Content-Type` makes Campfire upload an error page into the room.
- The `ignored` response carries **no `Content-Type` header at all**. That is the only way to say nothing.
- Every response to Campfire is HTTP 200, failures included.
- Identifiers crossing the transport boundary are `string`, never `number`.
- A column exists only if every plausible transport can fill it. Everything else lives in `payload`.
- Unknown identities resolve to `null`. They are **never** auto-created.
- Postgres is never read or written on the request path.
- Commit messages: conventional-commit style, lowercase imperative subject.
- Work happens on the branch `feat/phase-1-capture`, which already exists.

---

## File Structure

| File | Responsibility |
|---|---|
| `src/config.ts` | Parse and validate environment; fail loudly at boot |
| `src/http.ts` | Shared HTTP server, `/healthz`, the `HttpMount` handle |
| `src/capture/capture.ts` | `Capture` and `Outcome` types |
| `src/capture/policy.ts` | The guard: accept, ignore, or fail open |
| `src/capture/spool.ts` | Durable write, list, read, remove, quarantine, sweep |
| `src/capture/sink.ts` | `CaptureSink` joining policy to spool |
| `src/capture/drain.ts` | The loop moving spool files into Postgres |
| `src/people/identities.ts` | Seed people from config; resolve transport ids to people |
| `src/db/schema.ts` | Drizzle table definitions |
| `src/db/client.ts` | `pg` pool and Drizzle handle |
| `src/db/migrate.ts` | Apply migrations |
| `src/transports/transport.ts` | `Transport`, `HttpMount`, `Stop` |
| `src/transports/campfire.ts` | The only transport implementation |
| `src/index.ts` | Boot order: config, HTTP, transports, then Postgres in background |

---

### Task 1: Project scaffold

**Files:**
- Create: `package.json`, `tsconfig.json`, `biome.json`, `vitest.config.ts`, `mise.toml`, `.gitignore`, `.dockerignore`
- Create: `src/version.ts`
- Test: `test/unit/version.test.ts`

**Interfaces:**
- Consumes: nothing
- Produces: a working `npm test`, `npm run typecheck`, `npm run lint`. Every later task depends on these three commands existing.

- [ ] **Step 1: Create `package.json`**

```json
{
  "name": "squirrel",
  "version": "0.1.0",
  "private": true,
  "description": "A Campfire-driven external memory bot",
  "type": "module",
  "engines": { "node": ">=24" },
  "scripts": {
    "build": "tsc",
    "typecheck": "tsc --noEmit",
    "lint": "biome check .",
    "format": "biome check --write .",
    "test": "vitest run test/unit",
    "test:integration": "vitest run test/integration",
    "db:generate": "drizzle-kit generate",
    "start": "node dist/src/index.js"
  },
  "dependencies": {
    "@hono/node-server": "^1.13.7",
    "drizzle-orm": "^0.38.3",
    "hono": "^4.6.14",
    "pg": "^8.13.1"
  },
  "devDependencies": {
    "@biomejs/biome": "^1.9.4",
    "@types/node": "^22.10.2",
    "@types/pg": "^8.11.10",
    "drizzle-kit": "^0.30.1",
    "typescript": "^5.7.2",
    "vitest": "^2.1.8"
  }
}
```

- [ ] **Step 2: Create `tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2023",
    "lib": ["ES2023"],
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "outDir": "dist",
    "rootDir": ".",
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "exactOptionalPropertyTypes": true,
    "noImplicitOverride": true,
    "verbatimModuleSyntax": true,
    "skipLibCheck": true,
    "types": ["node"]
  },
  "include": ["src/**/*.ts", "test/**/*.ts", "vitest.config.ts", "drizzle.config.ts"]
}
```

- [ ] **Step 3: Create `biome.json`**

Deliberately plain. The core-must-not-import-transports rule is a test in Task 12 rather than a lint override, because a test asserting a real fact about the source tree does not depend on which Biome release a rule lives in.

```json
{
  "$schema": "https://biomejs.dev/schemas/1.9.4/schema.json",
  "organizeImports": { "enabled": true },
  "files": { "ignore": ["dist", "drizzle", "node_modules"] },
  "formatter": { "enabled": true, "indentStyle": "space", "indentWidth": 2, "lineWidth": 100 },
  "linter": {
    "enabled": true,
    "rules": { "recommended": true }
  }
}
```

- [ ] **Step 4: Create `vitest.config.ts`**

```ts
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    // Integration tests share one Postgres database and truncate between
    // cases, so they must not run concurrently with each other.
    fileParallelism: false,
    testTimeout: 20_000,
  },
});
```

- [ ] **Step 5: Create `mise.toml`, `.gitignore` and `.dockerignore`**

`mise.toml`:
```toml
[tools]
node = "24"
```

`.gitignore`:
```
node_modules
dist
```

`.dockerignore`:
```
node_modules
dist
.git
```

- [ ] **Step 6: Install dependencies**

Run: `npm install`
Expected: `package-lock.json` is created, no errors.

- [ ] **Step 7: Write the failing test**

`test/unit/version.test.ts`:
```ts
import { describe, expect, it } from "vitest";

import { NAME } from "../../src/version.js";

describe("version", () => {
  it("names the service", () => {
    expect(NAME).toBe("squirrel");
  });
});
```

- [ ] **Step 8: Run the test to verify it fails**

Run: `npm test`
Expected: FAIL — cannot resolve `../../src/version.js`.

- [ ] **Step 9: Write the minimal implementation**

`src/version.ts`:
```ts
export const NAME = "squirrel";
```

- [ ] **Step 10: Run the test, typecheck and lint**

Run: `npm test && npm run typecheck && npm run lint`
Expected: test PASS, typecheck clean, lint clean.

- [ ] **Step 11: Commit**

```bash
git add package.json package-lock.json tsconfig.json biome.json vitest.config.ts mise.toml .gitignore .dockerignore src/version.ts test/unit/version.test.ts
git commit -m "chore: scaffold the project"
```

---

### Task 2: Configuration

**Files:**
- Create: `src/config.ts`
- Test: `test/unit/config.test.ts`

**Interfaces:**
- Consumes: nothing
- Produces:
  - `interface PostgresConfig { host: string; port: number; database: string; user: string; password: string }`
  - `interface CampfireConfig { path: string; conversationId: string; senderId: string; baseUrl: string | null; botKey: string | null }`
  - `interface Config { port: number; transports: readonly string[]; ownerHandle: string; spoolDir: string; drainIntervalMs: number; postgres: PostgresConfig; campfire: CampfireConfig | null }`
  - `class ConfigError extends Error`
  - `function loadConfig(env: Record<string, string | undefined>): Config`

- [ ] **Step 1: Write the failing tests**

`test/unit/config.test.ts`:
```ts
import { describe, expect, it } from "vitest";

import { ConfigError, loadConfig } from "../../src/config.js";

const minimal = {
  CAMPFIRE_CONVERSATION_ID: "7",
  CAMPFIRE_SENDER_ID: "1",
  POSTGRES_SERVER: "db.example",
  POSTGRES_DB: "squirrel",
  POSTGRES_USER: "squirrel",
  POSTGRES_PASSWORD: "hunter2",
};

describe("loadConfig", () => {
  it("applies documented defaults", () => {
    const config = loadConfig(minimal);
    expect(config.port).toBe(8080);
    expect(config.transports).toEqual(["campfire"]);
    expect(config.ownerHandle).toBe("owner");
    expect(config.spoolDir).toBe("/var/spool/squirrel");
    expect(config.drainIntervalMs).toBe(1000);
    expect(config.postgres.port).toBe(5432);
    expect(config.campfire?.path).toBe("/transports/campfire");
  });

  it("leaves send unconfigured when no bot key is present", () => {
    expect(loadConfig(minimal).campfire?.botKey).toBeNull();
    expect(loadConfig(minimal).campfire?.baseUrl).toBeNull();
  });

  it("carries the bot key and base url when both are present", () => {
    const config = loadConfig({
      ...minimal,
      CAMPFIRE_BASE_URL: "https://campfire.example",
      CAMPFIRE_BOT_KEY: "3-abc",
    });
    expect(config.campfire?.baseUrl).toBe("https://campfire.example");
    expect(config.campfire?.botKey).toBe("3-abc");
  });

  it("rejects a bot key with no base url, because send would be broken rather than absent", () => {
    expect(() => loadConfig({ ...minimal, CAMPFIRE_BOT_KEY: "3-abc" })).toThrow(ConfigError);
  });

  it("rejects a missing required variable by name", () => {
    const { POSTGRES_PASSWORD, ...rest } = minimal;
    expect(() => loadConfig(rest)).toThrow(/POSTGRES_PASSWORD/);
  });

  it("rejects campfire settings that are absent while campfire is enabled", () => {
    const { CAMPFIRE_CONVERSATION_ID, ...rest } = minimal;
    expect(() => loadConfig(rest)).toThrow(/CAMPFIRE_CONVERSATION_ID/);
  });

  it("does not require campfire settings when campfire is not enabled", () => {
    const { CAMPFIRE_CONVERSATION_ID, CAMPFIRE_SENDER_ID, ...rest } = minimal;
    const config = loadConfig({ ...rest, TRANSPORTS: "" });
    expect(config.transports).toEqual([]);
    expect(config.campfire).toBeNull();
  });

  it("rejects a non-numeric port", () => {
    expect(() => loadConfig({ ...minimal, PORT: "eight" })).toThrow(/PORT/);
  });
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm test -- config`
Expected: FAIL — cannot resolve `../../src/config.js`.

- [ ] **Step 3: Write the implementation**

`src/config.ts`:
```ts
export class ConfigError extends Error {}

export interface PostgresConfig {
  readonly host: string;
  readonly port: number;
  readonly database: string;
  readonly user: string;
  readonly password: string;
}

export interface CampfireConfig {
  readonly path: string;
  readonly conversationId: string;
  readonly senderId: string;
  /** Both of these are null unless `send` is configured. */
  readonly baseUrl: string | null;
  readonly botKey: string | null;
}

export interface Config {
  readonly port: number;
  readonly transports: readonly string[];
  readonly ownerHandle: string;
  readonly spoolDir: string;
  readonly drainIntervalMs: number;
  readonly postgres: PostgresConfig;
  readonly campfire: CampfireConfig | null;
}

type Env = Record<string, string | undefined>;

function required(env: Env, name: string): string {
  const value = env[name];
  if (value === undefined || value === "") throw new ConfigError(`${name} is required`);
  return value;
}

function optional(env: Env, name: string): string | null {
  const value = env[name];
  return value === undefined || value === "" ? null : value;
}

function number(env: Env, name: string, fallback: number): number {
  const value = env[name];
  if (value === undefined || value === "") return fallback;
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed < 0) {
    throw new ConfigError(`${name} must be a non-negative integer, got ${JSON.stringify(value)}`);
  }
  return parsed;
}

function transportsFrom(env: Env): readonly string[] {
  const raw = env.TRANSPORTS ?? "campfire";
  return raw
    .split(",")
    .map((name) => name.trim())
    .filter((name) => name !== "");
}

function campfireFrom(env: Env): CampfireConfig {
  const baseUrl = optional(env, "CAMPFIRE_BASE_URL");
  const botKey = optional(env, "CAMPFIRE_BOT_KEY");
  // Half-configured outbound is worse than none: `send` would exist and fail at
  // the moment it is needed rather than being honestly absent.
  if (botKey !== null && baseUrl === null) {
    throw new ConfigError("CAMPFIRE_BOT_KEY requires CAMPFIRE_BASE_URL");
  }
  return {
    path: optional(env, "CAMPFIRE_PATH") ?? "/transports/campfire",
    conversationId: required(env, "CAMPFIRE_CONVERSATION_ID"),
    senderId: required(env, "CAMPFIRE_SENDER_ID"),
    baseUrl: botKey === null ? null : baseUrl,
    botKey,
  };
}

export function loadConfig(env: Env): Config {
  const transports = transportsFrom(env);
  return {
    port: number(env, "PORT", 8080),
    transports,
    ownerHandle: optional(env, "OWNER_HANDLE") ?? "owner",
    spoolDir: optional(env, "SPOOL_DIR") ?? "/var/spool/squirrel",
    drainIntervalMs: number(env, "DRAIN_INTERVAL_MS", 1000),
    postgres: {
      host: required(env, "POSTGRES_SERVER"),
      port: number(env, "POSTGRES_PORT", 5432),
      database: required(env, "POSTGRES_DB"),
      user: required(env, "POSTGRES_USER"),
      password: required(env, "POSTGRES_PASSWORD"),
    },
    campfire: transports.includes("campfire") ? campfireFrom(env) : null,
  };
}
```

- [ ] **Step 4: Run the tests**

Run: `npm test -- config && npm run typecheck`
Expected: 8 tests PASS, typecheck clean.

- [ ] **Step 5: Commit**

```bash
git add src/config.ts test/unit/config.test.ts
git commit -m "feat: load and validate configuration at boot"
```

---

### Task 3: Capture type and the guard

**Files:**
- Create: `src/capture/capture.ts`, `src/capture/policy.ts`
- Test: `test/unit/policy.test.ts`

**Interfaces:**
- Consumes: nothing
- Produces:
  - `interface Capture { transport: string; externalId: string | null; conversationId: string | null; senderId: string | null; text: string; receivedAt: Date; payload: unknown }`
  - `type Outcome = "stored" | "ignored" | "failed"`
  - `interface CaptureSink { accept(capture: Capture): Promise<Outcome> }`
  - `interface Allow { transport: string; conversationId: string; senderId: string }`
  - `type Verdict = "accept" | "ignore"`
  - `function decide(capture: Capture, allows: readonly Allow[]): Verdict`

- [ ] **Step 1: Write `src/capture/capture.ts`**

This file is types only, so it has no test of its own; Task 3's tests exercise it through `decide`.

```ts
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
```

- [ ] **Step 2: Write the failing tests**

`test/unit/policy.test.ts`:
```ts
import { describe, expect, it } from "vitest";

import type { Capture } from "../../src/capture/capture.js";
import { type Allow, decide } from "../../src/capture/policy.js";

const allows: readonly Allow[] = [
  { transport: "campfire", conversationId: "7", senderId: "1" },
];

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
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `npm test -- policy`
Expected: FAIL — cannot resolve `../../src/capture/policy.js`.

- [ ] **Step 4: Write the implementation**

`src/capture/policy.ts`:
```ts
import type { Capture } from "./capture.js";

export interface Allow {
  readonly transport: string;
  readonly conversationId: string;
  readonly senderId: string;
}

export type Verdict = "accept" | "ignore";

/**
 * The guard, and it fails in two directions on purpose.
 *
 * Understood the envelope and it is the wrong room or the wrong person: fail
 * closed, because the system genuinely was not addressed.
 *
 * Could not understand the envelope at all: fail open. A payload shape change
 * upstream would otherwise drop every capture silently, with the bot still
 * answering cheerfully. Junk rows in a table nobody reads yet is the cheaper
 * mistake by a wide margin.
 */
export function decide(capture: Capture, allows: readonly Allow[]): Verdict {
  if (capture.conversationId === null || capture.senderId === null) return "accept";

  const matched = allows.some(
    (allow) =>
      allow.transport === capture.transport &&
      allow.conversationId === capture.conversationId &&
      allow.senderId === capture.senderId,
  );
  return matched ? "accept" : "ignore";
}
```

- [ ] **Step 5: Run the tests**

Run: `npm test -- policy && npm run typecheck && npm run lint`
Expected: 8 tests PASS, typecheck clean, lint clean.

- [ ] **Step 6: Commit**

```bash
git add src/capture/capture.ts src/capture/policy.ts test/unit/policy.test.ts
git commit -m "feat: add the capture type and the guard"
```

---

### Task 4: The spool

**Files:**
- Create: `src/capture/spool.ts`
- Test: `test/unit/spool.test.ts`

**Interfaces:**
- Consumes: `Capture` from Task 3
- Produces:
  - `interface Spool { write(capture: Capture): Promise<string>; list(): Promise<string[]>; read(name: string): Promise<Capture>; remove(name: string): Promise<void>; quarantine(name: string): Promise<void>; sweep(): Promise<number>; writable(): Promise<boolean> }`
  - `function openSpool(dir: string): Promise<Spool>`
  - `write` returns the filename it created.

- [ ] **Step 1: Write the failing tests**

`test/unit/spool.test.ts`:
```ts
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
    expect(await readdir(dir)).toEqual([]);
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm test -- spool`
Expected: FAIL — cannot resolve `../../src/capture/spool.js`.

- [ ] **Step 3: Write the implementation**

`src/capture/spool.ts`:
```ts
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
```

- [ ] **Step 4: Run the tests**

Run: `npm test -- spool && npm run typecheck && npm run lint`
Expected: 10 tests PASS, typecheck clean, lint clean.

- [ ] **Step 5: Commit**

```bash
git add src/capture/spool.ts test/unit/spool.test.ts
git commit -m "feat: add the durable capture spool"
```

---

### Task 5: The sink

**Files:**
- Create: `src/capture/sink.ts`
- Test: `test/unit/sink.test.ts`

**Interfaces:**
- Consumes: `Capture`, `CaptureSink`, `Outcome` (Task 3); `Allow`, `decide` (Task 3); `Spool` (Task 4)
- Produces:
  - `interface SinkHooks { onError?: (error: unknown) => void; onIgnored?: (capture: Capture) => void }`
  - `function createSink(spool: Spool, allows: readonly Allow[], hooks?: SinkHooks): CaptureSink`

- [ ] **Step 1: Write the failing tests**

`test/unit/sink.test.ts`:
```ts
import { mkdtemp } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Capture } from "../../src/capture/capture.js";
import type { Allow } from "../../src/capture/policy.js";
import { createSink } from "../../src/capture/sink.js";
import { type Spool, openSpool } from "../../src/capture/spool.js";

const allows: readonly Allow[] = [
  { transport: "campfire", conversationId: "7", senderId: "1" },
];

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
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm test -- sink`
Expected: FAIL — cannot resolve `../../src/capture/sink.js`.

- [ ] **Step 3: Write the implementation**

`src/capture/sink.ts`:
```ts
import type { Capture, CaptureSink, Outcome } from "./capture.js";
import { type Allow, decide } from "./policy.js";
import type { Spool } from "./spool.js";

export interface SinkHooks {
  readonly onError?: (error: unknown) => void;
  /** An ignored capture is silent in the room. It should not be silent in the logs. */
  readonly onIgnored?: (capture: Capture) => void;
}

/**
 * The only place a capture becomes durable.
 *
 * Note what is absent: Postgres. The request path reaches this function and no
 * further, which is what lets a database outage pass without a lost thought.
 */
export function createSink(
  spool: Spool,
  allows: readonly Allow[],
  hooks: SinkHooks = {},
): CaptureSink {
  return {
    async accept(capture: Capture): Promise<Outcome> {
      if (decide(capture, allows) === "ignore") {
        hooks.onIgnored?.(capture);
        return "ignored";
      }
      try {
        await spool.write(capture);
        return "stored";
      } catch (error) {
        hooks.onError?.(error);
        return "failed";
      }
    },
  };
}
```

- [ ] **Step 4: Run the tests**

Run: `npm test -- sink && npm run typecheck && npm run lint`
Expected: 5 tests PASS, typecheck clean, lint clean.

- [ ] **Step 5: Commit**

```bash
git add src/capture/sink.ts test/unit/sink.test.ts
git commit -m "feat: add the capture sink over the spool"
```

---

### Task 6: HTTP server and health

**Files:**
- Create: `src/transports/transport.ts`, `src/http.ts`
- Test: `test/unit/http.test.ts`

**Interfaces:**
- Consumes: `CaptureSink` (Task 3), `Spool` (Task 4)
- Produces:
  - `type WebHandler = (request: Request) => Promise<Response>`
  - `interface HttpMount { post(path: string, handler: WebHandler): void }`
  - `type Stop = () => Promise<void>`
  - `interface Transport { readonly name: string; start(sink: CaptureSink, http: HttpMount): Promise<Stop>; readonly send: ((conversationId: string, text: string) => Promise<void>) | null }`
  - `function createHttp(spool: Pick<Spool, "writable">): { app: Hono; mount: HttpMount }`
  - `function startHttp(app: Hono, port: number): Promise<{ port: number; close: () => Promise<void> }>`

- [ ] **Step 1: Write `src/transports/transport.ts`**

Types only; exercised by Tasks 7 and 8.

```ts
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
```

- [ ] **Step 2: Write the failing tests**

`test/unit/http.test.ts`:
```ts
import { describe, expect, it } from "vitest";

import { createHttp, startHttp } from "../../src/http.js";

const writableSpool = { writable: () => Promise.resolve(true) };
const brokenSpool = { writable: () => Promise.resolve(false) };

describe("createHttp", () => {
  it("reports health when the spool is writable", async () => {
    const { app } = createHttp(writableSpool);
    const response = await app.request("/healthz");
    expect(response.status).toBe(200);
    expect(await response.text()).toBe("ok");
  });

  // Deliberately not a Postgres check. An unready pod stops receiving
  // webhooks, and Campfire never retries, so that would turn a survivable
  // outage into permanent loss.
  it("reports unhealthy only when the spool is unwritable", async () => {
    const { app } = createHttp(brokenSpool);
    expect((await app.request("/healthz")).status).toBe(503);
  });

  it("404s an unknown path", async () => {
    const { app } = createHttp(writableSpool);
    expect((await app.request("/nope")).status).toBe(404);
  });

  it("lets a transport mount a POST route", async () => {
    const { app, mount } = createHttp(writableSpool);
    mount.post("/transports/fake", async (request) => new Response(await request.text()));

    const response = await app.request("/transports/fake", { method: "POST", body: "hello" });
    expect(await response.text()).toBe("hello");
  });

  it("listens on an ephemeral port and closes cleanly", async () => {
    const { app } = createHttp(writableSpool);
    const server = await startHttp(app, 0);
    expect(server.port).toBeGreaterThan(0);

    const response = await fetch(`http://127.0.0.1:${server.port}/healthz`);
    expect(await response.text()).toBe("ok");
    await server.close();
  });
});
```

- [ ] **Step 3: Run the tests to verify they fail**

Run: `npm test -- http`
Expected: FAIL — cannot resolve `../../src/http.js`.

- [ ] **Step 4: Write the implementation**

`src/http.ts`:
```ts
import type { AddressInfo } from "node:net";

import { serve } from "@hono/node-server";
import { Hono } from "hono";

import type { Spool } from "./capture/spool.js";
import type { HttpMount, WebHandler } from "./transports/transport.js";

export interface Serving {
  readonly port: number;
  readonly close: () => Promise<void>;
}

/**
 * One server for the whole process: `/healthz`, plus whatever routes the
 * transports mount. A transport that polls simply never calls `mount.post`.
 */
export function createHttp(spool: Pick<Spool, "writable">): { app: Hono; mount: HttpMount } {
  const app = new Hono();

  // Liveness and readiness both, and deliberately not a database check. A
  // readiness probe that failed on a Postgres outage would pull this pod out
  // of its Service; webhook delivery would then fail, and Campfire does not
  // retry. That converts a survivable outage into permanent data loss.
  app.get("/healthz", async (context) =>
    (await spool.writable()) ? context.text("ok") : context.text("spool unwritable", 503),
  );

  const mount: HttpMount = {
    post(path: string, handler: WebHandler) {
      app.post(path, (context) => handler(context.req.raw));
    },
  };

  return { app, mount };
}

export function startHttp(app: Hono, port: number): Promise<Serving> {
  return new Promise((resolve) => {
    const server = serve({ fetch: app.fetch, port }, (address: AddressInfo) => {
      resolve({
        port: address.port,
        close: () => new Promise((closed) => server.close(() => closed())),
      });
    });
  });
}
```

- [ ] **Step 5: Run the tests**

Run: `npm test -- http && npm run typecheck && npm run lint`
Expected: 5 tests PASS, typecheck clean, lint clean.

- [ ] **Step 6: Commit**

```bash
git add src/transports/transport.ts src/http.ts test/unit/http.test.ts
git commit -m "feat: add the shared http server and health endpoint"
```

---

### Task 7: Campfire transport, inbound

**Files:**
- Create: `src/transports/campfire.ts`
- Test: `test/unit/campfire.test.ts`

**Interfaces:**
- Consumes: `Capture`, `CaptureSink`, `Outcome` (Task 3); `Transport`, `HttpMount`, `Stop` (Task 6); `CampfireConfig` (Task 2)
- Produces:
  - `function captureFrom(body: string, receivedAt: Date): Capture`
  - `function respond(outcome: Outcome): Response`
  - `function createCampfireTransport(config: CampfireConfig): Transport`

- [ ] **Step 1: Write the failing tests**

`test/unit/campfire.test.ts`:
```ts
import { describe, expect, it, vi } from "vitest";

import type { Capture, CaptureSink, Outcome } from "../../src/capture/capture.js";
import type { CampfireConfig } from "../../src/config.js";
import type { HttpMount, WebHandler } from "../../src/transports/transport.js";
import { captureFrom, createCampfireTransport, respond } from "../../src/transports/campfire.js";

const AT = new Date("2026-08-14T09:31:04.512Z");

const PAYLOAD = {
  user: { id: 1, name: "Ronald" },
  room: { id: 7, name: "Squirrel", path: "/rooms/7/3-abc/messages" },
  message: { id: 42, body: { html: "<div>buy milk</div>", plain: "buy milk" }, path: "/rooms/7/@42" },
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
      new Request("http://test/transports/campfire", { method: "POST", body: JSON.stringify(PAYLOAD) }),
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
      new Request("http://test/transports/campfire", { method: "POST", body: JSON.stringify(PAYLOAD) }),
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm test -- campfire`
Expected: FAIL — cannot resolve `../../src/transports/campfire.js`.

- [ ] **Step 3: Write the implementation**

`src/transports/campfire.ts`:
```ts
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
    payload = JSON.parse(body) as CampfirePayload;
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

    // Filled in by Task 8. Null here because no bot key is configured, which
    // is the honest answer to "can this bot start a conversation?".
    send: null,
  };
}
```

- [ ] **Step 4: Run the tests**

Run: `npm test -- campfire && npm run typecheck && npm run lint`
Expected: 12 tests PASS, typecheck clean, lint clean.

- [ ] **Step 5: Commit**

```bash
git add src/transports/campfire.ts test/unit/campfire.test.ts
git commit -m "feat: add the campfire transport, inbound"
```

---

### Task 8: Campfire transport, outbound

**Files:**
- Modify: `src/transports/campfire.ts`
- Test: `test/unit/campfire-send.test.ts`

**Interfaces:**
- Consumes: `createCampfireTransport` (Task 7), `CampfireConfig` (Task 2), `startHttp`/`createHttp` (Task 6)
- Produces: `Transport.send` is a function when `config.botKey` and `config.baseUrl` are both set, `null` otherwise. No new exported names.

- [ ] **Step 1: Write the failing tests**

`test/unit/campfire-send.test.ts`:
```ts
import { createServer, type Server } from "node:http";
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
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `npm test -- campfire-send`
Expected: FAIL — `transport.send` is null, so the second test receives nothing.

- [ ] **Step 3: Replace the `send: null` line in `src/transports/campfire.ts`**

Add this function above `createCampfireTransport`:

```ts
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
```

Then change the final property of the returned object from `send: null,` to:

```ts
    // Null unless a bot key is configured. Half-working outbound would fail at
    // exactly the moment it is needed; absent outbound is at least honest.
    send:
      config.baseUrl !== null && config.botKey !== null
        ? sendVia(config.baseUrl, config.botKey)
        : null,
```

- [ ] **Step 4: Run the tests**

Run: `npm test && npm run typecheck && npm run lint`
Expected: all unit tests PASS, typecheck clean, lint clean.

- [ ] **Step 5: Commit**

```bash
git add src/transports/campfire.ts test/unit/campfire-send.test.ts
git commit -m "feat: add campfire outbound send"
```

---

### Task 9: Database schema and migrations

**Files:**
- Create: `drizzle.config.ts`, `src/db/schema.ts`, `src/db/client.ts`, `src/db/migrate.ts`, `test/integration/support.ts`, `docs/testing.md`
- Create (generated): `drizzle/*.sql`
- Test: `test/integration/schema.test.ts`

**Interfaces:**
- Consumes: `PostgresConfig` (Task 2)
- Produces:
  - `src/db/schema.ts`: `people`, `identities`, `items` Drizzle tables
  - `src/db/client.ts`: `type Db = NodePgDatabase<typeof schema>`, `interface DbHandle { db: Db; close(): Promise<void> }`, `function openDb(config: PostgresConfig): DbHandle`, `function urlFor(config: PostgresConfig): string`
  - `src/db/migrate.ts`: `function applyMigrations(db: Db): Promise<void>`
  - `test/integration/support.ts`: `function testDatabaseUrl(): string`, `function withDb(): Promise<{ db: Db; close(): Promise<void> }>`, `function truncateAll(db: Db): Promise<void>`

- [ ] **Step 1: Write `docs/testing.md`**

```markdown
# Running the tests

Unit tests need nothing:

    npm test

Integration tests need a real Postgres. Start one:

    docker run --rm -d --name squirrel-test-db -p 55432:5432 \
      -e POSTGRES_USER=squirrel -e POSTGRES_PASSWORD=squirrel \
      -e POSTGRES_DB=squirrel_test postgres:17-alpine

Then:

    TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test \
      npm run test:integration

Stop it with `docker rm -f squirrel-test-db`.

The integration suite refuses to run without `TEST_DATABASE_URL` rather than
skipping, so a missing variable in CI is a failure rather than a green run over
nothing.
```

- [ ] **Step 2: Write `src/db/schema.ts`**

```ts
import { sql } from "drizzle-orm";
import { bigint, bigserial, index, jsonb, pgTable, text, timestamp, uniqueIndex } from "drizzle-orm/pg-core";

/**
 * A person is a row because a transport id is a transport-local coordinate:
 * "user 1, in Campfire" says nothing about the same human on Matrix.
 */
export const people = pgTable("people", {
  id: bigserial("id", { mode: "number" }).primaryKey(),
  handle: text("handle").notNull().unique(),
  createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
});

export const identities = pgTable(
  "identities",
  {
    id: bigserial("id", { mode: "number" }).primaryKey(),
    personId: bigint("person_id", { mode: "number" })
      .notNull()
      .references(() => people.id),
    transport: text("transport").notNull(),
    externalId: text("external_id").notNull(),
    createdAt: timestamp("created_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => ({
    unique: uniqueIndex("identities_transport_external_id_key").on(table.transport, table.externalId),
  }),
);

export const items = pgTable(
  "items",
  {
    id: bigserial("id", { mode: "number" }).primaryKey(),
    transport: text("transport").notNull(),
    // Identifiers are text, not bigint: Matrix room ids are strings and
    // Telegram chat ids are signed 64-bit. Nothing arithmetic is done with them.
    externalId: text("external_id"),
    conversationId: text("conversation_id"),
    // Kept alongside personId on purpose. personId is an interpretation and can
    // be absent or wrong; senderId is what the transport actually said.
    senderId: text("sender_id"),
    personId: bigint("person_id", { mode: "number" }).references(() => people.id),
    rawText: text("raw_text").notNull(),
    payload: jsonb("payload").notNull(),
    receivedAt: timestamp("received_at", { withTimezone: true }).notNull(),
    insertedAt: timestamp("inserted_at", { withTimezone: true }).notNull().defaultNow(),
  },
  (table) => ({
    // Partial, because the fail-open path has no external id to be unique on.
    external: uniqueIndex("items_transport_external_id_key")
      .on(table.transport, table.externalId)
      .where(sql`${table.externalId} is not null`),
    received: index("items_received_at_idx").on(table.receivedAt),
  }),
);
```

- [ ] **Step 3: Write `src/db/client.ts`**

```ts
import { drizzle, type NodePgDatabase } from "drizzle-orm/node-postgres";
import pg from "pg";

import type { PostgresConfig } from "../config.js";
import * as schema from "./schema.js";

export type Db = NodePgDatabase<typeof schema>;

export interface DbHandle {
  readonly db: Db;
  close(): Promise<void>;
}

export function urlFor(config: PostgresConfig): string {
  const user = encodeURIComponent(config.user);
  const password = encodeURIComponent(config.password);
  return `postgres://${user}:${password}@${config.host}:${config.port}/${config.database}`;
}

export function openDb(config: PostgresConfig): DbHandle {
  return fromUrl(urlFor(config));
}

export function fromUrl(url: string): DbHandle {
  const pool = new pg.Pool({ connectionString: url, max: 4 });
  return { db: drizzle(pool, { schema }), close: () => pool.end() };
}
```

- [ ] **Step 4: Write `drizzle.config.ts` and `src/db/migrate.ts`**

`drizzle.config.ts`:
```ts
import { defineConfig } from "drizzle-kit";

export default defineConfig({
  dialect: "postgresql",
  schema: "./src/db/schema.ts",
  out: "./drizzle",
});
```

`src/db/migrate.ts`:
```ts
import { migrate } from "drizzle-orm/node-postgres/migrator";

import type { Db } from "./client.js";

export function applyMigrations(db: Db): Promise<void> {
  return migrate(db, { migrationsFolder: "drizzle" });
}
```

- [ ] **Step 5: Generate the migration SQL**

Run: `npm run db:generate`
Expected: a new `drizzle/0000_*.sql` plus `drizzle/meta/`. Open the SQL and confirm it creates `people`, `identities`, `items`, and the partial unique index `items_transport_external_id_key`.

- [ ] **Step 6: Write `test/integration/support.ts`**

```ts
import { sql } from "drizzle-orm";

import { type Db, fromUrl } from "../../src/db/client.js";
import { applyMigrations } from "../../src/db/migrate.js";

/** Refuses rather than skips, so an unset variable in CI fails loudly. */
export function testDatabaseUrl(): string {
  const url = process.env.TEST_DATABASE_URL;
  if (!url) throw new Error("TEST_DATABASE_URL is required — see docs/testing.md");
  return url;
}

export async function withDb(): Promise<{ db: Db; close: () => Promise<void> }> {
  const handle = fromUrl(testDatabaseUrl());
  await applyMigrations(handle.db);
  await truncateAll(handle.db);
  return handle;
}

export async function truncateAll(db: Db): Promise<void> {
  await db.execute(sql`truncate table items, identities, people restart identity cascade`);
}
```

- [ ] **Step 7: Write the failing test**

`test/integration/schema.test.ts`:
```ts
import { sql } from "drizzle-orm";
import { afterAll, beforeAll, describe, expect, it } from "vitest";

import type { Db } from "../../src/db/client.js";
import { items, people } from "../../src/db/schema.js";
import { withDb } from "./support.js";

let db: Db;
let close: () => Promise<void>;

beforeAll(async () => {
  ({ db, close } = await withDb());
});

afterAll(async () => {
  await close();
});

describe("schema", () => {
  it("applies migrations and creates the three tables", async () => {
    const result = await db.execute(
      sql`select table_name from information_schema.tables where table_schema = 'public' order by table_name`,
    );
    const names = result.rows.map((row) => row.table_name);
    expect(names).toContain("people");
    expect(names).toContain("identities");
    expect(names).toContain("items");
  });

  it("stores an item with nulls in every fail-open column", async () => {
    await db.insert(items).values({
      transport: "campfire",
      externalId: null,
      conversationId: null,
      senderId: null,
      personId: null,
      rawText: "",
      payload: { unparseable: "nonsense" },
      receivedAt: new Date("2026-08-14T09:31:04.512Z"),
    });
    expect(await db.select().from(items)).toHaveLength(1);
  });

  it("allows many items with a null external id, since the index is partial", async () => {
    const row = {
      transport: "campfire",
      externalId: null,
      rawText: "",
      payload: {},
      receivedAt: new Date(),
    };
    await db.insert(items).values([row, row]);
    expect(await db.select().from(items)).toHaveLength(2);
  });

  it("rejects a duplicate handle", async () => {
    await db.insert(people).values({ handle: "owner" });
    await expect(db.insert(people).values({ handle: "owner" })).rejects.toThrow();
  });
});
```

- [ ] **Step 8: Start Postgres and run the test**

```bash
docker run --rm -d --name squirrel-test-db -p 55432:5432 \
  -e POSTGRES_USER=squirrel -e POSTGRES_PASSWORD=squirrel \
  -e POSTGRES_DB=squirrel_test postgres:17-alpine
```

Run: `TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test npm run test:integration`
Expected: 4 tests PASS.

- [ ] **Step 9: Commit**

```bash
git add drizzle drizzle.config.ts src/db test/integration docs/testing.md
git commit -m "feat: add the database schema and migrations"
```

---

### Task 10: People and identities

**Files:**
- Create: `src/people/identities.ts`
- Test: `test/integration/identities.test.ts`

**Interfaces:**
- Consumes: `Db` (Task 9), `people`/`identities` tables (Task 9)
- Produces:
  - `interface IdentitySeed { transport: string; externalId: string }`
  - `function seedOwner(db: Db, handle: string, seeds: readonly IdentitySeed[]): Promise<number>` — returns the owner's `person_id`
  - `function resolvePerson(db: Db, transport: string, externalId: string | null): Promise<number | null>`

- [ ] **Step 1: Write the failing tests**

`test/integration/identities.test.ts`:
```ts
import { afterAll, beforeAll, beforeEach, describe, expect, it } from "vitest";

import type { Db } from "../../src/db/client.js";
import { identities, people } from "../../src/db/schema.js";
import { resolvePerson, seedOwner } from "../../src/people/identities.js";
import { truncateAll, withDb } from "./support.js";

let db: Db;
let close: () => Promise<void>;

beforeAll(async () => {
  ({ db, close } = await withDb());
});

beforeEach(async () => {
  await truncateAll(db);
});

afterAll(async () => {
  await close();
});

describe("seedOwner", () => {
  it("creates the person and their identities", async () => {
    const personId = await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);

    expect(await db.select().from(people)).toHaveLength(1);
    expect(await db.select().from(identities)).toMatchObject([
      { personId, transport: "campfire", externalId: "1" },
    ]);
  });

  it("is idempotent, so every boot may run it", async () => {
    const seeds = [{ transport: "campfire", externalId: "1" }];
    const first = await seedOwner(db, "ronald", seeds);
    const second = await seedOwner(db, "ronald", seeds);

    expect(second).toBe(first);
    expect(await db.select().from(people)).toHaveLength(1);
    expect(await db.select().from(identities)).toHaveLength(1);
  });

  it("adds a second transport's identity to the same person", async () => {
    await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);
    const personId = await seedOwner(db, "ronald", [
      { transport: "campfire", externalId: "1" },
      { transport: "matrix", externalId: "@me:example" },
    ]);

    const rows = await db.select().from(identities);
    expect(rows).toHaveLength(2);
    expect(rows.every((row) => row.personId === personId)).toBe(true);
  });
});

describe("resolvePerson", () => {
  it("resolves a seeded identity", async () => {
    const personId = await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);
    expect(await resolvePerson(db, "campfire", "1")).toBe(personId);
  });

  it("returns null for an unknown identity", async () => {
    await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);
    expect(await resolvePerson(db, "campfire", "999")).toBeNull();
  });

  it("returns null for a null external id, which is the fail-open case", async () => {
    expect(await resolvePerson(db, "campfire", null)).toBeNull();
  });

  it("does not match the same external id under another transport", async () => {
    await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);
    expect(await resolvePerson(db, "matrix", "1")).toBeNull();
  });

  // The test that keeps the guard from being decorative: auto-creating a
  // person on first sight would re-admit whoever the allowlist turned away.
  it("never creates a person for an unknown identity", async () => {
    await resolvePerson(db, "campfire", "999");
    expect(await db.select().from(people)).toEqual([]);
    expect(await db.select().from(identities)).toEqual([]);
  });
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test npm run test:integration -- identities`
Expected: FAIL — cannot resolve `../../src/people/identities.js`.

- [ ] **Step 3: Write the implementation**

`src/people/identities.ts`:
```ts
import { and, eq } from "drizzle-orm";

import type { Db } from "../db/client.js";
import { identities, people } from "../db/schema.js";

export interface IdentitySeed {
  readonly transport: string;
  readonly externalId: string;
}

/**
 * Seeding is declarative rather than administrative: the desired state lives in
 * configuration, in Git, and every boot reconciles to it. There is no admin
 * screen to forget about.
 */
export async function seedOwner(
  db: Db,
  handle: string,
  seeds: readonly IdentitySeed[],
): Promise<number> {
  return db.transaction(async (tx) => {
    const inserted = await tx
      .insert(people)
      .values({ handle })
      .onConflictDoNothing({ target: people.handle })
      .returning({ id: people.id });

    const personId =
      inserted[0]?.id ??
      (await tx.select({ id: people.id }).from(people).where(eq(people.handle, handle)))[0]?.id;

    if (personId === undefined) throw new Error(`could not seed person ${handle}`);

    for (const seed of seeds) {
      await tx
        .insert(identities)
        .values({ personId, transport: seed.transport, externalId: seed.externalId })
        .onConflictDoNothing({ target: [identities.transport, identities.externalId] });
    }

    return personId;
  });
}

/**
 * Resolution happens in the drain loop, never on the request path — a person
 * lookup is a database read, and the request path not touching Postgres is what
 * makes an outage survivable.
 *
 * An unknown identity is null. It is never created: auto-vivifying a person on
 * first sight would quietly re-admit anyone the guard had just turned away.
 */
export async function resolvePerson(
  db: Db,
  transport: string,
  externalId: string | null,
): Promise<number | null> {
  if (externalId === null) return null;

  const rows = await db
    .select({ personId: identities.personId })
    .from(identities)
    .where(and(eq(identities.transport, transport), eq(identities.externalId, externalId)))
    .limit(1);

  return rows[0]?.personId ?? null;
}
```

- [ ] **Step 4: Run the tests**

Run: `TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test npm run test:integration && npm run typecheck && npm run lint`
Expected: all integration tests PASS, typecheck clean, lint clean.

- [ ] **Step 5: Commit**

```bash
git add src/people test/integration/identities.test.ts
git commit -m "feat: add people and transport identities"
```

---

### Task 11: The drain loop

**Files:**
- Create: `src/capture/drain.ts`
- Test: `test/integration/drain.test.ts`

**Interfaces:**
- Consumes: `Spool` (Task 4), `Capture` (Task 3), `Db` (Task 9), `resolvePerson` (Task 10), `items` (Task 9)
- Produces:
  - `interface DrainResult { inserted: number; quarantined: number; deferred: number }`
  - `interface Drain { drainOnce(): Promise<DrainResult>; start(): void; stop(): Promise<void> }`
  - `function createDrain(options: { spool: Spool; db: Db; intervalMs: number; maxBackoffMs?: number; onError?: (error: unknown) => void }): Drain`

- [ ] **Step 1: Write the failing tests**

`test/integration/drain.test.ts`:
```ts
import { mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import type { Capture } from "../../src/capture/capture.js";
import { createDrain } from "../../src/capture/drain.js";
import { type Spool, openSpool } from "../../src/capture/spool.js";
import { type Db, fromUrl } from "../../src/db/client.js";
import { items } from "../../src/db/schema.js";
import { seedOwner } from "../../src/people/identities.js";
import { truncateAll, withDb } from "./support.js";

let db: Db;
let close: () => Promise<void>;
let spool: Spool;
let dir: string;

beforeAll(async () => {
  ({ db, close } = await withDb());
});

beforeEach(async () => {
  await truncateAll(db);
  dir = await mkdtemp(join(tmpdir(), "squirrel-drain-"));
  spool = await openSpool(dir);
});

afterAll(async () => {
  await close();
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

describe("drain", () => {
  it("inserts a spooled capture and deletes the file", async () => {
    await spool.write(capture());
    const drain = createDrain({ spool, db, intervalMs: 1000 });

    expect(await drain.drainOnce()).toEqual({ inserted: 1, quarantined: 0, deferred: 0 });

    const rows = await db.select().from(items);
    expect(rows).toMatchObject([
      { transport: "campfire", externalId: "42", conversationId: "7", rawText: "buy milk" },
    ]);
    expect(await spool.list()).toEqual([]);
  });

  it("resolves a seeded identity to a person", async () => {
    const personId = await seedOwner(db, "ronald", [{ transport: "campfire", externalId: "1" }]);
    await spool.write(capture());
    await createDrain({ spool, db, intervalMs: 1000 }).drainOnce();

    expect((await db.select().from(items))[0]?.personId).toBe(personId);
  });

  // A capture is never held hostage to knowing whose it was.
  it("stores a capture from an unknown identity with a null person", async () => {
    const onUnknownIdentity = vi.fn();
    await spool.write(capture({ senderId: "999" }));
    const drain = createDrain({ spool, db, intervalMs: 1000, onUnknownIdentity });

    expect(await drain.drainOnce()).toMatchObject({ inserted: 1 });
    expect((await db.select().from(items))[0]?.personId).toBeNull();
    expect(await spool.list()).toEqual([]);
    expect(onUnknownIdentity).toHaveBeenCalledWith("campfire", "999");
  });

  it("inserts one row for a redelivered message", async () => {
    const drain = createDrain({ spool, db, intervalMs: 1000 });
    await spool.write(capture());
    await drain.drainOnce();
    await spool.write(capture());
    await drain.drainOnce();

    expect(await db.select().from(items)).toHaveLength(1);
  });

  it("keeps the same external id under two transports apart", async () => {
    await spool.write(capture());
    await spool.write(capture({ transport: "matrix" }));
    await createDrain({ spool, db, intervalMs: 1000 }).drainOnce();

    expect(await db.select().from(items)).toHaveLength(2);
  });

  it("defers everything and loses nothing while the database is unreachable", async () => {
    const unreachable = fromUrl("postgres://nobody:nobody@127.0.0.1:1/squirrel");
    await spool.write(capture());
    const drain = createDrain({ spool, db: unreachable.db, intervalMs: 1000 });

    expect(await drain.drainOnce()).toMatchObject({ inserted: 0, deferred: 1 });
    expect(await spool.list()).toHaveLength(1);
    await unreachable.close();

    // And it lands once the database comes back.
    expect(await createDrain({ spool, db, intervalMs: 1000 }).drainOnce()).toMatchObject({
      inserted: 1,
    });
  });

  it("quarantines an unreadable file rather than retrying it forever", async () => {
    await writeFile(join(dir, "000000000000001-campfire-9.json"), "{ not json");
    const drain = createDrain({ spool, db, intervalMs: 1000 });

    expect(await drain.drainOnce()).toMatchObject({ inserted: 0, quarantined: 1 });
    expect(await spool.list()).toEqual([]);
    expect(await drain.drainOnce()).toMatchObject({ quarantined: 0 });
  });

  it("drains on an interval once started, and stops cleanly", async () => {
    await spool.write(capture());
    const drain = createDrain({ spool, db, intervalMs: 10 });
    drain.start();

    await vi.waitFor(async () => expect(await db.select().from(items)).toHaveLength(1));
    await drain.stop();
  });
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test npm run test:integration -- drain`
Expected: FAIL — cannot resolve `../../src/capture/drain.js`.

- [ ] **Step 3: Write the implementation**

`src/capture/drain.ts`:
```ts
import { sql } from "drizzle-orm";

import type { Db } from "../db/client.js";
import { items } from "../db/schema.js";
import { resolvePerson } from "../people/identities.js";
import type { Spool } from "./spool.js";

export interface DrainResult {
  readonly inserted: number;
  readonly quarantined: number;
  readonly deferred: number;
}

export interface Drain {
  drainOnce(): Promise<DrainResult>;
  start(): void;
  stop(): Promise<void>;
}

export interface DrainOptions {
  readonly spool: Spool;
  readonly db: Db;
  readonly intervalMs: number;
  readonly maxBackoffMs?: number;
  readonly onError?: (error: unknown) => void;
  /** Not an error. The row still lands; nobody knows whose it is yet. */
  readonly onUnknownIdentity?: (transport: string, senderId: string) => void;
}

/**
 * Permanent means retrying cannot help: the file is not readable as a capture,
 * or the row violates a constraint that will still be violated next time.
 * Everything else — connection refused, a failover in progress, an error nobody
 * anticipated — is transient, because deferring costs a second and quarantining
 * a good capture costs the thought.
 */
function isPermanent(error: unknown): boolean {
  if (error instanceof SyntaxError) return true;
  const code = (error as { code?: unknown }).code;
  return typeof code === "string" && (code.startsWith("22") || code.startsWith("23"));
}

export function createDrain(options: DrainOptions): Drain {
  const { spool, db, intervalMs } = options;
  const maxBackoffMs = options.maxBackoffMs ?? 30_000;
  const onError = options.onError ?? (() => {});

  let timer: NodeJS.Timeout | undefined;
  let running = false;
  let backoffMs = intervalMs;

  async function drainOnce(): Promise<DrainResult> {
    let inserted = 0;
    let quarantined = 0;
    let deferred = 0;

    for (const name of await spool.list()) {
      try {
        const capture = await spool.read(name);
        const personId = await resolvePerson(db, capture.transport, capture.senderId);
        if (personId === null && capture.senderId !== null) {
          options.onUnknownIdentity?.(capture.transport, capture.senderId);
        }

        await db
          .insert(items)
          .values({
            transport: capture.transport,
            externalId: capture.externalId,
            conversationId: capture.conversationId,
            senderId: capture.senderId,
            personId,
            rawText: capture.text,
            payload: capture.payload,
            receivedAt: capture.receivedAt,
          })
          // Redelivery is harmless. The window between inserting a row and
          // deleting its file is small, but it is real.
          //
          // targetWhere is mandatory, not decoration: the index is partial, and
          // Postgres rejects an ON CONFLICT target whose predicate does not
          // match the index with "no unique or exclusion constraint matching
          // the ON CONFLICT specification".
          .onConflictDoNothing({
            target: [items.transport, items.externalId],
            targetWhere: sql`${items.externalId} is not null`,
          });

        await spool.remove(name);
        inserted += 1;
      } catch (error) {
        onError(error);
        if (isPermanent(error)) {
          // Never deleted. A file that cannot be inserted must not spin
          // forever, and it must not disappear either.
          await spool.quarantine(name);
          quarantined += 1;
        } else {
          deferred += 1;
        }
      }
    }

    return { inserted, quarantined, deferred };
  }

  function schedule(delayMs: number): void {
    timer = setTimeout(() => {
      void tick();
    }, delayMs);
    timer.unref();
  }

  async function tick(): Promise<void> {
    if (!running) return;
    let result: DrainResult = { inserted: 0, quarantined: 0, deferred: 0 };
    try {
      result = await drainOnce();
    } catch (error) {
      onError(error);
      result = { inserted: 0, quarantined: 0, deferred: 1 };
    }
    backoffMs = result.deferred > 0 ? Math.min(backoffMs * 2, maxBackoffMs) : intervalMs;
    if (running) schedule(backoffMs);
  }

  return {
    drainOnce,
    start() {
      if (running) return;
      running = true;
      backoffMs = intervalMs;
      schedule(0);
    },
    async stop() {
      running = false;
      if (timer) clearTimeout(timer);
      timer = undefined;
    },
  };
}
```

- [ ] **Step 4: Run the tests**

Run: `TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test npm run test:integration && npm run typecheck && npm run lint`
Expected: all integration tests PASS, typecheck clean, lint clean.

- [ ] **Step 5: Commit**

```bash
git add src/capture/drain.ts test/integration/drain.test.ts
git commit -m "feat: add the drain loop from spool to postgres"
```

---

### Task 12: Boot wiring

**Files:**
- Create: `src/index.ts`
- Test: `test/integration/boot.test.ts`, `test/unit/fake-transport.test.ts`, `test/unit/boundary.test.ts`

**Interfaces:**
- Consumes: everything from Tasks 2–11
- Produces: `interface Squirrel { port: number; stop(): Promise<void> }`, `function boot(env: Record<string, string | undefined>): Promise<Squirrel>`

- [ ] **Step 1: Write the failing core-boundary test**

This is the test that proves the core does not know what Campfire is. If it ever fails to compile, the boundary has leaked.

`test/unit/fake-transport.test.ts`:
```ts
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
    const sink = createSink(spool, [
      { transport: "fake", conversationId: "room", senderId: "me" },
    ]);

    const transport = fakeTransport();
    await transport.start(sink, { post: () => {} });

    expect(await transport.deliver("water the plants")).toBe("stored");
    expect(await spool.list()).toHaveLength(1);
  });

  it("cannot send, and says so", () => {
    expect(fakeTransport().send).toBeNull();
  });
});
```

- [ ] **Step 2: Run it to verify it passes already**

Run: `npm test -- fake-transport`
Expected: 2 tests PASS. This test needs no new production code — that is the point. If it does not compile, the interfaces from Tasks 3 and 6 are wrong.

- [ ] **Step 3: Write the boundary test**

The one rule that keeps the two halves apart, asserted against the source tree rather than trusted to reviewers.

`test/unit/boundary.test.ts`:
```ts
import { readFile, readdir } from "node:fs/promises";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const CORE = ["src/capture", "src/people", "src/db"];

async function sourceFiles(dir: string): Promise<string[]> {
  const entries = await readdir(dir, { withFileTypes: true, recursive: true });
  return entries
    .filter((entry) => entry.isFile() && entry.name.endsWith(".ts"))
    .map((entry) => join(entry.parentPath, entry.name));
}

describe("the core does not know what a transport is", () => {
  it.each(CORE)("%s imports nothing from src/transports", async (dir) => {
    for (const file of await sourceFiles(dir)) {
      const source = await readFile(file, "utf8");
      expect(source, `${file} imports from transports`).not.toMatch(/from\s+"[^"]*transports\//);
    }
  });

  it("finds files to check, so a passing run is not an empty one", async () => {
    for (const dir of CORE) expect(await sourceFiles(dir)).not.toHaveLength(0);
  });
});
```

- [ ] **Step 4: Run it**

Run: `npm test -- boundary`
Expected: 4 tests PASS. Like the fake transport, this needs no production code — it asserts a property the earlier tasks already have.

- [ ] **Step 5: Write the failing boot test**

`test/integration/boot.test.ts`:
```ts
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import type { Db } from "../../src/db/client.js";
import { items } from "../../src/db/schema.js";
import { boot, type Squirrel } from "../../src/index.js";
import { truncateAll, withDb } from "./support.js";

const PAYLOAD = {
  user: { id: 1, name: "Ronald" },
  room: { id: 7, name: "Squirrel", path: "/rooms/7/3-abc/messages" },
  message: { id: 42, body: { html: "<div>buy milk</div>", plain: "buy milk" }, path: "/rooms/7/@42" },
};

let db: Db;
let close: () => Promise<void>;
let squirrel: Squirrel;
let spoolDir: string;

function envFor(url: string, overrides: Record<string, string> = {}) {
  const parsed = new URL(url);
  return {
    PORT: "0",
    SPOOL_DIR: spoolDir,
    DRAIN_INTERVAL_MS: "10",
    OWNER_HANDLE: "ronald",
    CAMPFIRE_CONVERSATION_ID: "7",
    CAMPFIRE_SENDER_ID: "1",
    POSTGRES_SERVER: parsed.hostname,
    POSTGRES_PORT: parsed.port,
    POSTGRES_DB: parsed.pathname.slice(1),
    POSTGRES_USER: decodeURIComponent(parsed.username),
    POSTGRES_PASSWORD: decodeURIComponent(parsed.password),
    ...overrides,
  };
}

async function post(body: string): Promise<Response> {
  return fetch(`http://127.0.0.1:${squirrel.port}/transports/campfire`, { method: "POST", body });
}

beforeAll(async () => {
  ({ db, close } = await withDb());
});

beforeEach(async () => {
  await truncateAll(db);
  spoolDir = await mkdtemp(join(tmpdir(), "squirrel-boot-"));
});

afterEach(async () => {
  await squirrel?.stop();
  await rm(spoolDir, { recursive: true, force: true });
});

afterAll(async () => {
  await close();
});

describe("boot", () => {
  it("captures a message end to end and answers with a squirrel", async () => {
    squirrel = await boot(envFor(process.env.TEST_DATABASE_URL as string));

    const response = await post(JSON.stringify(PAYLOAD));
    expect(await response.text()).toBe("🐿️");

    await vi.waitFor(async () => {
      const rows = await db.select().from(items);
      expect(rows).toMatchObject([{ rawText: "buy milk", senderId: "1" }]);
      expect(rows[0]?.personId).not.toBeNull();
    });
  });

  it("says nothing at all to a message from another room", async () => {
    squirrel = await boot(envFor(process.env.TEST_DATABASE_URL as string));

    const elsewhere = { ...PAYLOAD, room: { ...PAYLOAD.room, id: 8 } };
    const response = await post(JSON.stringify(elsewhere));

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toBeNull();
    expect(await response.text()).toBe("");
  });

  it("is healthy", async () => {
    squirrel = await boot(envFor(process.env.TEST_DATABASE_URL as string));
    const response = await fetch(`http://127.0.0.1:${squirrel.port}/healthz`);
    expect(await response.text()).toBe("ok");
  });

  // The whole point of spooling. An unreachable database at boot must not stop
  // a single capture from being accepted, because Campfire will not retry it.
  it("serves, captures and stays healthy with the database unreachable", async () => {
    squirrel = await boot(
      envFor(process.env.TEST_DATABASE_URL as string, { POSTGRES_PORT: "1", POSTGRES_SERVER: "127.0.0.1" }),
    );

    const response = await post(JSON.stringify(PAYLOAD));
    expect(await response.text()).toBe("🐿️");
    expect(await (await fetch(`http://127.0.0.1:${squirrel.port}/healthz`)).text()).toBe("ok");
  });
});
```

- [ ] **Step 6: Run the test to verify it fails**

Run: `TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test npm run test:integration -- boot`
Expected: FAIL — cannot resolve `../../src/index.js`.

- [ ] **Step 7: Write the implementation**

`src/index.ts`:
```ts
import { createDrain, type Drain } from "./capture/drain.js";
import type { Allow } from "./capture/policy.js";
import { createSink } from "./capture/sink.js";
import { openSpool, type Spool } from "./capture/spool.js";
import { type Config, loadConfig } from "./config.js";
import { type DbHandle, openDb } from "./db/client.js";
import { applyMigrations } from "./db/migrate.js";
import { createHttp, startHttp } from "./http.js";
import { type IdentitySeed, seedOwner } from "./people/identities.js";
import { createCampfireTransport } from "./transports/campfire.js";
import type { Stop, Transport } from "./transports/transport.js";

export interface Squirrel {
  readonly port: number;
  stop(): Promise<void>;
}

function log(event: string, detail: Record<string, unknown> = {}): void {
  process.stdout.write(`${JSON.stringify({ event, ...detail })}\n`);
}

function transportsFrom(config: Config): Transport[] {
  const built: Transport[] = [];
  if (config.campfire !== null) built.push(createCampfireTransport(config.campfire));
  return built;
}

function allowsFrom(config: Config): Allow[] {
  if (config.campfire === null) return [];
  return [
    {
      transport: "campfire",
      conversationId: config.campfire.conversationId,
      senderId: config.campfire.senderId,
    },
  ];
}

function seedsFrom(config: Config): IdentitySeed[] {
  if (config.campfire === null) return [];
  return [{ transport: "campfire", externalId: config.campfire.senderId }];
}

/**
 * Postgres in the background, on purpose.
 *
 * Migrating before serving means a database outage during a pod restart
 * produces a service that refuses connections — and every message sent in that
 * window is gone, because Campfire does not retry. So the server binds first,
 * transports start, and the database is caught up with afterwards.
 */
async function connectAndDrain(
  config: Config,
  dbHandle: DbHandle,
  drainRef: { drain?: Drain },
  spool: Spool,
): Promise<void> {
  try {
    await applyMigrations(dbHandle.db);
    await seedOwner(dbHandle.db, config.ownerHandle, seedsFrom(config));
    const drain = createDrain({
      spool,
      db: dbHandle.db,
      intervalMs: config.drainIntervalMs,
      onError: (error) => log("drain.error", { error: String(error) }),
      onUnknownIdentity: (transport, senderId) =>
        log("identity.unknown", { transport, senderId }),
    });
    drainRef.drain = drain;
    drain.start();
    log("db.ready");
  } catch (error) {
    log("db.unavailable", { error: String(error) });
    setTimeout(() => {
      void connectAndDrain(config, dbHandle, drainRef, spool);
    }, config.drainIntervalMs).unref();
  }
}

export async function boot(env: Record<string, string | undefined>): Promise<Squirrel> {
  const config = loadConfig(env);

  const spool = await openSpool(config.spoolDir);
  const swept = await spool.sweep();
  if (swept > 0) log("spool.swept", { files: swept });

  const sink = createSink(spool, allowsFrom(config), {
    onError: (error) => log("spool.write_failed", { error: String(error) }),
    // Silent in the room, visible here. Being added to a room Squirrel does not
    // belong in should show up somewhere.
    onIgnored: (capture) =>
      log("capture.ignored", {
        transport: capture.transport,
        conversationId: capture.conversationId,
        senderId: capture.senderId,
      }),
  });

  const { app, mount } = createHttp(spool);
  const stops: Stop[] = [];
  for (const transport of transportsFrom(config)) {
    stops.push(await transport.start(sink, mount));
    log("transport.started", { transport: transport.name, sends: transport.send !== null });
  }

  const serving = await startHttp(app, config.port);
  log("http.listening", { port: serving.port });

  const dbHandle = openDb(config.postgres);
  const drainRef: { drain?: Drain } = {};
  void connectAndDrain(config, dbHandle, drainRef, spool);

  return {
    port: serving.port,
    async stop() {
      await drainRef.drain?.stop();
      for (const stop of stops) await stop();
      await serving.close();
      await dbHandle.close();
    },
  };
}

// Entry point when run as a program rather than imported by a test.
if (process.argv[1]?.endsWith("index.js")) {
  boot(process.env).catch((error: unknown) => {
    process.stderr.write(`${JSON.stringify({ event: "boot.failed", error: String(error) })}\n`);
    process.exit(1);
  });
}
```

- [ ] **Step 8: Run everything**

Run: `npm test && TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test npm run test:integration && npm run typecheck && npm run lint`
Expected: all tests PASS, typecheck clean, lint clean.

- [ ] **Step 9: Commit**

```bash
git add src/index.ts test/integration/boot.test.ts test/unit/fake-transport.test.ts test/unit/boundary.test.ts
git commit -m "feat: wire boot so the server serves before postgres is touched"
```

---

### Task 13: Container image and CI

**Files:**
- Create: `Dockerfile`, `.github/workflows/ci.yaml`
- Modify: `README.md`

**Interfaces:**
- Consumes: `npm run build`, `npm test`, `npm run test:integration` (Tasks 1–12)
- Produces: `ghcr.io/ronaldlokers/squirrel` on `v*` tags, `linux/amd64` and `linux/arm64`

- [ ] **Step 1: Write the `Dockerfile`**

```dockerfile
# Debian slim rather than Alpine, matching stringer: musl gives up the
# better-trodden arm64 path, and the size it saves is nothing on three Pis
# with NVMe.
FROM node:24-bookworm-slim AS build

WORKDIR /app
COPY package.json package-lock.json tsconfig.json ./
RUN npm ci
COPY src ./src
COPY test ./test
RUN npm run build && npm prune --omit=dev

FROM node:24-bookworm-slim

WORKDIR /app
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/dist/src ./dist/src
# Migrations are read from disk at boot, so they ship with the image rather
# than being applied by a separate Job that Flux would have to reconcile.
COPY drizzle ./drizzle
COPY package.json ./

# Nothing here needs to write outside the spool volume, and nothing needs a name.
USER node
CMD ["node", "dist/src/index.js"]
```

- [ ] **Step 2: Build the image locally and check it starts**

```bash
docker build -t squirrel:ci .
docker run --rm squirrel:ci node -e "console.log('ok')"
```
Expected: `ok`.

- [ ] **Step 3: Write `.github/workflows/ci.yaml`**

```yaml
name: CI

on:
  pull_request:
  push:
    branches:
      - main
    tags:
      - "v*"

permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-latest

    # The integration suite refuses to run without TEST_DATABASE_URL rather
    # than skipping, so a missing service here fails the build instead of
    # producing a green run over nothing.
    services:
      postgres:
        image: postgres:17-alpine
        env:
          POSTGRES_USER: squirrel
          POSTGRES_PASSWORD: squirrel
          POSTGRES_DB: squirrel_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 5s
          --health-timeout 5s
          --health-retries 10

    steps:
      - uses: actions/checkout@v5

      - uses: actions/setup-node@v4
        with:
          node-version: "24"
          cache: npm

      - run: npm ci
      - run: npm run lint
      - run: npm run typecheck
      - run: npm test
      - run: npm run test:integration
        env:
          TEST_DATABASE_URL: postgres://squirrel:squirrel@127.0.0.1:5432/squirrel_test

  image:
    runs-on: ubuntu-latest
    needs: test
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v5

      - uses: docker/setup-qemu-action@v3
      - uses: docker/setup-buildx-action@v3

      - name: Build for this architecture
        uses: docker/build-push-action@v6
        with:
          context: .
          load: true
          tags: squirrel:ci

      - name: Log in to ghcr
        if: startsWith(github.ref, 'refs/tags/v')
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      # Both architectures are mandatory: staging is amd64 and production is
      # arm64, so a single-arch image would pass in staging and fail on the Pis.
      - name: Build and push
        if: startsWith(github.ref, 'refs/tags/v')
        uses: docker/build-push-action@v6
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ghcr.io/${{ github.repository }}:${{ github.ref_name }}
            ghcr.io/${{ github.repository }}:latest
```

- [ ] **Step 4: Add a running section to `README.md`**

Append:

````markdown
## Running it

    npm ci
    npm run build
    node dist/src/index.js

Configuration is environment only; see the table in the design doc. The one
setting no code can check is `CAMPFIRE_CONVERSATION_ID`, which **must** name a
direct room — the webhook payload carries no room-type field, so Squirrel
cannot verify it.

Tests: see [`docs/testing.md`](docs/testing.md).
````

- [ ] **Step 5: Run everything one last time**

Run: `npm run lint && npm run typecheck && npm test && TEST_DATABASE_URL=postgres://squirrel:squirrel@127.0.0.1:55432/squirrel_test npm run test:integration`
Expected: all clean, all PASS.

- [ ] **Step 6: Commit and clean up**

```bash
git add Dockerfile .github/workflows/ci.yaml README.md
git commit -m "ci: build and test the image on both architectures"
docker rm -f squirrel-test-db
```

---

## Out of scope

Named so nobody adds them while passing through: `?` and `done` handling, the scheduler and firings, chores, the nightly clarification pass, LLM parsing, Home Assistant sensor evidence, context triggers, any web UI, authentication beyond the network boundary, `pg-boss`, and any transport other than Campfire.

The Kubernetes manifests are also out of scope for this repository. They belong in `ronaldlokers/homelab` as a separate change: the `Database` CR, the SOPS `pg-role-squirrel` Secret, the Deployment with its spool PVC, the ClusterIP Service, and the NetworkPolicy. Phase 1 ships **no** `CAMPFIRE_BOT_KEY` Secret, so `send` stays null in production until firings need it.
