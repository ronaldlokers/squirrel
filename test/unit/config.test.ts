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

  // Otherwise this boots with zero transports, a healthy /healthz, and every
  // webhook answered by the bare 404 — which Campfire uploads as a file.
  it("rejects an unrecognised transport name", () => {
    expect(() => loadConfig({ ...minimal, TRANSPORTS: "campfre" })).toThrow(ConfigError);
    expect(() => loadConfig({ ...minimal, TRANSPORTS: "campfre" })).toThrow(/campfre/);
  });

  it("rejects an unrecognised transport name alongside a valid one", () => {
    expect(() => loadConfig({ ...minimal, TRANSPORTS: "campfire,matrix" })).toThrow(/matrix/);
  });
});
