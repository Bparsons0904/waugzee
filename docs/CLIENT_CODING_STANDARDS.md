# Waugzee Client Coding Standards

This document provides comprehensive guidance for maintaining consistency and quality in the Waugzee client codebase. Follow these standards to ensure readable, maintainable, and consistent code across the SolidJS application.

## Table of Contents

1. [File Structure and Naming Conventions](#file-structure-and-naming-conventions)
2. [Component Development Standards](#component-development-standards)
3. [Styling Standards](#styling-standards)
4. [Service Layer Patterns](#service-layer-patterns)
5. [Simplified Discogs Proxy Service Usage](#simplified-discogs-proxy-service-usage)
6. [TypeScript Standards](#typescript-standards)
7. [Testing Guidelines](#testing-guidelines)

## File Structure and Naming Conventions

### Directory Structure

```
src/
├── components/           # React-like components
│   ├── common/          # Reusable UI components
│   │   ├── forms/       # Form-specific components
│   │   ├── ui/          # Basic UI components
│   │   └── layout/      # Layout components
│   ├── dashboard/       # Feature-specific components
│   └── [feature]/       # Feature-specific component groups
├── context/             # SolidJS contexts
├── pages/               # Route components
├── services/            # Business logic and API integration
├── styles/              # Global styles and design system
├── types/               # TypeScript type definitions
├── constants/           # Application constants
├── hooks/               # Custom hooks (if any)
└── utils/               # Utility functions
```

### File Naming Conventions

**CRITICAL: All naming must follow consistent camelCase/PascalCase standards. NO kebab-case allowed.**

#### Component Files
- **Components**: PascalCase - `Button.tsx`, `Modal.tsx`, `UserProfile.tsx`
- **Component styles**: camelCase - `button.module.scss`, `modal.module.scss`
- **Component tests**: Match component name - `Button.test.tsx`

#### Service Files
- **Services**: camelCase - `userService.ts`, `apiHooks.ts`, `discogsProxy.service.ts`
- **Types**: PascalCase - `User.ts`, `ApiResponse.ts`
- **Utilities**: camelCase - `dateUtils.ts`, `formatHelpers.ts`

#### Constants and Configuration
- **Constants**: camelCase - `api.constants.ts`, `routes.constants.ts`
- **Environment**: camelCase - `env.service.ts`

### Component Organization Pattern

Each component follows a consistent directory structure:

```
Button/
├── Button.tsx           # Component implementation
├── Button.module.scss   # Component styles
├── Button.test.tsx      # Component tests
└── index.ts            # Export (optional - prefer direct imports)
```

**Important**: Avoid creating `index.ts` files unless absolutely necessary. Use direct imports like:
```typescript
import { Button } from "@components/common/ui/Button/Button";
```

## Component Development Standards

### SolidJS Component Patterns

#### Basic Component Structure

```typescript
import { Component, JSX } from "solid-js";
import styles from "./ComponentName.module.scss";
import clsx from "clsx";

interface ComponentNameProps {
  // Use specific types, never 'any'
  title: string;
  isLoading?: boolean;
  onClick?: (event: MouseEvent) => void;
  children?: JSX.Element;
  class?: string; // For additional CSS classes
}

export const ComponentName: Component<ComponentNameProps> = (props) => {
  return (
    <div class={clsx(styles.component, props.class)}>
      {props.children}
    </div>
  );
};
```

#### Signal and State Management

```typescript
import { createSignal, createEffect, createMemo } from "solid-js";

export const ExampleComponent: Component = () => {
  // State management with signals
  const [isLoading, setIsLoading] = createSignal(false);
  const [data, setData] = createSignal<UserData | null>(null);

  // Computed values with memos
  const isDataValid = createMemo(() => {
    const currentData = data();
    return currentData && currentData.id && currentData.name;
  });

  // Side effects
  createEffect(() => {
    if (isDataValid()) {
      console.log("Data is valid:", data());
    }
  });

  return (
    <div class={styles.container}>
      {/* Component content */}
    </div>
  );
};
```

#### Props Interface Standards

- **Always define explicit interfaces** for component props
- **Use optional properties** with `?` for non-required props
- **Provide default values** in destructuring or with logical operators
- **Use union types** instead of `any` for multiple possible types

```typescript
interface UserCardProps {
  user: User;
  showActions?: boolean;
  variant?: "compact" | "expanded";
  onEdit?: (userId: string) => void;
  onDelete?: (userId: string) => void;
  class?: string;
}
```

#### Event Handling Patterns

```typescript
// Proper event typing
const handleButtonClick = (event: MouseEvent) => {
  event.preventDefault();
  // Handle click
};

const handleInputChange = (value: string, event: InputEvent & { target: HTMLInputElement }) => {
  // Handle input change
};

// Usage in JSX
<button onClick={handleButtonClick}>Click me</button>
<input onInput={(e) => handleInputChange(e.target.value, e)} />
```

#### Context Usage Patterns

```typescript
import { useAuth } from "@context/AuthContext";
import { useWebSocket } from "@context/WebSocketContext";

export const ExampleComponent: Component = () => {
  const { user, isAuthenticated } = useAuth();
  const { isConnected, sendMessage } = useWebSocket();

  // Use context values in effects and computed values
  createEffect(() => {
    if (isAuthenticated() && isConnected()) {
      // Safe to use WebSocket
    }
  });

  return (
    <Show when={isAuthenticated()}>
      <div>Welcome, {user()?.firstName}</div>
    </Show>
  );
};
```

## Styling Standards

### SCSS Module Patterns

#### Component Styling Structure

```scss
// ComponentName.module.scss

// Use design system variables exclusively
.component {
  display: flex;
  flex-direction: column;
  padding: $spacing-md;
  background-color: $bg-surface;
  border-radius: $border-radius-md;
  box-shadow: $shadow-sm;

  // State variations
  &.loading {
    opacity: 0.6;
    pointer-events: none;
  }

  &.error {
    border-color: $border-error;
    background-color: $bg-error-subtle;
  }
}

// Size variants
.small {
  padding: $spacing-sm;
  font-size: $font-size-sm;
}

.large {
  padding: $spacing-lg;
  font-size: $font-size-lg;
}

// Interactive states
.button {
  background-color: $button-primary-bg;
  color: $button-primary-text;
  border-radius: $border-radius-md;
  transition: all $transition-fast $transition-timing-default;

  &:hover:not(:disabled) {
    background-color: $button-primary-hover;
    transform: translateY(-1px);
  }

  &:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
}
```

#### Design System Variable Usage

**Always use design system variables instead of hardcoded values:**

```scss
// ✅ Good - Use design system variables
.card {
  padding: $spacing-lg;
  margin-bottom: $spacing-md;
  background-color: $bg-surface;
  border: $border-width-thin solid $border-default;
  border-radius: $border-radius-lg;
  color: $text-default;
}

// ❌ Avoid - Hardcoded values
.card {
  padding: 24px;
  margin-bottom: 16px;
  background-color: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  color: #111827;
}
```

#### Responsive Design Approach

Use mobile-first responsive design with design system breakpoints:

```scss
.component {
  // Mobile first - base styles
  padding: $spacing-sm;
  font-size: $font-size-sm;

  // Tablet and up
  @media (min-width: $breakpoint-md) {
    padding: $spacing-md;
    font-size: $font-size-md;
  }

  // Desktop and up
  @media (min-width: $breakpoint-lg) {
    padding: $spacing-lg;
    font-size: $font-size-lg;
  }
}
```

#### CSS Class Naming

- **Use camelCase** for CSS classes: `.userProfile`, `.primaryButton`
- **Use semantic names** that describe purpose, not appearance
- **Use BEM-like patterns** for complex components

```scss
// Component base
.userCard {
  // Base styles
}

// Element within component
.userCardHeader {
  // Header styles
}

// Modifier for state/variant
.userCardExpanded {
  // Expanded state styles
}
```

## Service Layer Patterns

### API Service Structure

#### Core API Service (api.ts)

The main API service provides typed HTTP methods with built-in error handling and retry logic:

```typescript
import { api } from "@services/api";

// Use typed API calls
const fetchUser = async (userId: string): Promise<User> => {
  return api.get<User>(`/users/${userId}`);
};

const updateUser = async (userId: string, userData: Partial<User>): Promise<User> => {
  return api.patch<User>(`/users/${userId}`, userData);
};
```

#### API Hooks Pattern (apiHooks.ts)

Use the provided API hooks for consistent query and mutation patterns:

```typescript
import { useApiGet, useApiPost, useApiPatch } from "@services/apiHooks";
import { USER_ENDPOINTS } from "@constants/api.constants";

// Query hook usage
export function useUserProfile(userId: string) {
  return useApiGet<User>(
    ["users", userId],
    USER_ENDPOINTS.PROFILE(userId),
    undefined,
    {
      enabled: () => Boolean(userId),
      staleTime: 5 * 60 * 1000, // 5 minutes
    }
  );
}

// Mutation hook usage
export function useUpdateUser() {
  return useApiPatch<User, Partial<User>>(
    (variables) => USER_ENDPOINTS.PROFILE(variables.id!),
    undefined,
    {
      successMessage: "User updated successfully",
      errorMessage: "Failed to update user",
      invalidateQueries: [["users"]],
    }
  );
}
```

#### Service File Organization

```typescript
// userService.ts
import { api } from "@services/api";
import { User } from "@types/User";

export class UserService {
  static async getProfile(): Promise<User> {
    return api.get<User>("/users/me");
  }

  static async updateDiscogsToken(token: string): Promise<User> {
    return api.patch<User>("/users/me/discogs", { token });
  }
}

// Export service instance
export const userService = new UserService();
```

### Context Integration Patterns

#### Authentication Context Usage

```typescript
import { useAuth } from "@context/AuthContext";

export const ExampleComponent: Component = () => {
  const { user, isAuthenticated, logout } = useAuth();

  const handleSecureAction = async () => {
    if (!isAuthenticated()) {
      console.warn("User not authenticated");
      return;
    }

    // Proceed with authenticated action
    const userData = user();
    if (userData?.id) {
      // Use user data safely
    }
  };

  return (
    <Show when={isAuthenticated()} fallback={<LoginPrompt />}>
      <UserDashboard user={user()!} />
    </Show>
  );
};
```

#### WebSocket Context Integration

```typescript
import { useWebSocket } from "@context/WebSocketContext";

export const ExampleComponent: Component = () => {
  const { isConnected, isAuthenticated, sendMessage, onSyncMessage } = useWebSocket();

  onMount(() => {
    // Set up WebSocket message handlers
    const unsubscribe = onSyncMessage((message) => {
      if (message.type === "sync_progress") {
        // Handle sync progress
        console.log("Sync progress:", message.data);
      }
    });

    onCleanup(unsubscribe);
  });

  const handleSendMessage = () => {
    if (isConnected() && isAuthenticated()) {
      sendMessage(JSON.stringify({
        type: "user_action",
        data: { action: "example" }
      }));
    }
  };

  return (
    <button onClick={handleSendMessage} disabled={!isConnected()}>
      Send Message
    </button>
  );
};
```

## Simplified Discogs Proxy Service Usage

The Discogs Proxy Service enables client-side API calls to Discogs while maintaining server coordination through WebSocket communication.

### Service Initialization

```typescript
import { discogsProxyService } from "@services/discogs/discogsProxy.service";
import { useWebSocket } from "@context/WebSocketContext";

export const ExampleComponent: Component = () => {
  const webSocket = useWebSocket();

  onMount(() => {
    // Initialize the proxy service with WebSocket context
    discogsProxyService.initialize(webSocket);

    // Cleanup on unmount
    onCleanup(() => {
      discogsProxyService.cleanup();
    });
  });

  // Check if service is ready for API calls
  const isServiceReady = () => discogsProxyService.isReady();

  return (
    <Show when={isServiceReady()} fallback={<ConnectingMessage />}>
      <DiscogsIntegration />
    </Show>
  );
};
```

### Integration Patterns

#### Service Readiness Check

Always verify the service is ready before making API calls:

```typescript
const handleSyncCollection = async () => {
  if (!discogsProxyService.isReady()) {
    console.warn("Discogs proxy service not ready");
    return;
  }

  // Service is ready - the backend will handle API requests through WebSocket
  // The client doesn't make direct HTTP calls to Discogs
  console.log("Sync will be handled by the proxy service");
};
```

#### WebSocket Message Handling

The proxy service automatically handles WebSocket messages for API requests:

```typescript
// The service automatically handles these message types:
// - "discogs_api_request" - Incoming API requests from server
// - Server requests are processed and responses sent back via WebSocket

// Your component should listen for sync-related messages:
const handleSyncMessages = () => {
  const unsubscribe = webSocket.onSyncMessage((message) => {
    switch (message.type) {
      case "sync_progress":
        setSyncProgress(message.data?.progress || 0);
        break;
      case "sync_complete":
        setSyncStatus("completed");
        break;
      case "sync_error":
        setSyncStatus("error");
        setErrorMessage(message.data?.error || "Sync failed");
        break;
    }
  });

  return unsubscribe;
};
```

#### Error Handling

```typescript
onMount(() => {
  try {
    discogsProxyService.initialize(webSocket);
  } catch (error) {
    console.error("Failed to initialize Discogs proxy service:", error);
    setError("Failed to connect to Discogs service");
  }
});

// Monitor service readiness
createEffect(() => {
  if (!discogsProxyService.isReady()) {
    setServiceStatus("disconnected");
  } else {
    setServiceStatus("connected");
  }
});
```

## TypeScript Standards

### Strict Type Safety

**Never use `any` type**. Use proper union types, generics, or `unknown` when needed:

```typescript
// ✅ Good - Proper typing
interface ApiResponse<T> {
  data: T;
  error?: string;
  metadata?: Record<string, unknown>;
}

// ✅ Good - Union types
type ButtonVariant = "primary" | "secondary" | "danger";

// ✅ Good - Generic with constraints
function processApiResponse<T extends Record<string, unknown>>(
  response: ApiResponse<T>
): T | null {
  return response.error ? null : response.data;
}

// ❌ Avoid - Using 'any'
function badFunction(data: any): any {
  return data.whatever;
}
```

### Interface and Type Definitions

```typescript
// Place in appropriate files in src/types/
export interface User {
  id: string;
  firstName: string;
  lastName: string;
  email?: string;
  isActive: boolean;
}

// Use Pick and Omit for derived types
export type CreateUserRequest = Omit<User, "id">;
export type UserProfile = Pick<User, "firstName" | "lastName" | "email">;

// Use extends for interface inheritance
export interface AdminUser extends User {
  permissions: Permission[];
  lastAdminAction?: string;
}
```

### Component Prop Typing

```typescript
import { JSX } from "solid-js";

interface ComponentProps {
  // Required props
  title: string;
  user: User;

  // Optional props with defaults
  showActions?: boolean;
  variant?: "compact" | "expanded";

  // Function props with proper signatures
  onEdit?: (user: User) => void;
  onDelete?: (userId: string) => Promise<void>;

  // JSX children
  children?: JSX.Element;

  // CSS class name
  class?: string;
}
```

## Testing Guidelines

### Testing Philosophy

Follow the principle: **Never add business logic to make tests pass - use mocks instead**.

### Component Testing

```typescript
import { render } from "@solidjs/testing-library";
import { Button } from "./Button";

describe("Button", () => {
  it("renders with correct text", () => {
    const { getByText } = render(() =>
      <Button>Click me</Button>
    );

    expect(getByText("Click me")).toBeInTheDocument();
  });

  it("calls onClick when clicked", () => {
    const handleClick = vi.fn();
    const { getByRole } = render(() =>
      <Button onClick={handleClick}>Click me</Button>
    );

    getByRole("button").click();
    expect(handleClick).toHaveBeenCalledOnce();
  });
});
```

### Service Testing with Mocks

```typescript
import { vi } from "vitest";
import { userService } from "./userService";
import { api } from "@services/api";

// Mock the API service
vi.mock("@services/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
  },
}));

describe("UserService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches user profile", async () => {
    const mockUser = { id: "1", name: "Test User" };
    vi.mocked(api.get).mockResolvedValue(mockUser);

    const result = await userService.getProfile();

    expect(api.get).toHaveBeenCalledWith("/users/me");
    expect(result).toEqual(mockUser);
  });
});
```

## Code Quality Standards

### Comments and Documentation

- **Limit comments** to critical and hard-to-understand areas only
- **Use self-documenting code** with descriptive names
- **Never add obvious comments** that restate what the code does

```typescript
// ✅ Good - Explains complex business logic
// Fallback to introspection for legacy tokens without 'sub' claim
if (!userClaims.sub && token) {
  return await this.introspectToken(token);
}

// ❌ Avoid - Obvious comment
// Set the background color to white
backgroundColor: "#ffffff"
```

### Import Organization

```typescript
// External library imports
import { Component, createSignal, onMount } from "solid-js";
import { useNavigate } from "@solidjs/router";

// Internal imports (services, contexts, types)
import { useAuth } from "@context/AuthContext";
import { api } from "@services/api";
import { User } from "@types/User";

// Component imports
import { Button } from "@components/common/ui/Button/Button";
import { Modal } from "@components/common/ui/Modal/Modal";

// Styles (always last)
import styles from "./Component.module.scss";
```

### Error Handling Patterns

```typescript
// Service error handling
const fetchUserData = async (userId: string): Promise<User | null> => {
  try {
    return await api.get<User>(`/users/${userId}`);
  } catch (error) {
    console.error("Failed to fetch user:", error);

    if (error instanceof ApiClientError && error.status === 404) {
      return null; // User not found
    }

    throw error; // Re-throw unexpected errors
  }
};

// Component error handling
const [error, setError] = createSignal<string | null>(null);

const handleAction = async () => {
  setError(null);

  try {
    await performAction();
  } catch (err) {
    setError(err instanceof Error ? err.message : "An error occurred");
  }
};
```

## Conclusion

These coding standards ensure consistency, maintainability, and quality across the Waugzee client codebase. When in doubt:

1. **Follow existing patterns** in the codebase
2. **Use TypeScript strictly** - no `any` types
3. **Leverage the design system** - no hardcoded values
4. **Write self-documenting code** - minimal comments
5. **Test behavior, not implementation** - use mocks for dependencies

For questions or clarifications on these standards, refer to existing code examples in the codebase or consult with the development team.