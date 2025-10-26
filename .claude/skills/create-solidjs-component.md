---
name: create-solidjs-component
description: Create SolidJS components following project standards, emphasizing reuse of existing components
---

# Create SolidJS Component Skill

When creating or modifying SolidJS components, follow these strict guidelines to ensure consistency with the Waugzee project's architecture and maximize component reuse.

## CRITICAL: Component Reuse First

**BEFORE creating any new component, you MUST:**

1. **Audit Existing Components**: Check `client/src/components/` for reusable components
2. **Prioritize Reuse**: Use existing components whenever possible, especially for forms and UI
3. **Only Create When Necessary**: New components should only be created when existing ones don't fit the use case

## Available Component Library

### Form Components (client/src/components/common/forms/)

**ALWAYS use these for forms instead of creating custom inputs:**

- **TextInput**: Text, email, password inputs with validation
  - Props: `label`, `value`, `placeholder`, `type`, `required`, `validationFunction`, `onInput`, `onBlur`
  - Features: Built-in email validation, password toggle, error messages
  - Example: `<TextInput label="Email" type="email" required />`

- **Textarea**: Multi-line text input with character count
  - Props: `label`, `value`, `placeholder`, `rows`, `maxLength`, `showCharacterCount`, `onChange`
  - Features: Character count, validation, error states
  - Example: `<Textarea label="Notes" rows={3} maxLength={500} />`

- **Select**: Basic dropdown select
  - Props: `label`, `options`, `value`, `placeholder`, `onChange`, `required`
  - Options format: `{ value: string, label: string, disabled?: boolean }[]`
  - Example: `<Select label="Category" options={categoryOptions} />`

- **SearchableSelect**: Dropdown with fuzzy search (uses Fuse.js)
  - Props: `label`, `options`, `value`, `searchPlaceholder`, `emptyMessage`, `onChange`
  - Options format: `{ value: string, label: string, metadata?: string }[]`
  - Features: Fuzzy search, keyboard navigation
  - Example: `<SearchableSelect label="Stylus" searchPlaceholder="Search..." options={stylusOptions()} />`

- **Checkbox**: Checkbox with custom styling
  - Props: `label`, `checked`, `onChange`, `required`, `disabled`
  - Features: Custom check icon, validation
  - Example: `<Checkbox label="Accept terms" checked={accepted()} onChange={setAccepted} />`

- **Toggle**: Toggle switch component
  - Props: `label`, `checked`, `onChange`, `disabled`
  - Example: `<Toggle label="Deep Clean" checked={isDeepClean()} onChange={setIsDeepClean} />`

- **DateInput**: Date picker input
  - Props: `label`, `value`, `onChange`, `required`
  - Example: `<DateInput label="Date" value={date()} onChange={setDate} />`

- **DateTimeInput**: Date and time picker
  - Props: `label`, `value`, `onChange`, `required`
  - Example: `<DateTimeInput label="Date & Time" value={dateTime()} onChange={setDateTime} />`

- **MultiSelect**: Multiple selection dropdown
  - Props: `label`, `options`, `value`, `onChange`
  - Example: `<MultiSelect label="Tags" options={tagOptions} />`

### UI Components (client/src/components/common/ui/)

**ALWAYS use these for common UI patterns:**

- **Button**: Primary UI button with variants
  - Props: `variant`, `size`, `type`, `disabled`, `onClick`, `children`
  - Variants: `"primary"` | `"secondary"` | `"tertiary"` | `"danger"` | `"gradient"` | `"ghost"` | `"warning"`
  - Sizes: `"sm"` | `"md"` | `"lg"`
  - Example: `<Button variant="primary" onClick={handleSubmit}>Submit</Button>`

