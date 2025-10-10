# Keycloak Multi-Tenant Setup Guide

This guide explains how to configure Keycloak to support multi-tenancy using organization IDs in JWT tokens.

## Overview

The Spooliq application supports multi-tenancy by associating users with organizations. Each user can belong to one organization, and this organization ID is passed through JWT tokens from Keycloak.

## Prerequisites

- Keycloak instance running
- Admin access to Keycloak
- Spooliq realm created
- Spooliq client configured

## Configuration Steps

### Step 1: Create User Attribute

1. Log in to Keycloak Admin Console
2. Select your realm (e.g., `spooliq`)
3. Go to **Realm Settings** → **User Profile**
4. Click **Create attribute**
5. Fill in the details:
   - **Attribute name**: `organization_id`
   - **Display name**: `Organization ID`
   - **Required**: No (optional, but recommended for production)
   - **Permissions**: Admin can view and edit
6. Click **Save**

### Step 2: Create Client Scope

1. In your realm, go to **Client Scopes**
2. Click **Create client scope**
3. Fill in the details:
   - **Name**: `organization`
   - **Type**: Default
   - **Display on consent screen**: OFF
   - **Protocol**: openid-connect
4. Click **Save**

### Step 3: Add Protocol Mapper to Client Scope

1. In the **organization** client scope, go to the **Mappers** tab
2. Click **Add mapper** → **By configuration**
3. Select **User Attribute**
4. Fill in the mapper details:
   - **Name**: `organization-id-mapper`
   - **User Attribute**: `organization_id`
   - **Token Claim Name**: `organization_id`
   - **Claim JSON Type**: String
   - **Add to ID token**: ON
   - **Add to access token**: ON
   - **Add to userinfo**: ON
   - **Multivalued**: OFF
   - **Aggregate attribute values**: OFF
5. Click **Save**

### Step 4: Assign Client Scope to Spooliq Client

1. Go to **Clients** and select your **spooliq** client
2. Go to the **Client scopes** tab
3. Click **Add client scope**
4. Select **organization** from the list
5. Choose **Default** (not Optional)
6. Click **Add**

### Step 5: Set Organization ID for Users

For each user in your system:

1. Go to **Users** and select a user
2. Go to the **Attributes** tab
3. Click **Add attribute**
4. Add:
   - **Key**: `organization_id`
   - **Value**: A unique organization identifier (e.g., `org_12345`, `acme-corp`, etc.)
5. Click **Save**

**Recommendation**: Use UUIDs or slugs for organization IDs for better security and uniqueness.

## Testing the Configuration

### Step 1: Get a Token

Use the following command to get a token for a user:

```bash
curl -X POST "https://your-keycloak-host/realms/spooliq/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=password" \
  -d "client_id=spooliq" \
  -d "client_secret=YOUR_CLIENT_SECRET" \
  -d "username=YOUR_USERNAME" \
  -d "password=YOUR_PASSWORD"
```

### Step 2: Decode the Token

Copy the `access_token` from the response and decode it at [jwt.io](https://jwt.io).

Verify that the token contains the `organization_id` claim:

```json
{
  "sub": "user-uuid-here",
  "email": "user@example.com",
  "organization_id": "org_12345",
  ...
}
```

### Step 3: Test API Call

Make an API call to Spooliq with the token:

```bash
curl -X GET "http://localhost:8000/v1/company" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

The API should now recognize the user's organization and return company-specific data.

## Organization ID Strategies

### Option 1: Manual Assignment
- Admins manually assign organization IDs to users
- Good for small teams or controlled environments

### Option 2: Registration Flow
- Create a custom registration flow in Keycloak
- Auto-assign organization ID during user registration
- Good for self-service scenarios

### Option 3: External Integration
- Use Keycloak's User Federation or Event Listeners
- Sync organization IDs from external systems
- Good for enterprise integrations

## Security Considerations

1. **Validation**: The Spooliq API validates the organization_id claim on every request
2. **Authorization**: Users can only access data within their organization
3. **Admin Override**: Admin users can access all organizations (if configured)
4. **Token Security**: Organization IDs are tamper-proof as they're signed in the JWT

## Troubleshooting

### Token doesn't contain organization_id

- Check if the user has the `organization_id` attribute set
- Verify the client scope is assigned to the client as **Default** (not Optional)
- Clear browser cache and get a fresh token
- Check the mapper configuration

### API returns 401 Unauthorized

- Verify the token is valid and not expired
- Check if the organization_id claim is present in the token
- Ensure the Spooliq API is correctly reading the claim

### Users from different organizations see each other's data

- Check the repository implementations filter by organization_id
- Verify middleware is setting the organization_id in context
- Review authorization logic in use cases

## Migration from Single-Tenant

If migrating from a single-tenant setup:

1. Assign all existing users to a default organization (e.g., `default_org`)
2. Update database to add organization_id to existing records
3. Test thoroughly before rolling out to production
4. Consider a maintenance window for the migration

## Additional Resources

- [Keycloak Documentation](https://www.keycloak.org/documentation)
- [JWT Token Claims](https://www.rfc-editor.org/rfc/rfc7519)
- [OpenID Connect Core](https://openid.net/specs/openid-connect-core-1_0.html)

