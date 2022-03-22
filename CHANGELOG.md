# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
## [1.1.7] - 2022-03-22
- Fixed missed V1->V2 of SSO.
- Fixed lint issues.
- Ported over fixes done to eve-quartermaster.
## [1.1.6] - 2021-10-28
- Updated to SSO V2 and V5 User API.
## [1.1.5] - 2021-05-13
- Added Dockerfile and docker image.
- Added Docker variant of the setup.
- Fixed auth.bin initial empty state bug.
- Fixed bug with not passing auth.bin path in "run" command.
## [1.1.4] - 2021-05-11
- Added estimated price of fuel based on 7 day rolling average of "The Forge" region.
- Added nice ice cube icon for the "Total fuel" block
- Added green, orange, red icons per structure, based on fuel remaining. Green=OK, Orange < 7 days, Red < 1 day.
## [1.1.3] - 2021-04-02
- Add estimated fuel consumption per structure and total per day and month.
## [1.1.2] - 2021-03-23
- Fixed panics when unable to connect to ESI or Discord.
## [1.0.1] - 2019-07-20
### Added
- Added VERSION file and version command.
### Fixed
- `fuelbot login` problems with "auth.bin" missing.
- Fixed systemd service file to restart with delay.
### Changed
- Import paths everywhere to `github.com`

## [1.0.0] - 2019-07-11
### Initial release of EVE-FuelBot

[Unreleased]: https://github.com/lunemec/eve-fuelbot/compare/v1.0.1...HEAD
[1.0.1]: https://github.com/lunemec/eve-fuelbot/compare/v1.0.0...1.0.1
[1.0.0]: https://github.com/lunemec/eve-fuelbot/releases/tag/1.0.0

