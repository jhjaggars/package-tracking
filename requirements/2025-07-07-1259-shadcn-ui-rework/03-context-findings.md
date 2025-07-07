# Context Findings

## Current UI Architecture Analysis

### Existing Stack
- **Framework**: React 19.1.0 with TypeScript
- **Build Tool**: Vite 7.0.0
- **Styling**: Tailwind CSS 4.1.11 (already latest version)
- **Router**: React Router DOM 7.6.3
- **State Management**: @tanstack/react-query 5.81.5
- **Component Library**: Custom components with some shadcn/ui foundations already present

### Current Button Component Analysis
- **Location**: `web/src/components/ui/button.tsx`
- **Dependencies**: Already uses @radix-ui/react-slot, class-variance-authority, and cn utility
- **Implementation**: Follows shadcn/ui patterns with buttonVariants using cva
- **Current Issue**: Uses basic styling, needs full shadcn/ui implementation

### Theme System Analysis
- **Location**: `web/src/contexts/ThemeContext.tsx`
- **Features**: 
  - Light/dark theme toggle
  - System preference detection
  - Local storage persistence
  - DOM class manipulation for dark mode
- **Implementation**: Custom React Context with useTheme hook
- **Compatibility**: Should integrate well with shadcn/ui theme system

### Dashboard Component Analysis
- **Location**: `web/src/pages/Dashboard.tsx`
- **Features**:
  - Animated StatCard components with loading states
  - Confetti effects for deliveries
  - Smart insights and greetings
  - Recent shipments list
  - Responsive design with mobile considerations
- **Dependencies**: Uses existing Button component, Lucide React icons
- **Data Integration**: Uses @tanstack/react-query hooks (useDashboardStats, useShipments)

### Layout Component Analysis
- **Location**: `web/src/components/layout/Layout.tsx`
- **Features**:
  - Sticky navigation with backdrop blur
  - Responsive mobile menu
  - Gradient backgrounds
  - Active navigation indicators
  - Theme toggle integration
- **Navigation Structure**: Dashboard, Shipments, Add Shipment pages
- **Mobile Pattern**: Hidden desktop nav, separate mobile menu

### Package Dependencies Already Present
- **shadcn/ui foundations**: @radix-ui/react-slot, class-variance-authority, clsx, tailwind-merge
- **Icons**: lucide-react (compatible with shadcn/ui)
- **Utilities**: DOMPurify for sanitization
- **Styling**: Tailwind CSS v4 (latest version)

## shadcn/ui Integration Requirements

### Installation Steps for Existing Project
1. **Already Have**: Tailwind CSS v4, @radix-ui/react-slot, class-variance-authority
2. **Need to Add**: @types/node (for path resolution)
3. **Update**: vite.config.ts with path alias
4. **Run**: `npx shadcn@latest init` to set up components.json
5. **Add Components**: Use `npx shadcn@latest add <component>` for each needed component

### Components Needed for Full Migration
Based on current usage patterns:
- **Button**: Already partially implemented, needs full shadcn version
- **Card**: For dashboard stat cards and shipment lists
- **Badge**: For shipment status indicators
- **Dialog**: For modals and confirmations
- **Form**: For add shipment form
- **Input**: For form inputs
- **Table**: For shipment lists
- **Sheet**: For mobile navigation drawer
- **Skeleton**: For loading states
- **Tabs**: For potential future features
- **Dropdown Menu**: For user actions
- **Alert**: For notifications and messages
- **Progress**: For loading indicators
- **Tooltip**: For enhanced UX

### Theme System Integration
- **Current**: Custom ThemeContext with localStorage persistence
- **shadcn/ui**: Supports CSS variables for theming
- **Integration**: Keep existing ThemeContext, update CSS variables for shadcn components
- **Dark Mode**: Current implementation compatible with shadcn/ui dark mode patterns

### File Structure Changes Required
- **Keep**: Existing page structure and routing
- **Update**: All component files to use shadcn/ui components
- **Add**: components.json configuration
- **Update**: vite.config.ts with path aliases
- **Preserve**: API hooks and data fetching patterns

### Testing Considerations
- **Current Tests**: Will need complete rewrite due to component changes
- **Testing Library**: Keep existing @testing-library/react setup
- **New Tests**: Write tests for shadcn/ui component integration
- **Coverage**: Maintain test coverage for critical functionality

## Files That Need Major Changes
- `web/src/components/ui/button.tsx` - Replace with shadcn/ui Button
- `web/src/components/layout/Layout.tsx` - Update with shadcn/ui navigation components
- `web/src/pages/Dashboard.tsx` - Update StatCard and layout with shadcn/ui Card, Badge
- `web/src/pages/ShipmentList.tsx` - Update with shadcn/ui Table
- `web/src/pages/AddShipment.tsx` - Update with shadcn/ui Form components
- `web/src/pages/ShipmentDetail.tsx` - Update with shadcn/ui layout components
- `web/src/index.css` - Update with shadcn/ui CSS variables
- `web/vite.config.ts` - Add path aliases for shadcn/ui
- `web/tsconfig.json` and `web/tsconfig.app.json` - Add path configuration

## Files That Can Be Preserved
- `web/src/services/api.ts` - Keep existing API service
- `web/src/hooks/api.ts` - Keep existing React Query hooks
- `web/src/contexts/ThemeContext.tsx` - Keep with minor updates for shadcn/ui integration
- `web/src/lib/utils.ts` - Keep existing cn utility and other utils
- `web/src/lib/sanitize.ts` - Keep existing sanitization logic
- `web/src/types/api.ts` - Keep existing type definitions