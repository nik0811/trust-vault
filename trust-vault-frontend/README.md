# TrustVault - Enterprise Data Governance Platform

A comprehensive Next.js-based frontend for an enterprise data governance platform that handles classification, compliance, privacy management, and AI governance.

## Features

### Core Pages (40+)
- **Dashboard**: Real-time overview of data governance metrics
- **Data Sources**: Discover, connect, and monitor data sources
- **Classification**: Automatic and rule-based data classification
- **Governance**: Policy management and compliance enforcement
- **AI Gate**: Control AI model access to sensitive data
- **Data Quality**: Monitor data quality metrics across datasets
- **Privacy & Compliance**: GDPR, CCPA, HIPAA, DPDP management
- **Audit Trail**: Complete audit logging of all activities
- **Observability**: System health and performance monitoring
- **AI Governance**: Manage AI model governance and policies
- **Documents**: Upload and process documents for classification
- **Notifications**: Real-time alerts and notifications
- **Jobs**: Schedule and manage automated tasks
- **Settings**: Configuration and administration
- **Lineage**: Interactive data lineage visualization

### Technology Stack
- **Framework**: Next.js 16 with App Router
- **UI Components**: shadcn/ui
- **Styling**: Tailwind CSS 4 with OKLCH color system
- **State Management**: Zustand
- **Data Fetching**: TanStack React Query
- **Forms**: React Hook Form + Zod validation
- **Charts**: Recharts
- **Authentication**: JWT-based with cookie storage
- **API Client**: Axios with automatic interceptors

### Architecture

#### Directory Structure
```
├── app/                          # Next.js App Router
│   ├── (auth)/                  # Authentication routes
│   │   ├── login/
│   │   └── forgot-password/
│   ├── (app)/                   # Protected app routes
│   │   ├── layout.tsx           # App layout with sidebar & topbar
│   │   ├── dashboard/
│   │   ├── data-sources/
│   │   ├── classification/
│   │   ├── governance/
│   │   ├── ai-gate/
│   │   ├── quality/
│   │   ├── privacy/
│   │   ├── audit/
│   │   ├── observability/
│   │   ├── ai-governance/
│   │   ├── documents/
│   │   ├── notifications/
│   │   ├── jobs/
│   │   ├── settings/
│   │   └── lineage/
│   └── layout.tsx               # Root layout with providers
│
├── components/
│   ├── base/                    # Reusable base components
│   │   ├── data-table.tsx
│   │   ├── stat-card.tsx
│   │   ├── status-badge.tsx
│   │   ├── skeleton.tsx
│   │   ├── modal.tsx
│   │   ├── empty-state.tsx
│   │   └── breadcrumbs.tsx
│   ├── layout/                  # Layout components
│   │   ├── top-bar.tsx
│   │   └── sidebar.tsx
│   └── providers.tsx            # React providers setup
│
├── lib/
│   ├── api.ts                   # Axios instance & API helpers
│   ├── schemas.ts               # Zod validation schemas
│   ├── types.ts                 # TypeScript type definitions
│   └── utils.ts                 # Utility functions
│
├── store/
│   ├── auth.ts                  # Auth state (Zustand)
│   ├── theme.ts                 # Theme state (Zustand)
│   └── ui.ts                    # UI state (Zustand)
│
├── app/
│   ├── globals.css              # Global styles with OKLCH theme
│   └── layout.tsx               # Root layout
│
└── package.json                 # Dependencies
```

### Key Components

#### Base Components
- **DataTable**: Sortable, filterable, paginated table
- **StatCard**: Metric cards with trends
- **StatusBadge**: Color-coded status indicators
- **Modal**: Reusable modal dialog with confirm variant
- **Skeleton**: Loading skeletons for data tables and cards
- **EmptyState**: Standardized empty state UI
- **Breadcrumbs**: Navigation breadcrumbs

#### Layout Components
- **TopBar**: Header with notifications, theme toggle, user menu
- **Sidebar**: Collapsible navigation with 10+ menu items
- **Responsive Design**: Mobile hamburger menu, responsive grid

