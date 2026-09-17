# Release artifacts

What a release candidate is, how it is produced, and what it is not.

## The property

**The artifact that was tested is the artifact that was scanned and the
artifact that is described.**

Without it, "we scanned the image" means "we scanned an image built from the
same source, probably". The two differ whenever a base image moved or a
dependency resolved differently between one job and the next, and a pipeline
that rebuilds at each stage cannot tell — so the evidence describes something
that was never run.

`.github/workflows/release.yml` builds each product image **once**, records its
identity, and refers to that identity from every later stage. The last step
re-reads them and fails if any moved.

## What is a release artifact

| | Release artifact | Why |
| --- | --- | --- |
| Integration Core image | **yes** | deployed |
| HUB image | **yes** | deployed |
| Mock upstreams | no | test fixtures; they exist so the platform can be exercised without credentials |
| E2E and load harnesses | no | they run in CI and are never deployed |

The E2E stack runs the **built** Core image rather than rebuilding, through
`docker-compose.e2e.images.yml`. The mocks are deliberately not overridden:
the manifest describes what is deployed.

## Identity

A local image has no registry digest until it is pushed, so the identity is the
**image ID** — the sha256 of its configuration, which covers its layers. It is
immutable for a given build and changes if anything about the image changes,
which is what has to be detected.

Images are tagged by the commit they were built from. A commit SHA is the only
name that cannot be reused for something else later; a tag like `latest` or
even `v1.2.3` can be moved.

## What the run produces

Uploaded as the `release-candidate` artifact, retained 90 days:

| File | What it is |
| --- | --- |
| `built.json` | every image, its tag and its identity |
| `release-manifest.json` | the four repository commits, schema level, OpenAPI hash, design system version, **and the image identities** |
| `provenance.json` | which workflow run, from which commits, produced which images |
| `sbom-*.cyclonedx.json` | what is inside each image, generated **from that image** |

## What it does not do

- **Nothing is pushed.** Publishing needs a registry credential, which is an
  owner decision. The images exist on the runner and are described; promotion
  is performed by a person — see below.
- **Provenance is unsigned.** A signature needs a key the owner has not
  configured. `provenance.json` records origin, not authenticity, and says so
  in the document itself rather than only here.
- **The HUB image is not deployable as built.** The HUB bakes its OIDC issuer
  and client id in at build time, so a release image is specific to the
  authentik it was built for. This pipeline proves the image builds, scans
  clean and is described; a deployable HUB image is built by the owner with
  their own issuer.

## Promotion

Performed by a person, deliberately. Nothing here deploys.

1. Take the `release-candidate` artifact from a **green** run on `main`.
2. Read `release-manifest.json`: it names the exact four commits, the schema
   level the release would apply, and the OpenAPI hash a client would have
   generated against.
3. Rebuild the HUB image with the deployment's own `VITE_OIDC_AUTHORITY` and
   `VITE_OIDC_CLIENT_ID`. Its identity will differ from the one in the
   manifest, and that is expected — record the new one.
4. Push both images to the registry, tagged by commit SHA. **Never** retag an
   existing tag onto a new image: the tag is how the manifest is joined to the
   thing running.
5. Deploy by digest where the registry supports it, or by the commit SHA tag.
6. Keep the manifest and the SBOMs with the deployment record. The next
   advisory is answerable from the SBOM without a re-scan: "were we shipping
   this package, and in which build?"

Steps 4 to 6 require a registry credential and a deployment target, both of
which are owner-controlled. They are **BLOCKED** for autonomous work and
listed in `docs/HANDOFF.md`.

## The check that makes it real

A check that only ever sees matching identities has never been shown to detect
a mismatch. `scripts/tests/stage-scripts.test.sh` exercises
`scripts/image-digests.py` from both directions: an unchanged pair verifies, an
image rebuilt between scanning and describing is caught by name, an image that
vanished is caught, and a manifest recording no image at all is refused rather
than trivially verified.
