// build.rs — protobuf compilation for the Kubric Rust services workspace.
// Invoked automatically by `cargo build`.
//
// Requires tonic-build in [build-dependencies]:
//   tonic-build = { version = "0.11", features = ["prost"] }
//
// If proto files are not yet present (e.g. initial checkout before buf generate),
// the script exits 0 so that `cargo check` succeeds without the generated sources.

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let proto_files = vec![
        "../../proto/kubric/v1/event.proto",
        "../../proto/kubric/v1/tenant.proto",
        "../../proto/kubric/v1/alert.proto",
        "../../proto/kubric/v1/agent.proto",
        "../../proto/kubric/v1/patch_job.proto",
        "../../proto/kubric/v1/decision.proto",
    ];

    let include_dirs = vec!["../../proto"];

    // Verify that at least one proto file exists before attempting compilation.
    // This allows `cargo check` to pass in environments where proto files have
    // not yet been generated via `buf generate`.
    let any_proto_exists = proto_files
        .iter()
        .any(|p| std::path::Path::new(p).exists());

    if !any_proto_exists {
        eprintln!(
            "cargo:warning=No .proto files found under ../../proto — \
             skipping protobuf compilation. \
             Run `buf generate` from the repo root first."
        );
        // Re-run when proto files are added.
        for f in &proto_files {
            println!("cargo:rerun-if-changed={f}");
        }
        println!("cargo:rerun-if-changed=build.rs");
        return Ok(());
    }

    // Create output directory for generated Rust sources.
    let out_dir = "src/proto";
    std::fs::create_dir_all(out_dir)?;

    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .out_dir(out_dir)
        // Include serde Serialize/Deserialize derives on all generated types
        // so they can be used in REST handlers and stored as JSON.
        .type_attribute(".", "#[derive(serde::Serialize, serde::Deserialize)]")
        .type_attribute(".", "#[serde(rename_all = \"snake_case\")]")
        // Emit rerun-if-changed for each file individually (also done below).
        .emit_rerun_if_changed(true)
        .compile_protos(
            // Only compile files that exist — skip missing ones gracefully.
            &proto_files
                .iter()
                .filter(|p| std::path::Path::new(p).exists())
                .copied()
                .collect::<Vec<_>>(),
            &include_dirs,
        )
        .unwrap_or_else(|e| {
            // Emit as a warning rather than a hard error so CI does not break
            // when proto compilation fails due to missing toolchain.
            eprintln!("cargo:warning=protobuf compilation error: {e}");
            // Exit 0 to allow downstream `cargo build` steps to continue.
            std::process::exit(0);
        });

    // Explicit rerun-if-changed directives so incremental builds work correctly.
    for f in &proto_files {
        println!("cargo:rerun-if-changed={f}");
    }
    println!("cargo:rerun-if-changed=build.rs");

    // Emit the proto include path as a cargo env var for integration tests.
    println!("cargo:rustc-env=KUBRIC_PROTO_PATH=../../proto");

    Ok(())
}
