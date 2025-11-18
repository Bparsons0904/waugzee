import { describe, expect, it } from "vitest";
import { ValidationChain, createValidation, runValidators, validators } from "./validation";

describe("validators.required", () => {
  it("passes for non-empty string", () => {
    const validator = validators.required("Username");
    const result = validator("john");
    expect(result.isValid).toBe(true);
  });

  it("fails for empty string", () => {
    const validator = validators.required("Username");
    const result = validator("");
    expect(result.isValid).toBe(false);
    expect(result.errorMessage).toBe("Username is required");
  });

  it("fails for whitespace only", () => {
    const validator = validators.required("Email");
    const result = validator("   ");
    expect(result.isValid).toBe(false);
    expect(result.errorMessage).toBe("Email is required");
  });

  it("uses default field name", () => {
    const validator = validators.required();
    const result = validator("");
    expect(result.errorMessage).toBe("Field is required");
  });
});

describe("validators.minLength", () => {
  it("passes for string meeting minimum length", () => {
    const validator = validators.minLength(5, "Password");
    const result = validator("password123");
    expect(result.isValid).toBe(true);
  });

  it("fails for string below minimum length", () => {
    const validator = validators.minLength(5, "Password");
    const result = validator("pass");
    expect(result.isValid).toBe(false);
    expect(result.errorMessage).toBe("Password must be at least 5 characters");
  });

  it("passes for exact minimum length", () => {
    const validator = validators.minLength(5, "Code");
    const result = validator("12345");
    expect(result.isValid).toBe(true);
  });

  it("uses default field name", () => {
    const validator = validators.minLength(3);
    const result = validator("ab");
    expect(result.errorMessage).toBe("Field must be at least 3 characters");
  });
});

describe("validators.maxLength", () => {
  it("passes for string within maximum length", () => {
    const validator = validators.maxLength(10, "Username");
    const result = validator("john");
    expect(result.isValid).toBe(true);
  });

  it("fails for string exceeding maximum length", () => {
    const validator = validators.maxLength(10, "Username");
    const result = validator("verylongusername");
    expect(result.isValid).toBe(false);
    expect(result.errorMessage).toBe("Username must be no more than 10 characters");
  });

  it("passes for exact maximum length", () => {
    const validator = validators.maxLength(5, "Code");
    const result = validator("12345");
    expect(result.isValid).toBe(true);
  });

  it("uses default field name", () => {
    const validator = validators.maxLength(5);
    const result = validator("toolong");
    expect(result.errorMessage).toBe("Field must be no more than 5 characters");
  });
});

describe("validators.email", () => {
  it("passes for valid email", () => {
    const validator = validators.email();
    expect(validator("test@example.com").isValid).toBe(true);
    expect(validator("user.name@domain.co.uk").isValid).toBe(true);
  });

  it("fails for invalid email formats", () => {
    const validator = validators.email();
    expect(validator("notanemail").isValid).toBe(false);
    expect(validator("missing@domain").isValid).toBe(false);
    expect(validator("@nodomain.com").isValid).toBe(false);
    expect(validator("no@.com").isValid).toBe(false);
  });

  it("provides helpful error message", () => {
    const validator = validators.email();
    const result = validator("invalid");
    expect(result.errorMessage).toBe("Please enter a valid email address");
  });
});

describe("validators.pattern", () => {
  it("passes for matching pattern", () => {
    const validator = validators.pattern(/^\d{5}$/, "Must be 5 digits");
    expect(validator("12345").isValid).toBe(true);
  });

  it("fails for non-matching pattern", () => {
    const validator = validators.pattern(/^\d{5}$/, "Must be 5 digits");
    const result = validator("1234");
    expect(result.isValid).toBe(false);
    expect(result.errorMessage).toBe("Must be 5 digits");
  });

  it("handles complex patterns", () => {
    const validator = validators.pattern(/^[A-Z]{2}\d{4}$/, "Format: AB1234");
    expect(validator("AB1234").isValid).toBe(true);
    expect(validator("ab1234").isValid).toBe(false);
  });
});

