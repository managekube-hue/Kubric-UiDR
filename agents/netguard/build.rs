use std::env;
use std::path::PathBuf;

fn main() {
    println!("cargo:rerun-if-env-changed=KUBRIC_NDPI_INCLUDE");
    println!("cargo:rerun-if-env-changed=KUBRIC_NDPI_LIB_DIR");
    println!("cargo:rerun-if-env-changed=KUBRIC_NDPI_BINDGEN");

    let manifest_dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR").unwrap_or_default());
    let repo_root = manifest_dir.parent().and_then(|p| p.parent()).map(PathBuf::from);

    let include_dir = env::var("KUBRIC_NDPI_INCLUDE")
        .map(PathBuf::from)
        .ok()
        .or_else(|| repo_root.as_ref().map(|r| r.join("vendor/ndpi/include")));

    let lib_dir = env::var("KUBRIC_NDPI_LIB_DIR")
        .map(PathBuf::from)
        .ok()
        .or_else(|| repo_root.as_ref().map(|r| r.join("vendor/ndpi/lib")));

    if let Some(ref lib) = lib_dir {
        if lib.exists() {
            println!("cargo:rustc-env=KUBRIC_NDPI_LIB_DIR={}", lib.display());
        }
    }

    let bindgen_enabled = env::var("KUBRIC_NDPI_BINDGEN")
        .map(|v| v == "1" || v.eq_ignore_ascii_case("true"))
        .unwrap_or(true);

    if !bindgen_enabled {
        return;
    }

    let Some(include_dir) = include_dir else {
        return;
    };
    let header = include_dir.join("ndpi_api.h");
    if !header.exists() {
        println!("cargo:warning=nDPI header not found at {} (bindgen skipped)", header.display());
        return;
    }

    let bindings = bindgen::Builder::default()
        .header(header.display().to_string())
        .clang_arg(format!("-I{}", include_dir.display()))
        .allowlist_function("ndpi_.*")
        .allowlist_type("ndpi_.*")
        .allowlist_var("NDPI_.*")
        .generate_comments(false)
        .generate();

    match bindings {
        Ok(bindings) => {
            let out_dir = PathBuf::from(env::var("OUT_DIR").unwrap_or_default());
            let out_file = out_dir.join("ndpi_bindings.rs");
            if let Err(e) = bindings.write_to_file(&out_file) {
                println!("cargo:warning=failed to write nDPI bindings: {e}");
            } else {
                println!("cargo:warning=nDPI bindings generated at {}", out_file.display());
            }
        }
        Err(e) => {
            println!("cargo:warning=bindgen failed for nDPI headers: {e}");
        }
    }
}
