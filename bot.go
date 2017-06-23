// bot.go
package main

import (
	"fmt"
	"hash/adler32"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Syfaro/telegram-bot-api"
)

var bot tgbotapi.BotAPI // a bot global variable

//snapper takes a snap of a webpage and return its adler32 hash
func snapper(url string) uint32 {
	resp, err := http.Get(url)
	if err != nil {
		return 0
	}
	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return 0
	}
	if strings.Contains(string(b), "Некорректные") {
		return 0
	}
	fmt.Println(b)
	return adler32.Checksum(b)
}

/*
Бот умеет чекать изменение содержимого интернет-страницы (например, на сайте ОРЦОКО). Если ты хочешь чекнуть ЕГЭ, то тапай /register. Бот *не сохраняет* и *не передает* никакие сведения о вас. Все данные уничтожаются после его закрытия.
*/

//Commands defs for regular user:
/*
	/register - запросит вашу фамилию и паспорт для регистрации на ОРЦОКО
	/check - начнет чекать ссылку
	/snap - запросит ссылку для чека. ДЛя работы с ОРЦОКО эта комнада НЕ НУЖНА
*/

func main() {
	bot, err := tgbotapi.NewBotAPI("411682696:AAHNvFpLyqFU9YlMvIVocv0KrK_rF6RSA7w")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	//Data structures: (all keyed by  user ID)
	//oldSnap := make(map[int]uint32)          // snap to compare other snaps with
	curUrl := make(map[int]string)           // stores current url
	isUrlRequired := make(map[int]bool)      // checks whether url was requested by "/snap"
	isRegisterRequired := make(map[int]bool) // checks whether name and pass number are needed to be passed
	requests := make(map[int]uint32)         // map stores the status of user's request keyed by id
	//TODO checking loop
	for update := range updates {
		if update.Message == nil {
			continue
		}

		sender := func(m string) {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, m)
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
		}
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		// check results in this loop
		for id, snap := range requests {
			if snap != snapper(curUrl[id]) {
				sender("Дождались! Смотри здесь: " + curUrl[id])
				/*maps : = []*map[int]interface{}{&curUrl, &isUrlRequired, &isRegisterRequired, &requests}
				for _, pointer := range maps{
					delete(*pointer, id)
				}*/
				delete(curUrl, id)
				delete(isUrlRequired, id)
				delete(isRegisterRequired, id)
				delete(requests, id)
			}
		}

		/*checkSuggestion() encourages user to press "/check"
		checkSuggestion := func() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "теперь тапай /check")
			msg.ReplyToMessageID = update.Message.MessageID
			bot.Send(msg)
		}*/
		curUserId := update.Message.From.ID
		switch {
		case isUrlRequired[curUserId]:
			isUrlRequired[curUserId] = false
			switch {
			case strings.HasPrefix(update.Message.Text, "http://"):
				curUrl[curUserId] = update.Message.Text
			default:
				curUrl[curUserId] = "http://" + update.Message.Text
			}
			requests[curUserId] = snapper(curUrl[curUserId])
			sender("твой запрос принят, жди обновлений")
			//sender(fmt.Sprint(oldSnap[curUserId]))
			//checkSuggestion()

		case strings.HasPrefix(update.Message.Text, "/snap"):
			sender("введи ссылку (желательно скопировать прямо из броузера)")
			isUrlRequired[curUserId] = true
		case isRegisterRequired[curUserId]:
			words := strings.Split(update.Message.Text, " ")
			v := url.Values{}
			v.Set("number", words[1])
			v.Set("surname", words[0])
			curUrl[curUserId] = "http://orcoko.ru/ege/results-ege/?" + v.Encode()
			isRegisterRequired[curUserId] = false // register completed
			snap := snapper(curUrl[curUserId])
			//check whether snap was taken successfully
			if snap == 0 {
				sender("перепроверь свои данные и попробуй еще /register")
				continue
			}
			requests[curUserId] = snap
			sender("Твой запрос принят, жди обновлений")
			//checkSuggestion()

		case update.Message.Text == "/register":
			sender("введите фамилию и номер паспорта через пробел")
			isRegisterRequired[curUserId] = true

			/*this case used to be important
			case update.Message.Text == "/check":
				sender("начинаю чекать. я напишу тебе, если что-то случится. но помни, без перерыва я могу работать только 6 часов!")
				newSnap := snapper(curUrl[curUserId])
				startTime := time.Now()
				for (newSnap == oldSnap[curUserId]) && (time.Since(startTime) < time.Duration(6*time.Hour)) {
					newSnap = snapper(curUrl[curUserId])
				}
				sender("кажется, что-то изменилось проверь на " + curUrl[curUserId])*/
		}
	}
}
