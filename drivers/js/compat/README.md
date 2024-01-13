# Bridge (drivers/js)

The bridge is a build-time package abstraction around syscall/js that enables to use
the same API to modify a UI tree on the server (where nodes are *html.Node instead of js.Value)


This is a drop-in replacement for syscall/js and as such, shares its status (experimental for now).