package main

import (
	"container/list"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/olahol/melody"
)

const TIMEDIFF = 5 //5秒

var mqLock sync.RWMutex
var pmqLock sync.RWMutex

var historyMsgHolder list.List                           //保留最新50记录子消息队列
var popularMsgHolder list.List                           //统计5秒词频子消息队列
var messageQueue chan string = make(chan string, 1000)   //主消息队列, 默认长度为1000
var boradcastQueue chan string = make(chan string, 1000) //广播消息队列, 默认长度为1000
var connectTimeHolder sync.Map                           //客户端连接时间

func main() {
	r := gin.Default()
	m := melody.New()

	r.GET("/", func(c *gin.Context) {
		http.ServeFile(c.Writer, c.Request, "index.html")
	})

	r.GET("/ws", func(c *gin.Context) {
		m.HandleRequest(c.Writer, c.Request)
	})

	go func() {
		for {
			doDistribute()
		}
	}()
	go func() {
		for {
			msg := <-boradcastQueue
			m.Broadcast([]byte(msg))
		}
	}()

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		switch strings.TrimSpace(string(msg)) {
		case "/clear":
			//清空历史消息，方便测试
			mqLock.Lock()
			defer mqLock.Unlock()
			historyMsgHolder = list.List{}

		case "/popular":
			pmqLock.RLock()
			defer pmqLock.RUnlock()

			w := getMostPopularWord()
			s.Write([]byte("/popular " + w))

		case "/stats":
			uid, exist := s.Get("uid")
			if !exist {
				return
			}

			if t, ok := connectTimeHolder.Load(uid); ok {
				d, err := time.ParseDuration(fmt.Sprintf("%ds", time.Now().Unix()-t.(int64)))
				if err != nil {
					fmt.Print(err)
					return
				}
				s.Write([]byte(fmt.Sprintf("/stats %s  %s", uid, d.String())))
			}

		default:
			messageQueue <- string(msg)
		}
	})

	m.HandleConnect(func(s *melody.Session) {
		uid := uuid.New().String()
		s.Set("uid", uid)

		connectTimeHolder.Store(uid, time.Now().Unix())

		mqLock.RLock()
		defer mqLock.RUnlock()

		//读取mq队列消息（50条）
		for e := historyMsgHolder.Back(); e != nil; e = e.Prev() {
			msg := e.Value.([]byte)
			s.Write(msg)
		}
	})

	r.Run(":5000")
}

//分发消息到不同的子消息队列
func doDistribute() {
	msg := <-messageQueue

	mqLock.Lock()
	defer mqLock.Unlock()

	//存入最新50条记录
	historyMsgHolder.PushFront([]byte(msgFilter(msg)))
	//修剪队列长度
	if historyMsgHolder.Len() > 50 {
		historyMsgHolder.Remove(historyMsgHolder.Back())
	}

	pmqLock.Lock()
	defer pmqLock.Unlock()
	//存入最新5秒记录
	popularMsgHolder.PushFront(Message{
		Msg:       msg,
		timestamp: time.Now().Unix(),
	})

	//修剪队列长度
	for popularMsgHolder.Back() != nil && time.Now().Unix()-popularMsgHolder.Back().Value.(Message).timestamp > TIMEDIFF {
		popularMsgHolder.Remove(popularMsgHolder.Back())
	}

	boradcastQueue <- msg
}

func msgFilter(msg string) string {
	words := strings.Split(msg, " ")
	for i, v := range words {
		_, ok := profanityWords[v]
		if ok {
			words[i] = "*"
		}
	}
	return strings.Join(words, " ")
}

func getMostPopularWord() string {
	timestamp := time.Now().Unix()
	m := make(map[string]int)
	for e := popularMsgHolder.Front(); e != nil; e = e.Next() {
		msg := e.Value.(Message)

		// 过滤超过5秒的消息
		if timestamp-msg.timestamp > TIMEDIFF {
			break
		}

		//统计词频
		words := strings.Split(msg.Msg, " ")
		for _, w := range words {
			_, ok := m[w]
			if ok {
				m[w] = m[w] + 1
			} else {
				m[w] = 1
			}
		}
	}
	if len(m) == 0 {
		return ""
	}

	wordCounter := make(WordCounter, len(m))
	i := 0
	for word, cnt := range m {
		wordCounter[i] = Word{
			word: word,
			cnt:  cnt,
		}
		i++
	}

	sort.Sort(wordCounter)

	fmt.Printf("%+v", wordCounter)
	return wordCounter[0].word
}