- **Modal**: Full-featured modal dialog
  - Props: `isOpen`, `onClose`, `children`, `size`, `title`, `showCloseButton`, `closeOnBackdropClick`, `closeOnEscape`
  - Sizes: `ModalSize.Small` | `ModalSize.Medium` | `ModalSize.Large` | `ModalSize.ExtraLarge`
  - Features: Focus trapping, keyboard navigation, portal rendering
  - Example: `<Modal isOpen={isOpen()} onClose={() => setIsOpen(false)} title="Edit Item">{content}</Modal>`

- **Card**: Card container with consistent styling
  - Props: `children`, `class`
  - Example: `<Card>{content}</Card>`

- **Avatar**: User avatar component
  - Props: `src`, `alt`, `size`
  - Example: `<Avatar src={user.avatar} alt={user.name} />`

- **Image**: Image with lazy loading and skeleton
  - Props: `src`, `alt`, `aspectRatio`, `showSkeleton`
  - Example: `<Image src={release.thumb} alt={release.title} aspectRatio="square" showSkeleton />`

- **ConfirmationModal**: Confirmation dialog
  - Props: `isOpen`, `onClose`, `onConfirm`, `title`, `message`
  - Example: `<ConfirmationModal isOpen={showConfirm()} onConfirm={handleDelete} title="Delete Item?" />`

- **ConfirmPopup**: Small inline confirmation popup
  - Props: `isOpen`, `onConfirm`, `onCancel`, `message`
  - Example: `<ConfirmPopup isOpen={showPopup()} onConfirm={proceed} message="Are you sure?" />`

### Icon Components (client/src/components/icons/)

**NEVER use inline SVG - ALWAYS create/use icon components:**

Available icons:
- `AlertCircleIcon`, `AlertTriangleIcon`, `CheckIcon`, `CheckCircleIcon`
- `ChevronDownIcon`, `EyeIcon`, `EyeOffIcon`, `FilterIcon`, `GridIcon`
- `LoadingSpinner`, `SearchIcon`, `XIcon`

All icons accept: `size?: number` and `class?: string`

Example: `<ChevronDownIcon size={16} />`

**If you need a new icon:**
1. Create a new component file in `client/src/components/icons/`
2. Export a component with `size` and `class` props
3. Use consistent naming: `[Name]Icon.tsx`

### Validation Hooks

**Use these hooks for form validation:**

- **useValidation**: Field-level validation
  - Props: `initialValue`, `required`, `minLength`, `maxLength`, `pattern`, `customValidators`, `fieldName`
  - Returns: `{ value, setValue, isValid, errorMessage, validate, markAsBlurred }`

- **useFormField**: Form context integration
  - Props: `name`, `required`, `initialValue`
  - Returns: `{ isConnectedToForm, updateFormField }`

## Component Creation Guidelines

### File Structure

```
client/src/components/
├── common/
│   ├── forms/          # Reusable form components
│   └── ui/             # Reusable UI components
├── [FeatureName]/      # Feature-specific components
│   ├── ComponentName.tsx
│   ├── ComponentName.module.scss
│   └── ComponentName.test.tsx (optional)
```

### Naming Conventions

- **Component Files**: PascalCase - `RecordActionModal.tsx`
- **SCSS Modules**: camelCase - `RecordActionModal.module.scss`
- **CSS Classes**: camelCase - `.modalOverlay`, `.submitButton`
- **Variables/Functions**: camelCase - `handleSubmit`, `isLoading`
- **Props Interfaces**: PascalCase - `RecordActionModalProps`

### TypeScript Standards

**All components must have:**

1. **Props Interface**: Define all props with proper types
2. **Component Type**: Use `Component<PropsInterface>` from SolidJS
3. **Full Type Safety**: No `any` types unless absolutely necessary

Example:
```typescript
import type { Component } from "solid-js";

interface MyComponentProps {
  title: string;
  onSubmit: (data: FormData) => void;
  isLoading?: boolean;
}

export const MyComponent: Component<MyComponentProps> = (props) => {
  // Component implementation
};
```

### SCSS Standards

**CRITICAL: Use design system variables - NO hardcoded values**

All SCSS files MUST use variables from the design system. Reference these files for available variables:

