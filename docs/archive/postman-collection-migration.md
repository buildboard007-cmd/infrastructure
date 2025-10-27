# Postman Collection Migration for Multi-API Architecture

## 🚨 **URGENT: Postman Collection Needs Updates**

The Postman collection is **NOT up-to-date** with the new multi-API Gateway architecture and will **fail** when tested against the deployed infrastructure.

## **Current Problem**
All endpoints currently use a single API Gateway ID:
```
74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/{endpoint}
```

## **Required Changes**
After deployment, you'll need to update the Postman collection to use **4 different API Gateway IDs**:

### **1. IAM API Endpoints**
**Replace**: `74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`
**With**: `{{IAM_API_ID}}.execute-api.us-east-2.amazonaws.com/main`

**Affected endpoints:**
- `/org` (2 requests)
- `/users` (2 requests) 
- `/users/{{user_id}}` (3 requests)
- `/users/{{user_id}}/reset-password` (1 request)
- `/locations` (2 requests)
- `/locations/{{location_id}}` (3 requests)
- `/roles` (2 requests)
- `/roles/{{role_id}}` (3 requests)
- `/roles/{{role_id}}/permissions` (2 requests)
- `/permissions` (2 requests)
- `/permissions/{{permission_id}}` (3 requests)

### **2. Projects API Endpoints**
**Replace**: `74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`
**With**: `{{PROJECTS_API_ID}}.execute-api.us-east-2.amazonaws.com/main`

**Affected endpoints:**
- `/projects` (2 requests)
- `/projects?location_id={{location_id}}` (1 request)
- `/projects/{{project_id}}` (3 requests)
- `/projects/{{project_id}}/managers` (2 requests)
- `/projects/{{project_id}}/managers/{{manager_id}}` (3 requests)
- `/projects/{{project_id}}/attachments` (2 requests)
- `/projects/{{project_id}}/attachments/{{attachment_id}}` (2 requests)
- `/projects/{{project_id}}/users` (2 requests)
- `/projects/{{project_id}}/users/{{assignment_id}}` (2 requests)

### **3. Issues API Endpoints**
**Replace**: `74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`
**With**: `{{ISSUES_API_ID}}.execute-api.us-east-2.amazonaws.com/main`

**Affected endpoints:**
- `/issues` (1 request)
- `/issues/{{issue_id}}` (3 requests)
- `/issues/{{issue_id}}/status` (1 request)
- `/projects/{{project_id}}/issues?status=open&priority=high` (2 requests)

### **4. RFIs API Endpoints**
**Replace**: `74zc1md7sc.execute-api.us-east-2.amazonaws.com/main`
**With**: `{{RFIS_API_ID}}.execute-api.us-east-2.amazonaws.com/main`

**Affected endpoints:**
- `/rfis` (1 request)
- `/rfis/{{rfi_id}}` (3 requests)
- `/rfis/{{rfi_id}}/status` (1 request)
- `/rfis/{{rfi_id}}/submit` (1 request)
- `/rfis/{{rfi_id}}/respond` (1 request)
- `/rfis/{{rfi_id}}/approve` (1 request)
- `/rfis/{{rfi_id}}/reject` (1 request)
- `/rfis/{{rfi_id}}/attachments` (2 requests)
- `/rfis/{{rfi_id}}/comments` (2 requests)
- `/projects/{{project_id}}/rfis?status=submitted&priority=high` (2 requests)

## **Steps to Update**

### **Step 1: Get API Gateway IDs**
After deployment, get the API Gateway IDs from AWS Console or CDK outputs:
```bash
aws cloudformation describe-stacks --stack-name Dev-Infrastructure-AppStage --query 'Stacks[0].Outputs'
```

### **Step 2: Set Postman Environment Variables**
Create/update these variables in your Postman environment:
- `IAM_API_ID` = [IAM API Gateway ID]
- `PROJECTS_API_ID` = [Projects API Gateway ID]  
- `ISSUES_API_ID` = [Issues API Gateway ID]
- `RFIS_API_ID` = [RFIs API Gateway ID]

### **Step 3: Update Collection URLs**
Replace all hardcoded API Gateway IDs with the appropriate variable placeholders as specified above.

## **Alternative: Quick Fix Script**
You can use find/replace in your JSON editor:

1. **Find**: `"74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/org"`
   **Replace**: `"{{IAM_API_ID}}.execute-api.us-east-2.amazonaws.com/main/org"`

2. **Find**: `"74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/users"`
   **Replace**: `"{{IAM_API_ID}}.execute-api.us-east-2.amazonaws.com/main/users"`

3. **Find**: `"74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/locations"`
   **Replace**: `"{{IAM_API_ID}}.execute-api.us-east-2.amazonaws.com/main/locations"`

4. **Find**: `"74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/roles"`
   **Replace**: `"{{IAM_API_ID}}.execute-api.us-east-2.amazonaws.com/main/roles"`

5. **Find**: `"74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/permissions"`
   **Replace**: `"{{IAM_API_ID}}.execute-api.us-east-2.amazonaws.com/main/permissions"`

6. **Find**: `"74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/projects"`
   **Replace**: `"{{PROJECTS_API_ID}}.execute-api.us-east-2.amazonaws.com/main/projects"`

7. **Find**: `"74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/issues"`
   **Replace**: `"{{ISSUES_API_ID}}.execute-api.us-east-2.amazonaws.com/main/issues"`

8. **Find**: `"74zc1md7sc.execute-api.us-east-2.amazonaws.com/main/rfis"`
   **Replace**: `"{{RFIS_API_ID}}.execute-api.us-east-2.amazonaws.com/main/rfis"`

## **⚠️ Critical Impact**
**Until these updates are made, the Postman collection will not work** with the new deployed infrastructure. All API calls will return 404 Not Found errors because they'll be hitting the old (non-existent) single API Gateway.

## **Testing Priority**
Update the collection **immediately after deployment** to ensure:
1. ✅ Authentication endpoints still work (Cognito - unchanged)
2. ✅ IAM endpoints work with new IAM API
3. ✅ Projects endpoints work with new Projects API  
4. ✅ Issues endpoints work with new Issues API
5. ✅ RFIs endpoints work with new RFIs API

The multi-API architecture will provide better performance and eliminate deployment issues, but requires this Postman collection update to be functional.