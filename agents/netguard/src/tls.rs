//! NetGuard TLS SNI extraction — parses TLS ClientHello to extract Server Name Indication.
//!
//! Used to identify destination hostnames from encrypted flows without decryption.
//! This is critical for threat intelligence lookups, domain-based IDS rules,
//! and DNS tunneling detection.

use serde::Serialize;

/// Extracted TLS ClientHello fields.
#[derive(Debug, Clone, Serialize)]
pub struct TlsClientHello {
    pub sni: String,
    pub tls_version: String,
    pub cipher_suites_count: usize,
}

/// Attempt to parse a TLS ClientHello from a TCP payload.
/// Returns None if the payload is not a valid ClientHello.
pub fn parse_client_hello(payload: &[u8]) -> Option<TlsClientHello> {
    // TLS record header: ContentType(1) + Version(2) + Length(2)
    if payload.len() < 5 {
        return None;
    }
    // ContentType 22 = Handshake
    if payload[0] != 22 {
        return None;
    }

    let tls_version = format!("{}.{}", payload[1], payload[2]);
    let record_len = u16::from_be_bytes([payload[3], payload[4]]) as usize;

    if payload.len() < 5 + record_len {
        return None;
    }

    let handshake = &payload[5..5 + record_len];

    // Handshake header: type(1) + length(3)
    if handshake.is_empty() || handshake[0] != 1 {
        // Type 1 = ClientHello
        return None;
    }
    if handshake.len() < 4 {
        return None;
    }

    let hs_len =
        ((handshake[1] as usize) << 16) | ((handshake[2] as usize) << 8) | (handshake[3] as usize);
    if handshake.len() < 4 + hs_len {
        return None;
    }

    let ch = &handshake[4..4 + hs_len];

    // ClientHello: Version(2) + Random(32) + SessionID(variable) + CipherSuites(variable) + ...
    if ch.len() < 34 {
        return None;
    }

    let mut pos = 2 + 32; // skip version + random

    // Session ID length
    if pos >= ch.len() {
        return None;
    }
    let session_id_len = ch[pos] as usize;
    pos += 1 + session_id_len;

    // Cipher suites
    if pos + 2 > ch.len() {
        return None;
    }
    let cipher_suites_len = u16::from_be_bytes([ch[pos], ch[pos + 1]]) as usize;
    let cipher_suites_count = cipher_suites_len / 2;
    pos += 2 + cipher_suites_len;

    // Compression methods
    if pos >= ch.len() {
        return None;
    }
    let comp_len = ch[pos] as usize;
    pos += 1 + comp_len;

    // Extensions
    if pos + 2 > ch.len() {
        return None;
    }
    let extensions_len = u16::from_be_bytes([ch[pos], ch[pos + 1]]) as usize;
    pos += 2;

    let extensions_end = pos + extensions_len;
    if extensions_end > ch.len() {
        return None;
    }

    // Walk extensions looking for SNI (type 0x0000)
    while pos + 4 <= extensions_end {
        let ext_type = u16::from_be_bytes([ch[pos], ch[pos + 1]]);
        let ext_len = u16::from_be_bytes([ch[pos + 2], ch[pos + 3]]) as usize;
        pos += 4;

        if ext_type == 0 {
            // SNI extension
            if let Some(sni) = parse_sni_extension(&ch[pos..pos + ext_len]) {
                return Some(TlsClientHello {
                    sni,
                    tls_version,
                    cipher_suites_count,
                });
            }
        }

        pos += ext_len;
    }

    None
}

fn parse_sni_extension(data: &[u8]) -> Option<String> {
    // SNI list length (2) + name type (1) + name length (2) + name
    if data.len() < 5 {
        return None;
    }
    let _list_len = u16::from_be_bytes([data[0], data[1]]) as usize;
    let name_type = data[2];
    if name_type != 0 {
        // Type 0 = hostname
        return None;
    }
    let name_len = u16::from_be_bytes([data[3], data[4]]) as usize;
    if data.len() < 5 + name_len {
        return None;
    }
    String::from_utf8(data[5..5 + name_len].to_vec()).ok()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_empty_returns_none() {
        assert!(parse_client_hello(&[]).is_none());
    }

    #[test]
    fn parse_non_tls_returns_none() {
        assert!(parse_client_hello(&[0x47, 0x45, 0x54, 0x20, 0x2f]).is_none());
    }

    #[test]
    fn parse_valid_client_hello() {
        // Minimal TLS 1.2 ClientHello with SNI for "example.com"
        let sni_name = b"example.com";
        let sni_ext = build_sni_extension(sni_name);
        let ch_body = build_client_hello(&sni_ext, &[0x00, 0x2f]); // TLS_RSA_WITH_AES_128_CBC_SHA
        let handshake = build_handshake(1, &ch_body);
        let record = build_tls_record(22, 3, 3, &handshake);

        let result = parse_client_hello(&record);
        assert!(result.is_some());
        let hello = result.unwrap();
        assert_eq!(hello.sni, "example.com");
        assert_eq!(hello.cipher_suites_count, 1);
    }

    fn build_tls_record(content_type: u8, major: u8, minor: u8, payload: &[u8]) -> Vec<u8> {
        let mut buf = vec![content_type, major, minor];
        buf.extend_from_slice(&(payload.len() as u16).to_be_bytes());
        buf.extend_from_slice(payload);
        buf
    }

    fn build_handshake(hs_type: u8, body: &[u8]) -> Vec<u8> {
        let len = body.len();
        let mut buf = vec![hs_type, (len >> 16) as u8, (len >> 8) as u8, len as u8];
        buf.extend_from_slice(body);
        buf
    }

    fn build_client_hello(extensions_payload: &[u8], cipher_suites: &[u8]) -> Vec<u8> {
        let mut buf = Vec::new();
        // Version
        buf.extend_from_slice(&[0x03, 0x03]); // TLS 1.2
        // Random (32 bytes)
        buf.extend_from_slice(&[0u8; 32]);
        // Session ID (0 length)
        buf.push(0);
        // Cipher suites
        buf.extend_from_slice(&(cipher_suites.len() as u16).to_be_bytes());
        buf.extend_from_slice(cipher_suites);
        // Compression methods (1 method: null)
        buf.push(1);
        buf.push(0);
        // Extensions
        buf.extend_from_slice(&(extensions_payload.len() as u16).to_be_bytes());
        buf.extend_from_slice(extensions_payload);
        buf
    }

    fn build_sni_extension(hostname: &[u8]) -> Vec<u8> {
        let mut sni_list = Vec::new();
        sni_list.push(0); // name type = hostname
        sni_list.extend_from_slice(&(hostname.len() as u16).to_be_bytes());
        sni_list.extend_from_slice(hostname);

        let mut ext = Vec::new();
        // Extension type 0x0000 = SNI
        ext.extend_from_slice(&[0x00, 0x00]);
        let ext_data_len = 2 + sni_list.len();
        ext.extend_from_slice(&(ext_data_len as u16).to_be_bytes());
        ext.extend_from_slice(&(sni_list.len() as u16).to_be_bytes());
        ext.extend_from_slice(&sni_list);
        ext
    }
}
