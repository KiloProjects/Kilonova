---
title: Problem parameters
---

# Introduction

Kilonova was designed to be relatively flexible with its support of problem types and parameters. As such, many mixes and matches are possible. This document hopes to explain the various categories and how well they match with other problem parameters.

## Checkers

Checkers (or, in CMS terminology, `comparators`) evaluate how close a submission's output matches the official testcase output. The similarity is on a scale from 0% to 100%, where 0% is a fully wrong answer and 100% is a correct answer. After checkers return a **verdict** (similarity percentage and eventual _comments_), that percentage is used to compute the submission's final score on the test.

For example, if a test is valued at $4$ points and a (custom) checker marks the output as 75% correct, the submission will receive $3$ out of the $4$ available points.

### Standard (white-diff) checker

The standard (built-in) checker uses the system's `diff` command to compare submission output with the official test output. Due to its limited checking abilities, it can only give either a 0% or 100% correct status (no partial scores).

More specifically, it runs the following command:

```sh
$ diff -qBbEa <submission_out_path> <test_out_path>
```

Here's the breakdown of each flag:

-   `-q` - do not show the actual diff, only return the exit code (`0` - inputs are the same, correct answer, `1` - inputs differ, wrong answer);
-   `-B` - ignore blank lines
-   `-b` - ignore changes in white space (2 spaces vs 4 spaces)
-   `-E` - ignore tab expansion (`\t` character vs 4 spaces)
-   `-a` - force treat the files as text

### Custom checker

