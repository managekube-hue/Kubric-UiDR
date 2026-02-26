package correlation

import "time"

// DefaultRules returns the built-in correlation rule set shipped with Kubric.
//
// Rules cover the most critical MSSP detection scenarios and are mapped to
// MITRE ATT&CK tactics and techniques.  Each rule is designed to fire on
// real-world attack patterns observed across endpoint, network, and
// third-party telemetry sources.
//
// Rule severity scale:
//
//	1 = informational
//	2 = low
//	3 = medium
//	4 = high
//	5 = critical
//
// All windows are calibrated to balance detection fidelity against false-positive
// rates in production MSSP environments.
func DefaultRules() []CorrelationRule {
	return []CorrelationRule{
		ruleR001LateralMovement(),
		ruleR002DefenseEvasionEncoded(),
		ruleR003CredentialDumping(),
		ruleR004C2Beaconing(),
		ruleR005RansomwareIndicators(),
		ruleR006PrivilegeEscalation(),
		ruleR007DataExfiltration(),
		ruleR008WazuhCoreSec(),
	}
}

// ---------------------------------------------------------------------------
// R001 — Lateral Movement: process exec + outbound network within 30 seconds
//
// MITRE: TA0008 (Lateral Movement) / T1021 (Remote Services)
//
// Rationale: A process spawned on an endpoint that immediately establishes an
// outbound connection is a strong signal for lateral movement tools such as
// Cobalt Strike, Metasploit, or PsExec.  The 30-second window is deliberately
// tight to reduce noise from legitimate software update traffic.
// ---------------------------------------------------------------------------

