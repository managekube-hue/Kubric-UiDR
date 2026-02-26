// Package zeek parses Zeek (formerly Bro) TSV log files into structured Go
// types.
//
// Zeek is a BSD-licensed network analysis framework that produces tab-separated
// log files (conn.log, dns.log, http.log, ssl.log, files.log, etc.).  This
// package contains no imported Zeek code -- it only parses the public TSV log
// format whose schema is self-described in the file header.
//
// Zeek log header lines:
//
//	#separator \x09
//	#set_separator	,
//	#empty_field	(empty)
//	#unset_field	-
//	#path	conn
//	#fields	ts	uid	id.orig_h	id.orig_p	...
//	#types	time	string	addr	port	...
//
// Data lines are tab-separated values matching the #fields order.
package zeek

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// ConnLog represents a Zeek conn.log entry.
type ConnLog struct {
	TS          time.Time `json:"ts"`
	UID         string    `json:"uid"`
	OrigH       string    `json:"id.orig_h"`
	OrigP       int       `json:"id.orig_p"`
	RespH       string    `json:"id.resp_h"`
	RespP       int       `json:"id.resp_p"`
	Proto       string    `json:"proto"` // tcp, udp, icmp
	Service     string    `json:"service"`
	Duration    float64   `json:"duration"`
	OrigBytes   int64     `json:"orig_bytes"`
	RespBytes   int64     `json:"resp_bytes"`
	ConnState   string    `json:"conn_state"` // S0, S1, SF, REJ, S2, S3, RSTO, RSTR, RSTOS0, etc.
	LocalOrig   bool      `json:"local_orig"`
	LocalResp   bool      `json:"local_resp"`
	MissedBytes int64     `json:"missed_bytes"`
	History     string    `json:"history"`
	OrigPkts    int64     `json:"orig_pkts"`
	OrigIPBytes int64     `json:"orig_ip_bytes"`
	RespPkts    int64     `json:"resp_pkts"`
	RespIPBytes int64     `json:"resp_ip_bytes"`
	CommunityID string   `json:"community_id,omitempty"`
}

// DNSLog represents a Zeek dns.log entry.
type DNSLog struct {
	TS         time.Time `json:"ts"`
	UID        string    `json:"uid"`
	OrigH      string    `json:"id.orig_h"`
	OrigP      int       `json:"id.orig_p"`
	RespH      string    `json:"id.resp_h"`
	RespP      int       `json:"id.resp_p"`
	Proto      string    `json:"proto"`
	TransID    int       `json:"trans_id"`
	RTT        float64   `json:"rtt"`
	Query      string    `json:"query"`
	QClass     int       `json:"qclass"`
	QClassName string    `json:"qclass_name"`
	QType      int       `json:"qtype"`
	QTypeName  string    `json:"qtype_name"`
	RCode      int       `json:"rcode"`
	RCodeName  string    `json:"rcode_name"`
	AA         bool      `json:"AA"`
	TC         bool      `json:"TC"`
	RD         bool      `json:"RD"`
	RA         bool      `json:"RA"`
	Answers    []string  `json:"answers"`
	TTLs       []float64 `json:"TTLs"`
}

// HTTPLog represents a Zeek http.log entry.
type HTTPLog struct {
	TS              time.Time `json:"ts"`
	UID             string    `json:"uid"`
	OrigH           string    `json:"id.orig_h"`
	OrigP           int       `json:"id.orig_p"`
	RespH           string    `json:"id.resp_h"`
	RespP           int       `json:"id.resp_p"`
	TransDepth      int       `json:"trans_depth"`
	Method          string    `json:"method"`
	Host            string    `json:"host"`
	URI             string    `json:"uri"`
	Referrer        string    `json:"referrer"`
	Version         string    `json:"version"`
	UserAgent       string    `json:"user_agent"`
	Origin          string    `json:"origin"`
	RequestBodyLen  int64     `json:"request_body_len"`
	ResponseBodyLen int64     `json:"response_body_len"`
	StatusCode      int       `json:"status_code"`
	StatusMsg       string    `json:"status_msg"`
	Tags            []string  `json:"tags"`
	MIMETypes       []string  `json:"resp_mime_types"`
}

