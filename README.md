## Build

Use MSYS2 MSYS Shell to build Windows binary

```
$ pacman -Syy
$ pacman -S mingw-w64-x86_64-go mingw-w64-x86_64-libvips mingw-w64-x86_64-pkg-config mingw-w64-x86_64-gcc
$ go build -o ccfolia-room-minifier.exe cmd/main.go
```
