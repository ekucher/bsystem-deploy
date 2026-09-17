#!/usr/bin/env python3
"""Record and verify the identity of the images a release is built from.

The point of this file is one property: **the artifact that was tested is the
artifact that was scanned and the artifact that is described**. Without it,
"we scanned the image" means "we scanned an image built from the same source,
probably", and the two differ whenever a base image moved, a dependency
resolved differently, or a later job rebuilt rather than reused.

A local image has no registry digest until it is pushed, so the identity used
here is the image ID — the sha256 of its configuration, which covers the
layers. It is immutable for a given build and changes if anything about the
image does, which is exactly what has to be detected.

    image-digests.py record built.json core=bsystem-integration-core:abc123
    image-digests.py verify built.json

`verify` re-reads each image and fails if any identity moved. `--from-file`
substitutes a recorded reading for the live one, so the failure path itself is
testable without building an image to corrupt.
"""

import json
import subprocess
import sys


def inspect(tag: str) -> str:
    """Return the image ID docker holds for a tag, or "" when there is none."""
    result = subprocess.run(
        ["docker", "image", "inspect", "--format", "{{.Id}}", tag],
        capture_output=True, text=True,
    )
    if result.returncode != 0:
        return ""
    return result.stdout.strip()


def record(destination: str, pairs: list[str]) -> int:
    images: dict[str, dict[str, str]] = {}
    findings: list[str] = []
    for pair in pairs:
        if "=" not in pair:
            findings.append(f"{pair!r} is not name=tag")
            continue
        name, tag = pair.split("=", 1)
        identity = inspect(tag)
        if not identity:
            findings.append(f"no image is tagged {tag!r}; nothing was built, or it was built elsewhere")
            continue
        images[name] = {"tag": tag, "id": identity}
    if findings:
        for finding in findings:
            print(f"  {finding}")
        return 1
    if not images:
        # The vacuity guard. A manifest recording no image would let every
        # later verification pass by having nothing to compare.
        print("  no images were recorded; a release manifest with no artifact describes nothing")
        return 1
    with open(destination, "w", encoding="utf-8") as handle:
        json.dump(images, handle, indent=2, sort_keys=True)
        handle.write("\n")
    for name, image in sorted(images.items()):
        print(f"  {name}: {image['tag']} is {image['id']}")
    return 0


def verify(source: str, from_file: str | None) -> int:
    with open(source, encoding="utf-8") as handle:
        recorded = json.load(handle)
    if not recorded:
        print("  the recorded manifest holds no image; there is nothing to verify")
        return 1

    current: dict[str, str] = {}
    if from_file:
        with open(from_file, encoding="utf-8") as handle:
            for name, image in json.load(handle).items():
                current[name] = image["id"]
    else:
        for name, image in recorded.items():
            current[name] = inspect(image["tag"])

    findings = []
    for name, image in sorted(recorded.items()):
        now = current.get(name, "")
        if not now:
            findings.append(
                f"{name} ({image['tag']}) no longer exists: it was tested and scanned, "
                f"and what a later step would describe is something else"
            )
        elif now != image["id"]:
            findings.append(
                f"{name} ({image['tag']}) changed identity: tested {image['id']}, "
                f"now {now}. A rebuild has silently substituted for the image the "
                f"evidence describes"
            )
    for finding in findings:
        print(f"  {finding}")
    if findings:
        return 1
    print(f"  {len(recorded)} image(s) unchanged since they were recorded")
    return 0


def main(argv: list[str]) -> int:
    if len(argv) < 3:
        print(__doc__)
        return 2
    command = argv[1]
    if command == "record":
        return record(argv[2], argv[3:])
    if command == "verify":
        from_file = None
        if "--from-file" in argv:
            from_file = argv[argv.index("--from-file") + 1]
        return verify(argv[2], from_file)
    print(f"unknown command {command!r}")
    return 2


if __name__ == "__main__":
    sys.exit(main(sys.argv))
