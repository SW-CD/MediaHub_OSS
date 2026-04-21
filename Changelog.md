# v2.0
Features:
- add video file support
- add fullscreen playback to frontend
- user roles now on database level

Breaking Changes:
- all timestamps are now stored with millisecond precision
- entry and database endpoint paths changed to use path variables
- files are now stored in folders by `id` rather than by `date`
- database schema changed significantly
- running the server now requires the `serve` command
- configuration file has renamed configs
- renamed environment variable prefix from `FDB_` to `MEDIAHUB_`

Other:
- publish multiarch docker build (ARM64, X86)
- add Windows ARM64 binaries to release

# v1.2
Features:
- make the "large file" threshold configurable.
- implement database schema versions to enable migrations in the future.
- implement a "recovery" command, for database consistency checks after, e.g., a server crash
- implement auditor logging, which has to be enabled explicitly
- implement bulk delete and bulk download functionalities

Bug fixes:
- fix swagger documentation not being up to date
- fix filename and status fields not being whitelisted in the search endpoint

# v1.1
- JWT implemented in the backend and frontend. Basic auth is still available for endpoints, but is to be avoided if possible.
- Drag and drop functionality added to the frontend. This functionality works in the list, grid and uploda views, and checks the mime type before upload as well.
- Reworked entry detail view, including Delete and Edit button, organized properties
- implement content negotiation to enable base64 responses, e.g., for Grafana display of images
- better preview handling and clearer status return flag for entry upload
