# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Updated Go GUI to v0.86.0 and adopted its explicit `NoSpacing` and `NoRadius`
  layout values.

### Added

- Initial resource timeline widget for Go GUI.
- Day, week, month, year, and custom time ranges.
- Overlap packing, event hit testing, selection, and truncated-title tooltips.
- Fixed time and resource headers with standard horizontal and vertical
  scrollbars.
- Automatic scene caching with an optional application-managed content version.
- Configurable event colors, resource header, dimensions, and week start.
- A runnable example with simulated data loading, resource pagination, calendar
  navigation, zooming, and a large-data mode.
- Keyboard navigation with a localized, application-defined active-event
  accessibility label and canvas focus indication.
