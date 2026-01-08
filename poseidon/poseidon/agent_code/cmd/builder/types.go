package main

// Config is the top-level configuration structure
type Config struct {
	UUID     string       `json:"uuid"`
	Debug    bool         `json:"debug"`
	Build    BuildConfig  `json:"build"`
	Profiles []string     `json:"profiles"`
	Egress   EgressConfig `json:"egress,omitempty"`
	UIClient *UIConfig    `json:"uiClient,omitempty"`

	HTTP        *HTTPConfig        `json:"http,omitempty"`
	Websocket   *WebsocketConfig   `json:"websocket,omitempty"`
	TCP         *TCPConfig         `json:"tcp,omitempty"`
	DNS         *DNSConfig         `json:"dns,omitempty"`
	DynamicHTTP *DynamicHTTPConfig `json:"dynamichttp,omitempty"`
	HTTPx       *HTTPxConfig       `json:"httpx,omitempty"`
}

type BuildConfig struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Output string `json:"output,omitempty"`
	Mode   string `json:"mode,omitempty"`
	Garble bool   `json:"garble,omitempty"`
	Static bool   `json:"static,omitempty"`
	CGO    bool   `json:"cgo,omitempty"`
}

type EgressConfig struct {
	Order           []string `json:"order,omitempty"`
	Failover        string   `json:"failover,omitempty"`
	FailedThreshold int      `json:"failedThreshold,omitempty"`
	BackoffDelay    int      `json:"backoffDelay,omitempty"`
	BackoffBase     int      `json:"backoffBase,omitempty"`
}

type UIConfig struct {
	BaseURL      string `json:"baseUrl"`
	CheckinPath  string `json:"checkinPath,omitempty"`
	PollPath     string `json:"pollPath,omitempty"`
	PollInterval int    `json:"pollInterval,omitempty"`
	HTTPTimeout  int    `json:"httpTimeout,omitempty"`
}

type HTTPConfig struct {
	CallbackHost           string            `json:"callbackHost"`
	CallbackPort           int               `json:"callbackPort"`
	AesPsk                 string            `json:"aesPsk"`
	Killdate               string            `json:"killdate"`
	Interval               int               `json:"interval"`
	Jitter                 int               `json:"jitter"`
	PostUri                string            `json:"postUri"`
	GetUri                 string            `json:"getUri"`
	QueryPathName          string            `json:"queryPathName,omitempty"`
	EncryptedExchangeCheck *bool             `json:"encryptedExchangeCheck,omitempty"`
	Headers                map[string]string `json:"headers,omitempty"`
	Proxy                  *ProxyConfig      `json:"proxy,omitempty"`
}

type ProxyConfig struct {
	Host   string `json:"host"`
	Port   int    `json:"port"`
	User   string `json:"user,omitempty"`
	Pass   string `json:"pass,omitempty"`
	Bypass bool   `json:"bypass,omitempty"`
}

type WebsocketConfig struct {
	CallbackHost           string `json:"callbackHost"`
	CallbackPort           int    `json:"callbackPort"`
	AesPsk                 string `json:"aesPsk"`
	Killdate               string `json:"killdate"`
	Interval               int    `json:"interval"`
	Jitter                 int    `json:"jitter"`
	Endpoint               string `json:"endpoint"`
	EncryptedExchangeCheck *bool  `json:"encryptedExchangeCheck,omitempty"`
	DomainFront            string `json:"domainFront,omitempty"`
	TaskingType            string `json:"taskingType,omitempty"`
	UserAgent              string `json:"userAgent,omitempty"`
}

type TCPConfig struct {
	Port                   int    `json:"port"`
	AesPsk                 string `json:"aesPsk"`
	Killdate               string `json:"killdate"`
	EncryptedExchangeCheck *bool  `json:"encryptedExchangeCheck,omitempty"`
}

type DNSConfig struct {
	Domains                []string `json:"domains"`
	AesPsk                 string   `json:"aesPsk"`
	Killdate               string   `json:"killdate"`
	Interval               int      `json:"interval"`
	Jitter                 int      `json:"jitter"`
	Server                 string   `json:"server,omitempty"`
	DomainRotation         string   `json:"domainRotation,omitempty"`
	FailoverThreshold      int      `json:"failoverThreshold,omitempty"`
	RecordType             string   `json:"recordType,omitempty"`
	MaxQueryLength         int      `json:"maxQueryLength,omitempty"`
	MaxSubdomainLength     int      `json:"maxSubdomainLength,omitempty"`
	EncryptedExchangeCheck *bool    `json:"encryptedExchangeCheck,omitempty"`
}

type DynamicHTTPConfig struct {
	AesPsk                 string `json:"aesPsk"`
	Killdate               string `json:"killdate"`
	Interval               int    `json:"interval"`
	Jitter                 int    `json:"jitter"`
	EncryptedExchangeCheck *bool  `json:"encryptedExchangeCheck,omitempty"`
	RawC2Config            string `json:"rawC2Config"`
}

type HTTPxConfig struct {
	CallbackDomains        []string `json:"callbackDomains"`
	AesPsk                 string   `json:"aesPsk"`
	Killdate               string   `json:"killdate"`
	Interval               int      `json:"interval"`
	Jitter                 int      `json:"jitter"`
	DomainRotationMethod   string   `json:"domainRotationMethod,omitempty"`
	FailoverThreshold      int      `json:"failoverThreshold,omitempty"`
	EncryptedExchangeCheck *bool    `json:"encryptedExchangeCheck,omitempty"`
	RawC2Config            string   `json:"rawC2Config"`
}
