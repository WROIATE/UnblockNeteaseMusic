package kuwo

import (
	"encoding/base64"
	"github.com/cnsilvan/UnblockNeteaseMusic/provider/base"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/cnsilvan/UnblockNeteaseMusic/common"
	"github.com/cnsilvan/UnblockNeteaseMusic/network"
	"github.com/cnsilvan/UnblockNeteaseMusic/utils"
)

type KuWo struct{}

func (m *KuWo) SearchSong(song common.SearchSong) (songs []*common.Song) {
	song = base.PreSearchSong(song)
	token := getToken(song.Keyword)
	header := make(http.Header, 4)
	header["referer"] = append(header["referer"], "http://www.kuwo.cn/search/list?key="+url.QueryEscape(song.Keyword))
	header["csrf"] = append(header["csrf"], token)
	header["cookie"] = append(header["cookie"], "kw_token="+token)
	result, err := base.Fetch(
		"http://www.kuwo.cn/api/www/search/searchMusicBykeyWord?key="+song.Keyword+"&pn=1&rn=30",
		nil, header, true)
	if err != nil {
		log.Println(err)
		return songs
	}
	data, ok := result["data"].(common.MapType)
	if ok {
		list, ok := data["list"].([]interface{})
		if ok && len(list) > 0 {
			listLength := len(list)
			maxIndex := listLength/2 + 1
			if maxIndex > 10 {
				maxIndex = 10
			}
			for index, matched := range list {
				if index >= 1 {//kuwo list order by score default
					break
				}
				kuWoSong, ok := matched.(common.MapType)
				if ok {
					rid, ok := kuWoSong["rid"].(string)
					if ok {
						songResult := &common.Song{}
						singerName, _ := kuWoSong["artist"].(string)
						songName, _ := kuWoSong["name"].(string)
						//musicSlice := strings.Split(musicrid, "_")
						//musicId := musicSlice[len(musicSlice)-1]
						songResult.PlatformUniqueKey = kuWoSong
						songResult.PlatformUniqueKey["UnKeyWord"] = song.Keyword
						songResult.Source = "kuwo"
						songResult.PlatformUniqueKey["header"] = header
						songResult.PlatformUniqueKey["musicId"] = rid
						songResult.Id = rid
						if len(songResult.Id) > 0 {
							songResult.Id = string(common.KuWoTag) + songResult.Id
						}
						songResult.Name = songName
						songResult.Artist = singerName
						songResult.AlbumName, _ = kuWoSong["album"].(string)
						songResult.Artist = strings.ReplaceAll(singerName, " ", "")
						songResult.MatchScore, ok = base.CalScore(song, songName, singerName, index, maxIndex)
						if !ok {
							continue
						}
						songs = append(songs, songResult)

					}
				}
			}

		}
	}

	return base.AfterSearchSong(song, songs)
}
func (m *KuWo) GetSongUrl(searchSong common.SearchMusic, song *common.Song) *common.Song {
	if id, ok := song.PlatformUniqueKey["musicId"]; ok {
		if musicId, ok := id.(string); ok {
			if httpHeader, ok := song.PlatformUniqueKey["header"]; ok {
				if header, ok := httpHeader.(http.Header); ok {
					header["user-agent"] = append(header["user-agent"], "okhttp/3.10.0")
					format := "flac|mp3"
					br := ""
					switch searchSong.Quality {
					case common.Standard:
						format = "mp3"
						br = "&br=128kmp3"
					case common.Higher:
						format = "mp3"
						br = "&br=192kmp3"
					case common.ExHigh:
						format = "mp3"
					case common.Lossless:
						format = "flac|mp3"
					default:
						format = "flac|mp3"
					}

					clientRequest := network.ClientRequest{
						Method:               http.MethodGet,
						ForbiddenEncodeQuery: true,
						RemoteUrl:            "http://mobi.kuwo.cn/mobi.s?f=kuwo&q=" + base64.StdEncoding.EncodeToString(Encrypt([]byte("corp=kuwo&p2p=1&type=convert_url2&sig=0&format="+format+"&rid="+musicId+br))),
						Header:               header,
						Proxy:                true,
					}
					resp, err := network.Request(&clientRequest)
					if err != nil {
						log.Println(err)
						return song
					}
					defer resp.Body.Close()
					body, err := network.GetResponseBody(resp, false)
					reg := regexp.MustCompile(`http[^\s$"]+`)
					address := string(body)
					params := reg.FindStringSubmatch(address)
					if len(params) > 0 {
						song.Url = params[0]
						return song
					}

				}
			}
		}
	}
	return song
}
func (m *KuWo) ParseSong(searchSong common.SearchSong) *common.Song {
	song := &common.Song{}
	songs := m.SearchSong(searchSong)
	if len(songs) > 0 {
		song = m.GetSongUrl(common.SearchMusic{Quality: searchSong.Quality}, songs[0])
	}
	return song
}
func getToken(keyword string) string {
	var token = ""
	clientRequest := network.ClientRequest{
		Method:    http.MethodGet,
		RemoteUrl: "http://kuwo.cn/search/list?key=" + keyword,
		Host:      "kuwo.cn",
		Header:    nil,
		Proxy:     false,
	}
	resp, err := network.Request(&clientRequest)
	if err != nil {
		log.Println(err)
		return token
	}
	defer resp.Body.Close()
	cookies := resp.Header.Get("set-cookie")
	if strings.Contains(cookies, "kw_token") {
		cookies = utils.ReplaceAll(cookies, ";.*", "")
		splitSlice := strings.Split(cookies, "=")
		token = splitSlice[len(splitSlice)-1]
	}
	return token
}