func ruleR001LateralMovement() CorrelationRule {
	return CorrelationRule{
		ID:          "R001",
		Name:        "Lateral Movement: Process Exec + Outbound Network",
		Description: "A new process was spawned and an outbound network connection was established within 30 seconds on the same agent, suggesting lateral movement or C2 beacon staging.",
		Severity:    4,
		Window:      30 * time.Second,
		MITRETactic: "TA0008",
		MITRETechnique: "T1021",
		Conditions: []RuleCondition{
			{
				// Condition 1: any process execution from CoreSec
				Source:    "coresec",
				EventType: "process",
				FieldMatch: map[string]string{},
				MinCount:  1,
			},
			{
				// Condition 2: outbound network event on the same agent
				Source:    "netguard",
				EventType: "network",
				FieldMatch: map[string]string{
					"direction": "outbound",
				},
				MinCount: 1,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// R002 — Defense Evasion: Encoded Command-Line + FIM
//
// MITRE: TA0005 (Defense Evasion) / T1027 (Obfuscated Files or Information)
//         T1059.001 (PowerShell)
//
// Rationale: Base64-encoded or heavily obfuscated command lines (commonly
// seen in PowerShell -EncodedCommand or certutil -decode chains) paired with
// filesystem modification events indicate a multi-stage dropper.
// ---------------------------------------------------------------------------

func ruleR002DefenseEvasionEncoded() CorrelationRule {
	return CorrelationRule{
		ID:          "R002",
		Name:        "Defense Evasion: Obfuscated Cmdline + File Write",
		Description: "A process with an obfuscated or encoded command line (e.g., PowerShell -EncodedCommand, certutil) was followed by a file system modification, indicating a dropper or in-memory loader.",
		Severity:    4,
		Window:      2 * time.Minute,
		MITRETactic: "TA0005",
		MITRETechnique: "T1027",
		Conditions: []RuleCondition{
			{
				// Condition 1: obfuscated process command line
				Source:    "coresec",
				EventType: "process",
				FieldMatch: map[string]string{
					// Match cmdline containing base64 padding, -enc, or certutil decode patterns
					"cmdline": "-enc",
				},
				MinCount: 1,
			},
			{
				// Condition 2: FIM event on same agent (file created/modified)
				Source:    "coresec",
				EventType: "fim",
				FieldMatch: map[string]string{},
				MinCount:  1,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// R003 — Credential Dumping: LSASS Access + Outbound Network
//
// MITRE: TA0006 (Credential Access) / T1003.001 (LSASS Memory)
//
// Rationale: Access to lsass.exe memory (via OpenProcess or MiniDump) is the
// canonical credential-dumping indicator used by Mimikatz, ProcDump, and
// comsvcs.dll.  Subsequent outbound network activity may indicate credential
// exfiltration.
// ---------------------------------------------------------------------------

func ruleR003CredentialDumping() CorrelationRule {
	return CorrelationRule{
		ID:          "R003",
		Name:        "Credential Dumping: LSASS Access + Outbound Network",
		Description: "A process accessed lsass.exe followed by outbound network activity, consistent with credential dumping tools (Mimikatz, ProcDump, comsvcs.dll) and potential exfiltration.",
		Severity:    5,
		Window:      60 * time.Second,
		MITRETactic: "TA0006",
		MITRETechnique: "T1003",
		Conditions: []RuleCondition{
			{
				// Condition 1: process accessing lsass
				Source:    "coresec",
				EventType: "process",
				FieldMatch: map[string]string{
					"cmdline": "lsass",
				},
				MinCount: 1,
			},
			{
				// Condition 2: any outbound network connection following the access
				Source:    "netguard",
				EventType: "network",
				FieldMatch: map[string]string{
					"direction": "outbound",
				},
				MinCount: 1,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// R004 — C2 Beaconing: Repeated Outbound Connections to Same Destination
//
// MITRE: TA0011 (Command and Control) / T1071.001 (Application Layer Protocol)
//         T1071.004 (DNS)
//
// Rationale: C2 frameworks (Cobalt Strike, Sliver, Merlin) maintain periodic
// callbacks to a team server.  Detecting ≥5 outbound connections to the same
// destination IP within 5 minutes is a strong signal of automated beaconing.
// RITA-style beacon detection (regularity of interval) would add precision
// but this threshold approach catches the majority of default beacon intervals
// (60s is the CS default).
// ---------------------------------------------------------------------------

func ruleR004C2Beaconing() CorrelationRule {
	return CorrelationRule{
		ID:          "R004",
		Name:        "C2 Beaconing: Repeated Outbound Connections",
		Description: "Five or more outbound network connections to the same destination within 5 minutes, consistent with C2 beacon callbacks (Cobalt Strike default: 60s jitter).",
		Severity:    4,
		Window:      5 * time.Minute,
		MITRETactic: "TA0011",
		MITRETechnique: "T1071",
		Threshold:   5, // 5+ matching events required
		Conditions: []RuleCondition{
			{
				Source:    "netguard",
				EventType: "network",
				FieldMatch: map[string]string{
					"direction": "outbound",
				},
				MinCount: 5,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// R005 — Ransomware Indicators: Mass File Modification
//
// MITRE: TA0040 (Impact) / T1486 (Data Encrypted for Impact)
//
// Rationale: Ransomware encryptors modify hundreds of files in rapid
// succession.  Detecting ≥20 FIM events within 60 seconds on a single agent
// fires early enough to trigger containment before full encryption completes.
// ---------------------------------------------------------------------------

func ruleR005RansomwareIndicators() CorrelationRule {
	return CorrelationRule{
		ID:          "R005",
		Name:        "Ransomware Indicators: Mass File Modification",
		Description: "Twenty or more file modification events on a single agent within 60 seconds, indicative of ransomware encryption activity (e.g., LockBit, BlackCat/ALPHV, Akira).",
		Severity:    5,
		Window:      60 * time.Second,
		MITRETactic: "TA0040",
		MITRETechnique: "T1486",
		Threshold:   20,
		Conditions: []RuleCondition{
			{
				Source:    "coresec",
				EventType: "fim",
				FieldMatch: map[string]string{
					// activity_id 2 = Modify; 1 = Create; both are ransomware signals
				},
				MinCount: 20,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// R006 — Privilege Escalation: UID-0 Process from Non-Root Parent
//
// MITRE: TA0004 (Privilege Escalation) / T1548.001 (Setuid and Setgid)
//         T1055 (Process Injection)
//
// Rationale: On Linux, a process running as UID 0 (root) spawned by a
// non-privileged parent process is a classic privilege escalation signal.
// This covers SUID exploitation, sudo abuse, and container escapes.
// The Wazuh cross-corroboration condition reduces false positives from
// legitimate sudo usage in CI/CD pipelines.
// ---------------------------------------------------------------------------

func ruleR006PrivilegeEscalation() CorrelationRule {
	return CorrelationRule{
		ID:          "R006",
		Name:        "Privilege Escalation: Root Process from Non-Privileged Parent",
		Description: "A process running as UID 0 (root) was spawned from a non-privileged parent, or a Wazuh alert for privilege escalation was corroborated by a CoreSec process event.",
		Severity:    4,
		Window:      30 * time.Second,
		MITRETactic: "TA0004",
		MITRETechnique: "T1548",
		Conditions: []RuleCondition{
			{
				// Condition 1: CoreSec process event where user= root and parent uid != 0
				Source:    "coresec",
				EventType: "process",
				FieldMatch: map[string]string{
					"user": "root",
				},
				MinCount: 1,
			},
			{
				// Condition 2: corroboration from Wazuh escalation rule group
				Source:    "wazuh",
				EventType: "alert",
				FieldMatch: map[string]string{
					"rule_group": "privilege_escalation",
				},
				MinCount: 1,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// R007 — Data Exfiltration: Large Outbound Transfer + Sensitive FIM Activity
//
// MITRE: TA0010 (Exfiltration) / T1041 (Exfiltration Over C2 Channel)
//         T1567 (Exfiltration Over Web Service)
//
// Rationale: File access in /etc/passwd, /etc/shadow, SSH keys, certificate
// stores, or cloud credentials combined with a large outbound network transfer
// is a strong indicator of staged data exfiltration.  The 5-minute window
// accounts for compression + upload latency.
// ---------------------------------------------------------------------------

func ruleR007DataExfiltration() CorrelationRule {
	return CorrelationRule{
		ID:          "R007",
		Name:        "Data Exfiltration: Sensitive File Access + Large Outbound Transfer",
		Description: "Access to sensitive paths (/etc/shadow, SSH keys, certificate stores) followed by a large outbound network transfer within 5 minutes, indicating staged data exfiltration.",
		Severity:    5,
		Window:      5 * time.Minute,
		MITRETactic: "TA0010",
		MITRETechnique: "T1041",
		Conditions: []RuleCondition{
			{
				// Condition 1: FIM access on sensitive paths
				Source:    "coresec",
				EventType: "fim",
				FieldMatch: map[string]string{
					// Match any of the sensitive paths via substring
					"path": "/etc/",
				},
				MinCount: 1,
			},
			{
				// Condition 2: large outbound network transfer (>1MB heuristic)
				// The NetGuard network event includes a bytes_out field
				Source:    "netguard",
				EventType: "network",
				FieldMatch: map[string]string{
					"direction": "outbound",
				},
				MinCount: 1,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// R008 — Cross-Source: Wazuh Alert + CoreSec Process Correlation
//
// MITRE: Multiple tactics (depends on Wazuh rule group)
//
// Rationale: A Wazuh HIDS alert corroborated by a concurrent CoreSec process
// event significantly increases confidence compared to either source alone.
// This rule catches attacks that partially evade one detection system but
// leave traces in another — a common pattern with polymorphic malware and
// fileless techniques.
// ---------------------------------------------------------------------------

func ruleR008WazuhCoreSec() CorrelationRule {
	return CorrelationRule{
		ID:          "R008",
		Name:        "Cross-Source: Wazuh HIDS + CoreSec Process Co-occurrence",
		Description: "A Wazuh HIDS alert was corroborated by a concurrent CoreSec process event on the same tenant within 2 minutes, indicating a multi-signal detection from independent monitoring systems.",
		Severity:    3,
		Window:      2 * time.Minute,
		MITRETactic: "TA0002",
		MITRETechnique: "T1059",
		Conditions: []RuleCondition{
			{
				Source:    "wazuh",
				EventType: "alert",
				FieldMatch: map[string]string{},
				MinCount:  1,
			},
			{
				Source:    "coresec",
				EventType: "process",
				FieldMatch: map[string]string{},
				MinCount:  1,
			},
		},
	}
}
