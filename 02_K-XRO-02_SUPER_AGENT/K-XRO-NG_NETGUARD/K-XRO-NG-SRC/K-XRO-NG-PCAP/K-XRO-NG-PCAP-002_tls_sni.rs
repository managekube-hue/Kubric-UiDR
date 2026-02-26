//! K-XRO-NG-PCAP-002 — TLS SNI extractor with JA3 fingerprinting.
//!
//! Parses TLS ClientHello packets to extract:
//! * Server Name Indication (SNI) — hostname the client is connecting to
//! * TLS version advertised by the client
//! * Complete cipher_suites and extensions lists
//! * JA3 fingerprint (MD5 of standardised ClientHello fields)
//!
//! JA3 is an industry-standard TLS fingerprinting technique:
//! https://github.com/salesforce/ja3
//!
//! # Cargo dependencies
//! ```toml
//! serde = { version = "1", features = ["derive"] }
//! md5   = "0.7"
//! ```

use serde::{Deserialize, Serialize};

// ─────────────────────────────────────────────────────────────────────────────
// Public types
// ─────────────────────────────────────────────────────────────────────────────

/// Parsed fields from a TLS ClientHello.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TlsClientHello {
    /// Server Name Indication hostname (empty string if absent).
    pub sni: String,
    /// TLS record-layer version string (e.g. "3.3" for TLS 1.2).
    pub version: String,
    /// Handshake-layer ClientHello version as a raw u16.
    pub handshake_version: u16,
    /// List of cipher suite IDs advertised.
    pub cipher_suites: Vec<u16>,
    /// List of TLS extension type IDs present.
    pub extensions: Vec<u16>,
    /// List of supported elliptic curve group IDs (extension 0x000a).
    pub elliptic_curves: Vec<u16>,
    /// List of EC point format IDs (extension 0x000b).
    pub ec_point_formats: Vec<u8>,
    /// JA3 fingerprint hex string (computed from standardised fields).
    pub ja3: String,
}

// ─────────────────────────────────────────────────────────────────────────────
// Top-level API functions
// ─────────────────────────────────────────────────────────────────────────────

/// Returns `true` if the first bytes indicate a TLS record containing a
/// ClientHello handshake.
///
/// Checks:
/// * `payload[0] == 0x16` — ContentType: Handshake
/// * `payload[1] == 0x03` — TLS major version 3.x
/// * Internal handshake type == 1 (ClientHello)
pub fn is_tls_client_hello(payload: &[u8]) -> bool {
    if payload.len() < 6 {
        return false;
    }
    if payload[0] != 0x16 || payload[1] != 0x03 {
        return false;
    }
    // Peek at handshake type inside record
    let record_len = u16::from_be_bytes([payload[3], payload[4]]) as usize;
    if payload.len() < 5 + record_len {
        return false;
    }
    let hs_type = payload[5];
    hs_type == 0x01
}

/// Extract the SNI hostname from a TLS ClientHello payload.
/// Returns `None` if the payload is not a valid ClientHello or contains no SNI.
pub fn extract_sni(payload: &[u8]) -> Option<String> {
    let hello = parse_client_hello(payload)?;
    if hello.sni.is_empty() {
        None
    } else {
        Some(hello.sni)
    }
}

