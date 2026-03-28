# Changelog

All notable changes to this project will be documented in this file.

## [v1.1.0] - 2026-03-28

### Changes since v0.0.1

- 1e3c085 (HEAD -> main, origin/main) chore: update Go version to 1.25 in workflows and benchmark test
- 0eb2288 chore(go.mod): update Go version to 1.25.0
- a0dad3f feat(cookie): add SameSite attribute to SetCookieWithOptions for enhanced security
- 0f44e84 fix(log): remove unnecessary nolint directive for uintptr conversion
- 5a0e47d refactor(tests): align tag formatting in TestStruct definitions for consistency
- cb55475 refactor(binder): update tag precedence logic and add tests for default tag handling
- 4012bb6 fix(log): remove unnecessary nolint directive for uintptr conversion
- c7a089f refactor: simplify tag parsing logic in multiple files
- 44b2929 refactor(service): improve documentation and structure of service interfaces
- 6f02c47 fix(metrics): handle potential error in CalculateStats method
- 7caed63 feat(metrics): enhance health report with statistics tracking
- 83440c5 fix(metrics): update JSON tags for Created and Updated fields to omit zero values
- 4537ca1 feat(metrics): enhance histogram and timer functionalities with examples and tests
- 3ab50cb feat(metrics): add support for default tags in metrics collection
- 452cd3d refactor(metrics): update metric creation functions to use consistent naming
- 1606e44 feat(metrics): enhance logging for metrics operations
- 1c0f6a3 feat(metrics): add comprehensive metrics implementation documentation
- 84e3c96 feat(metrics): add utility functions and validation for metric labels
- 686c89d feat(metrics): implement collectors for automatic metric collection
- e4d6283 feat(metrics): add example tests and benchmarks for metrics functionality
- cc4640a feat(metrics): introduce summary metric type and exemplar support
- 855cadc feat(metrics): add default bucket configurations and metadata options for metrics
- ba6df1f feat(metrics): make MetricsStorageConfig and MetricsExporterConfig generic
- 8ddd2f0 refactor(tests): remove unnecessary blank line in external_test.go
- 0eb309c feat(metrics): add mock implementations for metrics and health management
- 3e1c43d feat(log): add utility function to create new log fields
- 57a49b3 feat(errs): add ErrorHandler interface for handling HTTP errors
- e738cfe fix: update go.mod and go.sum to remove indirect dependency on validator and add assert package
- da05db0 feat(docs): update README with Go version, enhance package descriptions, and add new sections for DI, HTTP handling, and validation utilities
- 16e8aa2 Add validation framework with benchmarks and tests
- 1ff5ddc feat(validation): add validation error handling with structured response
- ea32af7 feat(di): implement dependency injection container with lifecycle management

