# Multi-API Gateway Architecture

## 🚀 **Overview**

This document outlines the new multi-API Gateway architecture designed to solve API Gateway resource limits and IAM policy size constraints.

## ⚠️ **Problems with Current Architecture**

### **Current Single API Structure**
```
https://api.domain.com/infra/
├── org/                    # Organization Management
├── users/                  # User Management  
├── locations/              # Location Management
├── roles/                  # Roles Management
├── permissions/            # Permissions Management
├── projects/               # Project Management
├── issues/                 # Issue Management
└── rfis/                   # RFI Management
```

### **Issues**
1. **API Gateway Resource Limits**: Too many resources under single API Gateway
2. **IAM Policy Size Limits**: Lambda functions exceed 20KB policy limit
3. **Deployment Complexity**: Changes to any module affect all modules  
4. **Poor Organization**: Hard to manage different modules independently

## ✅ **New Multi-API Architecture**

### **Final Structure**
```
https://api.domain.com/
├── iam/                  # IAM API Gateway (consolidated)
│   ├── users/           # User Management
│   ├── org/             # Organization Management
│   ├── locations/       # Location Management
│   ├── roles/           # Roles Management
│   └── permissions/     # Permissions Management
├── projects/            # Projects API Gateway
│   └── projects/        # Project Management
├── issues/             # Issues API Gateway
│   ├── issues/         # Issue Management
│   └── projects/{id}/issues/  # Project-specific issues
└── rfis/              # RFIs API Gateway
    ├── rfis/          # RFI Management  
    └── projects/{id}/rfis/    # Project-specific RFIs
```

## 🏗️ **Architecture Benefits**

### **1. Resource Distribution**
- **4 separate API Gateways** instead of 1
- **Reduced resources per API** (under limits)
- **Better scalability** and performance

### **2. IAM Policy Optimization**
- **Smaller policies per Lambda** (under 20KB limit)
- **Scoped permissions** per module
- **Reduced deployment failures**

### **3. Independent Deployments**
- **Module-specific deployments** possible
- **Faster iteration** on individual modules
- **Reduced blast radius** for changes

### **4. Logical Organization**
- **Related endpoints grouped** together
- **Cleaner API structure** and documentation
- **Easier maintenance** and updates

## 📋 **API Gateway Breakdown**

### **IAM API** (`/iam/`)
**Purpose**: Identity & Access Management (consolidated)
**Endpoints**:
- `GET|PUT /org` - Organization operations
- `GET|POST /users` - User listing and creation
- `GET|PUT|DELETE /users/{id}` - User operations
- `PATCH /users/{id}/reset-password` - Password reset
- `GET|POST /locations` - Location management
- `GET|PUT|DELETE /locations/{id}` - Location operations
- `GET|POST /roles` - Role management  
- `GET|PUT|DELETE /roles/{id}` - Role operations
- `POST|DELETE /roles/{id}/permissions` - Role permissions
- `GET|POST /permissions` - Permission management
- `GET|PUT|DELETE /permissions/{id}` - Permission operations

### **Projects API** (`/projects/`)
**Purpose**: Project and project-related resource management
**Endpoints**:
- `GET|POST /projects` - Project management
- `GET|PUT|DELETE /projects/{id}` - Project operations
- `GET|POST /projects/{id}/managers` - Project managers
- `GET|PUT|DELETE /projects/{id}/managers/{managerId}` - Manager operations
- `GET|POST /projects/{id}/attachments` - Project attachments
- `GET|DELETE /projects/{id}/attachments/{attachmentId}` - Attachment operations
- `GET|POST /projects/{id}/users` - Project user assignments
- `PUT|DELETE /projects/{id}/users/{assignmentId}` - Assignment operations

### **Issues API** (`/issues/`)
**Purpose**: Issue tracking and management
**Endpoints**:
- `POST /issues` - Create issue
- `GET|PUT|DELETE /issues/{id}` - Issue operations
- `PATCH /issues/{id}/status` - Status updates
- `GET|POST /projects/{projectId}/issues` - Project issues

### **RFIs API** (`/rfis/`)
**Purpose**: Request for Information management
**Endpoints**:
- `POST /rfis` - Create RFI
- `GET|PUT|DELETE /rfis/{id}` - RFI operations
- `PATCH /rfis/{id}/status` - Status updates
- `POST /rfis/{id}/submit` - Submit for review
- `POST /rfis/{id}/respond` - Provide response
- `POST /rfis/{id}/approve` - Approve RFI
- `POST /rfis/{id}/reject` - Reject RFI
- `GET|POST /rfis/{id}/attachments` - RFI attachments
- `GET|DELETE /rfis/{id}/attachments/{attachmentId}` - Attachment operations
- `GET|POST /rfis/{id}/comments` - RFI comments
- `GET|POST /projects/{projectId}/rfis` - Project RFIs

## 🔧 **Implementation Details**

### **Files Created**
1. **`multi-api-sub-stack.ts`** - New multi-API Gateway implementation
2. **`multi-api-main-stack.ts`** - Updated main stack for multi-API architecture
3. **`multi-api-architecture.md`** - This documentation file

### **Key Features**
- **Shared Cognito Authorizer** across all APIs
- **Shared CORS Lambda** for all OPTIONS requests
- **Individual Lambda integrations** per management module
- **Separate base path mappings** for domain setup
- **CloudFormation outputs** for all API URLs and IDs

### **Domain Mappings**
When deployed with a custom domain:
```
https://yourdomain.com/iam/         # IAM API (consolidated)
https://yourdomain.com/projects/    # Projects API
https://yourdomain.com/issues/      # Issues API
https://yourdomain.com/rfis/        # RFIs API
```

## 🚀 **Migration Path**

### **Option 1: Gradual Migration**
1. Deploy new multi-API architecture alongside existing
2. Update client applications module by module
3. Decommission old single API when migration complete

### **Option 2: Complete Replacement**
1. Replace current `SubStack` with `MultiApiSubStack`
2. Update Postman collection with new endpoints
3. Deploy all at once

### **Recommendation**
**Option 2 (Complete Replacement)** is recommended because:
- Avoids maintaining dual architecture
- Solves deployment issues immediately  
- Cleaner long-term solution
- API structure is logically better organized

## 📊 **Expected Improvements**

### **Before (Single API)**
- ❌ 1 API Gateway with 50+ resources
- ❌ Lambda IAM policies >20KB (deployment failures)
- ❌ All modules coupled together
- ❌ Complex endpoint structure

### **After (Multi-API)**
- ✅ 4 API Gateways with ~10-15 resources each
- ✅ Lambda IAM policies <10KB (no deployment failures)
- ✅ Independent module deployments
- ✅ Clean, logical endpoint structure

## 🎯 **Next Steps**

1. **Review Architecture**: Confirm the proposed structure meets requirements
2. **Update Postman Collection**: Modify collection for new API endpoints
3. **Replace Current Implementation**: Switch from `SubStack` to `MultiApiSubStack`
4. **Test Deployment**: Verify deployment succeeds without IAM policy errors
5. **Update Client Applications**: Modify frontend/client to use new API structure

## 📞 **Questions & Considerations**

1. **Are the proposed API groupings logical** for your use case?
2. **Should any endpoints be moved** between different APIs?
3. **Do you prefer gradual migration** or complete replacement?
4. **Any specific naming conventions** for the base paths?

This architecture will solve your current deployment issues while providing a more scalable and maintainable API structure! 🚀