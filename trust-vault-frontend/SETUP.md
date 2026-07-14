# SecureLens Frontend - Setup Guide

## Quick Start

### Prerequisites
- Node.js 18+ 
- pnpm 8+ (or npm/yarn)

### Installation

1. **Clone the repository**
```bash
git clone <repository-url>
cd trust-vault-frontend
```

2. **Install dependencies**
```bash
pnpm install
# or: npm install
# or: yarn install
```

3. **Start development server**
```bash
pnpm dev
```

Visit `http://localhost:3000` to see the application.

## Environment Configuration

### 1. Create `.env.local` file
```bash
cp .env.example .env.local
```

### 2. Update API endpoint (if using backend)
```env
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

### 3. Optional: Add analytics
```env
NEXT_PUBLIC_ANALYTICS_ID=your_analytics_id
```

## Project Structure

### `/app` - Pages and Routing
- `(auth)/` - Authentication pages (login, forgot password)
- `(app)/` - Protected application pages
  - `dashboard/` - Main dashboard
  - `data-sources/` - Data source management
  - `classification/` - Data classification
  - `governance/` - Policy management
  - `ai-gate/` - AI query control
  - And 10+ more sections...

### `/components` - React Components
- `base/` - Reusable UI components (DataTable, StatCard, etc.)
- `layout/` - Layout components (TopBar, Sidebar)
- `providers.tsx` - React context providers

### `/lib` - Utilities and Configuration
- `api.ts` - Axios API client with interceptors
- `types.ts` - TypeScript type definitions
- `schemas.ts` - Zod validation schemas
- `utils.ts` - Utility functions

### `/store` - State Management
- `auth.ts` - Authentication state (Zustand)
- `theme.ts` - Theme state (Zustand)
- `ui.ts` - UI state (Zustand)

## Authentication Flow

### 1. Login Page
- Navigate to `/login`
- Enter email and password
- Submit to backend `/auth/login` endpoint
- JWT token stored in httpOnly cookie
- Redirect to `/dashboard` on success

### 2. Protected Routes
- App layout checks `useAuthStore().isAuthenticated`
- Automatic redirect to login if not authenticated
- Token included in all API requests via interceptor

### 3. Logout
- Click user menu in top bar
- Select logout
- Clear cookies and auth store
- Redirect to login

## API Integration

### Current State
All pages include mock data for demonstration. To connect the backend:

### 1. Update API Calls
Replace API calls in each page component from:
```typescript
// Current: Mock data
const mockDataSources: DataSource[] = [...]
```

To:
```typescript
// API integration
const { data: dataSources } = useQuery({
  queryKey: ['dataSources'],
  queryFn: () => api.get('/data-sources')
})
```

### 2. Key API Endpoints to Implement
```
POST   /auth/login
POST   /auth/forgot-password
GET    /data-sources
POST   /data-sources
GET    /data-sources/:id
GET    /classification
GET    /governance/policies
POST   /governance/policies
GET    /privacy/dsar
POST   /privacy/dsar
GET    /audit
GET    /quality
GET    /ai-gate/queries
GET    /observability/health
```

### 3. Request/Response Format
```typescript
// Login request
POST /auth/login
{
  "email": "user@example.com",
  "password": "password123"
}

// Login response
{
  "token": "eyJhbGc...",
  "user": {
    "id": "123",
    "email": "user@example.com",
    "name": "John Doe",
    "role": "admin"
  }
}
```

## Form Validation

All forms use React Hook Form + Zod validation:

```typescript
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { loginSchema } from '@/lib/schemas'

const { register, handleSubmit, formState: { errors } } = useForm({
  resolver: zodResolver(loginSchema)
})
```

## Theme System

### Switching Themes
- Click sun/moon icon in top bar
- Theme persists in localStorage
- Auto-detects system theme preference

### Available Themes
- `light` - Light mode
- `dark` - Dark mode
- `system` - Use OS preference

### OKLCH Colors
All colors use OKLCH color space:
```css
--primary: oklch(0.488 0.243 264.376)
--background: oklch(0.145 0 0)
--foreground: oklch(0.985 0 0)
```

## State Management

### Using Zustand Stores

#### Auth Store
```typescript
import { useAuthStore } from '@/store/auth'

