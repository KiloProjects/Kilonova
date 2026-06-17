package browserlist

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mileusna/useragent"
)

func TestBrowserlist(t *testing.T) {
	userAgents := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/603.3.8 (KHTML, like Gecko) Version/10.1.2 Safari/603.3.8",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.115 Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_2 like Mac OS X) AppleWebKit/603.2.4 (KHTML, like Gecko) Version/10.0 Mobile/14F89 Safari/602.1",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_2 like Mac OS X) AppleWebKit/603.2.4 (KHTML, like Gecko) FxiOS/8.1.1b4948 Mobile/14F89 Safari/603.2.4",
		"Mozilla/5.0 (iPad; CPU OS 10_3_2 like Mac OS X) AppleWebKit/603.2.4 (KHTML, like Gecko) Version/10.0 Mobile/14F89 Safari/602.1",
		"Mozilla/5.0 (Linux; Android 4.3; GT-I9300 Build/JSS15J) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/59.0.3071.125 Mobile Safari/537.36",
		"Mozilla/5.0 (Android 4.3; Mobile; rv:54.0) Gecko/54.0 Firefox/54.0",
		"Mozilla/5.0 (Linux; Android 4.3; GT-I9300 Build/JSS15J) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/55.0.2883.91 Mobile Safari/537.36 OPR/42.9.2246.119956",
		"Opera/9.80 (Android; Opera Mini/28.0.2254/66.318; U; en) Presto/2.12.423 Version/12.16",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2.1 Mobile/15E148 Safari/604.1 [WAiOS/2.26.7]",
	}

	for _, s := range userAgents {
		ua := useragent.Parse(s)
		fmt.Println()
		fmt.Println(ua.String)
		fmt.Println(strings.Repeat("=", len(ua.String)))
		fmt.Println("Name:", ua.Name, "v", ua.VersionNo.Major)
		fmt.Println("OS:", ua.OS, "v", ua.OSVersion)
		fmt.Println("Device:", ua.Device)
		if ua.Mobile {
			fmt.Println("(Mobile)")
		}
		if ua.Tablet {
			fmt.Println("(Tablet)")
		}
		if ua.Desktop {
			fmt.Println("(Desktop)")
		}
		if ua.Bot {
			fmt.Println("(Bot)")
		}
		if ua.URL != "" {
			fmt.Println(ua.URL)
		}
	}
}
