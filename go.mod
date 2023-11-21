module aika

go 1.21

require (
	github.com/aws/aws-sdk-go v1.45.19
	github.com/buger/jsonparser v1.1.1
	github.com/bwmarrin/discordgo v0.27.1
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.1.0
	github.com/gocolly/colly v1.2.0
	github.com/google/uuid v1.3.1
	github.com/haguro/elevenlabs-go v0.2.2
	github.com/hegedustibor/htgo-tts v0.0.0-20230402053941-cd8d1a158135
	github.com/jellydator/ttlcache/v3 v3.1.0
	github.com/kkdai/youtube/v2 v2.9.0
	github.com/sashabaranov/go-openai v1.17.8
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.8.4
	golang.org/x/sync v0.3.0
	golang.org/x/time v0.3.0
	gopkg.in/yaml.v2 v2.4.0
	layeh.com/gopus v0.0.0-20210501142526-1ee02d434e32
)

// waiting on https://github.com/sashabaranov/go-openai/pull/580
replace github.com/sashabaranov/go-openai => github.com/rkintzi/go-openai v0.0.0-20231115154728-152021253721

require (
	github.com/PuerkitoBio/goquery v1.8.1 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/antchfx/htmlquery v1.3.0 // indirect
	github.com/antchfx/xmlquery v1.3.18 // indirect
	github.com/antchfx/xpath v1.2.4 // indirect
	github.com/bitly/go-simplejson v0.5.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/dop251/goja v0.0.0-20230919151941-fc55792775de // indirect
	github.com/ebitengine/purego v0.5.0 // indirect
	github.com/go-audio/riff v1.0.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/pprof v0.0.0-20230926050212-f7f687d19a98 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hajimehoshi/go-mp3 v0.3.4 // indirect
	github.com/hajimehoshi/oto/v2 v2.4.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kennygrant/sanitize v1.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/saintfish/chardet v0.0.0-20230101081208-5e3ef4b5456d // indirect
	github.com/temoto/robotstxt v1.1.2 // indirect
	golang.org/x/crypto v0.13.0 // indirect
	golang.org/x/net v0.15.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
