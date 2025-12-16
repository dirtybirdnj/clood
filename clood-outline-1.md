The file `clood-cli-spec.md` outlines the required features for a Rust CLI application similar to clood. These include:

1. **Tiered Model Inference**: Support fast models for context gathering and powerful models for reasoning.
2. **Project Awareness**: Integration with `projects_manifest.json` for project-specific configurations.
3. **Tool Integration**: MCP servers for filesystem, web search, and GitHub.
4. **Multi-Machine Support**: Ability to distribute workloads across multiple machines.

To implement these features in your Rust CLI app:

1. Use a Rust library like `clap` to define the command-line interface (CLI).
2. For Tiered Model Inference, consider using libraries like `tiny_actors` or `async-std`.
3. Integrate with `projects_manifest.json` using Rust's `serde` library for JSON serialization.
4. Use a crate like `reqwest` for HTTP requests to tool servers.
5. For multi-machine support, utilize Rust's standard library for networking and parallel processing.

Ensure you have the necessary dependencies installed by running `cargo build`.
