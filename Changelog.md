# v1.2
Features:
- make the "large file" threshold configurable.
- implement database schema versions to enable migrations in the future. (TODO)
- implement a "recovery" command, for database consistency checks after, e.g., a server crash (TODO)
- implement bulk delete and bulk download functionalities (TODO)
Bug fixes:
- fix swagger documentation not being up to date
- fix filename and status fields not being whitelisted in the search endpoint

# v1.1
- JWT implemented in the backend and frontend. Basic auth is still available for endpoints, but is to be avoided if possible.
- Drag and drop functionality added to the frontend. This functionality works in the list, grid and uploda views, and checks the mime type before upload as well.
- Reworked entry detail view, including Delete and Edit button, organized properties
- implement content negotiation to enable base64 responses, e.g., for Grafana display of images
- better preview handling and clearer status return flag for entry upload