- **`client/src/styles/_variables.scss`** - Spacing, typography, borders, shadows, transitions, breakpoints, z-index
- **`client/src/styles/_colors.scss`** - All color variables (text, backgrounds, borders, buttons, forms, etc.)

**Key Categories:**

**Spacing** (from `_variables.scss`):
- `$spacing-xs` through `$spacing-3xl` (4px to 64px)
- Component-specific: `$container-padding`, `$card-padding`, `$button-padding`, `$input-padding`

**Typography** (from `_variables.scss`):
- Font families: `$font-family-base`, `$font-family-heading`, `$font-family-mono`
- Font sizes: `$font-size-xs` through `$font-size-5xl` (12px to 48px)
  - **Note**: Use `$font-size-md` as base (there is NO `$font-size-base`)
- Font weights: `$font-weight-light` through `$font-weight-bold`
- Line heights: `$line-height-tight`, `$line-height-normal`, `$line-height-loose`

**Colors** (from `_colors.scss`):
- Semantic text colors: `$text-default`, `$text-muted`, `$text-light`, etc.
- Background colors: `$bg-body`, `$bg-surface`, `$bg-primary`, `$bg-success-subtle`, etc.
- Border colors: `$border-default`, `$border-strong`, `$border-focus`, `$border-error`, etc.
- Button colors: `$button-primary-bg`, `$button-danger-hover`, etc.
- Form colors: `$input-bg`, `$input-border-focus`, `$input-placeholder`, etc.

**Other Variables** (from `_variables.scss`):
- Border radius: `$border-radius-sm` through `$border-radius-full`
- Shadows: `$shadow-sm` through `$shadow-2xl`, `$focus-ring`
- Transitions: `$transition-fast` (150ms), `$transition-normal` (300ms), `$transition-slow` (500ms)
- Z-index: `$z-index-modal`, `$z-index-dropdown`, etc.
- Breakpoints: `$breakpoint-sm` through `$breakpoint-2xl`

**Bad vs Good Examples:**
```scss
// ❌ BAD - Hardcoded values
.modal {
  padding: 1.5rem;
  font-size: 16px;
  color: #333;
  border-radius: 8px;
}

// ✅ GOOD - Design system variables
.modal {
  padding: $spacing-lg;
  font-size: $font-size-md;
  color: $text-default;
  border-radius: $border-radius-lg;
}
```

**When in doubt, check the source files:**
- Read `client/src/styles/_variables.scss` for non-color variables
- Read `client/src/styles/_colors.scss` for all color-related variables

### API Integration

**CRITICAL: Always use TanStack Query hooks from `@services/apiHooks`**

**NEVER use `api.ts` directly** - it's only for internal use by hooks and AuthContext.

**Available Hooks:**
- `useApiQuery<T>` - GET requests with caching
- `useApiPut<Response, Request>` - PUT requests with invalidation
- `useApiPost<Response, Request>` - POST requests with invalidation
- `useApiPatch<Response, Request>` - PATCH requests with invalidation
- `useApiDelete<Response>` - DELETE requests with invalidation

**Declarative Pattern (Preferred):**
```typescript
const updateMutation = useApiPut<ResponseType, RequestType>(
  "/api/endpoint",
  undefined,
  {
    invalidateQueries: [["queryKey"]], // Auto-refetch
    successMessage: "Update successful!",
    errorMessage: "Update failed",
    onSuccess: (data) => {
      // Additional success logic (optional)
      console.log("Success:", data);
    },
    onError: (error) => {
      // Additional error handling (optional)
      console.error("Error:", error);
    },
  }
);

// Simple mutation call - no try/catch needed
updateMutation.mutate(data);
```

**Benefits:**
- Automatic toast notifications
- No manual try/catch blocks
- Cleaner, more readable code
- Consistent error handling

### Component Patterns

