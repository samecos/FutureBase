# Error Codes Reference

Standardized error codes across all ArchPlatform services.

## Format

```json
{
  "code": "ERROR_CODE",
  "message": "Human readable message",
  "details": [
    {
      "field": "fieldName",
      "code": "FIELD_ERROR",
      "message": "Field specific message"
    }
  ],
  "requestId": "uuid",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## Common Errors (1000-1999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INTERNAL_ERROR` | 500 | Internal server error |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable |
| `TIMEOUT_ERROR` | 504 | Request timeout |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INVALID_REQUEST` | 400 | Malformed request |
| `VALIDATION_ERROR` | 400 | Validation failed |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Permission denied |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource conflict |
| `GONE` | 410 | Resource permanently removed |
| `PAYLOAD_TOO_LARGE` | 413 | Request body too large |
| `UNSUPPORTED_MEDIA_TYPE` | 415 | Invalid content type |

## User Service Errors (2000-2999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `USER_NOT_FOUND` | 404 | User does not exist |
| `USER_ALREADY_EXISTS` | 409 | Username or email already taken |
| `USER_DISABLED` | 403 | User account is disabled |
| `USER_LOCKED` | 403 | User account is locked |
| `INVALID_CREDENTIALS` | 401 | Username or password incorrect |
| `AUTHENTICATION_FAILED` | 401 | Authentication failed |
| `TOKEN_EXPIRED` | 401 | JWT token expired |
| `TOKEN_INVALID` | 401 | Invalid JWT token |
| `REFRESH_TOKEN_EXPIRED` | 401 | Refresh token expired |
| `MFA_REQUIRED` | 403 | Multi-factor authentication required |
| `MFA_INVALID_CODE` | 400 | Invalid MFA code |
| `MFA_SETUP_REQUIRED` | 400 | MFA setup not completed |
| `PASSWORD_TOO_WEAK` | 400 | Password does not meet requirements |
| `PASSWORD_REUSED` | 400 | Cannot reuse recent passwords |
| `EMAIL_NOT_VERIFIED` | 403 | Email verification required |
| `INVALID_VERIFICATION_TOKEN` | 400 | Invalid or expired verification token |
| `RESET_TOKEN_INVALID` | 400 | Invalid or expired reset token |

## Project Service Errors (3000-3999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `PROJECT_NOT_FOUND` | 404 | Project does not exist |
| `PROJECT_ALREADY_EXISTS` | 409 | Project with this name already exists |
| `PROJECT_ACCESS_DENIED` | 403 | No permission to access project |
| `PROJECT_ARCHIVED` | 403 | Project is archived |
| `PROJECT_DELETED` | 404 | Project has been deleted |
| `PROJECT_LIMIT_REACHED` | 403 | Maximum projects limit reached |
| `MEMBER_NOT_FOUND` | 404 | Project member not found |
| `MEMBER_ALREADY_EXISTS` | 409 | User is already a project member |
| `CANNOT_REMOVE_OWNER` | 400 | Cannot remove project owner |
| `CANNOT_CHANGE_OWNER_ROLE` | 400 | Cannot change owner's role |
| `INVALID_ROLE` | 400 | Invalid member role specified |
| `INVITATION_EXPIRED` | 400 | Project invitation has expired |
| `INVITATION_ALREADY_USED` | 409 | Invitation has already been accepted |
| `DESIGN_FILE_NOT_FOUND` | 404 | Design file not found |
| `DESIGN_FILE_LOCKED` | 423 | File is locked by another user |
| `LOCK_EXPIRED` | 400 | File lock has expired |
| `VERSION_NOT_FOUND` | 404 | File version not found |

## Property Service Errors (4000-4999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `PROPERTY_NOT_FOUND` | 404 | Property not found |
| `PROPERTY_TEMPLATE_NOT_FOUND` | 404 | Property template not found |
| `INVALID_PROPERTY_VALUE` | 400 | Invalid property value |
| `INVALID_EXPRESSION` | 400 | Invalid calculation expression |
| `CIRCULAR_DEPENDENCY` | 400 | Circular dependency detected |
| `UNIT_CONVERSION_ERROR` | 400 | Unit conversion failed |
| `PROPERTY_GROUP_NOT_FOUND` | 404 | Property group not found |
| `PROPERTY_RULE_VIOLATION` | 400 | Property value violates rule |

## Version Control Errors (5000-5999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `BRANCH_NOT_FOUND` | 404 | Branch not found |
| `BRANCH_ALREADY_EXISTS` | 409 | Branch name already exists |
| `VERSION_NOT_FOUND` | 404 | Version not found |
| `MERGE_CONFLICT` | 409 | Merge conflict detected |
| `INVALID_MERGE` | 400 | Cannot merge branches |
| `CANNOT_DELETE_DEFAULT_BRANCH` | 400 | Cannot delete default branch |
| `BRANCH_BEHIND` | 409 | Branch is behind target |
| `MERGE_REQUEST_NOT_FOUND` | 404 | Merge request not found |
| `MERGE_REQUEST_ALREADY_EXISTS` | 409 | Active merge request already exists |
| `CONFLICT_RESOLUTION_REQUIRED` | 400 | Conflicts must be resolved |

## File Service Errors (6000-6999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `FILE_NOT_FOUND` | 404 | File not found |
| `FILE_TOO_LARGE` | 413 | File exceeds size limit |
| `INVALID_FILE_TYPE` | 415 | File type not allowed |
| `UPLOAD_FAILED` | 500 | File upload failed |
| `DOWNLOAD_FAILED` | 500 | File download failed |
| `STORAGE_QUOTA_EXCEEDED` | 403 | Storage quota exceeded |
| `FILE_CORRUPTED` | 400 | File checksum mismatch |
| `MULTIPART_UPLOAD_INVALID` | 400 | Invalid multipart upload |
| `THUMBNAIL_GENERATION_FAILED` | 500 | Failed to generate thumbnail |

## Search Service Errors (7000-7999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `SEARCH_UNAVAILABLE` | 503 | Search service unavailable |
| `INVALID_SEARCH_QUERY` | 400 | Invalid search syntax |
| `INDEXING_ERROR` | 500 | Failed to index document |
| `SEARCH_TIMEOUT` | 504 | Search query timeout |

## Geometry Service Errors (8000-8999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_GEOMETRY` | 400 | Invalid geometry data |
| `GEOMETRY_TOO_COMPLEX` | 400 | Geometry exceeds complexity limit |
| `BOOLEAN_OPERATION_FAILED` | 400 | Boolean operation failed |
| `IMPORT_FAILED` | 400 | Failed to import geometry |
| `EXPORT_FAILED` | 500 | Failed to export geometry |
| `UNSUPPORTED_FORMAT` | 415 | Unsupported file format |
| `GEOMETRY_SIMPLIFICATION_FAILED` | 500 | Simplification failed |

