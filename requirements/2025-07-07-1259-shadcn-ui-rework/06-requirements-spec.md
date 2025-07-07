# Requirements Specification: shadcn/ui Complete UI Rework

## Problem Statement
The current package tracking system UI needs a complete overhaul to adopt the shadcn/ui component library. The existing custom UI components and styling should be replaced with a consistent, professional design system while maintaining core functionality and user experience.

## Solution Overview
Completely rebuild the frontend interface using shadcn/ui components while preserving:
- Existing page structure and routing
- API integration patterns
- Theme toggle functionality
- Responsive mobile design
- Core user workflows

The new design will be cleaner, more professional, and remove visual flourishes in favor of shadcn/ui's minimal design principles.

## Functional Requirements

### FR1: Component Library Migration
- **Requirement**: Replace all custom UI components with shadcn/ui equivalents
- **Components Needed**: Button, Card, Badge, Dialog, Form, Input, Table, Sheet, Skeleton, Alert, Tooltip
- **Outcome**: Consistent design language across all UI elements

### FR2: Theme System Integration
- **Requirement**: Migrate from custom ThemeContext to shadcn/ui CSS variables
- **Preserve**: Theme toggle functionality and localStorage persistence
- **Implementation**: Update CSS variables for light/dark modes while keeping existing toggle behavior
- **Outcome**: Better component theming integration with shadcn/ui standards

### FR3: Mobile Navigation Redesign
- **Requirement**: Replace custom mobile menu with shadcn/ui Sheet component
- **Pattern**: Slide-out drawer for mobile navigation
- **Accessibility**: Improved keyboard navigation and screen reader support
- **Outcome**: More professional mobile navigation experience

### FR4: Simplified Dashboard Design
- **Requirement**: Remove animated StatCard number counting and confetti effects
- **Replace With**: Clean shadcn/ui Card components for statistics
- **Preserve**: Dashboard layout, data fetching, and smart insights
- **Outcome**: More professional, less distracting dashboard interface

### FR5: Visual Design Simplification
- **Requirement**: Remove gradient backgrounds and glassmorphism effects
- **Replace With**: Solid backgrounds with subtle shadows following shadcn/ui patterns
- **Preserve**: Layout structure and responsive design
- **Outcome**: Cleaner, more professional visual appearance

## Technical Requirements

### TR1: Installation and Setup
- **Location**: `/home/jhjaggars/code/package-tracking/web/`
- **Dependencies**: Install `@types/node` for path resolution
- **Configuration**: Update `vite.config.ts` with path aliases
- **Initialization**: Run `npx shadcn@latest init` to generate `components.json`

### TR2: File Structure Updates
- **Update**: All page components (`Dashboard.tsx`, `ShipmentList.tsx`, `AddShipment.tsx`, `ShipmentDetail.tsx`)
- **Update**: Layout component (`Layout.tsx`) with shadcn/ui navigation
- **Update**: Component files in `components/ui/` directory
- **Update**: CSS files for shadcn/ui theme variables

### TR3: Configuration Files
- **Update**: `vite.config.ts` with path aliases:
  ```typescript
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  }
  ```
- **Update**: `tsconfig.json` and `tsconfig.app.json` with baseUrl and paths
- **Update**: `src/index.css` with shadcn/ui imports

### TR4: Dependency Management
- **Preserve**: Existing dependencies (`@tanstack/react-query`, `react-router-dom`, `lucide-react`)
- **Add**: Individual shadcn/ui components as needed
- **Remove**: `canvas-confetti` and `react-confetti` dependencies
- **Update**: Button component to full shadcn/ui implementation

## Implementation Hints and Patterns

### IH1: Theme Context Integration
- **File**: `src/contexts/ThemeContext.tsx`
- **Pattern**: Keep existing ThemeContext but update to work with shadcn/ui CSS variables
- **Implementation**: Maintain `dark` class toggle on document root for shadcn/ui compatibility

### IH2: Dashboard StatCard Pattern
- **File**: `src/pages/Dashboard.tsx`
- **Pattern**: Replace StatCard with shadcn/ui Card component
- **Remove**: Animation logic, confetti effects, and complex hover effects
- **Keep**: Loading states, data display, and responsive grid layout

### IH3: Mobile Navigation Pattern
- **File**: `src/components/layout/Layout.tsx`
- **Pattern**: Implement hamburger menu that opens shadcn/ui Sheet
- **Remove**: Custom mobile menu with slide-down animation
- **Add**: Mobile menu trigger button for smaller screens

### IH4: Form Components Pattern
- **File**: `src/pages/AddShipment.tsx`
- **Pattern**: Use shadcn/ui Form, Input, and Button components
- **Preserve**: Form validation and API integration logic
- **Update**: Form styling to match shadcn/ui patterns

### IH5: Data Table Pattern
- **File**: `src/pages/ShipmentList.tsx`
- **Pattern**: Implement shadcn/ui Table component
- **Preserve**: Data fetching, sorting, and filtering logic
- **Update**: Table styling and responsive behavior

## Acceptance Criteria

### AC1: Component Consistency
- [ ] All UI components use shadcn/ui library
- [ ] No custom styled components remain
- [ ] Design language is consistent across all pages

### AC2: Functionality Preservation
- [ ] All existing page routes work correctly
- [ ] Theme toggle functionality preserved
- [ ] Mobile responsive design maintained
- [ ] API integration patterns unchanged

### AC3: Professional Design
- [ ] Gradient backgrounds and glassmorphism effects removed
- [ ] Clean, minimal design following shadcn/ui patterns
- [ ] Proper spacing and typography hierarchy
- [ ] Accessibility standards maintained

### AC4: Mobile Experience
- [ ] Sheet component used for mobile navigation
- [ ] Touch-friendly interface elements
- [ ] Responsive layout on all screen sizes
- [ ] Improved mobile navigation UX

### AC5: Performance
- [ ] No performance regression from current implementation
- [ ] Lazy loading preserved for page components
- [ ] Bundle size optimized with individual component imports

## Assumptions

### A1: Testing
- Existing component tests will be completely rewritten to match new shadcn/ui components
- Testing patterns will follow shadcn/ui component testing best practices
- Test coverage will be maintained for all critical functionality

### A2: Styling
- All custom CSS classes will be replaced with shadcn/ui utility classes
- Tailwind CSS configuration will be updated to work with shadcn/ui
- Color palette will follow shadcn/ui design tokens

### A3: Dependencies
- Current React 19 and TypeScript setup is compatible with shadcn/ui
- Existing build tools (Vite) will work with shadcn/ui components
- No major dependency conflicts will occur during migration

### A4: User Experience
- Users will adapt to the new, more professional design
- Removal of animations and effects will not negatively impact user engagement
- Mobile Sheet navigation will provide better UX than current custom menu

### A5: Development
- Migration will be done incrementally, page by page
- API integration patterns will remain unchanged
- Development workflow will not be disrupted by the component library change