**1. State Management:**
```typescript
import { createSignal, createStore } from "solid-js";

// Simple state
const [isOpen, setIsOpen] = createSignal(false);

// Complex form state
const [formState, setFormState] = createStore({
  email: "",
  password: "",
});
```

**2. Conditional Rendering:**
```typescript
import { Show, For } from "solid-js";

// Show/hide based on condition
<Show when={isLoading()} fallback={<Content />}>
  <LoadingSpinner />
</Show>

// List rendering
<For each={items()}>
  {(item) => <ItemComponent item={item} />}
</For>
```

**3. Form Example Using Existing Components:**
```typescript
const [formState, setFormState] = createStore({
  email: "",
  notes: "",
  category: "",
});

return (
  <form onSubmit={handleSubmit}>
    <TextInput
      label="Email"
      type="email"
      value={formState.email}
      onInput={(value) => setFormState("email", value)}
      required
    />

    <Select
      label="Category"
      options={categoryOptions}
      value={formState.category}
      onChange={(value) => setFormState("category", value)}
    />

    <Textarea
      label="Notes"
      value={formState.notes}
      onChange={(value) => setFormState("notes", value)}
      rows={3}
    />

    <Button type="submit" variant="primary">
      Submit
    </Button>
  </form>
);
```

## Best Practices

### Accessibility
- ✅ Use semantic HTML elements
- ✅ Add ARIA labels and roles
- ✅ Support keyboard navigation
- ✅ Ensure focus management in modals
- ✅ Provide alt text for images

### Performance
- ✅ Use `createMemo` for expensive computations
- ✅ Lazy load components when appropriate
- ✅ Use skeleton loading for better perceived performance
- ✅ Leverage TanStack Query caching

### Error Handling
- ✅ Show loading states during async operations
- ✅ Display error messages clearly
- ✅ Provide fallback UI for error states
- ✅ Use declarative pattern for mutations

### Code Quality
- ✅ Keep components single-responsibility
- ✅ Limit comments to critical/complex logic
- ✅ Use descriptive variable names
- ✅ Follow project linting rules (Biome)

## Reference Example: RecordActionModal

See `client/src/components/RecordActionModal/RecordActionModal.tsx` for a complete example demonstrating:

- ✅ Reuse of existing form components (DateTimeInput, SearchableSelect, Textarea, Toggle)
- ✅ Reuse of UI components (Button, Image)
- ✅ TanStack Query mutations with invalidation
- ✅ Proper TypeScript interfaces
- ✅ Design system variables in SCSS
- ✅ Form state management with createStore
- ✅ Conditional rendering with Show/For
- ✅ Loading and error states

## Workflow

When asked to create a component:

1. **Identify Requirements**: What does this component need to do?
2. **Audit Existing Components**: Can I reuse any existing components?
3. **Plan Component Structure**: What props, state, and children are needed?
4. **Check Design System**: What variables should I use for styling?
5. **Implement**: Create component using existing components where possible
6. **Test**: Ensure accessibility, loading states, and error handling work
7. **Document**: Add JSDoc comments for complex logic only

## Anti-Patterns to Avoid

❌ **Creating custom form inputs** - Use existing form components
❌ **Inline SVG elements** - Create icon components
❌ **Hardcoded colors/spacing** - Use design system variables
❌ **Direct API calls** - Use TanStack Query hooks
❌ **Manual try/catch for mutations** - Use declarative pattern with callbacks
❌ **Obvious comments** - Let code be self-documenting
❌ **Index files** - Use direct imports
❌ **`any` types** - Use proper TypeScript
❌ **`$font-size-base`** - Use `$font-size-md` instead (base doesn't exist!)

## Success Criteria

A well-built component should:
- ✅ Reuse existing components where possible
- ✅ Use design system variables exclusively
- ✅ Have full TypeScript type safety
- ✅ Follow naming conventions
- ✅ Use TanStack Query for API calls
- ✅ Handle loading and error states
- ✅ Be accessible and keyboard-navigable
- ✅ Have minimal, critical-only comments
