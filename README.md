# Kilonova

[![GoDoc](https://godoc.org/github.com/KiloProjects/Kilonova?status.svg)](https://godoc.org/github.com/KiloProjects/Kilonova)

Kilonova is a (work-in-progress) platform for competitive programming.

## For other developers, which may want to contribute

I added `.gitignore`s for `run.sh`, which could host your scripts for compilation, execution, everything.

Also, running `pkger` involves moving `sqldata` out of the project directory, i don't know how to mitigate this since pkger doesn't have an `-exclude` option yet (though it seems they are working on it).

## STUFF TO DO

-   For closed alpha:

    -   Finish grader (95%, only more testing is needed)
    -   Connect grader to DB (Done)
    -   Write minimal web interface (80% done)
    -   Design a user hierarchy
    -   Toasts

-   For closed beta:

    -   MAKE A TON OF TESTS (need to find something easy to develop tests with)
    -   Work on web interface
    -   Add other tools and rank
    -   Add various extensions to the task description markup (like MathJax)
    -   Add more functionality

-   For open beta:

    -   We'll see
    -   Remove gorm dependency, rely on pure SQL

-   For release: - If we get here, I'm sure we'll figure out what is missing
