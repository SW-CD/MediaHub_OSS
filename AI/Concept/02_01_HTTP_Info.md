### Info endpoints

#### GET /api/info

Retrieves general information about the software, i.e., the service name, software version and uptime. This is the only endpoint that does not require user authentication.

##### Request

`GET /api/info`

##### Success Response

**Status 200 - OK**

Returns general backend information, such as the software version or ffmpeg/ffprobe availability.

```json
{
  "service_name": "SWCD MediaHub-API",
  "version": "1.0.0-beta.1",
  "uptime_since": "2025-10-27T18:30:05Z",
  "ffmpeg": "true",
  "ffprobe": "true",
}
```

##### Error Responses

This endpoint does not require authentication and is not expected to return client-side validation errors or error codes.
