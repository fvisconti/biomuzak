---
description: 'React / Vite / Chakra UI rules for the biomuzak frontend'
applyTo: 'frontend/**/*.js,frontend/**/*.jsx,frontend/vite.config.js,frontend/package.json,frontend/nginx.conf'
---

# Frontend Instructions

Applies to all code in `frontend/`. The frontend is a React 18 SPA built with
Vite 5, styled with Chakra UI, and served by nginx in production.

## Stack

- **React 18** with functional components and hooks. No class components.
- **Vite 5** as the build tool and dev server. Config in `vite.config.js`.
- **Chakra UI 2** for styling and layout. Use the theme in `src/theme.js`; do
  not hardcode colors or spacing outside the theme.
- **react-router-dom 6** for routing. Routes are defined in `App.jsx`.
- **axios** for API calls. The API client is in `src/api/client.js`; do not
  create ad-hoc `fetch` calls outside that module.
- **@dnd-kit** for drag-and-drop (playlist reordering).
- **framer-motion** for animations.
- **react-dropzone** for file upload UI.

## Project structure

- `src/App.jsx` — root component, route definitions.
- `src/main.jsx` — entry point, ChakraProvider, BrowserRouter.
- `src/theme.js` — Chakra theme (colors, fonts, breakpoints).
- `src/api/` — API clients (`client.js` for the backend REST API, `tidal.js`
  for Tidal).
- `src/components/` — reusable UI components, organized by domain
  (`Auth/`, `Layout/`, `Music/`, `Upload/`).
- `src/context/` — React contexts (`AuthContext`, `PlayerContext`,
  `PlaylistContext`, `DragContext`).
- `src/pages/` — one file per route (Dashboard, Library, Albums, Playlists,
  Upload, Login, Register, AdminUsers, Tidal, etc.).
- `src/data/` — static/dummy data.

## Conventions

- **One component per file.** File name matches component name (PascalCase).
- **Pages are thin.** A page composes components and contexts; it does not
  contain business logic. Logic belongs in the context or a custom hook.
- **API calls go through `src/api/client.js`.** That module handles the base
  URL, auth token injection, and error normalization. Do not duplicate that
  logic in components.
- **Auth state** comes from `AuthContext`. Do not read `localStorage`
  directly in components.
- **Player state** (current track, queue, playback) comes from
  `PlayerContext`. Do not manage audio state in local component state.
- **No `any` / untyped props.** Use JSDoc `@typedef` or PropTypes for
  non-trivial component props.
- **ESLint** is configured (`eslint.config.js`). Run `npm run lint` before
  committing. Fix all errors; warnings should be justified.

## Vite specifics

- Dev server runs on port 3000 (`vite.config.js`).
- Build output is static files served by nginx (`frontend/nginx.conf`).
- Do not add a `base` path unless the deployment changes; the app is served
  from `/`.
- Environment variables are prefixed `VITE_` and accessed via
  `import.meta.env`. Do not hardcode API URLs.

## Security

- The frontend is a public asset. **No secrets, no API keys, no tokens in
  client-side code.** The JWT is stored in memory (via `AuthContext`) and
  sent as a `Bearer` header; it is not persisted to `localStorage`.
- All user input from the UI is sent to the backend for validation. The
  frontend does not sanitize SQL or build queries.
- CORS is handled server-side (`CORS_ALLOWED_ORIGINS`); the frontend does not
  set CORS headers.

## Testing

- Tests use the Jest + React Testing Library stack (configured via
  `package.json` scripts).
- Run with `npm test` (or `CI=true npm test -- --watchAll=false --runInBand`).
- Mock the API client (`src/api/client.js`) in component tests; do not hit a
  live backend.
- Test user-visible behaviour (what the user sees and can do), not
  implementation details (internal state, hook return values).

## Build & deploy

- `npm run build` produces `frontend/dist/`.
- The Dockerfile copies `dist/` into an nginx image.
- `nginx.conf` serves the SPA (all routes fall through to `index.html`) and
  proxies `/api/` and `/rest/` to the backend.
