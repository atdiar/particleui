 # zui REPL

 This is an implementation of a wasm in-memory repl, not unlike the 
 official go playground, but local and allowing to create UI code with 
 zui.
 It allows for the compilation of go code using a wasm version of the compiler, standard library and zui framework and loads the result in
 an iframe.

 As a component, it should be able to insert it in a webpage, as is or will be done on the 
 framework website zui.dev

 It should also allow for rapid prototyping, allowing features such as storybooks.

 ## ReplElement component

 As a component, it is composed of 3 elements:
 - the textarea which receives the code input
 - a div that displays the compilation status, errors etc.
 - an iframe which displays the result by creating an about:blank page that
 will load the generated wasm.

 In order to build a wasm file from the code inserted in the textarea input (or format said code), some wasm files (compile.wasm, link.wasm, gofmt.wasm) and package pre-compiled archives (standard library packages and zui framework library with the js driver + their dependencies) need to be fetched and loaded.

 These files should be made available from a CDN ideally.

 ### constructor function
 The repl component constructor function modifies the document to add repl support if it hasn't been done already by.
In order to add repl support, a script is appended to the dcoument head that applies the following:

 - add vfs.js as an import. This file implements the indexedDB backed Virtual File System
 - add the exec function that allows to run wasm commands (compile, link, gofmt)
 - add the logic that fetches and loads compile.wasm, link.wasm, gofmt.wasm
 - add the logic that fetches and loads the package archives such as runtime.a, log.a and so on.
 these will be used during the link phase of the build process
 - add the logic that verifies on page load the version of the cached wasm compiler versus the version required by index.html
 - add the logic that loads the compiler assets and caches them in indexedDB
 - add the logic that creates the importcfg file whcih will be added to the virtual file system

It is only needed to be done once. As such, detecting whether this has been done before is required.
It is easily done by checking the presence of an element with the script ID.

Lastly, we need to define the functions that allow to run the code (fmt, compile, link, execute, load in iframe, load compilation status) as methods of such a ReplElement.
We may want to define such functions fully in js, callable in Go via wasm, or build them incrementally on the go side.