## Script Service Errors (9000-9999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `SCRIPT_NOT_FOUND` | 404 | Script not found |
| `SCRIPT_EXECUTION_FAILED` | 500 | Script execution failed |
| `SCRIPT_TIMEOUT` | 504 | Script execution timeout |
| `SCRIPT_MEMORY_EXCEEDED` | 400 | Script exceeded memory limit |
| `SCRIPT_SYNTAX_ERROR` | 400 | Script has syntax errors |
| `PACKAGE_INSTALLATION_FAILED` | 500 | Failed to install package |
| `SANDBOX_ERROR` | 500 | Sandbox initialization failed |

## Collaboration Errors (10000-10999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `DOCUMENT_NOT_FOUND` | 404 | Collaboration document not found |
| `DOCUMENT_LOCKED` | 423 | Document is locked |
| `OPERATION_REJECTED` | 409 | Operation rejected due to conflict |
| `SESSION_INVALID` | 401 | Invalid or expired session |
| `CLIENT_OUT_OF_SYNC` | 409 | Client state out of sync |
| `PERMISSION_DENIED_FOR_OPERATION` | 403 | No permission for this operation |
| `HISTORY_NOT_AVAILABLE` | 404 | Operation history not available |

## Notification Errors (11000-11999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `NOTIFICATION_NOT_FOUND` | 404 | Notification not found |
| `INVALID_WEBHOOK_URL` | 400 | Invalid webhook URL |
| `WEBHOOK_DELIVERY_FAILED` | 500 | Failed to deliver webhook |
| `EMAIL_SEND_FAILED` | 500 | Failed to send email |
| `PUSH_NOTIFICATION_FAILED` | 500 | Failed to send push notification |

## Analytics Errors (12000-12999)

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `REPORT_NOT_FOUND` | 404 | Report not found |
| `INVALID_DATE_RANGE` | 400 | Invalid date range specified |
| `DATA_AGGREGATION_FAILED` | 500 | Failed to aggregate data |
| `EXPORT_GENERATION_FAILED` | 500 | Failed to generate export |
