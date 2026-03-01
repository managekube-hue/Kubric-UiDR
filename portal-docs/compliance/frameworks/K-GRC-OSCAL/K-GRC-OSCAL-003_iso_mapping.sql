-- K-GRC-OSCAL-003_iso_mapping.sql
-- PostgreSQL DDL: ISO 27001:2022 → NIST 800-53 control mapping tables
-- and OSCAL controls table (shared by all frameworks).

-- ============================================================
-- OSCAL Controls — stores controls from any NIST-style catalog
-- ============================================================
CREATE TABLE IF NOT EXISTS oscal_controls (
    control_id   TEXT        NOT NULL PRIMARY KEY,        -- e.g. "AC-2", "AU-6"
    catalog      TEXT        NOT NULL DEFAULT 'NIST SP 800-53 Rev 5',
    family       TEXT        NOT NULL,
    title        TEXT        NOT NULL,
    parameters   JSONB       NOT NULL DEFAULT '{}',
    guidance     TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oscal_controls_family   ON oscal_controls(family);
CREATE INDEX IF NOT EXISTS idx_oscal_controls_catalog  ON oscal_controls(catalog);

-- ============================================================
-- ISO 27001:2022 → NIST 800-53 Mapping
-- id SERIAL matches spec requirement
-- ============================================================
CREATE TABLE IF NOT EXISTS iso_nist_mapping (
    id                SERIAL  PRIMARY KEY,
    iso_control       TEXT    NOT NULL,               -- e.g. "5.1", "8.8"
    iso_title         TEXT    NOT NULL,
    nist_control      TEXT    NOT NULL,               -- e.g. "PL-1", "RA-5"
    nist_title        TEXT    NOT NULL,
    mapping_confidence TEXT   NOT NULL
        CHECK (mapping_confidence IN ('direct', 'partial', 'related')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_iso_nist_iso        ON iso_nist_mapping(iso_control);
CREATE INDEX IF NOT EXISTS idx_iso_nist_nist       ON iso_nist_mapping(nist_control);
CREATE INDEX IF NOT EXISTS idx_iso_nist_confidence ON iso_nist_mapping(mapping_confidence);

-- ============================================================
-- Seed NIST controls referenced in the mapping below (idempotent)
-- Run this before inserting iso_nist_mapping rows.
-- ============================================================
INSERT INTO oscal_controls (control_id, catalog, family, title, guidance) VALUES
    ('PL-1',  'NIST SP 800-53 Rev 5', 'Planning',                          'Policy and Procedures',                               ''),
    ('RA-3',  'NIST SP 800-53 Rev 5', 'Risk Assessment',                   'Risk Assessment',                                     ''),
    ('IA-5',  'NIST SP 800-53 Rev 5', 'Identification and Authentication', 'Authenticator Management',                            ''),
    ('SI-3',  'NIST SP 800-53 Rev 5', 'System and Information Integrity',  'Malicious Code Protection',                           ''),
    ('RA-5',  'NIST SP 800-53 Rev 5', 'Risk Assessment',                   'Vulnerability Monitoring and Scanning',               ''),
    ('CM-2',  'NIST SP 800-53 Rev 5', 'Configuration Management',          'Baseline Configuration',                              ''),
    ('SC-28', 'NIST SP 800-53 Rev 5', 'System and Communications Protection', 'Protection of Information at Rest',                ''),
    ('AU-2',  'NIST SP 800-53 Rev 5', 'Audit and Accountability',          'Event Logging',                                       ''),
    ('SI-4',  'NIST SP 800-53 Rev 5', 'System and Information Integrity',  'System Monitoring',                                   ''),
    ('SC-7',  'NIST SP 800-53 Rev 5', 'System and Communications Protection', 'Boundary Protection',                             ''),
    ('SC-8',  'NIST SP 800-53 Rev 5', 'System and Communications Protection', 'Transmission Confidentiality and Integrity',      ''),
    ('SA-11', 'NIST SP 800-53 Rev 5', 'System and Services Acquisition',   'Developer Testing and Evaluation',                    '')
ON CONFLICT (control_id) DO NOTHING;

-- ============================================================
-- Seed: 15 ISO 27001:2022 → NIST 800-53 Rev 5 mappings (exactly as spec)
-- ============================================================
INSERT INTO iso_nist_mapping (iso_control, iso_title, nist_control, nist_title, mapping_confidence) VALUES
    ('5.1',    'Policies for information security',
               'PL-1',   'Policy and Procedures',                           'direct'),
    ('6.1',    'Actions to address risks and opportunities',
               'RA-3',   'Risk Assessment',                                  'direct'),
    ('8.2',    'Information security risk assessment',
               'RA-3',   'Risk Assessment',                                  'direct'),
    ('8.5.1',  'Secure authentication',
               'IA-5',   'Authenticator Management',                         'direct'),
    ('8.7',    'Protection against malware',
               'SI-3',   'Malicious Code Protection',                        'direct'),
    ('8.8',    'Management of technical vulnerabilities',
               'RA-5',   'Vulnerability Monitoring and Scanning',            'direct'),
    ('8.9',    'Configuration management',
               'CM-2',   'Baseline Configuration',                           'direct'),
    ('8.12',   'Data leakage prevention',
               'SC-28',  'Protection of Information at Rest',                'direct'),
    ('8.15',   'Logging',
               'AU-2',   'Event Logging',                                    'direct'),
    ('8.16',   'Monitoring activities',
               'SI-4',   'System Monitoring',                                'direct'),
    ('8.19',   'Installation of software on operational systems',
               'SC-7',   'Boundary Protection',                              'direct'),
    ('8.24',   'Use of cryptography',
               'SC-8',   'Transmission Confidentiality and Integrity',       'direct'),
    ('8.28',   'Secure coding',
               'SA-11',  'Developer Testing and Evaluation',                 'direct'),
    ('6.1.2',  'Information security risk assessment (detailed process)',
               'RA-5',   'Vulnerability Monitoring and Scanning',            'partial'),
    ('A.14.1', 'Security requirements of information systems',
               'SC-8',   'Transmission Confidentiality and Integrity',       'related')
ON CONFLICT DO NOTHING;

COMMENT ON TABLE iso_nist_mapping IS
    'ISO 27001:2022 to NIST SP 800-53 Rev 5 control crosswalk. '
    'Populated by K-GRC-OSCAL-003. Used for compliance gap analysis.';

COMMENT ON TABLE oscal_controls IS
    'NIST SP 800-53 controls imported from OSCAL catalog via nist_ingest.py. '
    'Also used as FK target for iso_nist_mapping and soc2 assessment queries.';
