# Travel Diary backend

The Travel Diary API is a Go modular monolith using Fiber, MongoDB, S3-compatible object storage, Auth0, and an OpenAI-compatible AI adapter.

## Local setup

Requirements:

- Go 1.23 or newer
- MongoDB
- an S3-compatible bucket
- `heif-convert` from libheif if HEIC/HEIF uploads are enabled

```bash
cp .env.example .env
go run .
```

The API listens on port `8001` by default.

## Verification

```bash
gofmt -w .
go vet ./...
go test ./...
```

Generation is asynchronous through `generation_jobs`. A worker atomically leases queued work, reconstructs a JourneySpec, persists evidence-only story and creative fallbacks, plans a native ExperienceSpec, and persists an immutable `experience_versions` snapshot before moving the job to `render_ready`. Older and analysis-empty jobs retain the compatible album adapter path.

Workers renew their lease every 30 seconds while a generation is active. Infrastructure failures retry from the last persisted stage with bounded exponential backoff; schema, reference, and creative validation failures stop immediately. `GENERATION_MAX_ATTEMPTS` defaults to three and is bounded between one and ten. Expired final-attempt leases are marked failed rather than remaining active forever. SIGINT/SIGTERM shutdown stops new claims, waits up to 30 seconds for active jobs, then cancels and returns unfinished leases to the queue without consuming an attempt.

Uploaded photos retain the existing `trip_images` contract while a companion
`media_assets` document stores versioned visual metadata: orientation, EXIF
capture/GPS values, exact and perceptual hashes, dominant colors, quality
signals, and derivative references. Metadata extraction is supplemental to
upload completion and image/trip deletion removes both records.

Set `CREATIVE_PIPELINE_ENABLED=true` to enable the structured, model-backed Story Director. Its output must match `internal/ai/schemas/story-spec-v1.schema.json` and pass server-side reference validation. Provider/model/request/token/latency metadata is stored with the immutable story artifact. A failed or malformed model response emits a structured fallback event and stores the deterministic baseline with an explicit fallback reason.

The same flag enables Creative Explorer after Story Director. It produces 3–7 materially different concepts using `creative-exploration-v1.schema.json`, receives recent fingerprints from the user's other trips, and rejects concepts that strongly repeat multiple previous experiences. Explorations and the fingerprints considered during generation are stored immutably.

Creative Director then selects and refines one concept into `ExperienceDNA` using `creative-direction-v1.schema.json`. It validates the selected concept and signature assets against persisted inputs, requires a concrete answer to the memorable-moment test, records provider provenance, and stores the chosen direction immutably. Its normalized fingerprint is written to the user's anti-template history only after selection. When the creative pipeline flag is disabled or model output is invalid, the same stage persists an explicit deterministic fallback and provenance.

Experience Planner consumes the approved direction and StorySpec and emits a native ExperienceSpec using only registered runtime primitives. Server validation checks assets, story-chapter coverage, primitive keys, mobile choreography, reduced-motion fallbacks, and signature-moment placement before atomically activating an immutable version. New jobs with creative artifacts no longer call the legacy combined album prompt. The legacy path remains available for existing and analysis-empty jobs during migration.

`GET /api/v1/trips/:id/experience` returns the active owned version. During migration it lazily materializes a legacy album plan when required, so existing user data remains readable.

Version history is available from `GET /api/v1/trips/:id/experience/versions`; `PUT /api/v1/trips/:id/experience/versions/:version/activate` atomically restores an owned prior snapshot.

Presigned photo uploads use expiring single-use reservations. Upload completion must prove the key was issued to the authenticated user for the same trip before the API reads, records, or deletes the S3 object.

Architecture documentation lives in the frontend repository under `docs/architecture` because it describes both deployable repositories.