describe("validators.passwordStrength", () => {
  it("passes for strong password", () => {
    const validator = validators.passwordStrength();
    expect(validator("Password123").isValid).toBe(true);
    expect(validator("Str0ngP@ss").isValid).toBe(true);
  });

  it("fails for password too short", () => {
    const validator = validators.passwordStrength();
    const result = validator("Pass1");
    expect(result.isValid).toBe(false);
    expect(result.errorMessage).toBe("Password must be at least 8 characters long");
  });

  it("fails for password without lowercase", () => {
    const validator = validators.passwordStrength();
    const result = validator("PASSWORD123");
    expect(result.isValid).toBe(false);
    expect(result.errorMessage).toBe("Password must contain at least one lowercase letter");
  });

  it("fails for password without uppercase", () => {
    const validator = validators.passwordStrength();
    const result = validator("password123");
    expect(result.isValid).toBe(false);
    expect(result.errorMessage).toBe("Password must contain at least one uppercase letter");
  });

  it("fails for password without number", () => {
    const validator = validators.passwordStrength();
    const result = validator("PasswordOnly");
    expect(result.isValid).toBe(false);
    expect(result.errorMessage).toBe("Password must contain at least one number");
  });
});

describe("runValidators", () => {
  it("passes when all validators pass", () => {
    const validatorList = [validators.required("Field"), validators.minLength(3, "Field")];
    const result = runValidators("test", validatorList);
    expect(result.isValid).toBe(true);
  });

  it("fails on first validator failure", () => {
    const validatorList = [validators.required("Field"), validators.minLength(10, "Field")];
    const result = runValidators("short", validatorList);
    expect(result.isValid).toBe(false);
    expect(result.errorMessage).toBe("Field must be at least 10 characters");
  });

  it("returns first error encountered", () => {
    const validatorList = [
      validators.required("Field"),
      validators.minLength(5, "Field"),
      validators.maxLength(3, "Field"),
    ];
    const result = runValidators("", validatorList);
    expect(result.errorMessage).toBe("Field is required");
  });

  it("handles empty validator list", () => {
    const result = runValidators("anything", []);
    expect(result.isValid).toBe(true);
  });
});

describe("ValidationChain", () => {
  it("chains required and minLength validators", () => {
    const chain = new ValidationChain().required("Username").minLength(3, "Username");
    expect(chain.validate("ab").isValid).toBe(false);
    expect(chain.validate("abc").isValid).toBe(true);
  });

  it("chains multiple validators together", () => {
    const chain = new ValidationChain()
      .required("Password")
      .minLength(8, "Password")
      .passwordStrength();

    expect(chain.validate("").isValid).toBe(false);
    expect(chain.validate("short").isValid).toBe(false);
    expect(chain.validate("Password123").isValid).toBe(true);
  });

  it("supports email validation in chain", () => {
    const chain = new ValidationChain().required("Email").email();
    expect(chain.validate("").isValid).toBe(false);
    expect(chain.validate("invalid").isValid).toBe(false);
    expect(chain.validate("test@example.com").isValid).toBe(true);
  });

  it("supports pattern validation in chain", () => {
    const chain = new ValidationChain()
      .required("Zip")
      .pattern(/^\d{5}$/, "Must be 5 digits");

    expect(chain.validate("1234").isValid).toBe(false);
    expect(chain.validate("12345").isValid).toBe(true);
  });

  it("supports custom validators", () => {
    const customValidator = (value: string) => {
      if (value === "forbidden") {
        return { isValid: false, errorMessage: "This value is not allowed" };
      }
      return { isValid: true };
    };

    const chain = new ValidationChain().required("Field").custom(customValidator);

    expect(chain.validate("forbidden").isValid).toBe(false);
    expect(chain.validate("allowed").isValid).toBe(true);
  });

  it("returns validators array", () => {
    const chain = new ValidationChain().required("Field").minLength(5, "Field");
    const validators = chain.getValidators();
    expect(validators).toHaveLength(2);
  });

  it("supports maxLength in chain", () => {
    const chain = new ValidationChain()
      .required("Username")
      .minLength(3, "Username")
      .maxLength(10, "Username");

    expect(chain.validate("ab").isValid).toBe(false);
    expect(chain.validate("validuser").isValid).toBe(true);
    expect(chain.validate("verylongusername").isValid).toBe(false);
  });
});

describe("createValidation", () => {
  it("creates a new validation chain", () => {
    const chain = createValidation().required("Field").minLength(3, "Field");
    expect(chain.validate("ab").isValid).toBe(false);
    expect(chain.validate("abc").isValid).toBe(true);
  });

  it("creates independent chains", () => {
    const chain1 = createValidation().required("Field1");
    const chain2 = createValidation().minLength(5, "Field2");

    expect(chain1.validate("").isValid).toBe(false);
    expect(chain2.validate("ab").isValid).toBe(false);
  });
});
