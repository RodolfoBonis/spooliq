# Upload API Usage Guide

This guide explains how to use the file upload functionality in the Spooliq application.

## Overview

The Spooliq API integrates with rb-cdn (CDN service) to handle file uploads. This allows you to upload company logos, 3D model files, PDFs, and other assets.

## Available Endpoints

### 1. Upload Company Logo

**Endpoint**: `POST /v1/uploads/logo`

**Authentication**: Required (Bearer token)

**Description**: Upload a logo image for your company.

**Supported Formats**:
- JPG/JPEG
- PNG
- WEBP
- SVG

**Size Limit**: 5MB

**Request**:
```bash
curl -X POST "http://localhost:8000/v1/uploads/logo" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -F "file=@/path/to/logo.png"
```

**Response**:
```json
{
  "url": "https://rb-cdn.rodolfodebonis.com.br/v1/cdn/spooliq/logos/uuid-here.png",
  "message": "Logo uploaded successfully"
}
```

### 2. Upload Generic File

**Endpoint**: `POST /v1/uploads/file`

**Authentication**: Required (Bearer token)

**Description**: Upload any supported file type.

**Supported Formats**:
- JPG/JPEG
- PNG
- WEBP
- SVG
- PDF
- 3MF (3D model)
- STL (3D model)
- GCODE (3D printer instructions)

**Size Limit**: 50MB

**Request**:
```bash
curl -X POST "http://localhost:8000/v1/uploads/file" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -F "file=@/path/to/model.3mf" \
  -F "folder=models"
```

**Optional Parameters**:
- `folder`: Custom folder name (default: "files")

**Response**:
```json
{
  "url": "https://rb-cdn.rodolfodebonis.com.br/v1/cdn/spooliq/models/uuid-here.3mf",
  "message": "File uploaded successfully"
}
```

## Integration Examples

### JavaScript/TypeScript (Axios)

```typescript
import axios from 'axios';

// Upload logo
async function uploadLogo(file: File, token: string) {
  const formData = new FormData();
  formData.append('file', file);

  try {
    const response = await axios.post(
      'http://localhost:8000/v1/uploads/logo',
      formData,
      {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'multipart/form-data',
        },
      }
    );
    
    console.log('Logo URL:', response.data.url);
    return response.data.url;
  } catch (error) {
    console.error('Upload failed:', error);
    throw error;
  }
}

// Upload 3D model
async function upload3DModel(file: File, token: string) {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('folder', '3d-models');

  try {
    const response = await axios.post(
      'http://localhost:8000/v1/uploads/file',
      formData,
      {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'multipart/form-data',
        },
      }
    );
    
    console.log('Model URL:', response.data.url);
    return response.data.url;
  } catch (error) {
    console.error('Upload failed:', error);
    throw error;
  }
}
```

### React Component Example

```jsx
import React, { useState } from 'react';

function LogoUploader({ token }) {
  const [uploading, setUploading] = useState(false);
  const [logoUrl, setLogoUrl] = useState('');

  const handleFileChange = async (event) => {
    const file = event.target.files[0];
    if (!file) return;

    setUploading(true);

    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await fetch('http://localhost:8000/v1/uploads/logo', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
        },
        body: formData,
      });

      const data = await response.json();
      setLogoUrl(data.url);
      console.log('Logo uploaded successfully!');
    } catch (error) {
      console.error('Upload error:', error);
    } finally {
      setUploading(false);
    }
  };

  return (
    <div>
      <input
        type="file"
        accept=".jpg,.jpeg,.png,.webp,.svg"
        onChange={handleFileChange}
        disabled={uploading}
      />
      {uploading && <p>Uploading...</p>}
      {logoUrl && <img src={logoUrl} alt="Uploaded logo" />}
    </div>
  );
}
```

## Workflow Integration

### Company Settings Flow

1. User uploads a logo via `/v1/uploads/logo`
2. Get the returned CDN URL
3. Update company settings with the logo URL via `/v1/company` (PUT)

```javascript
async function updateCompanyWithLogo(file, token) {
  // Step 1: Upload logo
  const formData = new FormData();
  formData.append('file', file);
  
  const uploadResponse = await fetch('/v1/uploads/logo', {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}` },
    body: formData,
  });
  
  const { url: logoUrl } = await uploadResponse.json();
  
  // Step 2: Update company with logo URL
  const companyResponse = await fetch('/v1/company', {
    method: 'PUT',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      logo_url: logoUrl,
    }),
  });
  
  return await companyResponse.json();
}
```

## Error Handling

Common error responses:

### 400 Bad Request
```json
{
  "error": "File is required"
}
```
or
```json
{
  "error": "Invalid file type. Allowed: jpg, jpeg, png, webp, svg"
}
```
or
```json
{
  "error": "File size exceeds 5MB limit"
}
```

### 401 Unauthorized
```json
{
  "error": "Unauthorized"
}
```

### 500 Internal Server Error
```json
{
  "error": "Failed to upload file"
}
```

## Best Practices

1. **Validate files client-side** before uploading to provide faster feedback
2. **Show upload progress** to improve UX
3. **Handle errors gracefully** with user-friendly messages
4. **Store URLs immediately** after successful upload
5. **Use appropriate folders** to organize files (e.g., "logos", "models", "documents")
6. **Optimize images** before uploading (compress, resize if possible)
7. **Check file sizes** before uploading to avoid hitting limits

## Security Considerations

- All upload endpoints require authentication
- File types are validated server-side
- File sizes are enforced server-side
- Uploaded files are stored with random UUIDs to prevent conflicts
- CDN URLs are publicly accessible once uploaded

## CDN Configuration

The upload service uses the following environment variables:

```bash
CDN_BASE_URL=https://rb-cdn.rodolfodebonis.com.br
CDN_API_KEY=your-api-key-here
CDN_BUCKET=spooliq
```

Make sure these are properly configured in your environment.

## Troubleshooting

### Upload fails with network error
- Check if CDN service is accessible
- Verify API key is correct
- Check network connectivity

### File not appearing after upload
- Verify the returned URL is accessible
- Check CDN bucket permissions
- Ensure file was actually uploaded (check CDN dashboard)

### "File too large" error
- For logos: reduce file size to under 5MB
- For other files: reduce to under 50MB
- Consider compressing images or optimizing 3D models

## URL Structure

Uploaded files follow this URL structure:

```
https://rb-cdn.rodolfodebonis.com.br/v1/cdn/{bucket}/{folder}/{uuid}{extension}
```

Example:
```
https://rb-cdn.rodolfodebonis.com.br/v1/cdn/spooliq/logos/a1b2c3d4-e5f6-7890-abcd-ef1234567890.png
```