// SSLLog represents a Zeek ssl.log entry.
type SSLLog struct {
	TS             time.Time `json:"ts"`
	UID            string    `json:"uid"`
	OrigH          string    `json:"id.orig_h"`
	OrigP          int       `json:"id.orig_p"`
	RespH          string    `json:"id.resp_h"`
	RespP          int       `json:"id.resp_p"`
	Version        string    `json:"version"`
	Cipher         string    `json:"cipher"`
	Curve          string    `json:"curve"`
	ServerName     string    `json:"server_name"`
	Resumed        bool      `json:"resumed"`
	Established    bool      `json:"established"`
	Subject        string    `json:"subject"`
	Issuer         string    `json:"issuer"`
	NotValidBefore string    `json:"not_valid_before"`
	NotValidAfter  string    `json:"not_valid_after"`
	JA3            string    `json:"ja3"`
	JA3S           string    `json:"ja3s"`
}

// LogHeader stores Zeek log metadata parsed from the header lines
// (#separator, #set_separator, #empty_field, #unset_field, #path,
// #fields, #types).
type LogHeader struct {
	Separator    string
	SetSeparator string
	EmptyField   string
	UnsetField   string
	Fields       []string
	Types        []string
	Path         string // e.g. "conn", "dns", "http", "ssl"
}

// Parser reads Zeek TSV log files using the self-describing header.
type Parser struct {
	header LogHeader
}

// ---------------------------------------------------------------------------
// Header parsing
// ---------------------------------------------------------------------------

// ParseHeader reads the Zeek log header from a buffered reader.  It consumes
// all lines beginning with '#' and returns the parsed LogHeader.  The reader
// is left positioned at the first data line.
func ParseHeader(reader *bufio.Reader) (*LogHeader, error) {
	if reader == nil {
		return nil, fmt.Errorf("zeek: reader must not be nil")
	}

	h := &LogHeader{
		Separator:    "\t",
		SetSeparator: ",",
		EmptyField:   "(empty)",
		UnsetField:   "-",
	}

	for {
		// Peek to check if the next line starts with '#'.
		peek, err := reader.Peek(1)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("zeek: peek header: %w", err)
		}
		if peek[0] != '#' {
			break // reached data lines
		}

		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("zeek: read header line: %w", err)
		}
		line = strings.TrimRight(line, "\r\n")

		if strings.HasPrefix(line, "#close") {
			// #close is the log trailer; stop here.
			break
		}

		parts := strings.SplitN(line, "\t", 2)
		if len(parts) < 1 {
			continue
		}

		directive := parts[0]
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}

		switch directive {
		case "#separator":
			// The separator value is often written as a literal like "\x09".
			h.Separator = unescapeZeekSeparator(value)
		case "#set_separator":
			h.SetSeparator = value
		case "#empty_field":
			h.EmptyField = value
		case "#unset_field":
			h.UnsetField = value
		case "#path":
			h.Path = value
		case "#fields":
			h.Fields = strings.Split(value, "\t")
		case "#types":
			h.Types = strings.Split(value, "\t")
		}

		if err == io.EOF {
			break
		}
	}

	if len(h.Fields) == 0 {
		return nil, fmt.Errorf("zeek: no #fields directive found in header")
	}

	return h, nil
}

// unescapeZeekSeparator converts the Zeek #separator value (e.g. "\\x09")
// into the actual rune.
func unescapeZeekSeparator(s string) string {
	s = strings.TrimSpace(s)
	// Zeek writes the separator as a literal like \x09 (without surrounding
	// quotes).  Go's strconv.Unquote expects surrounding quotes.
	unquoted, err := strconv.Unquote(`"` + s + `"`)
	if err != nil {
		// Fall back to the raw value.
		return s
	}
	return unquoted
}

// ---------------------------------------------------------------------------
// Generic parser
// ---------------------------------------------------------------------------

