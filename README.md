# 0.11-cloud

This will hopefully be cloud software specifically designed for old Vector software. All support for newer bots is removed.

This is essentially just a bunch of stuff copied from wire-pod. This is more designed to be run as server software rather than software meant to be run on the same network as Vector. So, a bunch of stuff is taken out.

It isn't working yet, but it's reasonably close.

I will probably implement a public web interface. Anyone can register their bot's ESN with a password. Weather API will be handled by me, but you can enter your knowledge graph credentials.

## Env vars

```
	CertFileEnv      = "TLS_CERT_PATH"
	KeyFileEnv       = "TLS_KEY_PATH"
	WeatherAPIKeyEnv = "WEATHER_API_KEY"
	VoskModelPathEnv = "VOSK_MODEL_PATH"
	ChipperPortEnv   = "CHIPPER_PORT"
	WebPortEnv       = "WEB_PORT"
```