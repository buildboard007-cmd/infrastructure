# Super Admin Workflow - Quick Reference

## TL;DR - The Complete Flow

```
1. User Signs Up → pending_org_setup + pending org
2. User Sets Up Organization → org details updated
3. User Creates Location → BOTH user & org become active
4. User Access Dashboard → full access granted
```

## Status Transitions

| Component | Initial | After Org Setup | After Location | Final |
|-----------|---------|-----------------|----------------|-------|
| User | `pending_org_setup` | `pending_org_setup` | `active` | `active` |
| Organization | `pending` | `pending` | `active` | `active` |

## Key Files & Functions

### Backend Activation Logic
```
/src/lib/data/org_repository.go:checkAndUpdateUserStatus()
/src/lib/data/location_repository.go:checkAndUpdateUserStatusAfterLocation()
```

### Frontend Setup Pages
```
/src/app/setup-organization/page.tsx → PUT /org
/src/app/setup-location/page.tsx → POST /locations
```

### Lambdas Involved
```
infrastructure-user-signup (Post-Confirmation)
infrastructure-token-customizer (JWT Claims)
infrastructure-organization-management (Org CRUD)
infrastructure-location-management (Location CRUD)
```

## Test Email Format
```
buildboard007+{anything}@gmail.com
```

## Quick Debug Queries

### Check Status
```sql
SELECT u.email, u.status as user_status, o.status as org_status, o.name
FROM iam.users u 
JOIN iam.organizations o ON u.org_id = o.id
WHERE u.email = 'test@example.com';
```

### Manual Activation
```sql
BEGIN;
UPDATE iam.users SET status = 'active' WHERE email = 'test@example.com';
UPDATE iam.organizations SET status = 'active' 
WHERE id = (SELECT org_id FROM iam.users WHERE email = 'test@example.com');
COMMIT;
```

## API Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| PUT | `/org` | Update organization details |
| POST | `/locations` | Create first location |
| GET | `/org` | Get organization info |
| GET | `/locations` | List locations |

## Recent Bug Fix (2025-01-13)

**Problem**: Organization status stayed `pending` after setup
**Solution**: Added org status update to both activation methods
**Files**: `org_repository.go:340-380`, `location_repository.go:404-444`

## Error Troubleshooting

| Error | Cause | Solution |
|-------|-------|----------|
| Access Denied on setup pages | Not super admin or wrong status | Check JWT claims |
| User stuck pending | Location not created or activation failed | Check location count & manual activation |
| 500 on API calls | Database/Lambda errors | Check CloudWatch logs |

## Development Commands

```bash
# Start frontend
cd /Users/mayur/git_personal/ui/frontend && npm run dev

# Deploy infrastructure  
npm run build && cdk deploy --profile dev

# Check logs
aws logs tail /aws/lambda/infrastructure-organization-management --follow --profile dev
```