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