### Theme System

#### OKLCH Color Palette
The application uses a modern OKLCH color system with light/dark modes:

**Light Mode**
- Background: White (oklch(1 0 0))
- Foreground: Near black (oklch(0.145 0 0))
- Primary: Dark gray (oklch(0.205 0 0))
- Accent: Light gray (oklch(0.97 0 0))

**Dark Mode**
- Background: Near black (oklch(0.145 0 0))
- Foreground: White (oklch(0.985 0 0))
- Primary: Light blue (oklch(0.488 0.243 264.376))
- Accent: Dark gray (oklch(0.269 0 0))

#### Theme Features
- System theme detection
- Manual light/dark toggle
- High-contrast mode support
- Reduced motion support
- Automatic OS theme sync

### Authentication Flow

1. **Login**: Email + password authentication
2. **JWT Token**: Stored in httpOnly cookies
3. **Protected Routes**: Middleware checks token validity
4. **Auto-redirect**: Unauthenticated users redirected to login
5. **Token Expiration**: Automatic 401 handling

### API Integration

#### Base URL
```
http://localhost:8080/api/v1
```

#### Request Format
- JWT token automatically included in Authorization header
- Automatic retry on network failures
- Global error handling with toast notifications
- 30-second request timeout

#### Mock Data
All pages include realistic mock data for demonstration purposes.

### Development

#### Dependencies
```bash
pnpm add next react react-dom
pnpm add @tanstack/react-query axios zod react-hook-form zustand
pnpm add sonner date-fns recharts uuid
pnpm add tailwindcss @tailwindcss/postcss postcss
pnpm add lucide-react class-variance-authority clsx tailwind-merge
```

#### Starting Development Server
```bash
pnpm dev
```

Server runs on `http://localhost:3000`

#### Building for Production
```bash
pnpm build
pnpm start
```

### Deployment

This project is optimized for deployment to Vercel:

1. Connect GitHub repository
2. Import project to Vercel
3. Set environment variables (if needed):
   - `NEXT_PUBLIC_API_URL`: Backend API URL
4. Deploy with `pnpm build`

### Form Validation

All forms use Zod schemas with React Hook Form:

```typescript
// Example schema
const loginSchema = z.object({
  email: z.string().email('Invalid email'),
  password: z.string().min(6, 'Minimum 6 characters'),
})

// Usage in component
const { register, handleSubmit, formState: { errors } } = useForm({
  resolver: zodResolver(loginSchema)
})
```

### State Management

#### Zustand Stores
- **Auth Store**: User data, JWT token, authentication status
- **Theme Store**: Theme mode (system/light/dark), resolved theme
- **UI Store**: Sidebar state, active filters, pagination

```typescript
import { useAuthStore } from '@/store/auth'
import { useThemeStore } from '@/store/theme'
import { useUIStore } from '@/store/ui'
```

### Data Fetching

Uses TanStack React Query for:
- Automatic caching (5 minutes stale time)
- Background refetching
- Error handling
- Loading states
- Automatic invalidation on mutations

```typescript
const { data, isLoading, error } = useQuery({
  queryKey: ['dataSources'],
  queryFn: () => api.get('/data-sources'),
})
```

### Performance Optimizations

1. **Code Splitting**: Route-based code splitting
2. **Image Optimization**: Next.js Image component
3. **CSS Optimization**: Tailwind CSS tree-shaking
4. **React 19**: Latest React features and optimizations
5. **Turbopack**: Fast builds with Turbopack
6. **Caching**: Query caching and HTTP caching

### Accessibility

- WCAG 2.1 AA compliance
- Semantic HTML elements
- Keyboard navigation support
- Screen reader friendly
- High-contrast mode support
- Reduced motion support
- ARIA labels and roles

### Browser Support

- Chrome (latest)
- Firefox (latest)
- Safari (latest)
- Edge (latest)

### License

Proprietary - Enterprise Data Governance Platform

### Support

For issues or feature requests, please contact the development team.
