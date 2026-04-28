package proxy

import (
	"os"
)

const ProxyAddr = "127.0.0.1:7890"

func SetSystemProxy() error {
	os.Setenv("HTTP_PROXY", "http://"+ProxyAddr)
	os.Setenv("HTTPS_PROXY", "http://"+ProxyAddr)
	os.Setenv("ALL_PROXY", "socks5://"+ProxyAddr)
	os.Setenv("NO_PROXY", "localhost,127.0.0.1")
	return nil
}

func UnsetSystemProxy() error {
	os.Unsetenv("HTTP_PROXY")
	os.Unsetenv("HTTPS_PROXY")
	os.Unsetenv("ALL_PROXY")
	os.Unsetenv("NO_PROXY")
	os.Unsetenv("no_proxy")
	return nil
}