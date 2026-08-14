import { describe, expect, it } from "vitest";

import { NAME } from "../../src/version.js";

describe("version", () => {
  it("names the service", () => {
    expect(NAME).toBe("squirrel");
  });
});
