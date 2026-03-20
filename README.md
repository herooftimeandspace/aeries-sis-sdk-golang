# Aeries SIS Go SDK

`aeries-sis-sdk-golang` is a Go SDK for the Aeries SIS API.

This repository is being built as a junior-friendly codebase. The goal is to make the Aeries API easier to understand, safer to call, and easier to test from Go.

## What this repo will contain

- A Go module named `github.com/herooftimeandspace/aeries-sis-sdk-golang`
- A public `aeries` package for application code
- A `contract` package that describes the documented API surface
- Contract snapshots and generated reference docs
- GitHub Pages documentation built with MkDocs Material

## Documentation

The published docs will live on GitHub Pages and start from this README.

- [Quickstart](docs/quickstart.md)
- [Configuration](docs/configuration.md)
- [API Contract](docs/contract.md)
- [Testing Guide](docs/testing.md)
- [Generated API Reference](docs/reference/index.md)

## Build expectations

The implementation plan requires:

- JSON-only behavior for v1
- typed Go request and response models
- a generated contract manifest derived from the Aeries support articles
- beginner-friendly comments on handwritten code
- 95% or better test coverage
- GitHub Pages publishing from `main`

## Aeries docs used as source material

- [Full Documentation](https://support.aeries.com/support/solutions/articles/14000077926-aeries-api-full-documentation)
- [Building a Request](https://support.aeries.com/support/solutions/articles/14000113681-aeries-api-building-a-request)
- [Quick Reference](https://support.aeries.com/support/solutions/articles/14000070569-api-quick-reference-guide)
- [School-related End Points](https://support.aeries.com/support/solutions/articles/14000113682-aeries-api-school-related-end-points)
- [Pre-Enroll Students](https://support.aeries.com/support/solutions/articles/14000120862-aeries-api-pre-enroll-students)
- [Student-related End Points](https://support.aeries.com/support/solutions/articles/14000113683-aeries-api-student-related-end-points)
- [Student Grades-related End Points](https://support.aeries.com/support/solutions/articles/14000113685-aeries-api-student-grades-related-end-points)
- [Attendance and Enrollment-related End Points](https://support.aeries.com/support/solutions/articles/14000113684-aeries-api-attendance-and-enrollment-related-end-points)
- [Staff-related End Points](https://support.aeries.com/support/solutions/articles/14000113687-aeries-api-staff-related-end-points)
- [Scheduling-related End Points](https://support.aeries.com/support/solutions/articles/14000113686-aeries-api-scheduling-related-end-points)
- [Gradebook-related End Points](https://support.aeries.com/support/solutions/articles/14000113688-gradebook-related-end-points)
- [Alerts](https://support.aeries.com/support/solutions/articles/14000129264-alerts)

## LLM Usage Disclaimer

This is an LLM-driven project. The docs and code are designed to be readable and maintainable by humans, but generated content may appear during development and publication.
