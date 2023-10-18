# Changelog

## [1.0.0](https://github.com/jfortunato/wp-zip/compare/v0.2.0...v1.0.0) (2023-10-18)


### ⚠ BREAKING CHANGES

* Use site-url instead of domain
* Major refactor that splits responsibilities more logically

### Features

* Ability to export database without shell access ([ef738e5](https://github.com/jfortunato/wp-zip/commit/ef738e5e43189fc7b9be45ffbeeea277a8149c02))
* Auto detect site-url/domain at runtime ([c1f3f82](https://github.com/jfortunato/wp-zip/commit/c1f3f8291e84eedaf5096984ddcd01bb21626421))
* Auto detect webroot at runtime ([0031c34](https://github.com/jfortunato/wp-zip/commit/0031c34b6a460c2c1eb6b857dc6e5a6b47ecefb1))
* Parse table prefix from wp-config dynamically ([456e41b](https://github.com/jfortunato/wp-zip/commit/456e41be11e6d7d4089cfb33d5d4616dabea4ed2))
* Prompt for password if not given ([4cde8e5](https://github.com/jfortunato/wp-zip/commit/4cde8e521f994e407319ccfb86d02c8ac01f04a7))


### Code Refactoring

* Major refactor that splits responsibilities more logically ([9802d70](https://github.com/jfortunato/wp-zip/commit/9802d70f55e1768cb32814d81ac3ae5fbea28430))
* Use site-url instead of domain ([8e37e93](https://github.com/jfortunato/wp-zip/commit/8e37e93ea872336447edac6970e7ff48a915ccc8))

## [0.2.0](https://github.com/jfortunato/wp-zip/compare/v0.1.1...v0.2.0) (2023-09-25)


### Features

* Use site domain as localwp imported name ([c9da74d](https://github.com/jfortunato/wp-zip/commit/c9da74d67532a6679ad95f61213614e97049e58a))


### Bug Fixes

* Allow wp-config fields to use spaces ([748405a](https://github.com/jfortunato/wp-zip/commit/748405ad19d9f7e5836977c5b71ad0371d1054ff))
* Handle additional wp-config parsing cases ([40aebb0](https://github.com/jfortunato/wp-zip/commit/40aebb040cddfddf794ceb715001a0e19f38519a))
* Handle special chars in mysql cli password ([3733dff](https://github.com/jfortunato/wp-zip/commit/3733dffb6cc49264ae0b85eab81d88c2cb101bb9))
* Properly calculate size of public site files for progress bar ([69ff722](https://github.com/jfortunato/wp-zip/commit/69ff7225ce65f2f568efa71ca9dbb6b12e8d86a7))

## [0.1.1](https://github.com/jfortunato/wp-zip/compare/v0.1.0...v0.1.1) (2023-09-22)


### Miscellaneous Chores

* release 0.1.1 ([0ad9017](https://github.com/jfortunato/wp-zip/commit/0ad9017a3107fc27ccdcb4a24ac7c6a6b8369e01))

## [0.1.0](https://github.com/jfortunato/wp-zip/compare/0.0.1-alpha...v0.1.0) (2023-09-22)


### Features

* Allow user to set their own output file ([df6db51](https://github.com/jfortunato/wp-zip/commit/df6db511572b3af2e90c0da390f9ed63f4828925))
* Show progress bar for download_files operation ([c22e6b5](https://github.com/jfortunato/wp-zip/commit/c22e6b5d4b6dc7e50e693bde04b0bb567dfadffa))


### Bug Fixes

* Always use unix filepaths ([7f7eae3](https://github.com/jfortunato/wp-zip/commit/7f7eae3449c4037aa650d06bfe10fc0beaf11f22))


### Miscellaneous Chores

* release 0.1.0 ([69c772a](https://github.com/jfortunato/wp-zip/commit/69c772a610b6e1e0e257018de06056d425cc6d8c))
