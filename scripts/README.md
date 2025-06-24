# Scripts Directory

This directory contains utility scripts related to the `codebase-syncer` project. These scripts are designed to automate common development tasks, such as building the application for different platforms.

## Available Scripts

*   **[`build.sh`](scripts/build.sh:1):**
    *   A Bash script for compiling the Go application.
    *   It allows specifying the target operating system (`GOOS`), architecture (`GOARCH`), and a version string (`VERSION`).
    *   The script injects the version string into the binary and names the output fileConventionally, including version, OS, and architecture.
    *   Build artifacts are placed in the `bin` directory at the project root.
    *   **Usage:** `./build.sh <GOOS> <GOARCH> <VERSION>`
    *   **Example:** `./build.sh linux amd64 v1.0.0`

---

Feel free to add more scripts to this directory as needed to streamline other development or operational tasks.