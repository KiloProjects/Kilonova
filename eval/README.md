# package eval

Eval is the part of kilonova that handles grading of programming tasks (hence why I think it should be renamed, see below). It is currently split in 4 parts: 
- the `box` package, which handles lower-level communication with the isolate program.
- the `isolate` directory, which has a stripped (and maybe will be modified more significantly in the future) version of [this repo](https://github.com/ioi/isolate). This contains the effective sandboxer that handles the OS-level stuff.
- the `boxmanager` package, which has a higher-level role of setting up, compiling and running tasks sent by users.
- the main.go package, which (for now) is just a sandbox (pun not intended) to test stuff up while I work on the other parts.


### TODO: change name to grader or something better, eval doesn't sound well.
