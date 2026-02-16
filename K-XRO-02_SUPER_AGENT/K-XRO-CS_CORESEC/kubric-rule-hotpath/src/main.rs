use regex::Regex;
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ValidationError {
    #[error("io error: {0}")]
    Io(#[from] std::io::Error),
    #[error("schema error: {0}")]
    Schema(String),
    #[error("regex error: {0}")]
    Regex(#[from] regex::Error),
}

#[derive(Debug, Serialize, Deserialize)]
pub struct InternalRule {
    pub id: String,
    pub category_uid: u16,
    pub class_uid: u16,
    pub activity_id: u8,
    pub pattern: String,
    pub severity_id: u8,
}

fn validate_and_warm(rule_path: &Path) -> Result<InternalRule, ValidationError> {
    let content = fs::read_to_string(rule_path)?;
    let rule: InternalRule =
        serde_json::from_str(&content).map_err(|error| ValidationError::Schema(error.to_string()))?;

    if rule.class_uid == 0 {
        return Err(ValidationError::Schema("class_uid must be set".to_string()));
    }

    if rule.category_uid == 0 {
        return Err(ValidationError::Schema("category_uid must be set".to_string()));
    }

    Regex::new(&rule.pattern)?;

    Ok(rule)
}

fn main() {
    let candidate = std::env::args()
        .nth(1)
        .unwrap_or_else(|| "rules/compiled_rule_101.json".to_string());

    match validate_and_warm(Path::new(&candidate)) {
        Ok(rule) => println!("rule-ready:{} class_uid:{}", rule.id, rule.class_uid),
        Err(error) => {
            eprintln!("rule-invalid:{}", error);
            std::process::exit(1);
        }
    }
}
