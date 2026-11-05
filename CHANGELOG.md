# Changelog
All notable changes to this project will be documented in this file. See [conventional commits](https://www.conventionalcommits.org/) for commit guidelines.

- - -
## [v0.4.6](https://github.com/rtuszik/sex/compare/225f4f93aabbbc23a8e5d32166dc6f6ff13e5197..v0.4.6) - 2026-11-05
#### Bug Fixes
- (**connect**) match session name before icon strip - ([5be6467](https://github.com/rtuszik/sex/commit/5be64674dcc2cf9e98a4642e8c2009ab836208f6)) - Robin Tuszik
#### Documentation
- (**readme**) add installation steps for plugin loader - ([fc3bae3](https://github.com/rtuszik/sex/commit/fc3bae3871eaca9b6337974896d1cb2dfcf911bd)) - Federico, Copilot Autofix powered by AI
- (**readme**) add picker and activity screenshots - ([225f4f9](https://github.com/rtuszik/sex/commit/225f4f93aabbbc23a8e5d32166dc6f6ff13e5197)) - Robin Tuszik
- update plugin command description - ([ef1dfed](https://github.com/rtuszik/sex/commit/ef1dfedd00bd9fc3b5bbcac0b51c67c54e4a3161)) - Robin Tuszik
- simplify rex plugin install instructions - ([0f1169e](https://github.com/rtuszik/sex/commit/0f1169e25eb002bb675c80bfa47706862153f46f)) - Robin Tuszik
#### Continuous Integration
- add conventional commit check and dev task - ([d0787ea](https://github.com/rtuszik/sex/commit/d0787eaa326c29cf0dca73db399f4fa09b197e6d)) - Robin Tuszik
#### Refactoring
- (**plugin**) add lua/sex.lua loader - ([b3b20c0](https://github.com/rtuszik/sex/commit/b3b20c0476b07d8a4b3dd155a33c316dadcc651d)) - Robin Tuszik
#### Miscellaneous Chores
- (**deps**) update jdx/mise-action action to v5.2.5 (#10) - ([a070b94](https://github.com/rtuszik/sex/commit/a070b94551d9c154fa1900cc180eaeb239481bbf)) - koalabot-rt[bot], koalabot-rt[bot]
- (**deps**) update jdx/mise-action action to v5.2.4 - ([2cd474a](https://github.com/rtuszik/sex/commit/2cd474a707d7a86ad09274657c8fa16515205d67)) - koalabot-rt[bot]
- (**deps**) update goreleaser/goreleaser-action action to v7.3.0 - ([3131a0a](https://github.com/rtuszik/sex/commit/3131a0a67fcc6c8da8e9df08a21614d980a8bdc8)) - koalabot-rt[bot]
- (**deps**) add renovate.json - ([2a9931a](https://github.com/rtuszik/sex/commit/2a9931a3d50be40163e4d6beb0a7068699afe584)) - koalabot-rt[bot]
- make sex amber compatible - ([7130a29](https://github.com/rtuszik/sex/commit/7130a2962004e24ad021414f7b0c0b9160a34e3c)) - Federico

- - -

## [v0.4.5](https://github.com/rtuszik/sex/compare/7cf6e3187f3f7673693a4784fc9e54bf69c597e5..v0.4.5) - 2026-11-01
#### Bug Fixes
- (**rex**) decode empty lua tables as empty lists - ([7cf6e31](https://github.com/rtuszik/sex/commit/7cf6e3187f3f7673693a4784fc9e54bf69c597e5)) - Robin Tuszik

- - -

## [v0.4.4](https://github.com/rtuszik/sex/compare/5a405de3b92b0793a9283503f24bf7cab2c9d1a8..v0.4.4) - 2026-11-01
#### Bug Fixes
- (**plugin**) start the picker with sex found on seeion host - ([ffedc30](https://github.com/rtuszik/sex/commit/ffedc3052c39df90418134d1e3db79465f868b1d)) - Robin Tuszik
#### Documentation
- (**readme**) improve prose and add links - ([5a405de](https://github.com/rtuszik/sex/commit/5a405de3b92b0793a9283503f24bf7cab2c9d1a8)) - Robin Tuszik
#### Tests
- (**plugin**) tolerate missing fzf in script - ([96bb2be](https://github.com/rtuszik/sex/commit/96bb2bea0f59a7120a1949764caad9f9fbaedfda)) - Robin Tuszik

- - -

## [v0.4.3](https://github.com/rtuszik/sex/compare/23a24277f05ee2f0d9f70fd92ccfab897a2e56df..v0.4.3) - 2026-10-31
#### Bug Fixes
- (**pick**) restore source icons - ([1ee5e6e](https://github.com/rtuszik/sex/commit/1ee5e6ef525dc44d2503e6e4864f80784a97bb01)) - Robin Tuszik
- (**pick**) color running sessions green and current bold - ([23a2427](https://github.com/rtuszik/sex/commit/23a24277f05ee2f0d9f70fd92ccfab897a2e56df)) - Robin Tuszik

- - -

## [v0.4.2](https://github.com/rtuszik/sex/compare/aea7459c91b94310e613a847418e9d3340051215..v0.4.2) - 2026-10-31
#### Bug Fixes
- (**pick**) drop fzf frames inside the Rex pane and label picker panes - ([aea7459](https://github.com/rtuszik/sex/commit/aea7459c91b94310e613a847418e9d3340051215)) - Robin Tuszik

- - -

## [v0.4.1](https://github.com/rtuszik/sex/compare/819642da363ce5bc954501b915cd3d3298d86e04..v0.4.1) - 2026-10-31
#### Bug Fixes
- (**plugin**) search for sex when SEX_BIN is unset and report load failures - ([819642d](https://github.com/rtuszik/sex/commit/819642da363ce5bc954501b915cd3d3298d86e04)) - Robin Tuszik

- - -

## [v0.4.0](https://github.com/rtuszik/sex/compare/529b9c947494a07d354e7466ae9de33ca917f44c..v0.4.0) - 2026-10-31
#### Features
- (**plugin**) load plugin form installed binary and reload after upgrades - ([529b9c9](https://github.com/rtuszik/sex/commit/529b9c947494a07d354e7466ae9de33ca917f44c)) - Robin Tuszik

- - -

## [v0.3.0](https://github.com/rtuszik/sex/compare/e9ae59bf7a8f4444ccb31f126970c7de0228029a..v0.3.0) - 2026-10-31
#### Features
- add activity picker for terminals that report a program status - ([e9ae59b](https://github.com/rtuszik/sex/commit/e9ae59bf7a8f4444ccb31f126970c7de0228029a)) - Robin Tuszik

- - -

## [v0.2.0](https://github.com/rtuszik/sex/compare/5f26c3eb24a01151d089231a5121cfb24821030f..v0.2.0) - 2026-10-31
#### Features
- open fzf before seeions list is ready - ([2783181](https://github.com/rtuszik/sex/commit/2783181c82b6fc4ce03c7679cc96ad21390f1f09)) - Robin Tuszik
- switch app showing current terminal and avoid autostart on reading - ([92ed66d](https://github.com/rtuszik/sex/commit/92ed66d07c616e08b868e5fa4f3fd189b7b0086c)) - Robin Tuszik
#### Documentation
- (**readme**) rewrite and expand documentation - ([5f26c3e](https://github.com/rtuszik/sex/commit/5f26c3eb24a01151d089231a5121cfb24821030f)) - Robin Tuszik

- - -

## [v0.1.0](https://github.com/rtuszik/sex/compare/61813ee876f2a411d185ac237e295295fba8fa9f..v0.1.0) - 2026-10-27
#### Features
- (**cmd/sex**) resolve version from go build info - ([593b342](https://github.com/rtuszik/sex/commit/593b3425b1b0e0fc06a39f024dddee101afefcac)) - Robin Tuszik
#### Documentation
- add readme - ([2de926c](https://github.com/rtuszik/sex/commit/2de926c62cb61d14bd82723b91152082cce1cf17)) - Robin Tuszik
#### Miscellaneous Chores
- ![BREAKING](https://img.shields.io/badge/BREAKING-red) use cog for releases and remove clip fix - ([8ca6b08](https://github.com/rtuszik/sex/commit/8ca6b089d95f1ff651a6c269aeeb6d451155904e)) - Robin Tuszik

- - -

Changelog generated by [cocogitto](https://github.com/cocogitto/cocogitto).