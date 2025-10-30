# Super Admin Workflow Documentation

## Overview

This document describes the complete super admin signup and organization setup workflow for BuildBoard. The workflow handles the initial account creation, email verification, organization setup, and location setup for super administrator users.

---

## ⚡ Quick Reference (TL;DR)

### The Complete Flow
```
1. User Signs Up → pending_org_setup + pending org
2. User Sets Up Organization → org details updated
3. User Creates Location → BOTH user & org become active
4. User Access Dashboard → full access granted
```

### Status Transitions
| Component | Initial | After Org Setup | After Location | Final |
|-----------|---------|-----------------|----------------|-------|
| User | `pending_org_setup` | `pending_org_setup` | `active` | `active` |
| Organization | `pending` | `pending` | `active` | `active` |

### Key Files
```
Backend: /src/lib/data/org_repository.go:checkAndUpdateUserStatus()
Backend: /src/lib/data/location_repository.go:checkAndUpdateUserStatusAfterLocation()
Frontend: /src/app/setup-organization/page.tsx
Frontend: /src/app/setup-location/page.tsx
```

### Quick Debug Commands
```bash
# Start frontend
cd /Users/mayur/git_personal/ui/frontend && npm run dev

# Deploy infrastructure
npm run build && cdk deploy --profile dev

# Check logs
aws logs tail /aws/lambda/infrastructure-organization-management --follow --profile dev
```

---

## Table of Contents

