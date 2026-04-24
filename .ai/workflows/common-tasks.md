# Common task workflows

These sequences are the orchestration layer for this project. Follow them in order — skipping steps, particularly updating `api/openapi.yaml` before touching code, causes the spec and implementation to drift.

When a task spans both backend and frontend, always work **contract-first**: the OpenAPI spec is the source of truth and must be updated before any implementation begins.

---

## 1. Add a new API endpoint

**Order: spec → backend → tests → frontend client → UI**

1. **Update `api/openapi.yaml`** — define the path, method, request body schema, response schema(s), and all error cases. Do not proceed until this is done; the spec is the contract.
2. **Implement the Echo handler** — validate all input at the boundary, call the service layer, return structured JSON matching the spec exactly.
3. **Add OTel instrumentation** — add a span to the handler and any service functions it calls; attach relevant attributes (user ID, resource ID, etc.).
4. **Write tests** — unit tests for service logic, integration tests for the handler against a real or in-memory data store.
5. **Update `frontend/src/api/`** — add the typed client function(s) for the new endpoint. This is the only place in the frontend that may know the backend URL.
6. **Implement the Vue component or composable** — consume the new client function; no raw `fetch` calls.

---

## 2. Change an existing API contract

**Determine first: breaking or non-breaking?**

A change is **breaking** if it removes fields, renames fields, changes field types, or makes previously optional fields required. Breaking changes require a new version; non-breaking changes (adding optional fields, adding new endpoints) update in place.

**Non-breaking change:**
1. Update `api/openapi.yaml`.
2. Update the backend handler and service layer.
3. Update `frontend/src/api/` to match the new shape.
4. Update any frontend components that depend on the changed response.

**Breaking change:**
1. Create a new versioned path (`/api/v2/...`) in `api/openapi.yaml`.
2. Implement the v2 handler in the backend alongside the existing v1 handler.
3. Mark the v1 path as `deprecated: true` in the spec.
4. Update `frontend/src/api/` to call v2.
5. Update frontend components to use the new response shape.
6. Remove v1 only after confirming no active clients depend on it.

---

## 3. Add a frontend-only feature

**Order: confirm no new API needed → use existing client → implement UI**

1. **Confirm scope** — verify the feature needs no new backend data. If it does, run workflow 1 first, then return here.
2. **Check `frontend/src/api/`** — use the existing typed client methods. Do not add raw `fetch` calls anywhere outside this layer.
3. **Implement in Vue** — component(s) using `<script setup lang="ts">`, composable(s) for reusable logic, a new route entry if needed.
4. **Verify** — run the Vite dev server against the real backend (or an MSW mock if the backend isn't ready) and walk through the feature manually.

---

## 4. Add or change a data model

**Models live entirely in the backend; the frontend only ever sees the JSON shape the API exposes.**

1. **Define the Go struct** in the appropriate backend package. Design the zero value to be useful.
2. **Update `api/openapi.yaml`** — add or update the schema component that represents this model in API responses.
3. **Update any handlers** that produce or consume the model to match the new struct.
4. **Follow workflow 1 or 2** for any new or changed endpoints that expose the model.