function MyComponent() {
  const { user, token, isAuthenticated, login, logout } = useAuthStore()
  
  return <div>{user?.email}</div>
}
```

#### Theme Store
```typescript
import { useThemeStore } from '@/store/theme'

function MyComponent() {
  const { mode, toggle } = useThemeStore()
  
  return <button onClick={toggle}>Toggle Theme</button>
}
```

#### UI Store
```typescript
import { useUIStore } from '@/store/ui'

function MyComponent() {
  const { sidebarOpen, toggleSidebar } = useUIStore()
  
  return <button onClick={toggleSidebar}>Toggle Sidebar</button>
}
```

## Data Fetching

### Using React Query

```typescript
import { useQuery, useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'

// Fetch data
function DataComponent() {
  const { data, isLoading } = useQuery({
    queryKey: ['dataSources'],
    queryFn: () => api.get('/data-sources')
  })
  
  if (isLoading) return <div>Loading...</div>
  return <div>{data?.length} sources</div>
}

// Mutate data
function CreateComponent() {
  const queryClient = useQueryClient()
  const { mutate } = useMutation({
    mutationFn: (data) => api.post('/data-sources', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dataSources'] })
    }
  })
  
  return <button onClick={() => mutate(newData)}>Create</button>
}
```

## Error Handling

### API Errors
```typescript
// Automatic error handling in interceptor
// 401 errors redirect to login
// Other errors show toast notifications
```

### Form Validation Errors
```typescript
{errors.email && (
  <p className="text-red-500">{errors.email.message}</p>
)}
```

## Deployment

### To Vercel
```bash
vercel deploy
```

### Environment Variables
Set in Vercel project settings:
- `NEXT_PUBLIC_API_URL` - Backend API URL
- Any other environment variables needed

### Build Command
```bash
pnpm build
```

### Start Command
```bash
pnpm start
```

## Development Tips

### Hot Reload
Changes automatically reload in browser during development.

### Console Logs
Use `console.log()` for debugging - visible in browser console.

### Mock Data
Mock data is included in each page for testing. Replace with API calls later.

### Component Reusability
Create new components in `/components/base/` for reuse across pages.

### Type Safety
Always add TypeScript types - check `/lib/types.ts` for common types.

## Troubleshooting

### Port Already in Use
```bash
# Find process using port 3000
lsof -i :3000
# Kill the process
kill -9 <PID>
# Or use different port
pnpm dev -p 3001
```

### Build Errors
```bash
# Clear Next.js cache
rm -rf .next
pnpm dev
```

### Dependencies Issues
```bash
# Reinstall dependencies
rm -rf node_modules pnpm-lock.yaml
pnpm install
```

### API Connection Issues
- Verify backend is running at `NEXT_PUBLIC_API_URL`
- Check network tab in browser DevTools
- Verify API response format matches expected types

## Additional Resources

- [Next.js Documentation](https://nextjs.org/docs)
- [React Documentation](https://react.dev)
- [Tailwind CSS Documentation](https://tailwindcss.com/docs)
- [shadcn/ui Components](https://ui.shadcn.com)
- [Zustand Documentation](https://github.com/pmndrs/zustand)
- [React Query Documentation](https://tanstack.com/query/latest)

## Support

For issues or questions:
1. Check the README.md file
2. Review the BUILD_SUMMARY.md for overview
3. Check browser console for errors
4. Contact development team

## Version Information

- Next.js: 16.2.6
- React: 19.2.4
- Node.js: 18+ required
- pnpm: 8+ recommended

---

Last Updated: July 8, 2026