- [Quick Reference](#-quick-reference-tldr)
- [Architecture Overview](#architecture-overview)
- [Complete Workflow](#complete-workflow)
- [Technical Components](#technical-components)
- [Database Schema](#database-schema)
- [API Endpoints](#api-endpoints)
- [Frontend Flow](#frontend-flow)
- [Security Considerations](#security-considerations)
- [Recent Bug Fix](#recent-bug-fix)
- [Testing Guide](#testing-guide)
- [Troubleshooting](#troubleshooting)

## Architecture Overview

The super admin workflow involves multiple AWS services and components:

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Super Admin Workflow Architecture                │
└─────────────────────────────────────────────────────────────────────┘

Frontend (Next.js)                    Backend (AWS Lambda + RDS)
┌─────────────────┐                   ┌─────────────────────────────┐
│                 │                   │                             │
│ 1. Signup Form  │ ──────────────► │ AWS Cognito User Pool       │
│                 │                   │ - Email verification        │
│                 │                   │ - custom:isSuperAdmin=true  │
│                 │                   │                             │
│ 2. Verification │ ◄──────────────── │ Post-Confirmation Lambda    │
│    Email        │                   │ - Creates user in database  │
│                 │                   │ - Status: pending_org_setup │
│                 │                   │                             │
│ 3. Login        │ ──────────────► │ Token Customizer Lambda     │
│                 │                   │ - Adds user profile to JWT │
│                 │                   │ - Status determines routing │
│                 │                   │                             │
│ 4. Org Setup    │ ──────────────► │ Organization Management     │
│    Screen       │                   │ Lambda (PUT /org)           │
│                 │                   │                             │
│ 5. Location     │ ──────────────► │ Location Management         │
│    Setup Screen │                   │ Lambda (POST /locations)    │
│                 │                   │                             │
│ 6. Dashboard    │ ◄──────────────── │ Status Activation Logic     │
│    (Active)     │                   │ - User: active              │
│                 │                   │ - Organization: active      │
└─────────────────┘                   └─────────────────────────────┘
```

## Complete Workflow

### Phase 1: User Signup & Verification

#### Step 1: Initial Signup
- **User Action**: Fills out signup form with email and password
- **Frontend**: Calls AWS Cognito SignUp API with `custom:isSuperAdmin=true` attribute
- **Cognito**: Creates user account and sends verification email
- **User Status**: Account exists but unverified

#### Step 2: Email Verification
- **User Action**: Receives verification code via email and enters it
- **Cognito**: Verifies email and triggers Post-Confirmation Lambda
- **Post-Confirmation Lambda**: 
  - Creates user record in `iam.users` table
  - Status: `pending_org_setup`
  - Creates organization record in `iam.organizations` table  
  - Organization status: `pending`
  - Organization name: `NULL` (to be filled during setup)

### Phase 2: Organization Setup

#### Step 3: First Login & Routing
- **User Action**: Signs in with verified credentials
- **Token Customizer Lambda**: 
  - Fetches user profile from database
  - Adds user status and profile data to JWT token
  - Returns token with `status: "pending_org_setup"`
- **Frontend**: 
  - Reads status from JWT token
  - Redirects user to `/setup-organization` page
  - Blocks access to dashboard until setup complete

#### Step 4: Organization Setup Form
- **User Action**: Fills out organization details form
  - Organization name (required)
  - Organization type (required)
  - License number (optional)
  - Address, phone, email, website
- **Frontend**: Submits via `PUT /org` endpoint
- **Organization Lambda**: Updates organization record with provided details
- **Activation Check**: Checks if user should be activated (requires location too)
- **Frontend**: Redirects to location setup page

### Phase 3: Location Setup & Activation

#### Step 5: Location Setup Form
- **User Action**: Creates first business location
  - Location name (required)
  - Location type (office/warehouse/job_site/yard)
  - Address details (required)
  - Status (active/inactive/under_construction/closed)
- **Frontend**: Submits via `POST /locations` endpoint  
- **Location Lambda**: Creates location record
- **Activation Trigger**: Calls activation logic after location creation

#### Step 6: Status Activation
- **System**: Checks activation conditions:
  - User status is `pending_org_setup`
  - Organization has been updated (name != NULL)
  - At least one location exists
- **Database Transaction**: 
  - Updates user status: `pending_org_setup` → `active`
  - Updates organization status: `pending` → `active`
- **Frontend**: Redirects to dashboard with `?setup=completed` parameter

### Phase 4: Active Operation

#### Step 7: Dashboard Access
- **User**: Now has full access to BuildBoard features
- **JWT Token**: Contains `status: "active"` and full profile data
- **System**: User can manage projects, users, locations, etc.

## Technical Components

### 1. Post-Confirmation Lambda
**Location**: `/src/infrastructure-user-signup/main.go`

**Key Functions**:
- `Handler()`: Main entry point for Cognito trigger
- `processSuperAdminSignup()`: Creates user and organization records
- `extractSignupData()`: Parses Cognito event data

**Database Operations**:
```sql
-- Create organization with NULL name (to be filled during setup)
INSERT INTO iam.organizations (name, org_type, status, created_by, updated_by)
VALUES (NULL, NULL, 'pending', 1, 1)

-- Create super admin user
INSERT INTO iam.users (
    cognito_id, org_id, email, first_name, last_name, phone,
    status, is_super_admin, created_by, updated_by
) VALUES ($1, $2, $3, $4, $5, $6, 'pending_org_setup', true, 1, 1)
```

### 2. Token Customizer Lambda  
**Location**: `/src/infrastructure-token-customizer/main.go`

**Key Functions**:
- `Handler()`: Main JWT customization entry point
- `buildCustomClaims()`: Converts database profile to JWT claims
- Auto-activates pending users on authentication

**JWT Claims Added**:
```json
{
  "user_id": "123",
  "cognito_id": "uuid",
  "email": "user@company.com", 
  "first_name": "John",
  "last_name": "Doe",
  "full_name": "John Doe",
  "status": "pending_org_setup|active",
  "org_id": "456",
  "org_name": "ACME Construction",
  "last_selected_location_id": "789",
  "isSuperAdmin": true,
  "locations": "base64encodedJSON"
}
```

### 3. Organization Management Lambda
**Location**: `/src/infrastructure-organization-management/main.go`

**Endpoints**:
- `PUT /org`: Update organization details
- `GET /org`: Retrieve organization info

**Key Functions**:
- `handleUpdateOrganization()`: Updates org and triggers activation check
- `checkAndUpdateUserStatus()`: Activates user/org when conditions met

### 4. Location Management Lambda  
**Location**: `/src/infrastructure-location-management/main.go`

**Endpoints**:
- `POST /locations`: Create new location
- `GET /locations`: List locations
- `PUT /locations/{id}`: Update location
- `DELETE /locations/{id}`: Delete location

**Activation Logic**: Triggers after successful location creation

## Database Schema

### Users Table (`iam.users`)
```sql
CREATE TABLE iam.users (
    id BIGSERIAL PRIMARY KEY,
    cognito_id VARCHAR(255) UNIQUE NOT NULL,
    org_id BIGINT NOT NULL REFERENCES iam.organizations(id),
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255), 
    phone VARCHAR(20),
    mobile VARCHAR(20),
    job_title VARCHAR(255),
    employee_id VARCHAR(100),
    avatar_url TEXT,
    last_selected_location_id BIGINT,
    status VARCHAR(50) DEFAULT 'pending',  -- pending|active|inactive|suspended|pending_org_setup
    is_super_admin BOOLEAN DEFAULT FALSE,
    email_verified BOOLEAN DEFAULT FALSE,
    phone_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT,
    updated_by BIGINT,
    is_deleted BOOLEAN DEFAULT FALSE
);
```

### Organizations Table (`iam.organizations`)
```sql
CREATE TABLE iam.organizations (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255),  -- NULL initially, filled during setup
    org_type VARCHAR(100),
    license_number VARCHAR(255),
    address TEXT,
    phone VARCHAR(20),
    email VARCHAR(255),
    website VARCHAR(500),
    status VARCHAR(50) DEFAULT 'pending_setup',  -- pending|active|inactive|suspended
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,  
    created_by BIGINT,
    updated_by BIGINT,
    is_deleted BOOLEAN DEFAULT FALSE
);
```

### Locations Table (`iam.locations`)
```sql
CREATE TABLE iam.locations (
    id BIGSERIAL PRIMARY KEY,
    org_id BIGINT NOT NULL REFERENCES iam.organizations(id),
    name VARCHAR(255) NOT NULL,
    location_type VARCHAR(50), -- office|warehouse|job_site|yard
    address TEXT,
    city VARCHAR(255),
    state VARCHAR(100), 
    zip_code VARCHAR(20),
    country VARCHAR(100) DEFAULT 'USA',
    timezone VARCHAR(100),
    status VARCHAR(50) DEFAULT 'active', -- active|inactive|under_construction|closed
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT,
    updated_by BIGINT,
    is_deleted BOOLEAN DEFAULT FALSE
);
```

## API Endpoints

### Organization Management
```
PUT /org
Authorization: Bearer JWT
Content-Type: application/json

Request Body:
{
  "name": "ACME Construction Company",
  "org_type": "general_contractor", 
  "license_number": "GC123456",
  "address": "123 Main St, Suite 100\nAnytown, ST 12345",
  "phone": "+1 (555) 123-4567",
  "email": "contact@acmeconstruction.com",
  "website": "https://acmeconstruction.com"
}

Response:
{
  "id": 123,
  "name": "ACME Construction Company",
  "org_type": "general_contractor",
  "status": "active",  // Updated after location creation
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Location Management
```
POST /locations  
Authorization: Bearer JWT
Content-Type: application/json

Request Body:
{
  "name": "Main Office",
  "location_type": "office",
  "address": "123 Business Blvd", 
  "city": "Business City",
  "state": "ST",
  "zip_code": "12345",
  "country": "USA",
  "status": "active"
}

Response:
{
  "id": 456,
  "org_id": 123,
  "name": "Main Office", 
  "location_type": "office",
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z"
}
```

## Frontend Flow

### 1. Authentication State Management
**File**: `/src/store/authSlice.ts`

```typescript
interface AuthState {
  user: UserProfile | null;
  isAuthenticated: boolean;
  status: 'pending_org_setup' | 'active' | 'inactive' | 'suspended';
}
```

### 2. Route Protection
**Files**: 
- `/src/app/setup-organization/page.tsx`
- `/src/app/setup-location/page.tsx`

**Logic**:
```typescript
// Only super admins with pending_org_setup can access
if (!user?.isSuperAdmin || user?.status !== 'pending_org_setup') {
  return <AccessDenied />;
}
```

### 3. Setup Flow Navigation
```typescript
// Organization setup success
router.push('/setup-location');

// Location setup success  
router.push('/dashboard?setup=completed');
```

## Security Considerations

### 1. Authentication & Authorization
- **JWT Tokens**: All API calls require valid JWT with user profile
- **Super Admin Check**: Only users with `isSuperAdmin: true` can access setup
- **Organization Isolation**: Users can only modify their own organization
- **Status Validation**: Frontend enforces proper setup sequence

### 2. Input Validation
- **Required Fields**: Name, org_type, email for organization
- **Email Format**: Valid email format validation
- **SQL Injection**: All queries use parameterized statements
- **XSS Protection**: Input sanitization on both frontend and backend

### 3. Database Security
- **Transactions**: Atomic updates for status changes
- **Soft Deletes**: `is_deleted` flag prevents data loss
- **Audit Trail**: `created_by`, `updated_by`, timestamps for all changes
- **Connection Security**: SSL connections with credential rotation via SSM

## Recent Bug Fix

### Issue Identified
After successful organization and location setup, users were being activated but **organizations remained in `pending` status** instead of becoming `active`.

### Root Cause
Two activation methods existed but both only updated user status:
- `OrgDao.checkAndUpdateUserStatus()` - triggered after org update  
- `LocationDao.checkAndUpdateUserStatusAfterLocation()` - triggered after location creation

### Solution Applied

**Files Modified**:
- `/src/lib/data/org_repository.go` (lines 340-380)
- `/src/lib/data/location_repository.go` (lines 404-444)  

**Fix Details**:
```go
// BEFORE: Only user status updated
_, err = dao.DB.ExecContext(ctx, `
    UPDATE iam.users SET status = 'active' 
    WHERE id = $1 AND status = 'pending_org_setup'
`, userID, userID)

// AFTER: Both user and organization status updated atomically
tx, err := dao.DB.BeginTx(ctx, nil)
// Update user status
_, err = tx.ExecContext(ctx, `
    UPDATE iam.users SET status = 'active' 
    WHERE id = $1 AND status = 'pending_org_setup'
`, userID, userID)
// Update organization status  
_, err = tx.ExecContext(ctx, `
    UPDATE iam.organizations SET status = 'active'
    WHERE id = $2 AND status = 'pending'  
`, userID, orgID)
tx.Commit()
```

### Impact
- ✅ Users properly transition: `pending_org_setup` → `active`
- ✅ Organizations properly transition: `pending` → `active`  
- ✅ JWT tokens reflect correct status for frontend routing
- ✅ Database consistency maintained via atomic transactions

## Testing Guide

### Manual Testing Flow

1. **Start Frontend**:
   ```bash
   cd /Users/mayur/git_personal/ui/frontend
   npm run dev
   ```
   Navigate to: http://localhost:3001

2. **Super Admin Signup**:
   - Go to signup page
   - Use test email: `buildboard007+test1@gmail.com`
   - Enter password (minimum 8 characters with uppercase, lowercase, number)
   - Verify account with email code (request from developer)

3. **Organization Setup**:
   - Should automatically redirect to `/setup-organization`
   - Fill required fields: name, org_type, email
   - Submit form → should redirect to `/setup-location`

4. **Location Setup**:
   - Fill required fields: name, location_type, address, city, state
   - Submit form → should redirect to `/dashboard?setup=completed`
   - Check JWT token contains `status: "active"`

5. **Verification**:
   - Dashboard should be fully accessible
   - User should be able to create additional locations/users
   - Database should show both user and organization with `status: "active"`

### Database Verification Queries

```sql
-- Check user status after setup
SELECT id, email, status, is_super_admin, org_id 
FROM iam.users 
WHERE email = 'buildboard007+test1@gmail.com';

-- Check organization status  
SELECT id, name, org_type, status
FROM iam.organizations 
WHERE id = (SELECT org_id FROM iam.users WHERE email = 'buildboard007+test1@gmail.com');

-- Check locations created
SELECT id, name, location_type, status, org_id
FROM iam.locations 
WHERE org_id = (SELECT org_id FROM iam.users WHERE email = 'buildboard007+test1@gmail.com');
```

### API Testing with Postman

1. **Get JWT Token**: Sign in through frontend, copy token from browser dev tools
2. **Test Organization Update**:
   ```
   PUT {{baseUrl}}/org
   Authorization: Bearer {{jwt_token}}
   ```
3. **Test Location Creation**:
   ```
   POST {{baseUrl}}/locations  
   Authorization: Bearer {{jwt_token}}
   ```

## Troubleshooting

### Common Issues

#### 1. User Stuck in `pending_org_setup` Status
**Symptoms**: User can access org setup but not dashboard after completion
**Causes**: 
- Organization update failed
- Location creation failed
- Activation logic not triggered

**Debug Steps**:
```sql
-- Check user status
SELECT status FROM iam.users WHERE email = 'user@example.com';

-- Check organization status  
SELECT status, name FROM iam.organizations WHERE id = (
  SELECT org_id FROM iam.users WHERE email = 'user@example.com'
);

-- Check location count
SELECT COUNT(*) FROM iam.locations WHERE org_id = (
  SELECT org_id FROM iam.users WHERE email = 'user@example.com'
);
```

**Manual Fix**:
```sql
-- Manually activate user and organization
BEGIN;
UPDATE iam.users SET status = 'active' WHERE email = 'user@example.com';
UPDATE iam.organizations SET status = 'active' WHERE id = (
  SELECT org_id FROM iam.users WHERE email = 'user@example.com'  
);
COMMIT;
```

#### 2. Access Denied on Setup Pages
**Symptoms**: "Access Denied" message on setup screens
**Causes**:
- User not marked as super admin
- User status not `pending_org_setup`
- JWT token missing required claims

**Debug Steps**:
```javascript
// In browser console, check JWT token
const token = localStorage.getItem('idToken');
const payload = JSON.parse(atob(token.split('.')[1]));
console.log('Status:', payload.status);
console.log('isSuperAdmin:', payload.isSuperAdmin);
```

#### 3. Organization/Location API Errors
**Symptoms**: API calls failing with 400/500 errors
**Causes**:
- Missing required fields
- Database connection issues
- Authorization failures

**Debug Steps**:
1. Check CloudWatch logs for Lambda errors
2. Verify JWT token validity
3. Test API endpoints with Postman
4. Check database connectivity

### Logs and Monitoring

#### CloudWatch Log Groups
- `/aws/lambda/infrastructure-user-signup`
- `/aws/lambda/infrastructure-token-customizer`  
- `/aws/lambda/infrastructure-organization-management`
- `/aws/lambda/infrastructure-location-management`

#### Key Log Messages
```
// Successful user creation
"User status updated to active after organization update and location creation"

// Token customization
"Successfully added custom claims to token"

// Activation triggers
"Successfully activated pending user on first login"
```

---

## Conclusion

The super admin workflow provides a robust, secure way for new organizations to onboard to BuildBoard. The recent bug fix ensures both users and organizations are properly activated after setup completion, providing a seamless experience from signup to active dashboard use.

For additional support or questions, refer to the API documentation in `/docs/api/` or contact the development team.