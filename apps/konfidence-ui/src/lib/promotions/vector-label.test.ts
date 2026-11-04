import { describe, expect, it } from "vitest";
import { vectorLabel } from "./vector-label.ts";

describe("vector presentation", () => {
  it.each([
    ["v2.14.0", "v2.14.0"],
    [
      "https://registry.example.com:5000/konfidence//delivery-vector:2026.9.29-201426000Z",
      "delivery-vector:2026.9.29-201426000Z",
    ],
    ["registry.example.com:5000/project/image:v1", "image:v1"],
    ["oci://registry.example.com/project/image:v1", "image:v1"],
    ["registry.example.com/project/image@sha256:abcdef0123456789", "image@sha256:abcdef0123456789"],
    ["https://registry.example.com:5000", "https://registry.example.com:5000"],
    ["https://registry.example.com/project/", "https://registry.example.com/project/"],
    [" arbitrary-vector ", "arbitrary-vector"],
    ["", ""],
  ])("formats %s without fabricating a version", (reference, expected) => {
    expect(vectorLabel(reference)).toBe(expected);
  });
});