/// Fully parse a TLS ClientHello.  Returns `None` on any parse failure.
pub fn parse_client_hello(payload: &[u8]) -> Option<TlsClientHello> {
    // ── TLS record header ────────────────────────────────────────────────
    if payload.len() < 5 {
        return None;
    }
    if payload[0] != 0x16 {
        // ContentType must be Handshake (22)
        return None;
    }
    let record_major = payload[1];
    let record_minor = payload[2];
    let version = format!("{record_major}.{record_minor}");

    let record_len = u16::from_be_bytes([payload[3], payload[4]]) as usize;
    if payload.len() < 5 + record_len {
        return None;
    }
    let handshake = &payload[5..5 + record_len];

    // ── Handshake header ─────────────────────────────────────────────────
    if handshake.len() < 4 {
        return None;
    }
    if handshake[0] != 0x01 {
        // HandshakeType must be ClientHello (1)
        return None;
    }
    let hs_body_len = u24_to_usize(&handshake[1..4]);
    if handshake.len() < 4 + hs_body_len {
        return None;
    }
    let ch = &handshake[4..4 + hs_body_len];

    // ── ClientHello body ─────────────────────────────────────────────────
    // Version(2) + Random(32) = 34 bytes minimum before session ID
    if ch.len() < 34 {
        return None;
    }
    let handshake_version = u16::from_be_bytes([ch[0], ch[1]]);
    let mut pos = 34; // skip version(2) + random(32)

    // Session ID
    if pos >= ch.len() {
        return None;
    }
    let session_id_len = ch[pos] as usize;
    pos += 1 + session_id_len;

    // Cipher suites
    if pos + 2 > ch.len() {
        return None;
    }
    let cs_len = u16::from_be_bytes([ch[pos], ch[pos + 1]]) as usize;
    pos += 2;
    if pos + cs_len > ch.len() {
        return None;
    }
    let mut cipher_suites: Vec<u16> = Vec::with_capacity(cs_len / 2);
    {
        let mut i = 0;
        while i + 1 < cs_len {
            let cs = u16::from_be_bytes([ch[pos + i], ch[pos + i + 1]]);
            // Exclude GREASE values from JA3
            if !is_grease(cs) {
                cipher_suites.push(cs);
            }
            i += 2;
        }
    }
    pos += cs_len;

    // Compression methods
    if pos >= ch.len() {
        return None;
    }
    let comp_len = ch[pos] as usize;
    pos += 1 + comp_len;

    // Extensions (optional)
    let mut extensions: Vec<u16> = Vec::new();
    let mut elliptic_curves: Vec<u16> = Vec::new();
    let mut ec_point_formats: Vec<u8> = Vec::new();
    let mut sni = String::new();

    if pos + 2 <= ch.len() {
        let ext_total_len = u16::from_be_bytes([ch[pos], ch[pos + 1]]) as usize;
        pos += 2;
        let ext_end = pos + ext_total_len;
        if ext_end <= ch.len() {
            while pos + 4 <= ext_end {
                let ext_type = u16::from_be_bytes([ch[pos], ch[pos + 1]]);
                let ext_len = u16::from_be_bytes([ch[pos + 2], ch[pos + 3]]) as usize;
                pos += 4;

                if pos + ext_len > ext_end {
                    break;
                }
                let ext_data = &ch[pos..pos + ext_len];

                if !is_grease(ext_type) {
                    extensions.push(ext_type);
                }

                match ext_type {
                    // SNI (0x0000)
                    0x0000 => {
                        if let Some(s) = parse_sni_extension(ext_data) {
                            sni = s;
                        }
                    }
                    // Supported Groups / Elliptic Curves (0x000a)
                    0x000a => {
                        elliptic_curves = parse_u16_list(ext_data, 2);
                        // Filter GREASE
                        elliptic_curves.retain(|&c| !is_grease(c));
                    }
                    // EC Point Formats (0x000b)
                    0x000b => {
                        if !ext_data.is_empty() {
                            let count = ext_data[0] as usize;
                            if ext_data.len() >= 1 + count {
                                ec_point_formats = ext_data[1..1 + count].to_vec();
                            }
                        }
                    }
                    _ => {}
                }

                pos += ext_len;
            }
        }
    }

    let mut hello = TlsClientHello {
        sni,
        version,
        handshake_version,
        cipher_suites,
        extensions,
        elliptic_curves,
        ec_point_formats,
        ja3: String::new(),
    };
    hello.ja3 = compute_ja3(&hello);
    Some(hello)
}

// ─────────────────────────────────────────────────────────────────────────────
// JA3 fingerprinting
// ─────────────────────────────────────────────────────────────────────────────

