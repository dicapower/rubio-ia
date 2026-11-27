# Security policy

## Reporting a vulnerability

Please do not open a public issue for a security problem. Use GitHub's private reporting: **Security → Report a vulnerability** on this repository. If that is not available to you, write to alexis.diaz@dicapower.com with "rubio-ia security" in the subject.

Include the Rubio-IA version (`rubio version`), what you did, and what you expected. Rubio-IA is maintained by one person: expect a first reply within about a week. There is no bug bounty.

## What counts

- Rubio-IA changing or deleting anything outside its own `<!-- rubio-ia:... -->` blocks or a skill that carries its marker.
- `rubio restore` not returning a file byte for byte, or a backup that is missing before a write.
- `rubio migrate` changing anything without `--yes`, or touching files that belong to another tool outside what the plan lists.
- The installer or a release asset that does not match `SHA256SUMS`.

## What does not count

- Anything that needs write access to your own agent configuration files or your own machine.

## Supported versions

Only the latest release gets fixes. Update with `install.sh`, `install.ps1`, Homebrew or Scoop; Rubio-IA never updates itself.

## Verifying a download

Each release has a `SHA256SUMS` file. Check the archive you downloaded against it before running it (`shasum -a 256 -c SHA256SUMS --ignore-missing`).
