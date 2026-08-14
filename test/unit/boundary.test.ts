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