When there are multiple correct outputs (or when there's partial scoring), the standard checker is not powerful enough. In such cases, a **custom checker** can be used to perform validation.

The best practice is for checkers to be written in C++, since their compilation is cached between multiple submissions and they are fastest.

In an effort to improve localization, it is encouraged that standard messages are returned by translation keys as follows:

-   `translate:success` -> `Correct`
-   `translate:partial` -> `Partially correct`
-   `translate:wrong` -> `Wrong answer`

Kilonova supports 2 subtly different checker types. Their difference lies just in the way they are supposed to interact with the external environment:

=== "Standard checkers"

    -   Can be uploaded as attachments of the form `checker.cpp`, `checker.cpp20`, `checker.pas`, etc;
    -   This is the official custom checker format for Kilonova and all problems should implement this one.
    -   They are compatible with the CMS format. That is:
        -   Upon execution, the program receives 3 arguments, in this order:
            1. path to the test input - it is assumed to be always valid;
            2. path to the correct test output;
            3. path to the user submission's output.
        -   The checker's verdict is returned according to the [CMS Standard manager output](https://cms.readthedocs.io/en/latest/Task%20types.html#tasktypes-standard-manager-output):
            -   `stdout` (`cout`) contains a single line, containing a floating point number from $0$ to $1$, proportional to the score received, expressed as a percentage.
            -   `stderr` (`cerr`) contains a single line, containing the message for the contestant.

    Example snippet:

    ``` c++ title="checker.cpp"
    #include <bits/stdc++.h>

    using namespace std;

    float score = 0;

    void result(const char* msg, float pts) {
        //"{score}" to stdout
        //"{message}" to stderr
        // score is a float between 0 and 1
        printf("%f", pts);
        fprintf(stderr, "%s", msg);
        exit(0);
    }

    void Assert(bool cond, string str) {
        if (!cond)
            result(str.c_str(), 0);
    }
    void Success(float pts, string str) {
        result(str.c_str(), pts);
    }

    ifstream out, ok, in;

    int main(int argc, char* argv[]) {
        in.open(argv[1]); // test input; assumed to be valid
        ok.open(argv[2]); // correct output
        out.open(argv[3]); // user output
        int task;
        in >> task;
        if(task == 1){
            int ansok, ansout;
            ok >> ansok;
            Assert(!!(out >> ansout), "translate:wrong"); // try to read contestant's answer
            Assert(ansout == ansok, "translate:wrong"); // check if values match
            score += 1;
            Success(score, "translate:success");
        }
        else {
            int nrok, nrout;
            // not giving partial points if the first part of the answer is wrong, even if the second one might be right
            Assert(!!(out >> nrout), "Wrong answer. Wrong format for number");
            ok >> nrok;
            Assert(nrout == nrok, "Wrong answer. Incorrect number");
            score += 0.5;
            string message = "Correct number. ";
            double valok, valout;
            ok >> valok;
            // giving partial points even if the second part of the answer is wrong
            if(!!(out >> valout) && abs(valok - valout) <= 1e-5)
                score += 0.5, message += "Correct value";
            else
                message += "Wrong value";
            Success(score, message);
        }
    }

    ```

=== "Legacy checkers"

    -   Can be uploaded as attachments of the form `checker_legacy.cpp`, `checker_legacy.cpp20`, etc;
    -   These are still available for compatibility reasons, but their usage should be avoided wherever possible and new problems might not be published if they make use of this checker.
    -   They are compatible with the legacy Romanian Olympiad format:
        -   Upon execution, the program receives 3 arguments, in this order:
            1. path to the user submission's output;
            2. path to the correct test output;
            3. path to the test input - it is assumed to be always valid.
                - Historical fun fact! This is a later addition by us, in order for legacy checkers to also have access to the test input. The original format did not offer this parameter and contest organizers would sometimes need to copy-paste the test input into the output.
        -   The checker's verdict is returned as follows:
            -   The score and verdict are read from `stdout` (`cout`), separated by a space. The score is a number from $0$ to $100$, showing how correct the output is, as a readable percentage.
            -   For example, if the checker outputs `75 Almost there!`, the test will be graded with 75% correctness and `Almost there!` will appear in the UI.

    Example snippet:

    ``` c++ title="checker_legacy.cpp"
    #include <bits/stdc++.h>

    using namespace std;

    int score = 0;

    void result(const char* msg, int pts) {
        // "{score} {message}" to stdout
        // score is an integer between 0 and 100
        printf("%d %s", pts, msg);
        exit(0);
    }

    void Assert(bool cond, string str) {
        if (!cond)
            result(str.c_str(), 0);
    }
    void Success(int pts, string str) {
        result(str.c_str(), pts);
    }

    ifstream out, ok, in;

    int main(int argc, char* argv[]) {
        // NOTE: Newer problems should use the standard checker instead of the legacy one.
        out.open(argv[1]); // user output
        ok.open(argv[2]); // correct output
        in.open(argv[3]); // test input; assumed to be valid
        int task;
        in >> task;
        if(task == 1){
            int ansok, ansout;
            ok >> ansok;
            Assert(!!(out >> ansout), "translate:wrong"); // try to read contestant's answer
            Assert(ansout == ansok, "translate:wrong"); // check if values match
            score += 100;
            Success(score, "translate:success");
        }
        else {
            int nrok, nrout;
            // not giving partial points if the first part of the answer is wrong, even if the second one might be right
            Assert(!!(out >> nrout), "Wrong answer. Wrong format for number");
            ok >> nrok;
            Assert(nrout == nrok, "Wrong answer. Incorrect number");
            score += 50;
            string message = "Correct number. ";
            double valok, valout;
            ok >> valok;
            // giving partial points even if the second part of the answer is wrong
            if(!!(out >> valout) && abs(valok - valout) <= 1e-5)
                score += 50, message += "Correct value";
            else
                message += "Wrong value";
            Success(score, message);
        }
    }
    ```

Besides the input/output files, checkers also have access to the contestant's source code in a special file called `contestant.txt`, available from the checker's working directory. It can be used to disallow certain keywords or create bespoke source code requirements for certain problems.

!!! note

    Despite the compatibility between Kilonova and CMS checkers, the biggest difference between the platforms is how they are distributed:

    - Kilonova requires uploading the source code for the checkers, which is afterwards compiled and cached, whereas
    - CMS requires the admins to build a Linux executable that will be run by the grading system.
