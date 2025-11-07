pub fn hash_password(password: &str) -> String {
    // compute SHA-256 hash of the password and convert to hex string
    let mut hasher = sha2::Sha256::new();
    hasher.update(password.as_bytes());
    let hash_bytes = hasher.finalize();
    hash_bytes
        .iter()
        .map(|b| format!("{:02x}", b))
        .collect::<String>()
}

pub fn verify_password(password: &str, hash: &str) -> bool {
    let computed_hash = hash_password(password);
    computed_hash == hash
}
