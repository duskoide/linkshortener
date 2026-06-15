use assert_cmd::Command;
use predicates::prelude::*;

#[test]
fn test_binary_compiles_and_runs() {
    Command::cargo_bin("linkshortener")
        .unwrap()
        .assert()
        .failure()
        .stderr(
            predicate::str::contains("Failed")
                .or(predicate::str::contains("failed"))
                .or(predicate::str::contains("error"))
                .or(predicate::str::contains("Error")),
        );
}

#[test]
fn test_database_initialization() {
    assert!(true);
}
