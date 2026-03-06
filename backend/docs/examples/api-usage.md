# API Usage Examples

Complete examples for using ArchPlatform API.

## Table of Contents

- [Authentication](#authentication)
- [User Management](#user-management)
- [Project Management](#project-management)
- [File Operations](#file-operations)
- [Collaboration](#collaboration)

---

## Authentication

### Register a New User

```bash
curl -X POST https://api.archplatform.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john.doe",
    "email": "john@example.com",
    "password": "SecurePass123",
    "firstName": "John",
    "lastName": "Doe"
  }'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "john.doe",
  "email": "john@example.com",
  "firstName": "John",
  "lastName": "Doe",
  "roles": ["USER"],
  "createdAt": "2024-01-15T10:30:00Z"
}
```

### Login

```bash
curl -X POST https://api.archplatform.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john.doe",
    "password": "SecurePass123"
  }'
```

Response:
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
  "expiresIn": 3600,
  "tokenType": "Bearer",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "john.doe",
    "email": "john@example.com"
  }
}
```

### Using the Token

Include the token in all subsequent requests:

```bash
curl https://api.archplatform.com/api/v1/users/me \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

---

## User Management

### Get Current User

```bash
curl https://api.archplatform.com/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN"
```

### Update Profile

```bash
curl -X PUT https://api.archplatform.com/api/v1/users/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "firstName": "Johnny",
    "lastName": "Doe",
    "avatarUrl": "https://example.com/avatar.jpg"
  }'
```

### Change Password

```bash
curl -X POST https://api.archplatform.com/api/v1/users/550e8400-e29b-41d4-a716-446655440000/change-password \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "currentPassword": "SecurePass123",
    "newPassword": "NewSecurePass456"
  }'
```

### Setup MFA

```bash
# 1. Setup MFA
curl -X POST https://api.archplatform.com/api/v1/auth/mfa/setup \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qrCode": "data:image/png;base64,iVBORw0KGgo..."
}
```

```bash
# 2. Verify MFA code
curl -X POST https://api.archplatform.com/api/v1/auth/mfa/verify \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "123456"
  }'
```

---

## Project Management

### Create Project

```bash
curl -X POST https://api.archplatform.com/api/v1/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Downtown Office Building",
    "description": "15-story commercial office building",
    "location": "123 Main St, New York, NY",
    "clientName": "ABC Corporation",
    "tags": ["commercial", "high-rise", "office"]
  }'
```

Response:
```json
{
  "id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "name": "Downtown Office Building",
  "description": "15-story commercial office building",
  "ownerId": "550e8400-e29b-41d4-a716-446655440000",
  "status": "ACTIVE",
  "createdAt": "2024-01-15T10:35:00Z",
  "memberCount": 1
}
```

### List Projects

```bash
# List all projects
curl "https://api.archplatform.com/api/v1/projects?page=0&size=20" \
  -H "Authorization: Bearer $TOKEN"

# Filter by status
curl "https://api.archplatform.com/api/v1/projects?status=ACTIVE" \
  -H "Authorization: Bearer $TOKEN"

# Search by name
curl "https://api.archplatform.com/api/v1/projects?q=office" \
  -H "Authorization: Bearer $TOKEN"
```

### Get Project Details

```bash
curl https://api.archplatform.com/api/v1/projects/6ba7b810-9dad-11d1-80b4-00c04fd430c8 \
  -H "Authorization: Bearer $TOKEN"
```

### Update Project

```bash
curl -X PUT https://api.archplatform.com/api/v1/projects/6ba7b810-9dad-11d1-80b4-00c04fd430c8 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Building Name",
    "description": "Updated description"
  }'
```

### Archive Project

```bash
curl -X POST https://api.archplatform.com/api/v1/projects/6ba7b810-9dad-11d1-80b4-00c04fd430c8/archive \
  -H "Authorization: Bearer $TOKEN"
```

### Delete Project

```bash
curl -X DELETE https://api.archplatform.com/api/v1/projects/6ba7b810-9dad-11d1-80b4-00c04fd430c8 \
  -H "Authorization: Bearer $TOKEN"
```

---

## Project Members

### Add Member

```bash
curl -X POST https://api.archplatform.com/api/v1/projects/6ba7b810-9dad-11d1-80b4-00c04fd430c8/members \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "6ba7b811-9dad-11d1-80b4-00c04fd430c8",
    "role": "EDITOR"
  }'
```

Roles: `OWNER`, `ADMIN`, `EDITOR`, `VIEWER`

### List Members

```bash
curl https://api.archplatform.com/api/v1/projects/6ba7b810-9dad-11d1-80b4-00c04fd430c8/members \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
[
  {
    "userId": "550e8400-e29b-41d4-a716-446655440000",
    "role": "OWNER",
    "joinedAt": "2024-01-15T10:35:00Z"
  },
  {
    "userId": "6ba7b811-9dad-11d1-80b4-00c04fd430c8",
    "role": "EDITOR",
    "joinedAt": "2024-01-15T11:00:00Z"
  }
]
```

### Update Member Role

```bash
curl -X PUT "https://api.archplatform.com/api/v1/projects/6ba7b810-9dad-11d1-80b4-00c04fd430c8/members/6ba7b811-9dad-11d1-80b4-00c04fd430c8?role=ADMIN" \
  -H "Authorization: Bearer $TOKEN"
```

### Remove Member

```bash
curl -X DELETE https://api.archplatform.com/api/v1/projects/6ba7b810-9dad-11d1-80b4-00c04fd430c8/members/6ba7b811-9dad-11d1-80b4-00c04fd430c8 \
  -H "Authorization: Bearer $TOKEN"
```

---

## File Operations

### Get Upload URL

```bash
curl -X POST https://api.archplatform.com/api/v1/files/upload-url \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "filename": "floor-plan.dwg",
    "contentType": "application/acad",
    "projectId": "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
  }'
```

Response:
```json
{
  "uploadUrl": "https://storage.archplatform.com/upload/...",
  "fileId": "6ba7b812-9dad-11d1-80b4-00c04fd430c8",
  "expiresAt": "2024-01-15T11:05:00Z"
}
```

### Upload File

```bash
# Using the presigned URL
curl -X PUT "https://storage.archplatform.com/upload/..." \
  -H "Content-Type: application/acad" \
  --data-binary @floor-plan.dwg
```

### List Files

```bash
curl "https://api.archplatform.com/api/v1/files?projectId=6ba7b810-9dad-11d1-80b4-00c04fd430c8" \
  -H "Authorization: Bearer $TOKEN"
```

### Download File

```bash
# Get download URL
curl https://api.archplatform.com/api/v1/files/6ba7b812-9dad-11d1-80b4-00c04fd430c8/download \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "url": "https://storage.archplatform.com/download/...",
  "expiresAt": "2024-01-15T11:15:00Z"
}
```

```bash
# Download the file
curl -O "https://storage.archplatform.com/download/..."
```

---

## Collaboration

### Connect via WebSocket

```javascript
const ws = new WebSocket('wss://api.archplatform.com/ws/collaboration', [], {
  headers: {
    'Authorization': 'Bearer ' + token
  }
});

ws.onopen = () => {
  // Join document
  ws.send(JSON.stringify({
    type: 'JOIN',
    projectId: '6ba7b810-9dad-11d1-80b4-00c04fd430c8',
    documentId: 'floor-plan-v1'
  }));
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};
```

### Send Update

```javascript
ws.send(JSON.stringify({
  type: 'UPDATE',
  projectId: '6ba7b810-9dad-11d1-80b4-00c04fd430c8',
  documentId: 'floor-plan-v1',
  operation: {
    type: 'ADD_ELEMENT',
    data: {
      elementType: 'WALL',
      position: { x: 100, y: 200 },
      dimensions: { width: 500, height: 10 }
    }
  },
  clientId: 'client-123',
  timestamp: Date.now()
}));
```

### Lock Element

```javascript
ws.send(JSON.stringify({
  type: 'LOCK',
  projectId: '6ba7b810-9dad-11d1-80b4-00c04fd430c8',
  documentId: 'floor-plan-v1',
  elementId: 'wall-001',
  userId: '550e8400-e29b-41d4-a716-446655440000'
}));
```

---

## Version Control

### Create Branch

```bash
curl -X POST https://api.archplatform.com/api/v1/versions/branches \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "feature/new-wing",
    "projectId": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
    "sourceVersionId": "6ba7b813-9dad-11d1-80b4-00c04fd430c8"
  }'
```

### List Versions

```bash
curl "https://api.archplatform.com/api/v1/versions?projectId=6ba7b810-9dad-11d1-80b4-00c04fd430c8" \
  -H "Authorization: Bearer $TOKEN"
```

### Create Merge Request

```bash
curl -X POST https://api.archplatform.com/api/v1/versions/merges \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sourceBranchId": "6ba7b814-9dad-11d1-80b4-00c04fd430c8",
    "targetBranchId": "6ba7b815-9dad-11d1-80b4-00c04fd430c8",
    "title": "Add new wing design",
    "description": "This merge adds the east wing design"
  }'
```

---

## Search

### Full-text Search

```bash
curl "https://api.archplatform.com/api/v1/search?q=office+building&page=0&size=20" \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "query": "office building",
  "total": 15,
  "hits": [
    {
      "id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "type": "project",
      "title": "Downtown Office Building",
      "highlight": "15-story commercial <em>office</em> <em>building</em>"
    }
  ]
}
```

### Filter by Type

```bash
curl "https://api.archplatform.com/api/v1/search?q=john&type=user" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Python Example

```python
import requests

class ArchPlatformClient:
    def __init__(self, base_url, token=None):
        self.base_url = base_url
        self.token = token
    
    def _headers(self):
        headers = {'Content-Type': 'application/json'}
        if self.token:
            headers['Authorization'] = f'Bearer {self.token}'
        return headers
    
    def login(self, username, password):
        response = requests.post(
            f'{self.base_url}/api/v1/auth/login',
            json={'username': username, 'password': password}
        )
        data = response.json()
        self.token = data['accessToken']
        return data
    
    def create_project(self, name, description=None):
        return requests.post(
            f'{self.base_url}/api/v1/projects',
            headers=self._headers(),
            json={'name': name, 'description': description}
        ).json()

# Usage
client = ArchPlatformClient('https://api.archplatform.com')
client.login('john.doe', 'SecurePass123')
project = client.create_project('New Project', 'Description')
print(project['id'])
```
