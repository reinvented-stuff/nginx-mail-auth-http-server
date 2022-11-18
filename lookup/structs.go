package lookup

var Debug bool = false
var ShowSecretsInLog bool = false

type LookupStruct struct {
	ShowSecretsInLog bool
}

type authResultStruct struct {
	AuthStatus    string
	AuthServer    string
	AuthPort      int
	AuthWait      string
	AuthErrorCode string
	AuthViaRelay  bool
	AuthViaLogin  bool
}

type UpstreamStruct struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type QueryParamsStruct struct {
	User     string `db:"User"`
	Pass     string `db:"Pass"`
	RcptTo   string `db:"RcptTo"`
	MailFrom string `db:"MailFrom"`
	ClientIP string `db:"ClientIP"`
}