type Message struct {
	Msg       string
	timestamp int64
}

type Word struct {
	word string
	cnt  int
}

type WordCounter []Word

func (h WordCounter) Len() int {
	return len(h)
}

func (h WordCounter) Less(i, j int) bool {
	return h[i].cnt > h[j].cnt
}

func (h WordCounter) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

var profanityWords = map[string]string{
	"4r5e":                    "4r5e",
	"5h1t":                    "5h1t",
	"5hit":                    "5hit",
	"a55":                     "a55",
	"anal":                    "anal",
	"anus":                    "anus",
	"ar5e":                    "ar5e",
	"arrse":                   "arrse",
	"arse":                    "arse",
	"ass":                     "ass",
	"ass-fucker":              "ass-fucker",
	"asses":                   "asses",
	"assfucker":               "assfucker",
	"assfukka":                "assfukka",
	"asshole":                 "asshole",
	"assholes":                "assholes",
	"asswhole":                "asswhole",
	"a_s_s":                   "a_s_s",
	"b!tch":                   "b!tch",
	"b00bs":                   "b00bs",
	"b17ch":                   "b17ch",
	"b1tch":                   "b1tch",
	"ballbag":                 "ballbag",
	"balls":                   "balls",
	"ballsack":                "ballsack",
	"bastard":                 "bastard",
	"beastial":                "beastial",
	"beastiality":             "beastiality",
	"bellend":                 "bellend",
	"bestial":                 "bestial",
	"bestiality":              "bestiality",
	"bi+ch":                   "bi+ch",
	"biatch":                  "biatch",
	"bitch":                   "bitch",
	"bitcher":                 "bitcher",
	"bitchers":                "bitchers",
	"bitches":                 "bitches",
	"bitchin":                 "bitchin",
	"bitching":                "bitching",
	"bloody":                  "bloody",
	"blow job":                "blow job",
	"blowjob":                 "blowjob",
	"blowjobs":                "blowjobs",
	"boiolas":                 "boiolas",
	"bollock":                 "bollock",
	"bollok":                  "bollok",
	"boner":                   "boner",
	"boob":                    "boob",
	"boobs":                   "boobs",
	"booobs":                  "booobs",
	"boooobs":                 "boooobs",
	"booooobs":                "booooobs",
	"booooooobs":              "booooooobs",
	"breasts":                 "breasts",
	"buceta":                  "buceta",
	"bugger":                  "bugger",
	"bum":                     "bum",
	"bunny fucker":            "bunny fucker",
	"butt":                    "butt",
	"butthole":                "butthole",
	"buttmunch":               "buttmunch",
	"buttplug":                "buttplug",
	"c0ck":                    "c0ck",
	"c0cksucker":              "c0cksucker",
	"carpet muncher":          "carpet muncher",
	"cawk":                    "cawk",
	"chink":                   "chink",
	"cipa":                    "cipa",
	"cl1t":                    "cl1t",
	"clit":                    "clit",
	"clitoris":                "clitoris",
	"clits":                   "clits",
	"cnut":                    "cnut",
	"cock":                    "cock",
	"cock-sucker":             "cock-sucker",
	"cockface":                "cockface",
	"cockhead":                "cockhead",
	"cockmunch":               "cockmunch",
	"cockmuncher":             "cockmuncher",
	"cocks":                   "cocks",
	"cocksuck ":               "cocksuck ",
	"cocksucked ":             "cocksucked ",
	"cocksucker":              "cocksucker",
	"cocksucking":             "cocksucking",
	"cocksucks ":              "cocksucks ",
	"cocksuka":                "cocksuka",
	"cocksukka":               "cocksukka",
	"cok":                     "cok",
	"cokmuncher":              "cokmuncher",
	"coksucka":                "coksucka",
	"coon":                    "coon",
	"cox":                     "cox",
	"crap":                    "crap",
	"cum":                     "cum",
	"cummer":                  "cummer",
	"cumming":                 "cumming",
	"cums":                    "cums",
	"cumshot":                 "cumshot",
	"cunilingus":              "cunilingus",
	"cunillingus":             "cunillingus",
	"cunnilingus":             "cunnilingus",
	"cunt":                    "cunt",
	"cuntlick ":               "cuntlick ",
	"cuntlicker ":             "cuntlicker ",
	"cuntlicking ":            "cuntlicking ",
	"cunts":                   "cunts",
	"cyalis":                  "cyalis",
	"cyberfuc":                "cyberfuc",
	"cyberfuck ":              "cyberfuck ",
	"cyberfucked ":            "cyberfucked ",
	"cyberfucker":             "cyberfucker",
	"cyberfuckers":            "cyberfuckers",
	"cyberfucking ":           "cyberfucking ",
	"d1ck":                    "d1ck",
	"damn":                    "damn",
	"dick":                    "dick",
	"dickhead":                "dickhead",
	"dildo":                   "dildo",
	"dildos":                  "dildos",
	"dink":                    "dink",
	"dinks":                   "dinks",
	"dirsa":                   "dirsa",
	"dlck":                    "dlck",
	"dog-fucker":              "dog-fucker",
	"doggin":                  "doggin",
	"dogging":                 "dogging",
	"donkeyribber":            "donkeyribber",
	"doosh":                   "doosh",
	"duche":                   "duche",
	"dyke":                    "dyke",
	"ejaculate":               "ejaculate",
	"ejaculated":              "ejaculated",
	"ejaculates ":             "ejaculates ",
	"ejaculating ":            "ejaculating ",
	"ejaculatings":            "ejaculatings",
	"ejaculation":             "ejaculation",
	"ejakulate":               "ejakulate",
	"f u c k":                 "f u c k",
	"f u c k e r":             "f u c k e r",
	"f4nny":                   "f4nny",
	"fag":                     "fag",
	"fagging":                 "fagging",
	"faggitt":                 "faggitt",
	"faggot":                  "faggot",
	"faggs":                   "faggs",
	"fagot":                   "fagot",
	"fagots":                  "fagots",
	"fags":                    "fags",
	"fanny":                   "fanny",
	"fannyflaps":              "fannyflaps",
	"fannyfucker":             "fannyfucker",
	"fanyy":                   "fanyy",
	"fatass":                  "fatass",
	"fcuk":                    "fcuk",
	"fcuker":                  "fcuker",
	"fcuking":                 "fcuking",
	"feck":                    "feck",
	"fecker":                  "fecker",
	"felching":                "felching",
	"fellate":                 "fellate",
	"fellatio":                "fellatio",
	"fingerfuck ":             "fingerfuck ",
	"fingerfucked ":           "fingerfucked ",
	"fingerfucker ":           "fingerfucker ",
	"fingerfuckers":           "fingerfuckers",
	"fingerfucking ":          "fingerfucking ",
	"fingerfucks ":            "fingerfucks ",
	"fistfuck":                "fistfuck",
	"fistfucked ":             "fistfucked ",
	"fistfucker ":             "fistfucker ",
	"fistfuckers ":            "fistfuckers ",
	"fistfucking ":            "fistfucking ",
	"fistfuckings ":           "fistfuckings ",
	"fistfucks ":              "fistfucks ",
	"flange":                  "flange",
	"fook":                    "fook",
	"fooker":                  "fooker",
	"fuck":                    "fuck",
	"fucka":                   "fucka",
	"fucked":                  "fucked",
	"fucker":                  "fucker",
	"fuckers":                 "fuckers",
	"fuckhead":                "fuckhead",
	"fuckheads":               "fuckheads",
	"fuckin":                  "fuckin",
	"fucking":                 "fucking",
	"fuckings":                "fuckings",
	"fuckingshitmotherfucker": "fuckingshitmotherfucker",
	"fuckme ":                 "fuckme ",
	"fucks":                   "fucks",
	"fuckwhit":                "fuckwhit",
	"fuckwit":                 "fuckwit",
	"fudge packer":            "fudge packer",
	"fudgepacker":             "fudgepacker",
	"fuk":                     "fuk",
	"fuker":                   "fuker",
	"fukker":                  "fukker",
	"fukkin":                  "fukkin",
	"fuks":                    "fuks",
	"fukwhit":                 "fukwhit",
	"fukwit":                  "fukwit",
	"fux":                     "fux",
	"fux0r":                   "fux0r",
	"f_u_c_k":                 "f_u_c_k",
	"gangbang":                "gangbang",
	"gangbanged ":             "gangbanged ",
	"gangbangs ":              "gangbangs ",
	"gaylord":                 "gaylord",
	"gaysex":                  "gaysex",
	"goatse":                  "goatse",
	"God":                     "God",
	"god-dam":                 "god-dam",
	"god-damned":              "god-damned",
	"goddamn":                 "goddamn",
	"goddamned":               "goddamned",
	"hardcoresex ":            "hardcoresex ",
	"hell":                    "hell",
	"heshe":                   "heshe",
	"hoar":                    "hoar",
	"hoare":                   "hoare",
	"hoer":                    "hoer",
	"homo":                    "homo",
	"hore":                    "hore",
	"horniest":                "horniest",
	"horny":                   "horny",
	"hotsex":                  "hotsex",
	"jack-off ":               "jack-off ",
	"jackoff":                 "jackoff",
	"jap":                     "jap",
	"jerk-off ":               "jerk-off ",
	"jism":                    "jism",
	"jiz ":                    "jiz ",
	"jizm ":                   "jizm ",
	"jizz":                    "jizz",
	"kawk":                    "kawk",
	"knob":                    "knob",
	"knobead":                 "knobead",
	"knobed":                  "knobed",
	"knobend":                 "knobend",
	"knobhead":                "knobhead",
	"knobjocky":               "knobjocky",
	"knobjokey":               "knobjokey",
	"kock":                    "kock",
	"kondum":                  "kondum",
	"kondums":                 "kondums",
	"kum":                     "kum",
	"kummer":                  "kummer",
	"kumming":                 "kumming",
	"kums":                    "kums",
	"kunilingus":              "kunilingus",
	"l3i+ch":                  "l3i+ch",
	"l3itch":                  "l3itch",
	"labia":                   "labia",
	"lmfao":                   "lmfao",
	"lust":                    "lust",
	"lusting":                 "lusting",
	"m0f0":                    "m0f0",
	"m0fo":                    "m0fo",
	"m45terbate":              "m45terbate",
	"ma5terb8":                "ma5terb8",
	"ma5terbate":              "ma5terbate",
	"masochist":               "masochist",
	"master-bate":             "master-bate",
	"masterb8":                "masterb8",
	"masterbat*":              "masterbat*",
	"masterbat3":              "masterbat3",
	"masterbate":              "masterbate",
	"masterbation":            "masterbation",
	"masterbations":           "masterbations",
	"masturbate":              "masturbate",
	"mo-fo":                   "mo-fo",
	"mof0":                    "mof0",
	"mofo":                    "mofo",
	"mothafuck":               "mothafuck",
	"mothafucka":              "mothafucka",
	"mothafuckas":             "mothafuckas",
	"mothafuckaz":             "mothafuckaz",
	"mothafucked ":            "mothafucked ",
	"mothafucker":             "mothafucker",
	"mothafuckers":            "mothafuckers",
	"mothafuckin":             "mothafuckin",
	"mothafucking ":           "mothafucking ",
	"mothafuckings":           "mothafuckings",
	"mothafucks":              "mothafucks",
	"mother fucker":           "mother fucker",
	"motherfuck":              "motherfuck",
	"motherfucked":            "motherfucked",
	"motherfucker":            "motherfucker",
	"motherfuckers":           "motherfuckers",
	"motherfuckin":            "motherfuckin",
	"motherfucking":           "motherfucking",
	"motherfuckings":          "motherfuckings",
	"motherfuckka":            "motherfuckka",
	"motherfucks":             "motherfucks",
	"muff":                    "muff",
	"mutha":                   "mutha",
	"muthafecker":             "muthafecker",
	"muthafuckker":            "muthafuckker",
	"muther":                  "muther",
	"mutherfucker":            "mutherfucker",
	"n1gga":                   "n1gga",
	"n1gger":                  "n1gger",
	"nazi":                    "nazi",
	"nigg3r":                  "nigg3r",
	"nigg4h":                  "nigg4h",
	"nigga":                   "nigga",
	"niggah":                  "niggah",
	"niggas":                  "niggas",
	"niggaz":                  "niggaz",
	"nigger":                  "nigger",
	"niggers ":                "niggers ",
	"nob":                     "nob",
	"nob jokey":               "nob jokey",
	"nobhead":                 "nobhead",
	"nobjocky":                "nobjocky",
	"nobjokey":                "nobjokey",
	"numbnuts":                "numbnuts",
	"nutsack":                 "nutsack",
	"orgasim ":                "orgasim ",
	"orgasims ":               "orgasims ",
	"orgasm":                  "orgasm",
	"orgasms ":                "orgasms ",
	"p0rn":                    "p0rn",
	"pawn":                    "pawn",
	"pecker":                  "pecker",
	"penis":                   "penis",
	"penisfucker":             "penisfucker",
	"phonesex":                "phonesex",
	"phuck":                   "phuck",
	"phuk":                    "phuk",
	"phuked":                  "phuked",
	"phuking":                 "phuking",
	"phukked":                 "phukked",
	"phukking":                "phukking",
	"phuks":                   "phuks",
	"phuq":                    "phuq",
	"pigfucker":               "pigfucker",
	"pimpis":                  "pimpis",
	"piss":                    "piss",
	"pissed":                  "pissed",
	"pisser":                  "pisser",
	"pissers":                 "pissers",
	"pisses ":                 "pisses ",
	"pissflaps":               "pissflaps",
	"pissin ":                 "pissin ",
	"pissing":                 "pissing",
	"pissoff ":                "pissoff ",
	"poop":                    "poop",
	"porn":                    "porn",
	"porno":                   "porno",
	"pornography":             "pornography",
	"pornos":                  "pornos",
	"prick":                   "prick",
	"pricks ":                 "pricks ",
	"pron":                    "pron",
	"pube":                    "pube",
	"pusse":                   "pusse",
	"pussi":                   "pussi",
	"pussies":                 "pussies",
	"pussy":                   "pussy",
	"pussys ":                 "pussys ",
	"rectum":                  "rectum",
	"retard":                  "retard",
	"rimjaw":                  "rimjaw",
	"rimming":                 "rimming",
	"s hit":                   "s hit",
	"s.o.b.":                  "s.o.b.",
	"sadist":                  "sadist",
	"schlong":                 "schlong",
	"screwing":                "screwing",
	"scroat":                  "scroat",
	"scrote":                  "scrote",
	"scrotum":                 "scrotum",
	"semen":                   "semen",
	"sex":                     "sex",
	"sh!+":                    "sh!+",
	"sh!t":                    "sh!t",
	"sh1t":                    "sh1t",
	"shag":                    "shag",
	"shagger":                 "shagger",
	"shaggin":                 "shaggin",
	"shagging":                "shagging",
	"shemale":                 "shemale",
	"shi+":                    "shi+",
	"shit":                    "shit",
	"shitdick":                "shitdick",
	"shite":                   "shite",
	"shited":                  "shited",
	"shitey":                  "shitey",
	"shitfuck":                "shitfuck",
	"shitfull":                "shitfull",
	"shithead":                "shithead",
	"shiting":                 "shiting",
	"shitings":                "shitings",
	"shits":                   "shits",
	"shitted":                 "shitted",
	"shitter":                 "shitter",
	"shitters ":               "shitters ",
	"shitting":                "shitting",
	"shittings":               "shittings",
	"shitty ":                 "shitty ",
	"skank":                   "skank",
	"slut":                    "slut",
	"sluts":                   "sluts",
	"smegma":                  "smegma",
	"smut":                    "smut",
	"snatch":                  "snatch",
	"son-of-a-bitch":          "son-of-a-bitch",
	"spac":                    "spac",
	"spunk":                   "spunk",
	"s_h_i_t":                 "s_h_i_t",
	"t1tt1e5":                 "t1tt1e5",
	"t1tties":                 "t1tties",
	"teets":                   "teets",
	"teez":                    "teez",
	"testical":                "testical",
	"testicle":                "testicle",
	"tit":                     "tit",
	"titfuck":                 "titfuck",
	"tits":                    "tits",
	"titt":                    "titt",
	"tittie5":                 "tittie5",
	"tittiefucker":            "tittiefucker",
	"titties":                 "titties",
	"tittyfuck":               "tittyfuck",
	"tittywank":               "tittywank",
	"titwank":                 "titwank",
	"tosser":                  "tosser",
	"turd":                    "turd",
	"tw4t":                    "tw4t",
	"twat":                    "twat",
	"twathead":                "twathead",
	"twatty":                  "twatty",
	"twunt":                   "twunt",
	"twunter":                 "twunter",
	"v14gra":                  "v14gra",
	"v1gra":                   "v1gra",
	"vagina":                  "vagina",
	"viagra":                  "viagra",
	"vulva":                   "vulva",
	"w00se":                   "w00se",
	"wang":                    "wang",
	"wank":                    "wank",
	"wanker":                  "wanker",
	"wanky":                   "wanky",
	"whoar":                   "whoar",
	"whore":                   "whore",
	"willies":                 "willies",
	"willy":                   "willy",
	"xrated":                  "xrated",
	"xxx":                     "xxx",
}
