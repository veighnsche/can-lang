package check

import (
	"errors"
	"fmt"
	"github.com/veighnsche/can-lang/compiler/internal/source"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ConnectionSetting contains literal evidence supplied by the native declaration
// checker. Expressions and environment values cannot enter this static schema.
// The syntax wiring belongs to I16; this policy is shared by all native forms.
type ConnectionSetting struct {
	Invalid             error
	File                string
	NameSpan, ValueSpan source.Span
	Name                string
	Text                *string
	Integer             *big.Int
	Entries             []ConnectionSetting
}
type ConnectionPolicy struct {
	Endpoint, BearerEnvironment, Protocol, Model       string
	TimeoutMilliseconds, MaxBodyBytes, MaxOutputTokens int64
	Headers                                            []ConnectionHeader
}
type ConnectionHeader struct{ Name, Value string }

func CheckConnection(settings []ConnectionSetting) (ConnectionPolicy, error) {
	p := ConnectionPolicy{MaxBodyBytes: 8388608, MaxOutputTokens: 2048}
	seen := map[string]ConnectionSetting{}
	var problems []error
	var origin ConnectionSetting
	fail := func(message string) error {
		return source.Locate(origin.File, origin.ValueSpan, fmt.Errorf("connection: %s", message))
	}
	text := func(s ConnectionSetting) (string, bool) {
		returnValue := ""
		if s.Text != nil {
			returnValue = *s.Text
		}
		return returnValue, s.Text != nil && s.Integer == nil && len(s.Entries) == 0 && utf8.ValidString(returnValue)
	}
	integer := func(s ConnectionSetting, max int64) (int64, bool) {
		if s.Integer == nil || s.Text != nil || len(s.Entries) != 0 || !s.Integer.IsInt64() {
			return 0, false
		}
		n := s.Integer.Int64()
		return n, n >= 1 && n <= max
	}
	for _, s := range settings {
		origin = s
		validate := func() error {
			if prior, exists := seen[s.Name]; exists {
				origin.ValueSpan = s.NameSpan
				return source.Relate(prior.File, prior.NameSpan, "first setting declared here", fail("duplicate setting "+s.Name))
			}
			seen[s.Name] = s
			if s.Invalid != nil {
				return s.Invalid
			}
			switch s.Name {
			case "endpoint":
				v, ok := text(s)
				if !ok || v == "" {
					return fail("endpoint requires a nonempty string literal")
				}
				u, err := url.Parse(v)
				if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || strings.ContainsAny(v, "?#") {
					return fail("invalid endpoint")
				}
				if u.Hostname() == "" {
					return fail("invalid endpoint host")
				}
				if port := u.Port(); port != "" {
					n, err := strconv.ParseUint(port, 10, 16)
					if err != nil || n > 65535 {
						return fail("invalid endpoint port")
					}
				}
				if strings.Contains(u.Hostname(), ":") && net.ParseIP(u.Hostname()) == nil {
					return fail("invalid endpoint IP")
				}
				p.Endpoint = v
			case "auth":
				v, ok := text(s)
				if !ok || !regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`).MatchString(v) {
					return fail("invalid bearer environment name")
				}
				p.BearerEnvironment = v
			case "timeout_ms":
				v, ok := integer(s, 2147483647)
				if !ok {
					return fail("timeout_ms out of range")
				}
				p.TimeoutMilliseconds = v
			case "max_body_bytes":
				v, ok := integer(s, 67108864)
				if !ok {
					return fail("max_body_bytes out of range")
				}
				p.MaxBodyBytes = v
			case "headers":
				if s.Text != nil || s.Integer != nil {
					return fail("headers requires static entries")
				}
				names := map[string]ConnectionSetting{}
				for _, h := range s.Entries {
					origin = h
					validateHeader := func() error {
						var findings []error
						name := strings.ToLower(strings.ReplaceAll(h.Name, "_", "-"))
						v, ok := text(h)
						origin.ValueSpan = h.NameSpan
						if prior, exists := names[name]; exists {
							findings = append(findings, source.Relate(prior.File, prior.NameSpan, "first header declared here", fail("duplicate header")))
						} else {
							names[name] = h
						}
						if !connectionHeaderName(h.Name) {
							findings = append(findings, fail("invalid header"))
						}
						origin.ValueSpan = h.ValueSpan
						if h.Invalid != nil {
							findings = append(findings, h.Invalid)
						} else if !ok {
							findings = append(findings, fail("header requires string literal"))
						} else {
							for _, r := range v {
								if r > 255 || r == '\r' || r == '\n' || r == 0 {
									findings = append(findings, fail("invalid literal header value"))
									break
								}
							}
						}
						if len(findings) != 0 {
							return errors.Join(findings...)
						}
						p.Headers = append(p.Headers, ConnectionHeader{name, v})
						return nil
					}
					if err := validateHeader(); err != nil {
						problems = append(problems, err)
					}
				}
			case "metadata":
				if s.Text != nil || s.Integer != nil {
					return fail("metadata requires typed entries")
				}
				keys := map[string]ConnectionSetting{}
				invalidMetadata := map[string]bool{}
				for _, m := range s.Entries {
					origin = m
					validateMetadata := func() error {
						var findings []error
						if prior, exists := keys[m.Name]; exists {
							origin.ValueSpan = m.NameSpan
							findings = append(findings, source.Relate(prior.File, prior.NameSpan, "first metadata declared here", fail("duplicate metadata "+m.Name)))
						} else {
							keys[m.Name] = m
						}
						origin.ValueSpan = m.ValueSpan
						validateValue := func() error {
							if m.Invalid != nil {
								return m.Invalid
							}
							switch m.Name {
							case "protocol":
								v, ok := text(m)
								if !ok || (v != "typesafe_systemone_v1" && v != "openai_responses_v1") {
									return fail("unknown protocol")
								}
								if len(findings) == 0 {
									p.Protocol = v
								}
							case "model":
								v, ok := text(m)
								if !ok || v == "" {
									return fail("model requires nonempty text")
								}
								if len(findings) == 0 {
									p.Model = v
								}
							case "max_output_tokens":
								v, ok := integer(m, 32768)
								if !ok {
									return fail("max_output_tokens out of range")
								}
								if len(findings) == 0 {
									p.MaxOutputTokens = v
								}
							default:
								origin.ValueSpan = m.NameSpan
								return fail("unknown metadata " + m.Name)
							}
							return nil
						}
						if err := validateValue(); err != nil {
							findings = append(findings, err)
						}
						return errors.Join(findings...)
					}
					if err := validateMetadata(); err != nil {
						invalidMetadata[m.Name] = true
						problems = append(problems, err)
					}
				}
				_, hasProtocol := keys["protocol"]
				if setting, exists := keys["max_output_tokens"]; exists && !invalidMetadata["max_output_tokens"] && !invalidMetadata["protocol"] && (!hasProtocol || p.Protocol != "" && p.Protocol != "openai_responses_v1") {
					origin = setting
					return fail("max_output_tokens requires OpenAI profile")
				}
			default:
				origin.ValueSpan = s.NameSpan
				return fail("unknown setting " + s.Name)
			}
			return nil
		}
		if err := validate(); err != nil {
			problems = append(problems, err)
		}
	}
	origin = ConnectionSetting{}
	_, hasEndpoint := seen["endpoint"]
	_, hasTimeout := seen["timeout_ms"]
	if !hasEndpoint || !hasTimeout {
		problems = append(problems, fail("endpoint and timeout_ms are required"))
	}
	for _, h := range p.Headers {
		if p.BearerEnvironment != "" && h.Name == "authorization" {
			for _, setting := range seen["headers"].Entries {
				if strings.EqualFold(setting.Name, "authorization") {
					origin = setting
					origin.ValueSpan = setting.NameSpan
				}
			}
			problems = append(problems, fail("bearer owns Authorization"))
		}
	}
	if len(problems) != 0 {
		return ConnectionPolicy{}, errors.Join(problems...)
	}
	return p, nil
}
func connectionHeaderName(source string) bool {
	if source == "" {
		return false
	}
	for i, r := range source {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	switch strings.ToLower(strings.ReplaceAll(source, "_", "-")) {
	case "host", "content-length", "transfer-encoding", "connection", "upgrade", "proxy-authorization", "proxy-connection", "keep-alive", "te", "trailer":
		return false
	}
	return true
}
func (p ConnectionPolicy) CheckAI(protocol string) error {
	if p.Protocol != protocol || p.Model == "" {
		return fmt.Errorf("connection requires %s and a model", protocol)
	}
	for _, h := range p.Headers {
		if (h.Name == "content-type" || h.Name == "accept") && h.Value != "application/json" {
			return fmt.Errorf("AI connection owns JSON headers")
		}
	}
	return nil
}
