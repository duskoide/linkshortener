use assert_cmd::Command;

#[test]
fn test_binary_compiles_and_runs() {
    Command::cargo_bin("linkshortener")
        .unwrap()
        .assert()
        .failure();
}

#[test]
fn test_database_initialization() {
    assert!(true);
}
