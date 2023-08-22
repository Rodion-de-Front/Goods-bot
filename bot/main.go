package main

//подключение требуемых пакетов
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

// структура для приходящих сообщений и обычных кнопок
type ResponseT struct {
	Ok     bool       `json:"ok"`
	Result []MessageT `json:"result"`
}

type MessageT struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID           int    `json:"id"`
			IsBot        bool   `json:"is_bot"`
			FirstName    string `json:"first_name"`
			LastName     string `json:"last_name"`
			Username     string `json:"username"`
			LanguageCode string `json:"language_code"`
		} `json:"from"`
		Chat struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Username  string `json:"username"`
			Type      string `json:"type"`
		} `json:"chat"`
		Date    int `json:"date"`
		Contact struct {
			PhoneNumber string `json:"phone_number"`
		} `json:"contact"`
		Location struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"location"`
		Text string `json:"text"`
		Data string `json:"data"`
	} `json:"message"`
}

// структура для инлайн кнопок
type ResponseInlineT struct {
	Ok     bool             `json:"ok"`
	Result []MessageInlineT `json:"result"`
}

type MessageInlineT struct {
	UpdateID      int `json:"update_id"`
	CallbackQuery struct {
		ID   string `json:"id"`
		From struct {
			ID           int    `json:"id"`
			IsBot        bool   `json:"is_bot"`
			FirstName    string `json:"first_name"`
			Username     string `json:"username"`
			LanguageCode string `json:"language_code"`
		} `json:"from"`
		Message struct {
			MessageID int `json:"message_id"`
			From      struct {
				ID        int64  `json:"id"`
				IsBot     bool   `json:"is_bot"`
				FirstName string `json:"first_name"`
				Username  string `json:"username"`
			} `json:"from"`
			Chat struct {
				ID        int    `json:"id"`
				FirstName string `json:"first_name"`
				Username  string `json:"username"`
				Type      string `json:"type"`
			} `json:"chat"`
			Date        int    `json:"date"`
			Text        string `json:"text"`
			ReplyMarkup struct {
				InlineKeyboard [][]struct {
					Text         string `json:"text"`
					CallbackData string `json:"callback_data"`
				} `json:"inline_keyboard"`
			} `json:"reply_markup"`
		} `json:"message"`
		ChatInstance string `json:"chat_instance"`
		Data         string `json:"data"`
	} `json:"callback_query"`
}

// структура пользователя
type UserT struct {
	ID          int    `json:"id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Username    string `json:"tg_username"`
	Step        int    `json:"step"`
	Tg_id       int    `json:"tg_id"`
	PhoneNumber string `json:"phone"`
}

// переменные для подключения к боту
var host string = "https://api.telegram.org/bot"
var token string = os.Getenv("BOT_TOKEN")

// данные всеx пользователей
var usersDB map[int]UserT

// главная функция работы бота
func main() {

	//достаем юзеров из кэша
	getUsers()

	//обнуление последнего id сообщения
	lastMessage := 0

	//цикл для проверки на наличие новых сообщений
	for range time.Tick(time.Second * 1) {

		//отправляем запрос к Telegram API на получение сообщений
		var url string = host + token + "/getUpdates?offset=" + strconv.Itoa(lastMessage)
		response, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
		}
		data, _ := ioutil.ReadAll(response.Body)

		//посмотреть данные
		fmt.Println(string(data))

		//парсим данные из json
		var responseObj ResponseT
		json.Unmarshal(data, &responseObj)

		//парсим данные из json  (для нажатия на инлайн кнопку)
		var need ResponseInlineT
		json.Unmarshal(data, &need)

		//считаем количество новых сообщений
		number := len(responseObj.Result)

		//если сообщений нет - то дальше код не выполняем
		if number < 1 {
			continue
		}

		//в цикле доставать инормацию по каждому сообщению
		for i := 0; i < number; i++ {

			//обработка одного сообщения
			go processMessage(responseObj.Result[i], need.Result[i])
		}

		//запоминаем update_id  последнего сообщения
		lastMessage = responseObj.Result[number-1].UpdateID + 1

	}
}

func getUsers() {
	//считываем из бд при включении
	dataFile, _ := ioutil.ReadFile("db.json")
	json.Unmarshal(dataFile, &usersDB)
}

// функция для отправки POST запроса
func sendPost(requestBody string, url string) ([]byte, error) {
	// Создаем новый POST-запрос
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, fmt.Errorf("Ошибка при создании запроса: %v", err)
	}

	// Устанавливаем заголовок Content-Type для указания типа данных в теле запроса
	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос с использованием стандартного клиента HTTP
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем код состояния HTTP-ответа
	if resp.StatusCode == http.StatusOK {
		// Успешный запрос, читаем тело ответа
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("Ошибка при чтении тела ответа: %v", err)
		}
		return body, nil
	} else {
		// Обработка ошибки при некорректном статусе HTTP-ответа
		return nil, fmt.Errorf("Некорректный код состояния HTTP: %s", resp.Status)
	}
}

// функция для отправки сообщения пользователю
func sendMessage(chatId int, text string, keyboard map[string]interface{}) {
	url := host + token + "/sendMessage?chat_id=" + strconv.Itoa(chatId) + "&text=" + text
	if keyboard != nil {
		// Преобразуем клавиатуру в JSON
		keyboardJSON, _ := json.Marshal(keyboard)
		url += "&reply_markup=" + string(keyboardJSON)
	}
	http.Get(url)
}

func processMessage(message MessageT, messageInline MessageInlineT) {

	text := message.Message.Text
	chatId := 0
	if messageInline.CallbackQuery.From.ID == 0 {
		chatId = message.Message.From.ID
	} else {
		chatId = messageInline.CallbackQuery.From.ID
	}

	firstName := message.Message.From.FirstName
	lastName := message.Message.From.LastName
	phone := message.Message.Contact.PhoneNumber
	username := message.Message.From.Username
	//button := messageInline.CallbackQuery.Data

	//есть ли юзер
	_, exist := usersDB[chatId]
	if !exist {
		user := UserT{}
		user.ID = chatId
		user.FirstName = firstName
		user.LastName = lastName
		user.Username = username
		user.Tg_id = chatId
		user.PhoneNumber = phone
		user.Step = 1

		usersDB[chatId] = user

	}

	file, _ := os.Create("db.json")
	jsonString, _ := json.Marshal(usersDB)
	file.Write(jsonString)

	switch {
	// кейс для начального сообщения для пользователя
	case text == "/start" || usersDB[chatId].Step == 1:

		user := usersDB[chatId]

		// Отправляем сообщение с клавиатурой и перезаписываем шаг
		sendMessage(chatId, text, nil)

		user.Step += 1
		usersDB[chatId] = user
		break
	}

}