/// Compute the JA3 fingerprint for a parsed `TlsClientHello`.
///
/// JA3 string format:
/// `SSLVersion,Ciphers,Extensions,EllipticCurves,EllipticCurvePointFormats`
///
/// All fields are dash-separated lists of decimal values.  The final
/// fingerprint is the MD5 hex digest of this string.
pub fn compute_ja3(hello: &TlsClientHello) -> String {
    let ja3_str = build_ja3_string(hello);
    md5_hex(ja3_str.as_bytes())
}

fn build_ja3_string(hello: &TlsClientHello) -> String {
    let ssl_version = hello.handshake_version;

    let ciphers = hello
        .cipher_suites
        .iter()
        .map(|c| c.to_string())
        .collect::<Vec<_>>()
        .join("-");

    let exts = hello
        .extensions
        .iter()
        .map(|e| e.to_string())
        .collect::<Vec<_>>()
        .join("-");

    let curves = hello
        .elliptic_curves
        .iter()
        .map(|c| c.to_string())
        .collect::<Vec<_>>()
        .join("-");

    let formats = hello
        .ec_point_formats
        .iter()
        .map(|f| f.to_string())
        .collect::<Vec<_>>()
        .join("-");

    format!("{ssl_version},{ciphers},{exts},{curves},{formats}")
}

/// Tiny self-contained MD5 implementation to avoid a crate dependency.
/// This is the reference RFC-1321 algorithm.
fn md5_hex(data: &[u8]) -> String {
    // MD5 constants
    #[rustfmt::skip]
    const K: [u32; 64] = [
        0xd76aa478, 0xe8c7b756, 0x242070db, 0xc1bdceee,
        0xf57c0faf, 0x4787c62a, 0xa8304613, 0xfd469501,
        0x698098d8, 0x8b44f7af, 0xffff5bb1, 0x895cd7be,
        0x6b901122, 0xfd987193, 0xa679438e, 0x49b40821,
        0xf61e2562, 0xc040b340, 0x265e5a51, 0xe9b6c7aa,
        0xd62f105d, 0x02441453, 0xd8a1e681, 0xe7d3fbc8,
        0x21e1cde6, 0xc33707d6, 0xf4d50d87, 0x455a14ed,
        0xa9e3e905, 0xfcefa3f8, 0x676f02d9, 0x8d2a4c8a,
        0xfffa3942, 0x8771f681, 0x6d9d6122, 0xfde5380c,
        0xa4beea44, 0x4bdecfa9, 0xf6bb4b60, 0xbebfbc70,
        0x289b7ec6, 0xeaa127fa, 0xd4ef3085, 0x04881d05,
        0xd9d4d039, 0xe6db99e5, 0x1fa27cf8, 0xc4ac5665,
        0xf4292244, 0x432aff97, 0xab9423a7, 0xfc93a039,
        0x655b59c3, 0x8f0ccc92, 0xffeff47d, 0x85845dd1,
        0x6fa87e4f, 0xfe2ce6e0, 0xa3014314, 0x4e0811a1,
        0xf7537e82, 0xbd3af235, 0x2ad7d2bb, 0xeb86d391,
    ];
    #[rustfmt::skip]
    const S: [u32; 64] = [
        7,12,17,22, 7,12,17,22, 7,12,17,22, 7,12,17,22,
        5, 9,14,20, 5, 9,14,20, 5, 9,14,20, 5, 9,14,20,
        4,11,16,23, 4,11,16,23, 4,11,16,23, 4,11,16,23,
        6,10,15,21, 6,10,15,21, 6,10,15,21, 6,10,15,21,
    ];

    let orig_len_bits = (data.len() as u64).wrapping_mul(8);
    let mut msg = data.to_vec();
    msg.push(0x80);
    while msg.len() % 64 != 56 {
        msg.push(0);
    }
    msg.extend_from_slice(&orig_len_bits.to_le_bytes());

    let mut a0: u32 = 0x67452301;
    let mut b0: u32 = 0xefcdab89;
    let mut c0: u32 = 0x98badcfe;
    let mut d0: u32 = 0x10325476;

    for chunk in msg.chunks(64) {
        let mut m = [0u32; 16];
        for (i, word) in m.iter_mut().enumerate() {
            *word = u32::from_le_bytes([
                chunk[i * 4],
                chunk[i * 4 + 1],
                chunk[i * 4 + 2],
                chunk[i * 4 + 3],
            ]);
        }
        let (mut a, mut b, mut c, mut d) = (a0, b0, c0, d0);
        for i in 0usize..64 {
            let (f, g) = match i {
                0..=15  => ((b & c) | (!b & d),           i),
                16..=31 => ((d & b) | (!d & c),           (5 * i + 1) % 16),
                32..=47 => (b ^ c ^ d,                    (3 * i + 5) % 16),
                _       => (c ^ (b | !d),                 (7 * i) % 16),
            };
            let temp = d;
            d = c;
            c = b;
            b = b.wrapping_add(
                (a.wrapping_add(f).wrapping_add(K[i]).wrapping_add(m[g]))
                    .rotate_left(S[i]),
            );
            a = temp;
        }
        a0 = a0.wrapping_add(a);
        b0 = b0.wrapping_add(b);
        c0 = c0.wrapping_add(c);
        d0 = d0.wrapping_add(d);
    }

    format!(
        "{:08x}{:08x}{:08x}{:08x}",
        a0.to_le(), b0.to_le(), c0.to_le(), d0.to_le()
    )
}

