// ─────────────────────────────────────────────────────────────────────────────
// Kubric-UiDR — Commitlint Configuration (Conventional Commits)
// Enforces consistent commit message format across the monorepo.
// ─────────────────────────────────────────────────────────────────────────────

/** @type {import('@commitlint/types').UserConfig} */
module.exports = {
  extends: ["@commitlint/config-conventional"],

  rules: {
    // ── Type ─────────────────────────────────────────────────────────────
    "type-enum": [
      2,
      "always",
      [
        "feat",     // New feature
        "fix",      // Bug fix
        "chore",    // Maintenance / tooling
        "docs",     // Documentation only
        "style",    // Formatting, no logic change
        "refactor", // Code restructure, no behaviour change
        "perf",     // Performance improvement
        "test",     // Adding or updating tests
        "ci",       // CI/CD pipeline changes
        "build",    // Build system or dependency changes
        "revert",   // Reverts a previous commit
      ],
    ],
    "type-case": [2, "always", "lower-case"],
    "type-empty": [2, "never"],

    // ── Scope ────────────────────────────────────────────────────────────
    "scope-enum": [
      2,
      "always",
      [
        // Rust agents
        "coresec",
        "netguard",
        "perftrace",
        "provisioning",
        "watchdog",
        // Go services
        "ksvc",
        "kic",
        "noc",
        "vdr",
        "kbridges",
        // Python AI
        "kai",
        "kai-rag",
        "kai-crew",
        // Frontend
        "portal",
        // Infrastructure
        "infra",
        "k8s",
        "docker",
        // Cross-cutting
        "ci",
        "proto",
        "db",
        "deps",
        "release",
      ],
    ],
    "scope-case": [2, "always", "lower-case"],
    "scope-empty": [1, "never"],

    // ── Subject ──────────────────────────────────────────────────────────
    "subject-case": [2, "never", ["upper-case", "pascal-case", "start-case"]],
    "subject-empty": [2, "never"],
    "subject-max-length": [2, "always", 100],
    "subject-full-stop": [2, "never", "."],

    // ── Header ───────────────────────────────────────────────────────────
    "header-max-length": [2, "always", 120],

    // ── Body ─────────────────────────────────────────────────────────────
    "body-leading-blank": [2, "always"],
    "body-max-line-length": [1, "always", 200],

    // ── Footer ───────────────────────────────────────────────────────────
    "footer-leading-blank": [2, "always"],
    "footer-max-line-length": [1, "always", 200],
  },

  // ── Help URL ───────────────────────────────────────────────────────────
  helpUrl:
    "https://github.com/managekube-hue/Kubric-UiDR/blob/main/DEVELOPER-BOOTSTRAP.md#commit-messages",
};
