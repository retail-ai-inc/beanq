package capture

import "github.com/retail-ai-inc/beanq/v3/helper/json"

type Config struct {
	Google   GoogleCredential `json:"google" redis:"google"`
	SMTP     SMTP             `json:"smtp" redis:"smtp"`
	SendGrid SendGrid         `json:"sendGrid" redis:"sendGrid"`
	Rule     Rule             `json:"rule" redis:"rule"`
}

type GoogleCredential struct {
	ClientId     string `json:"clientId" redis:"clientId"`
	ClientSecret string `json:"clientSecret" redis:"clientSecret"`
	CallBackUrl  string `json:"callBackUrl" redis:"callBackUrl"`
	Scheme       string `json:"scheme" redis:"scheme"`
}

func (t GoogleCredential) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}
func (t GoogleCredential) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &t)
}

type SMTP struct {
	Host     string `json:"host" redis:"host"`
	Port     string `json:"port" redis:"port"`
	User     string `json:"user" redis:"user"`
	Password string `json:"password" redis:"password"`
}

func (t SMTP) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}
func (t SMTP) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &t)
}

type SendGrid struct {
	Key         string `json:"key" redis:"key"`
	FromName    string `json:"fromName" redis:"fromName"`
	FromAddress string `json:"fromAddress" redis:"fromAddress"`
}

func (t SendGrid) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}
func (t SendGrid) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &t)
}

type Then struct {
	Key   string `json:"key" redis:"key"`
	Value string `json:"value" redis:"value"`
}
type When struct {
	Key   string `json:"key" redis:"key"`
	Value string `json:"value" redis:"value"`
	Text  string `json:"text" redis:"text"`
}
type Topic struct {
	Channel  string `json:"channel" redis:"channel"`
	Topic    string `json:"topic" redis:"topic"`
	MoodType string `json:"moodType" redis:"moodType"`
}
type If struct {
	Key   string  `json:"key" redis:"key"`
	Value string  `json:"value" redis:"value"`
	Topic []Topic `json:"topic" redis:"topic"`
}
type Rule struct {
	When []When `json:"when" redis:"when"`
	If   []If   `json:"if" redis:"if"`
	Then []Then `json:"then" redis:"then"`
}

func (t Rule) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}
func (t Rule) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &t)
}
