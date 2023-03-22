# Changelog

## [v0.58.0](https://github.com/mergermarket/cdflow2/releases/tag/v0.58.0) - 2023-03-22

### Fixed

- Fix volume in use error during release command ([#43](https://github.com/mergermarket/cdflow2/pull/43))

## [v0.57.0](https://github.com/mergermarket/cdflow2/releases/tag/v0.57.0) - 2023-02-17

### Fixed

- Fix terraform providers upload during release command ([#33](https://github.com/mergermarket/cdflow2/pull/33))

  > WARNING!  
  > Using terraform 0.13.x image version is not recommended anymore for cdflow2.  
  > Please upgrade to 0.14 or newer otherwise you will see increased build times when running 'release' command.

## [v0.56.0](https://github.com/mergermarket/cdflow2/releases/tag/v0.56.0) - 2023-01-25

### Fixed

- Create infra directory if not exists before using terraform container ([#34](https://github.com/mergermarket/cdflow2/pull/34))
- Fix docker registry parsing ([#35](https://github.com/mergermarket/cdflow2/pull/35))
- Remove container volumes automatically ([#38](https://github.com/mergermarket/cdflow2/pull/38))

### Deprecated

- Deprecate quiet global argument ([#37](https://github.com/mergermarket/cdflow2/pull/37))  
  It's not used anywhere in the code, so currently doesn't do anything, please remove from commands.

## [v0.55.0](https://github.com/mergermarket/cdflow2/releases/tag/v0.55.0) - 2023-01-11

### Fixed

- Fix unknow option handling in commands ([#32](https://github.com/mergermarket/cdflow2/pull/32))

## [v0.54.0](https://github.com/mergermarket/cdflow2/releases/tag/v0.54.0) - 2023-01-04

### Fixed

- Unify missing argument handling ([#31](https://github.com/mergermarket/cdflow2/pull/31))

## [v0.53.1](https://github.com/mergermarket/cdflow2/releases/tag/v0.53.1) - 2022-12-15

### Changed

- Disable CGO for release binary ([#30](https://github.com/mergermarket/cdflow2/pull/30))

## [v0.53.0](https://github.com/mergermarket/cdflow2/releases/tag/v0.53.0) - 2022-12-14

### Added

- Add Datadog monitoring ([#28](https://github.com/mergermarket/cdflow2/pull/28))

### Fixed

- Fix panic when release env map nil ([#29](https://github.com/mergermarket/cdflow2/pull/29))

## [v0.52.1](https://github.com/mergermarket/cdflow2/releases/tag/v0.52.1) - 2022-11-25

### Fixed

- Fix global state loading for init command ([#27](https://github.com/mergermarket/cdflow2/pull/27))

## [v0.52.0](https://github.com/mergermarket/cdflow2/releases/tag/v0.52.0) - 2022-11-23

### Added

- Create init command ([#26](https://github.com/mergermarket/cdflow2/pull/26))  
  Docs: https://developer-preview.acuris.com/opensource/cdflow2/commands/init

## [v0.51.0](https://github.com/mergermarket/cdflow2/releases/tag/v0.51.0) - 2022-11-09

### Changed

- Set container terminal width/height based on host settings for shell command interactive mode ([#25](https://github.com/mergermarket/cdflow2/pull/25))
