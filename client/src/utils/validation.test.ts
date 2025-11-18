import { describe, expect, it } from "vitest";
import { ValidationChain, validators } from "./validation";

describe("validators.required", () => {
  it("passes for non-empty string", () => {
    const validator = validators.required("Username");
    expect(validator("john").isValid).toBe(true);
  });

  it("fails for empty string", () => {
    const validator = validators.required("Username");
    const result = validator("");
    expect(result.isValid).toBe(false);
    expect(result.errorMessage).toBe("Username is required");
  });
});

describe("validators.email", () => {
  it("passes for valid email", () => {
    const validator = validators.email();
    expect(validator("test@example.com").isValid).toBe(true);
  });

  it("fails for invalid email", () => {
    const validator = validators.email();
    expect(validator("notanemail").isValid).toBe(false);
  });
});

describe("validators.passwordStrength", () => {
  it("passes for strong password", () => {
    const validator = validators.passwordStrength();
    expect(validator("Password123").isValid).toBe(true);
  });

  it("fails for weak password", () => {
    const validator = validators.passwordStrength();
    expect(validator("weak").isValid).toBe(false);
  });
});

describe("ValidationChain", () => {
  it("chains multiple validators together", () => {
    const chain = new ValidationChain().required("Password").passwordStrength();

    expect(chain.validate("").isValid).toBe(false);
    expect(chain.validate("Password123").isValid).toBe(true);
  });
});
