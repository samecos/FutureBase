# ArchPlatform TypeScript Client SDK

Official TypeScript/JavaScript client for ArchPlatform API.

## Installation

```bash
npm install @archplatform/client
# or
yarn add @archplatform/client
```

## Quick Start

```typescript
import { ArchPlatformClient } from '@archplatform/client';

const client = new ArchPlatformClient({
  baseURL: 'https://api.archplatform.com',
  apiKey: 'your-api-key'
});

// Authentication
await client.auth.login({
  username: 'user@example.com',
  password: 'password'
});

// Create a project
const project = await client.projects.create({
  name: 'My New Project',
  description: 'A test project'
});

console.log(project.id);
```

## Configuration

```typescript
const client = new ArchPlatformClient({
  baseURL: process.env.API_URL || 'https://api.archplatform.com',
  apiKey: process.env.API_KEY,
  timeout: 30000,
  retries: 3,
  
  // Optional: custom HTTP client
  httpClient: axios.create({ ... }),
  
  // Optional: error handler
  onError: (error) => {
    console.error('API Error:', error);
  }
});
```

## Services

### Auth Service

```typescript
// Login
const loginResult = await client.auth.login({
  username: 'user@example.com',
  password: 'password'
});

// Refresh token
const newToken = await client.auth.refreshToken(refreshToken);

// Logout
await client.auth.logout();

// Setup MFA
const mfaSetup = await client.auth.setupMFA();
// Display QR code: mfaSetup.qrCode

// Verify MFA
await client.auth.verifyMFA({
  code: '123456'
});
```

### User Service

```typescript
// Get current user
const user = await client.users.getCurrentUser();

// Get user by ID
const user = await client.users.getById('user-id');

// Update user
await client.users.update('user-id', {
  firstName: 'John',
  lastName: 'Doe'
});

// Change password
await client.users.changePassword({
  currentPassword: 'old',
  newPassword: 'new'
});
```

### Project Service

```typescript
// Create project
const project = await client.projects.create({
  name: 'Project Name',
  description: 'Description',
  location: 'New York'
});

// List projects
const projects = await client.projects.list({
  page: 0,
  size: 20,
  status: 'ACTIVE'
});

// Get project
const project = await client.projects.get('project-id');

// Update project
await client.projects.update('project-id', {
  name: 'New Name'
});

// Archive project
await client.projects.archive('project-id');

// Delete project
await client.projects.delete('project-id');
```

### Project Members

```typescript
// Add member
await client.projectMembers.add('project-id', {
  userId: 'user-id',
  role: 'EDITOR'
});

// List members
const members = await client.projectMembers.list('project-id');

// Update role
await client.projectMembers.updateRole('project-id', 'user-id', 'ADMIN');

// Remove member
await client.projectMembers.remove('project-id', 'user-id');
```

### File Service

```typescript
// Upload file
const file = await client.files.upload({
  file: fileBlob,
  projectId: 'project-id',
  onProgress: (progress) => {
    console.log(`${progress.percentage}%`);
  }
});

// Get download URL
const url = await client.files.getDownloadUrl('file-id');

// Delete file
await client.files.delete('file-id');
```

### Collaboration (WebSocket)

```typescript
// Connect to collaboration session
const session = await client.collaboration.connect({
  projectId: 'project-id',
  documentId: 'document-id',
  
  onMessage: (message) => {
    console.log('Received:', message);
  },
  
  onStateChange: (state) => {
    console.log('State:', state);
  }
});

// Send update
session.sendUpdate({
  type: 'UPDATE',
  data: { ... }
});

// Disconnect
session.disconnect();
```

## Error Handling

```typescript
try {
  await client.projects.create({ name: '' });
} catch (error) {
  if (error instanceof ValidationError) {
    console.log('Validation failed:', error.details);
  } else if (error instanceof NotFoundError) {
    console.log('Resource not found');
  } else if (error instanceof RateLimitError) {
    console.log('Rate limit exceeded, retry after:', error.retryAfter);
  } else {
    console.log('Unknown error:', error);
  }
}
```

## React Hooks

```typescript
import { useUser, useProject, useCollaboration } from '@archplatform/client/react';

// User hook
function UserProfile() {
  const { user, loading, error } = useUser();
  
  if (loading) return <Spinner />;
  if (error) return <Error message={error.message} />;
  
  return <div>{user.firstName}</div>;
}

// Project hook
function ProjectList() {
  const { projects, createProject } = useProject({
    status: 'ACTIVE'
  });
  
  return (
    <ul>
      {projects.map(p => <li key={p.id}>{p.name}</li>)}
    </ul>
  );
}

// Collaboration hook
function DesignCanvas() {
  const { state, sendOperation, activeUsers } = useCollaboration({
    projectId: 'project-id',
    documentId: 'doc-id'
  });
  
  return (
    <div>
      <Canvas data={state} />
      <UserList users={activeUsers} />
    </div>
  );
}
```

## TypeScript Types

```typescript
import type { 
  User, 
  Project, 
  ProjectMember, 
  DesignFile,
  LoginRequest,
  CreateProjectRequest 
} from '@archplatform/client';

const user: User = await client.users.getCurrentUser();
```

## License

MIT
