import { test, expect } from "bun:test";
import { checkCatalogue } from "./catalogue-checks.ts";

test("closed generated runtime catalogue", () => {
  expect(checkCatalogue()).toHaveLength(9);
});
