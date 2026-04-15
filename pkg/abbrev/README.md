# abbrev
This is a fork of [xmonader/abbrev](https://github.com/xmonader/abbrev), Abbrev package ported from ruby to Go land.

## Fork changes
- better collision detection
- empty string is not a valid abbreviation
- do not delete input keywords on collision with an abbreviation
- tests for the above