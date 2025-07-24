package forge_connect

func GetClientInfoKey(appId string) string {
	return "client:" + appId + ":info"
}
