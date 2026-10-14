package capture

import "github.com/retail-ai-inc/beanq/v4/helper/json"

type Config struct {
	Google          GoogleCredential `json:"google" redis:"google" bson:"google"`
	GoogleReCAPTCHA GoogleReCAPTCHA  `json:"googleReCAPTCHA" redis:"googleReCAPTCHA" bson:"googleReCAPTCHA"`
	Email           Email            `json:"email" bson:"email"`
	Rule            Rule             `json:"rule" redis:"rule" bson:"rule"`
	Slack           Slack            `json:"slack" redis:"slack" bson:"slack"`
}

type Email struct {
	Used     string   `json:"used" bson:"used"`
	SMTP     SMTP     `json:"smtp" bson:"SMTP"`
	SendGrid SendGrid `json:"sendGrid" bson:"sendGrid"`
}

func (t Email) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}
func (t Email) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &t)
}

type GoogleCredential struct {
	ClientId     string `json:"clientId" redis:"clientId" bson:"clientId"`
	ClientSecret string `json:"clientSecret" redis:"clientSecret" bson:"clientSecret"`
	CallBackUrl  string `json:"callBackUrl" redis:"callBackUrl" bson:"callBackUrl"`
}

func (t GoogleCredential) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}
func (t GoogleCredential) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &t)
}

type GoogleReCAPTCHA struct {
	Active    bool   `json:"active" bson:"active"`
	ProjectId string `json:"projectId" bson:"projectId"`
	SiteKey   string `json:"siteKey" bson:"siteKey"`
	ApiKey    string `json:"apiKey" bson:"apiKey"`
}

func (t GoogleReCAPTCHA) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}

func (t GoogleReCAPTCHA) UnmarshalBinary(data []byte) error {
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
	Key         string `json:"key" redis:"key" bson:"key"`
	FromName    string `json:"fromName" redis:"fromName" bson:"fromName"`
	FromAddress string `json:"fromAddress" redis:"fromAddress" bson:"fromAddress"`
}

func (t SendGrid) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}
func (t SendGrid) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &t)
}

type ThenParameters struct {
	WorkSpace string `json:"workSpace" redis:"workSpace" bson:"workSpace"`
	Channel   string `json:"channel" redis:"channel" bson:"channel"`
	Tags      string `json:"tags" redis:"tags" bson:"tags"`
	Notes     string `json:"notes" redis:"notes" bson:"notes"`
}

type Then struct {
	Key        string         `json:"key" redis:"key" bson:"key"`
	Value      string         `json:"value" redis:"value" bson:"value"`
	Parameters ThenParameters `json:"parameters" redis:"parameters" bson:"parameters"`
}
type When struct {
	Key   string `json:"key" redis:"key" bson:"key"`
	Value string `json:"value" redis:"value" bson:"value"`
	Text  string `json:"text" redis:"text" bson:"text"`
}
type Topic struct {
	Channel  string `json:"channel" redis:"channel" bson:"channel"`
	Topic    string `json:"topic" redis:"topic" bson:"topic"`
	MoodType string `json:"moodType" redis:"moodType" bson:"moodType"`
}
type If struct {
	Key   string  `json:"key" redis:"key" bson:"key"`
	Value string  `json:"value" redis:"value" bson:"value"`
	Topic []Topic `json:"topic" redis:"topic" bson:"topic"`
}
type Rule struct {
	When []When `json:"when" redis:"when" bson:"when"`
	If   []If   `json:"if" redis:"if" bson:"if"`
	Then []Then `json:"then" redis:"then" bson:"then"`
}

func (t Rule) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}
func (t Rule) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &t)
}

type Slack struct {
	BotAuthToken string `json:"botAuthToken" redis:"botAuthToken" bson:"botAuthToken"`
}

func (t Slack) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}
func (t Slack) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &t)
}