// ─────────────────────────────────────────────────────────────────────────────
// Parser helpers
// ─────────────────────────────────────────────────────────────────────────────

fn u24_to_usize(b: &[u8]) -> usize {
    ((b[0] as usize) << 16) | ((b[1] as usize) << 8) | (b[2] as usize)
}

fn parse_sni_extension(data: &[u8]) -> Option<String> {
    // ServerNameList length (2 bytes), then entries:
    //   name_type (1) + length (2) + name_bytes
    if data.len() < 5 {
        return None;
    }
    let _list_len = u16::from_be_bytes([data[0], data[1]]) as usize;
    let name_type = data[2];
    if name_type != 0 {
        return None; // 0 = host_name
    }
    let name_len = u16::from_be_bytes([data[3], data[4]]) as usize;
    if data.len() < 5 + name_len {
        return None;
    }
    String::from_utf8(data[5..5 + name_len].to_vec()).ok()
}

/// Parse a list of u16 values from `data` with a leading `prefix_size` length
/// field (in bytes, so divide by 2 to get count).
fn parse_u16_list(data: &[u8], prefix_size: usize) -> Vec<u16> {
    if data.len() < prefix_size {
        return Vec::new();
    }
    let list_len = match prefix_size {
        1 => data[0] as usize,
        2 => u16::from_be_bytes([data[0], data[1]]) as usize,
        _ => return Vec::new(),
    };
    let mut out = Vec::with_capacity(list_len / 2);
    let mut i = prefix_size;
    while i + 1 <= prefix_size + list_len && i + 1 < data.len() {
        out.push(u16::from_be_bytes([data[i], data[i + 1]]));
        i += 2;
    }
    out
}

/// GREASE values (RFC 8701) should be excluded from JA3 computation.
fn is_grease(v: u16) -> bool {
    matches!(
        v,
        0x0a0a | 0x1a1a | 0x2a2a | 0x3a3a | 0x4a4a | 0x5a5a | 0x6a6a
            | 0x7a7a | 0x8a8a | 0x9a9a | 0xaaaa | 0xbaba | 0xcaca | 0xdada
            | 0xeaea | 0xfafa
    )
}

