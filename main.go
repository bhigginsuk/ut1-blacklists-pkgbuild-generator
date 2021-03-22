package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type blocklist struct {
	url  string
	name string
	desc string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	blocklist_regexp := regexp.MustCompile("^ftp:\\/\\/ftp\\.ut-capitole\\.fr\\/pub\\/reseau\\/cache\\/squidguard_contrib\\/(\\w+)\\.tar\\.gz")

	response, err := http.Get("https://dsi.ut-capitole.fr/blacklists/index_en.php")
	check(err)

	tokenizer := html.NewTokenizer(response.Body)
	defer response.Body.Close()

	lists := make([]*blocklist, 0)

	ftp_url := ""
	name := ""
	desc := ""
	count := 0
	skip_tags := 0
	for {
		token_type := tokenizer.Next()

		if skip_tags > 0 {
			skip_tags--
			continue
		}

		if token_type == html.ErrorToken {
			break
		} else if token_type == html.StartTagToken {
			token := tokenizer.Token()

			if token.Data == "a" {
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						match := blocklist_regexp.FindStringSubmatch(attr.Val)
						if len(match) > 0 {
							// The first matches a unique HTML element with a blocklist that
							// contains them all
							if count != 0 {
								ftp_url = attr.Val
								name = match[1]
								skip_tags = 9
							}
							count++
							break
						}
					}
				}
			}
		} else if ftp_url != "" {
			token := tokenizer.Token()
			desc = token.Data
		}

		if ftp_url != "" && desc != "" {
			lists = append(lists, &blocklist{url: ftp_url, name: name, desc: desc})
			ftp_url = ""
			name = ""
			desc = ""
		}
	}

	data, err := ioutil.ReadFile("PKGBUILD-format")
	check(err)
	pkgbuild_str := string(data)

	for _, list := range lists {
		output := strings.Replace(pkgbuild_str, "{{LISTNAME}}", list.name, 1)
		output = strings.Replace(output, "{{DESCRIPTION}}", list.desc, 1)

		_ = os.MkdirAll("out/"+list.name, 0750)
		err := ioutil.WriteFile(fmt.Sprintf("out/%s/PKGBUILD", list.name), []byte(output), 0644)
		check(err)

		fmt.Printf("Generated %s\n", list.name)
	}
	fmt.Println("Done")
}
