# Infrastructure API Documentation

This directory contains the complete API documentation for the Infrastructure application.

## 📋 Available Documentation

### 🌐 Interactive API Documentation
- **Main Documentation**: [docs/api/index.html](./api/index.html) - Complete interactive Swagger UI
- **Alternative Swagger**: [docs/api/swagger.html](./api/swagger.html) - Alternative Swagger UI

### 📄 API Specifications
- **OpenAPI 3.0 Spec**: [docs/api/openapi.json](./api/openapi.json) - Raw OpenAPI specification

## 🚀 API Base URL
```
https://74zc1md7sc.execute-api.us-east-2.amazonaws.com/main
```

## 📚 API Endpoints Overview

### Organization Management
- `GET /org` - Get organization information
- `PUT /org` - Update organization name

### Location Management  
- `GET /locations` - List all locations
- `POST /locations` - Create new location
- `GET /locations/{id}` - Get location by ID
- `PUT /locations/{id}` - Update location
- `DELETE /locations/{id}` - Delete location

### Roles Management (Super Admin Only)
- `GET /roles` - List all roles
- `POST /roles` - Create new role
- `GET /roles/{id}` - Get role with permissions
- `PUT /roles/{id}` - Update role
- `DELETE /roles/{id}` - Delete role
- `POST /roles/{id}/permissions` - Assign permission to role
- `DELETE /roles/{id}/permissions` - Unassign permission from role

### Permissions Management (Super Admin Only)
- `GET /permissions` - List all permissions
- `POST /permissions` - Create new permission
- `GET /permissions/{id}` - Get permission by ID
- `PUT /permissions/{id}` - Update permission
- `DELETE /permissions/{id}` - Delete permission

## 🔐 Authentication
All endpoints require AWS Cognito JWT authentication via Bearer token in Authorization header.

## 🎯 How to Share Documentation

### Option 1: GitHub Pages (Recommended)
1. Enable GitHub Pages for this repository
2. Set source to `docs` folder
3. Share URL: `https://[username].github.io/[repository]/api/`

### Option 2: Local Hosting
```bash
cd docs
python -m http.server 8000
# Access at: http://localhost:8000/api/
```

### Option 3: Online Swagger Editor
1. Go to https://editor.swagger.io/
2. Upload `docs/api/openapi.json`
3. Share the browser URL

## 📖 Testing the API
- Use the included Postman collection: `postman/Infrastructure.postman_collection.json`
- Test credentials are provided in the Postman collection for dev environment