// ─────────────────────────────────────────────────────────────────────────────
// Test helpers (packet builders)
// ─────────────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;

    // ── Packet builders ────────────────────────────────────────────────────

    fn build_tls_record(content_type: u8, major: u8, minor: u8, payload: &[u8]) -> Vec<u8> {
        let mut buf = vec![content_type, major, minor];
        buf.extend_from_slice(&(payload.len() as u16).to_be_bytes());
        buf.extend_from_slice(payload);
        buf
    }

    fn build_handshake(hs_type: u8, body: &[u8]) -> Vec<u8> {
        let len = body.len();
        let mut buf = vec![
            hs_type,
            (len >> 16) as u8,
            (len >> 8) as u8,
            len as u8,
        ];
        buf.extend_from_slice(body);
        buf
    }

    fn build_client_hello_body(extensions: &[u8], cipher_suites: &[u8]) -> Vec<u8> {
        let mut buf = Vec::new();
        buf.extend_from_slice(&[0x03, 0x03]); // ClientHello version TLS 1.2
        buf.extend_from_slice(&[0u8; 32]);    // Random
        buf.push(0);                           // Session ID length = 0
        buf.extend_from_slice(&(cipher_suites.len() as u16).to_be_bytes());
        buf.extend_from_slice(cipher_suites);
        buf.push(1); // compression methods: 1
        buf.push(0); // null compression
        buf.extend_from_slice(&(extensions.len() as u16).to_be_bytes());
        buf.extend_from_slice(extensions);
        buf
    }

    fn build_sni_extension(hostname: &[u8]) -> Vec<u8> {
        // name_type(1) + name_len(2) + name
        let mut sni_entry = vec![0x00]; // host_name
        sni_entry.extend_from_slice(&(hostname.len() as u16).to_be_bytes());
        sni_entry.extend_from_slice(hostname);
        // ServerNameList: list_len(2) + entries
        let mut sni_list = Vec::new();
        sni_list.extend_from_slice(&(sni_entry.len() as u16).to_be_bytes());
        sni_list.extend_from_slice(&sni_entry);
        // Extension header: type(2) + len(2) + data
        let mut ext = vec![0x00, 0x00]; // type = SNI
        ext.extend_from_slice(&(sni_list.len() as u16).to_be_bytes());
        ext.extend_from_slice(&sni_list);
        ext
    }

    fn make_client_hello_record(sni: &[u8], cipher_suites: &[u8]) -> Vec<u8> {
        let exts = build_sni_extension(sni);
        let ch = build_client_hello_body(&exts, cipher_suites);
        let hs = build_handshake(0x01, &ch);
        build_tls_record(0x16, 0x03, 0x01, &hs)
    }

    // ── Tests ──────────────────────────────────────────────────────────────

    #[test]
    fn is_tls_client_hello_valid() {
        let pkt = make_client_hello_record(b"example.com", &[0x00, 0x2f]);
        assert!(is_tls_client_hello(&pkt));
    }

    #[test]
    fn is_tls_client_hello_rejects_http() {
        assert!(!is_tls_client_hello(b"GET / HTTP/1.1\r\n"));
    }

    #[test]
    fn is_tls_client_hello_empty() {
        assert!(!is_tls_client_hello(&[]));
    }

    #[test]
    fn extract_sni_returns_hostname() {
        let pkt = make_client_hello_record(b"myserver.internal", &[0x00, 0x2f]);
        let sni = extract_sni(&pkt);
        assert_eq!(sni, Some("myserver.internal".to_string()));
    }

    #[test]
    fn extract_sni_no_sni_extension() {
        // Build a hello with no extensions
        let ch = build_client_hello_body(&[], &[0x00, 0x2f]);
        let hs = build_handshake(0x01, &ch);
        let pkt = build_tls_record(0x16, 0x03, 0x01, &hs);
        assert!(extract_sni(&pkt).is_none());
    }

    #[test]
    fn parse_client_hello_full() {
        let pkt = make_client_hello_record(b"api.example.com", &[0x00, 0x2f, 0x00, 0x35]);
        let hello = parse_client_hello(&pkt).expect("parse should succeed");
        assert_eq!(hello.sni, "api.example.com");
        assert_eq!(hello.cipher_suites.len(), 2);
        assert_eq!(hello.cipher_suites[0], 0x002f);
        assert_eq!(hello.cipher_suites[1], 0x0035);
    }

    #[test]
    fn parse_client_hello_version_field() {
        let pkt = make_client_hello_record(b"example.com", &[0x00, 0x2f]);
        let hello = parse_client_hello(&pkt).expect("should parse");
        assert_eq!(hello.version, "3.1"); // record layer TLS 1.0 = 3.1
        assert_eq!(hello.handshake_version, 0x0303); // ClientHello says TLS 1.2
    }

    #[test]
    fn parse_client_hello_invalid_input() {
        assert!(parse_client_hello(&[]).is_none());
        assert!(parse_client_hello(b"deadbeef").is_none());
    }

    #[test]
    fn parse_client_hello_wrong_content_type() {
        let mut pkt = make_client_hello_record(b"example.com", &[0x00, 0x2f]);
        pkt[0] = 0x17; // Application Data, not Handshake
        assert!(parse_client_hello(&pkt).is_none());
    }

    #[test]
    fn ja3_deterministic() {
        let pkt = make_client_hello_record(b"test.com", &[0x00, 0x2f, 0x00, 0x35]);
        let h1 = parse_client_hello(&pkt).unwrap();
        let h2 = parse_client_hello(&pkt).unwrap();
        assert_eq!(h1.ja3, h2.ja3, "JA3 must be deterministic");
    }

    #[test]
    fn ja3_is_32_char_hex() {
        let pkt = make_client_hello_record(b"test.com", &[0x00, 0x2f]);
        let hello = parse_client_hello(&pkt).unwrap();
        assert_eq!(hello.ja3.len(), 32, "JA3 should be 32 hex chars (MD5)");
        assert!(hello.ja3.chars().all(|c| c.is_ascii_hexdigit()));
    }

    #[test]
    fn compute_ja3_different_ciphers_different_hashes() {
        let p1 = make_client_hello_record(b"host.com", &[0x00, 0x2f]);
        let p2 = make_client_hello_record(b"host.com", &[0x00, 0x35]);
        let h1 = parse_client_hello(&p1).unwrap();
        let h2 = parse_client_hello(&p2).unwrap();
        assert_ne!(h1.ja3, h2.ja3, "Different cipher suites should yield different JA3");
    }

    #[test]
    fn md5_known_value() {
        // MD5("") = "d41d8cd98f00b204e9800998ecf8427e"
        let hash = md5_hex(b"");
        assert_eq!(hash, "d41d8cd98f00b204e9800998ecf8427e");
    }

    #[test]
    fn md5_abc() {
        // MD5("abc") = "900150983cd24fb0d6963f7d28e17f72"
        let hash = md5_hex(b"abc");
        assert_eq!(hash, "900150983cd24fb0d6963f7d28e17f72");
    }

    #[test]
    fn grease_values_filtered() {
        // Inject a GREASE cipher suite 0x0a0a and verify it's excluded from cipher_suites
        let grease_cs: &[u8] = &[0x0a, 0x0a, 0x00, 0x2f]; // GREASE + TLS_RSA_WITH_AES_128
        let pkt = make_client_hello_record(b"grease.test", grease_cs);
        let hello = parse_client_hello(&pkt).unwrap();
        assert!(
            !hello.cipher_suites.contains(&0x0a0a),
            "GREASE cipher should be filtered"
        );
        assert!(hello.cipher_suites.contains(&0x002f));
    }

    #[test]
    fn sni_unicode_invalid_utf8_returns_none() {
        // Build an extension with invalid UTF-8
        let bad_name: Vec<u8> = vec![0xff, 0xfe, 0x00];
        let exts = build_sni_extension(&bad_name);
        let ch = build_client_hello_body(&exts, &[0x00, 0x2f]);
        let hs = build_handshake(0x01, &ch);
        let pkt = build_tls_record(0x16, 0x03, 0x01, &hs);
        assert!(extract_sni(&pkt).is_none());
    }
}
