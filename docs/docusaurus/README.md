# MediaHub OSS Documentation (Docusaurus)

This directory contains the [Docusaurus](https://docusaurus.io/) setup for building and rendering the static documentation site for MediaHub OSS.

---

## 📁 Directory Structure

- **`docs/manual/`**: Source documentation content written in Markdown (`.md` / `.mdx`).
- **`docs/docusaurus/`**: Docusaurus configuration, themes, sidebars, dependencies (`package.json`), and static assets (`static/`).
- **`docs/docusaurus/build/`**: The output directory generated during local builds containing the final static HTML, CSS, and JS files. *(This directory is git-ignored and clearly separated from source files).*

---

## 📋 Prerequisites

- **Node.js**: `v18.0.0` or higher
- **npm**: Installed with Node.js

---

## 🚀 How to Build and Run Locally

### 1. Navigate to the Docusaurus directory

From the root of the repository, change directory to `docs/docusaurus`:

```bash
cd docs/docusaurus
```

### 2. Install Dependencies

Install the required npm packages:

```bash
npm install
```

### 3. Local Development (Live Preview with Hot Reload)

To start a local development server with real-time preview and hot reloading as you edit markdown files in `docs/manual/`:

```bash
npm start
```

This will open your browser at **`http://localhost:3000/`**.

### 4. Build Static Site Locally

To compile and build the static website files:

```bash
npm run build
```

The compiled production build output will be stored in:
`docs/docusaurus/build/`

*Note: The `build/` directory is automatically ignored by Git and will not pollute your version control changes.*

### 5. Preview the Built Static Site Locally

To preview the built static site locally (testing the contents of `docs/docusaurus/build/`):

```bash
npm run serve
```

This starts a local web server serving the contents of `build/` at **`http://localhost:3000/`**.

---

## 🐳 Alternative: Build and Run with Docker

You can also build and run the documentation container using Docker without installing Node.js locally:

1. **Build Docker Image** (run from repository root):
   ```bash
   docker build -t mediahub-docs -f docs/docusaurus/Dockerfile .
   ```

2. **Run Docker Container**:
   ```bash
   docker run -d -p 8081:80 mediahub-docs
   ```

3. Open **`http://localhost:8081`** in your browser.

---

## 📌 Documentation Versioning

Versioning is configured and enabled for this project.

- **`v3.1` (Current/Latest Release)**: Snapshot stored in `docs/docusaurus/versioned_docs/version-3.1/` and served by default at `/`.
- **`Development (Next)`**: Unreleased working draft in `docs/manual/` served at `/next`.

### Creating a New Version (e.g. `3.2` or `4.0`)

1. Make sure your active manual files in `docs/manual/` are up to date.
2. From `docs/docusaurus`, run:
   ```bash
   npm run docusaurus docs:version 3.2
   ```
3. Update `docusaurus.config.js` to set `lastVersion: '3.2'` and add the new version label under `versions`.
4. Commit the generated version files (`versioned_docs/`, `versioned_sidebars/`, `versions.json`).

### Updating an Existing Version (e.g. Fixing typos in `v3.1`)

To edit or patch an existing released version (like `v3.1`):
Edit the markdown files directly inside:
```
docs/docusaurus/versioned_docs/version-3.1/
```
*(And if you add/remove pages, update `docs/docusaurus/versioned_sidebars/version-3.1-sidebars.json`)*.

### Overwriting an Existing Version Completely

> **Note:** Running `npm run docusaurus docs:version 3.1` when version `3.1` already exists will throw an error (`[ERROR] Error: [docs]: this version already exists!`).

If you want to **completely replace/overwrite** `v3.1` with the current contents of `docs/manual/`:

1. Delete the snapshot folder: `docs/docusaurus/versioned_docs/version-3.1/`
2. Delete the sidebar file: `docs/docusaurus/versioned_sidebars/version-3.1-sidebars.json`
3. Remove `"3.1"` from `docs/docusaurus/versions.json`
4. Re-run the versioning command:
   ```bash
   npm run docusaurus docs:version 3.1
   ```