// ParseGeneric reads any Zeek log and returns the header together with each
// row represented as a map[string]string keyed by field name.  Unset and
// empty sentinel values are normalised to Go zero values (empty string).
func ParseGeneric(path string) (*LogHeader, []map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("zeek: open %s: %w", path, err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	header, err := ParseHeader(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("zeek: parse header %s: %w", path, err)
	}

	var rows []map[string]string
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue // skip blank lines and closing header
		}

		fields := strings.Split(line, header.Separator)
		row := make(map[string]string, len(header.Fields))
		for i, name := range header.Fields {
			if i < len(fields) {
				val := fields[i]
				// Normalise sentinel values to empty string.
				if val == header.UnsetField || val == header.EmptyField {
					val = ""
				}
				row[name] = val
			} else {
				row[name] = ""
			}
		}
		rows = append(rows, row)
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("zeek: scan %s: %w", path, err)
	}

	return header, rows, nil
}

// ---------------------------------------------------------------------------
// Typed parsers
// ---------------------------------------------------------------------------

// ParseConnLog reads a Zeek conn.log file and returns parsed entries.
func ParseConnLog(path string) ([]ConnLog, error) {
	header, rows, err := ParseGeneric(path)
	if err != nil {
		return nil, fmt.Errorf("zeek: parse conn.log: %w", err)
	}
	_ = header

	entries := make([]ConnLog, 0, len(rows))
	for i, row := range rows {
		ts, err := parseZeekTimestamp(row["ts"])
		if err != nil {
			return nil, fmt.Errorf("zeek: conn.log row %d: parse timestamp: %w", i, err)
		}
		entry := ConnLog{
			TS:          ts,
			UID:         row["uid"],
			OrigH:       row["id.orig_h"],
			OrigP:       safeAtoi(row["id.orig_p"]),
			RespH:       row["id.resp_h"],
			RespP:       safeAtoi(row["id.resp_p"]),
			Proto:       row["proto"],
			Service:     row["service"],
			Duration:    safeParseFloat(row["duration"]),
			OrigBytes:   safeAtoi64(row["orig_bytes"]),
			RespBytes:   safeAtoi64(row["resp_bytes"]),
			ConnState:   row["conn_state"],
			LocalOrig:   row["local_orig"] == "T",
			LocalResp:   row["local_resp"] == "T",
			MissedBytes: safeAtoi64(row["missed_bytes"]),
			History:     row["history"],
			OrigPkts:    safeAtoi64(row["orig_pkts"]),
			OrigIPBytes: safeAtoi64(row["orig_ip_bytes"]),
			RespPkts:    safeAtoi64(row["resp_pkts"]),
			RespIPBytes: safeAtoi64(row["resp_ip_bytes"]),
			CommunityID: row["community_id"],
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ParseDNSLog reads a Zeek dns.log file and returns parsed entries.
func ParseDNSLog(path string) ([]DNSLog, error) {
	header, rows, err := ParseGeneric(path)
	if err != nil {
		return nil, fmt.Errorf("zeek: parse dns.log: %w", err)
	}

	entries := make([]DNSLog, 0, len(rows))
	for i, row := range rows {
		ts, err := parseZeekTimestamp(row["ts"])
		if err != nil {
			return nil, fmt.Errorf("zeek: dns.log row %d: parse timestamp: %w", i, err)
		}
		entry := DNSLog{
			TS:         ts,
			UID:        row["uid"],
			OrigH:      row["id.orig_h"],
			OrigP:      safeAtoi(row["id.orig_p"]),
			RespH:      row["id.resp_h"],
			RespP:      safeAtoi(row["id.resp_p"]),
			Proto:      row["proto"],
			TransID:    safeAtoi(row["trans_id"]),
			RTT:        safeParseFloat(row["rtt"]),
			Query:      row["query"],
			QClass:     safeAtoi(row["qclass"]),
			QClassName: row["qclass_name"],
			QType:      safeAtoi(row["qtype"]),
			QTypeName:  row["qtype_name"],
			RCode:      safeAtoi(row["rcode"]),
			RCodeName:  row["rcode_name"],
			AA:         row["AA"] == "T",
			TC:         row["TC"] == "T",
			RD:         row["RD"] == "T",
			RA:         row["RA"] == "T",
			Answers:    parseZeekSet(row["answers"], header.SetSeparator),
			TTLs:       parseZeekFloatSet(row["TTLs"], header.SetSeparator),
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ParseHTTPLog reads a Zeek http.log file and returns parsed entries.
func ParseHTTPLog(path string) ([]HTTPLog, error) {
	header, rows, err := ParseGeneric(path)
	if err != nil {
		return nil, fmt.Errorf("zeek: parse http.log: %w", err)
	}

	entries := make([]HTTPLog, 0, len(rows))
	for i, row := range rows {
		ts, err := parseZeekTimestamp(row["ts"])
		if err != nil {
			return nil, fmt.Errorf("zeek: http.log row %d: parse timestamp: %w", i, err)
		}
		entry := HTTPLog{
			TS:              ts,
			UID:             row["uid"],
			OrigH:           row["id.orig_h"],
			OrigP:           safeAtoi(row["id.orig_p"]),
			RespH:           row["id.resp_h"],
			RespP:           safeAtoi(row["id.resp_p"]),
			TransDepth:      safeAtoi(row["trans_depth"]),
			Method:          row["method"],
			Host:            row["host"],
			URI:             row["uri"],
			Referrer:        row["referrer"],
			Version:         row["version"],
			UserAgent:       row["user_agent"],
			Origin:          row["origin"],
			RequestBodyLen:  safeAtoi64(row["request_body_len"]),
			ResponseBodyLen: safeAtoi64(row["response_body_len"]),
			StatusCode:      safeAtoi(row["status_code"]),
			StatusMsg:       row["status_msg"],
			Tags:            parseZeekSet(row["tags"], header.SetSeparator),
			MIMETypes:       parseZeekSet(row["resp_mime_types"], header.SetSeparator),
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ParseSSLLog reads a Zeek ssl.log file and returns parsed entries.
func ParseSSLLog(path string) ([]SSLLog, error) {
	_, rows, err := ParseGeneric(path)
	if err != nil {
		return nil, fmt.Errorf("zeek: parse ssl.log: %w", err)
	}

	entries := make([]SSLLog, 0, len(rows))
	for i, row := range rows {
		ts, err := parseZeekTimestamp(row["ts"])
		if err != nil {
			return nil, fmt.Errorf("zeek: ssl.log row %d: parse timestamp: %w", i, err)
		}
		entry := SSLLog{
			TS:             ts,
			UID:            row["uid"],
			OrigH:          row["id.orig_h"],
			OrigP:          safeAtoi(row["id.orig_p"]),
			RespH:          row["id.resp_h"],
			RespP:          safeAtoi(row["id.resp_p"]),
			Version:        row["version"],
			Cipher:         row["cipher"],
			Curve:          row["curve"],
			ServerName:     row["server_name"],
			Resumed:        row["resumed"] == "T",
			Established:    row["established"] == "T",
			Subject:        row["subject"],
			Issuer:         row["issuer"],
			NotValidBefore: row["not_valid_before"],
			NotValidAfter:  row["not_valid_after"],
			JA3:            row["ja3"],
			JA3S:           row["ja3s"],
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseZeekTimestamp converts a Zeek epoch timestamp string like
// "1234567890.123456" into a time.Time.  Zeek timestamps are always Unix
// epoch seconds with microsecond precision.
func parseZeekTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	// Try parsing as a float (the standard Zeek format).
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("zeek: parse timestamp %q: %w", s, err)
	}

	sec := int64(f)
	nsec := int64((f - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).UTC(), nil
}

// parseZeekSet splits a Zeek set/vector value (e.g. "a,b,c") by the set
// separator.  Returns nil for empty or unset values.
func parseZeekSet(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// parseZeekFloatSet splits a Zeek set of floats (e.g. TTLs) and parses each.
func parseZeekFloatSet(s, sep string) []float64 {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, sep)
	out := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		f, err := strconv.ParseFloat(p, 64)
		if err != nil {
			continue
		}
		out = append(out, f)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// safeAtoi parses a string to int, returning 0 on failure (unset/empty fields).
func safeAtoi(s string) int {
	if s == "" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}

// safeAtoi64 parses a string to int64, returning 0 on failure.
func safeAtoi64(s string) int64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// safeParseFloat parses a string to float64, returning 0 on failure.
func safeParseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
