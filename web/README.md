# Package Tracking Web Frontend

A modern React/TypeScript web application for the package tracking system.

## 🏗️ Architecture

### Technology Stack
- **React 18** + **TypeScript** for UI components
- **Vite** for fast development and build tooling
- **Tailwind CSS** for styling
- **React Router** for client-side routing
- **TanStack Query (React Query)** for server state management
- **Axios** for HTTP API calls
- **Lucide React** for icons

### Project Structure
```
web/
├── src/
│   ├── components/
│   │   ├── layout/          # Layout components (Layout.tsx)
│   │   └── ui/              # Reusable UI components (Button.tsx)
│   ├── pages/               # Main page components
│   │   ├── Dashboard.tsx    # Dashboard with stats and recent shipments
│   │   ├── ShipmentList.tsx # Shipment list with search/filtering
│   │   ├── AddShipment.tsx  # Add new shipment form
│   │   └── ShipmentDetail.tsx # Shipment detail with tracking timeline
│   ├── hooks/
│   │   └── api.ts           # React Query hooks for API calls
│   ├── services/
│   │   └── api.ts           # Axios HTTP client and API service
│   ├── types/
│   │   └── api.ts           # TypeScript type definitions
│   ├── lib/
│   │   └── utils.ts         # Utility functions (classname merging)
│   ├── App.tsx              # Main app component with routing
│   └── main.tsx             # React entry point
├── public/                  # Static assets
├── dist/                    # Built files (served by Go backend)
├── package.json             # Dependencies and scripts
├── tailwind.config.js       # Tailwind CSS configuration
├── vite.config.ts           # Vite build configuration
└── tsconfig.json            # TypeScript configuration
```

## 🚀 Development

### Prerequisites
- Node.js 18+ (for frontend development)
- Go 1.21+ (for backend API)

### Getting Started

1. **Install dependencies:**
   ```bash
   cd web
   npm install
   ```

2. **Start the backend API server:**
   ```bash
   # From project root
   go run cmd/server/main.go
   # Server runs on http://localhost:8080
   ```

3. **Start the frontend development server:**
   ```bash
   cd web
   npm run dev
   # Frontend runs on http://localhost:5173
   # Automatically proxies API calls to :8080
   ```

### Build Commands

```bash
# Development server
npm run dev

# Type checking
npm run type-check

# Production build
npm run build

# Preview production build
npm run preview
```

## 🎨 Features Implemented

### ✅ Core Features
- **Dashboard**: Overview with shipment statistics and recent activity
- **Shipment Management**: Add, view, edit, and delete shipments
- **Shipment List**: Table view with search, filtering, and sorting
- **Shipment Detail**: Detailed view with tracking timeline
- **Manual Refresh**: On-demand tracking data refresh with rate limiting
- **Responsive Design**: Mobile-first design that works on all devices

### ✅ Technical Features
- **Real-time Updates**: Auto-refresh with configurable intervals
- **Error Handling**: Comprehensive error boundaries and user feedback
- **Loading States**: Visual feedback for all async operations
- **Form Validation**: Client-side validation with helpful error messages
- **Search & Filter**: Multi-field search and carrier/status filtering
- **Type Safety**: Full TypeScript coverage with strict mode

## 🔌 API Integration

### Endpoints Used
- `GET /api/shipments` - List all shipments
- `POST /api/shipments` - Create new shipment  
- `GET /api/shipments/{id}` - Get shipment details
- `PUT /api/shipments/{id}` - Update shipment
- `DELETE /api/shipments/{id}` - Delete shipment
- `GET /api/shipments/{id}/events` - Get tracking events
- `POST /api/shipments/{id}/refresh` - Manual refresh tracking
- `GET /api/carriers` - List available carriers
- `GET /api/health` - Health check

### State Management
React Query handles all server state with:
- **Caching**: Smart caching with 1-minute stale time
- **Auto-refresh**: Background updates every 2 minutes
- **Optimistic Updates**: Immediate UI updates with rollback on error
- **Error Recovery**: Automatic retry with exponential backoff

## 🎯 User Experience

### Dashboard
- **Quick Stats**: Total, in-transit, delivered, and problem shipments
- **Recent Activity**: Last 5 shipments with quick actions
- **Empty State**: Clear call-to-action when no shipments exist

### Shipment List
- **Search**: Real-time search by tracking number or description
- **Filters**: Filter by carrier, status, or delivery state
- **Table View**: Sortable columns with status indicators
- **Actions**: Quick view and manage buttons

### Add Shipment
- **Form Validation**: Real-time validation with helpful error messages
- **Carrier Selection**: Dropdown with all supported carriers
- **Auto-redirect**: Returns to shipment list after successful creation

### Shipment Detail
- **Timeline View**: Visual timeline of tracking events
- **Manual Refresh**: Force refresh with rate limiting feedback
- **Status Indicators**: Clear visual status with color coding
- **Quick Actions**: Edit and delete with confirmation

## 🔧 Configuration

### Environment Variables
The frontend automatically detects the environment:
- **Development**: API calls go to `http://localhost:8080/api`
- **Production**: API calls go to `/api` (served by Go backend)

### Customization
- **Colors**: Update `tailwind.config.js` for custom color scheme
- **API Base URL**: Modify `src/services/api.ts` for different backend URL
- **Refresh Intervals**: Adjust query options in `src/hooks/api.ts`

## 🚀 Production Deployment

### Build Process
```bash
cd web
npm run build
```

The built files in `web/dist/` are automatically served by the Go backend through the static file handler at `internal/handlers/static.go`.

### Single Binary Deployment
The Go server serves both:
1. **API endpoints** at `/api/*`
2. **Static files** for the React app at all other routes
3. **SPA routing** - serves `index.html` for client-side routes

## 🧪 Testing Strategy

### Planned Tests (Future)
- **Unit Tests**: Component testing with React Testing Library
- **Integration Tests**: API integration testing with MSW
- **E2E Tests**: End-to-end user flows with Playwright
- **Visual Tests**: Component visual regression testing

### Current Status
The foundation is in place for comprehensive testing. All components are designed to be testable with clear separation of concerns.

## 🔜 Future Enhancements

### Phase 2 Features
- **Dark Mode**: Toggle between light and dark themes
- **Notifications**: Browser notifications for status changes
- **Bulk Operations**: Multi-select and bulk actions
- **Advanced Search**: Saved searches and complex filters
- **Export**: CSV/PDF export of shipment data

### Performance Optimizations
- **Code Splitting**: Route-based lazy loading
- **Virtual Scrolling**: For large shipment lists
- **Service Worker**: Offline support and caching
- **Bundle Analysis**: Optimize package size

---

**Built with ❤️ using React, TypeScript, and modern web technologies**