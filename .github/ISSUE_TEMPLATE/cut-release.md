---
name: Cut a new release
about: Create a tracking issue for cutting a new provider release
title: Cut vX.Y.Z release
---

Scheduled for: <!-- YYYY-MM-DD -->

## Pre-release Verification

- [ ] All PRs targeted for this release are merged
- [ ] All tests are passing
  <!-- Run regular tests: go test -v ./...
       For integration tests: TEST_INTEGRATION=1 go test -v ./... -->
- [ ] Lint checks pass
  <!-- Run either: go run mage.go lint:go or simply: mage lint:go -->
- [ ] OpenAPI spec is up-to-date
  <!-- Run either: go run mage.go openapi:update or simply: mage openapi:update -->
- [ ] Verify no branch exists with the same name as the intended tag (e.g., `v1.2.3`)

## Documentation Review

- [ ] Documentation is up-to-date, including:
  - [ ] `docs/resources/`
  - [ ] `docs/data-sources/`
  - [ ] Inline HCL examples work
  - [ ] Standalone examples in `examples/` work
- [ ] CHANGELOG.md is updated

<!-- Common changelog entries to consider:
  - New features, bug fixes, breaking changes, deprecation/docs/dependencies
    updates, security updates
  
  Note: While GoReleaser can auto-generate changelogs, for better quality:
  1. Let GoReleaser generate an initial changelog from commits
  2. Manually curate it following Keep a Changelog format
     (https://keepachangelog.com)
  3. Either:
     - Use --release-notes=CHANGELOG.md to provide the curated changelog
     - Or if draft releases are enabled, review/edit before publishing -->

## Release Type Determination

Choose the appropriate version bump based on changes since last release:

- [ ] Major (X) - Breaking changes
- [ ] Minor (Y) - New features, non-breaking
- [ ] Patch (Z) - Bug fixes, documentation updates

## Release Process

- [ ] Create and push release tag
  <!-- Run:
       git tag vX.Y.Z
       git push origin vX.Y.Z -->
- [ ] Monitor GitHub Actions release workflow:
  - [ ] GoReleaser builds successfully
    <!-- Check the workflow run at:
         https://github.com/authzed/terraform-provider-authzed/actions/workflows/release.yml -->
  - [ ] GitHub release is published
    <!-- Add link to the release page:
         e.g. https://github.com/authzed/terraform-provider-authzed/releases/tag/v0.1.3 -->
  - [ ] Changelog is visible and formatted properly
  - [ ] Slack notifications are sent
    <!-- AuthZed employees only: Internal notification step -->

## Post-release QA Verification

- [ ] Provider shows up in registry
  <!-- Check the provider page at:
       https://registry.terraform.io/providers/authzed/authzed/latest -->
- [ ] Documentation renders correctly
- [ ] Provider can be installed (`terraform init`)

## Communication

- Post release announcements:
  - [ ] GitHub Release Notes are clear and complete
  - [ ] Internal team notification
  - [ ] External announcement (if applicable)

## Important Notes

⚠️ Never delete a pushed tag. If changes are needed after pushing a tag:

1. Fix the issue in a new commit
2. Create a new tag with an incremented patch version

This applies to both post-release changes and GitHub Actions workflow failures.

## Action Items

<!--
During the release, you may find a few things that require updates
(process changes, documentation updates, fixes to release tooling).

Please list them here.

It will be your responsibility to open issues/PRs to resolve these
issues/improvements. Keep this issue open until these action items
are filed.

- [ ] Item 1
- [ ] Item 2
- [ ] Item 3
-->

### Additional Context

<!-- Add any release-specific notes, special instructions, or context below -->