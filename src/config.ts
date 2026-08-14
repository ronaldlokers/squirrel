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
