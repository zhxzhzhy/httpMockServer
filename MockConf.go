package main

type MockConf struct {
	Port  int    `json:"port"`
	Mocks []Mock `json:"mocks""`
}

type Mock struct {
	Url     string                 `json:"url"`
	Method  string                 `json:"method"`
	ReqBody map[string]interface{} `json:"reqBody"`
	Resp    interface{}            `json:"resp"`
}
