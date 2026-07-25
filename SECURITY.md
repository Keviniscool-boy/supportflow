# Security Policy

## Project Status

SupportFlow is in the design and MVP development stage. No release is currently supported for production use, and the project does not claim any industry compliance certification.

| Version | Supported |
| --- | --- |
| `main` | Best effort |
| `< 1.0` | No production security support |

## Reporting a Vulnerability

Do not open a public issue for a suspected vulnerability or include credentials, customer data, access tokens, or exploit details in public discussions.

Use GitHub Private Vulnerability Reporting when it is enabled for the repository. If it is unavailable, contact the maintainers privately through the contact method listed on their GitHub profiles.

Please include:

- The affected component and revision.
- Reproduction steps or a minimal proof of concept.
- The expected impact and required preconditions.
- Any suggested mitigation, if known.

Maintainers will acknowledge reports and coordinate remediation on a best-effort basis. There is no guaranteed response SLA before a stable release.

## Security Scope

Security-sensitive areas include tenant isolation, customer identity, Tool permissions, prompt injection boundaries, secret handling, document upload validation, Agent Trace redaction, and authorization checks.

Demo data and Mock Connectors must never be treated as a substitute for production identity, access control, audit, backup, or compliance controls.
