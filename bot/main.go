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
	"strings"
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
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Step      int    `json:"step"`
	Tg_id     int    `json:"tg_id"`
	Order     Order  `json:"order"`
}

type Order struct {
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	RegDate          string    `json:"reg_date"`
	OrganizationName string    `json:"organization_name"`
	Products         []Product `json:"product"`
	Shipment         string    `json:"shipment"`
	Supply           string    `json:"supply"`
	Comment          string    `json:"comment"`
}

type Product struct {
	Marketplace       string   `json:"marketplace"`
	Type              string   `json:"type"`
	Size              string   `json:"size"`
	Weight            string   `json:"weight"`
	Description       string   `json:"color"`
	Count             string   `json:"count"`
	SelectedPackaging []string `json:"selected_packaging"`
	Marking           string   `json:"marking"`
	Barcode           string   `json:"barcode"`
}

// переменные для подключения к боту
var host string = "https://api.telegram.org/bot"
var token string = os.Getenv("BOT_TOKEN")

// данные всеx пользователей
var usersDB map[int]UserT
var newProduct = Product{}

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
	button := messageInline.CallbackQuery.Data

	//есть ли юзер
	_, exist := usersDB[chatId]
	if !exist {
		user := UserT{}
		user.ID = chatId
		user.FirstName = firstName
		user.LastName = lastName
		user.Tg_id = chatId
		user.Step = 1

		usersDB[chatId] = user

	}

	file, _ := os.Create("db.json")
	jsonString, _ := json.Marshal(usersDB)
	file.Write(jsonString)

	switch {
	// кейс для начального сообщения для пользователя
	case text == "/start" || usersDB[chatId].Step == 1 || text == "Заказать ещё":

		user := usersDB[chatId]
		user.Step = 1
		usersDB[chatId] = user

		// Отправляем сообщение с клавиатурой и перезаписываем шаг
		sendMessage(chatId, "Здраствуйте! Добро пожаловать в <Название бота>. Введите название организации (ИП или ООО)", nil)

		user.Order.FirstName = firstName
		user.Order.LastName = lastName
		user.Step += 1
		usersDB[chatId] = user
		break

	//кейс для выбора маркетплейса
	case usersDB[chatId].Step == 2:

		newProduct = Product{}

		if strings.Contains(text, "ИП") || strings.Contains(text, "ООО") {

			user := usersDB[chatId]

			user.Order.OrganizationName = text

			//собираем объект клавиатуры для выбора языка
			buttons := [][]map[string]interface{}{
				{{"text": "Ozon", "callback_data": "ozon"}},
				{{"text": "Wildberris", "callback_data": "wildberris"}},
				{{"text": "Yandex", "callback_data": "yandex"}},
			}

			inlineKeyboard := map[string]interface{}{
				"inline_keyboard": buttons,
			}

			sendMessage(chatId, "Выберите маркетплейс", inlineKeyboard)

			user.Step += 1
			usersDB[chatId] = user

		} else {
			sendMessage(chatId, "Имя организации должно начинаться с ИП или ООО", nil)
		}

		break

	case usersDB[chatId].Step == 3:

		user := usersDB[chatId]
		// Создаем новый экземпляр Product
		newProduct = Product{
			Marketplace: button,
		}
		user.Order.Products = append(user.Order.Products, newProduct)

		sendMessage(chatId, "Введите вид товара. Например: Худи Найк, MacBook, Телевизор Sumsung и др", nil)

		user.Step += 1
		usersDB[chatId] = user
		break

	case button == "another":
		sendMessage(chatId, "Введите нужную информацию", nil)

	case usersDB[chatId].Step == 4:

		user := usersDB[chatId]
		productIndex := len(user.Order.Products) - 1
		if text != "" {
			user.Order.Products[productIndex].Type = text
		} else {
			user.Order.Products[productIndex].Type = button
		}

		sendMessage(chatId, "Отправьте размеры товара в см (120Х48)", nil)

		user.Step += 1
		usersDB[chatId] = user
		break

	case usersDB[chatId].Step == 5:

		user := usersDB[chatId]
		productIndex := len(user.Order.Products) - 1
		user.Order.Products[productIndex].Size = text

		sendMessage(chatId, "Отправьте вес товара в кг (3кг)", nil)

		user.Step += 1
		usersDB[chatId] = user
		break

	case usersDB[chatId].Step == 6:

		user := usersDB[chatId]
		productIndex := len(user.Order.Products) - 1
		user.Order.Products[productIndex].Weight = text

		sendMessage(chatId, "Введите характеристики товара. Например: белый цвет, Оперативная память 16Гб и тд", nil)

		user.Step += 1
		usersDB[chatId] = user
		break

	case usersDB[chatId].Step == 7:

		user := usersDB[chatId]
		productIndex := len(user.Order.Products) - 1
		if text != "" {
			user.Order.Products[productIndex].Description = text
		} else {
			user.Order.Products[productIndex].Description = button
		}

		sendMessage(chatId, "Отправьте количество товара", nil)

		user.Step += 1
		usersDB[chatId] = user
		break

	case usersDB[chatId].Step == 8 || button == "yes":
		user := usersDB[chatId]

		if button == "" {
			productIndex := len(user.Order.Products) - 1
			user.Order.Products[productIndex].Count = text
		}
		user.Step = 8
		usersDB[chatId] = user

		// Собираем объект клавиатуры для выбора упаковки
		buttons := [][]map[string]interface{}{
			{{"text": "Пупырчатая пленка", "callback_data": "Пупырчатая пленка"}},
			{{"text": "Стрейч пленка", "callback_data": "Стрейч пленка"}},
			{{"text": "Коробка", "callback_data": "Коробка"}},
			{{"text": "Термоусадочный пакет", "callback_data": "Термоусадочный пакет"}},
			{{"text": "ПВД рукав", "callback_data": "ПВД рукав"}},
			{{"text": "БОПП пакет", "callback_data": "БОПП пакет"}},
			{{"text": "Почтовый пакет", "callback_data": "Почтовый пакет"}},
			{{"text": "ЗИП-лок пакет", "callback_data": "ЗИП-лок пакет"}},
		}

		inlineKeyboard := map[string]interface{}{
			"inline_keyboard": buttons,
		}

		sendMessage(chatId, "Выберите упаковку", inlineKeyboard)

		user.Step += 1
		usersDB[chatId] = user
		break

	case button == "Коробка" || button == "БОПП пакет" || button == "Почтовый пакет" || button == "ЗИП-лок пакет":

		sendMessage(chatId, "Введите требуемые размеры выбранной упаковки в см", nil)

		user := usersDB[chatId]

		productIndex := len(user.Order.Products) - 1
		user.Order.Products[productIndex].SelectedPackaging = append(user.Order.Products[productIndex].SelectedPackaging, button)

		user.Step = 9
		usersDB[chatId] = user
		break

	case button == "no":
		user := usersDB[chatId]

		buttons := [][]map[string]interface{}{
			{{"text": "Маркировка на упаковку", "callback_data": "Марк на упаковку"}},
			{{"text": "Маркировка на товар и упаковку", "callback_data": " Марк на товар и упаковку"}},
			{{"text": "Честный знак на упаковку", "callback_data": "чз на упаковку"}},
			{{"text": "Честный знак на бирку", "callback_data": "чз на бирку"}},
			{{"text": "Честный знак на бирку и упаковку", "callback_data": "чз на бирку и упаковку"}},
		}

		inlineKeyboard := map[string]interface{}{
			"inline_keyboard": buttons,
		}

		sendMessage(chatId, "Выберите подходящую маркировку", inlineKeyboard)
		fmt.Println(usersDB[chatId].Step)
		user.Step += 1
		usersDB[chatId] = user
		fmt.Println(usersDB[chatId].Step)
		break

	case usersDB[chatId].Step == 9:

		user := usersDB[chatId]

		if text != "" {
			productIndex := len(user.Order.Products) - 1
			packageIndex := len(user.Order.Products[productIndex].SelectedPackaging) - 1
			user.Order.Products[productIndex].SelectedPackaging[packageIndex] += " " + text
		} else {
			productIndex := len(user.Order.Products) - 1
			user.Order.Products[productIndex].SelectedPackaging = append(user.Order.Products[productIndex].SelectedPackaging, button)
		}

		usersDB[chatId] = user

		// Создайте объект клавиатуры для выбора между "Да" и "Нет"
		buttons := [][]map[string]interface{}{
			{{"text": "Да", "callback_data": "yes"}},
			{{"text": "Нет", "callback_data": "no"}},
		}

		inlineKeyboard := map[string]interface{}{
			"inline_keyboard": buttons,
		}

		sendMessage(chatId, "Нужна ещё одна упаковка?", inlineKeyboard)
		break

	case usersDB[chatId].Step == 10:

		user := usersDB[chatId]

		productIndex := len(user.Order.Products) - 1
		user.Order.Products[productIndex].Marking = button

		if user.Order.Products[productIndex].Marketplace == "yandex" || user.Order.Products[productIndex].Marketplace == "wildberris" {
			sendMessage(chatId, "Отправьте числовой баркод из маркетплейса", nil)
		} else {
			sendMessage(chatId, "Отправьте файл pdf c баркодом из маркетплейса", nil)
		}

		user.Step += 1
		usersDB[chatId] = user
		break

	case usersDB[chatId].Step == 11:

		user := usersDB[chatId]

		productIndex := len(user.Order.Products) - 1
		user.Order.Products[productIndex].Barcode = text

		// Собираем объект клавиатуры для выбора упаковки
		buttons := [][]map[string]interface{}{
			{{"text": "Добавить", "callback_data": "add_good"}},
			{{"text": "Дальше", "callback_data": "continue"}},
		}

		inlineKeyboard := map[string]interface{}{
			"inline_keyboard": buttons,
		}

		sendMessage(chatId, "Добавить ещё товар?", inlineKeyboard)

		user.Step += 1
		usersDB[chatId] = user
		break

	case usersDB[chatId].Step == 12 && button == "add_good":
		user := usersDB[chatId]
		user.Step = 2
		usersDB[chatId] = user

		//собираем объект клавиатуры для выбора языка
		buttons := [][]map[string]interface{}{
			{{"text": "Ozon", "callback_data": "ozon"}},
			{{"text": "Wildberris", "callback_data": "wildberris"}},
			{{"text": "Yandex", "callback_data": "yandex"}},
		}

		inlineKeyboard := map[string]interface{}{
			"inline_keyboard": buttons,
		}

		sendMessage(chatId, "Выберите маркетплейс", inlineKeyboard)

		user.Step += 1
		usersDB[chatId] = user

	case usersDB[chatId].Step == 12 && button == "continue":

		user := usersDB[chatId]

		sendMessage(chatId, "Когда планиурется отгрузка", nil)

		user.Step += 1
		usersDB[chatId] = user

	case usersDB[chatId].Step == 14:
		user := usersDB[chatId]
		user.Order.Shipment = text

		//собираем объект клавиатуры для выбора языка
		buttons := [][]map[string]interface{}{
			{{"text": "Клиент формирует поставку сам", "callback_data": "формирует сам"}},
			{{"text": "Формирование поставки нашими силами", "callback_data": "нашими силами"}},
		}

		inlineKeyboard := map[string]interface{}{
			"inline_keyboard": buttons,
		}

		sendMessage(chatId, "Дата поставки на маркетплейс", inlineKeyboard)

		user.Step += 1
		usersDB[chatId] = user
		break

	case usersDB[chatId].Step == 14:
		user := usersDB[chatId]
		user.Order.Supply = button

		//собираем объект клавиатуры для выбора языка
		buttons := [][]map[string]interface{}{
			{{"text": "Нет", "callback_data": "finish"}},
		}

		inlineKeyboard := map[string]interface{}{
			"inline_keyboard": buttons,
		}

		sendMessage(chatId, "Особые коментарии к заказу", inlineKeyboard)
		user.Step += 1
		usersDB[chatId] = user
		break

	case usersDB[chatId].Step == 15:

		user := usersDB[chatId]
		user.Order.Comment = text
		// Получите текущее время
		currentTime := time.Now()

		// Определите, сколько часов вы хотите прибавить
		hoursToAdd := 3

		// Прибавьте указанное количество часов
		newTime := currentTime.Add(time.Duration(hoursToAdd) * time.Hour)

		// Определите желаемый формат времени
		format := "2006-01-02 15:04" // Например, "год-месяц-день час:минута"

		// Преобразуйте новое время в строку с помощью метода Format
		formattedTime := newTime.Format(format)

		user.Order.RegDate = formattedTime
		// Создаем объект клавиатуры
		keyboard := map[string]interface{}{
			"keyboard": [][]map[string]interface{}{
				{
					{
						"text": "Заказать ещё",
					},
				},
			},
			"resize_keyboard":   true,
			"one_time_keyboard": true,
		}
		sendMessage(chatId, "Спасибо за Ваш заказ. Для ещё одного заказа нажмите на кнопку ниже", keyboard)
		user.Step = 1
		usersDB[chatId] = user

		file, _ := os.Create("db.json")
		jsonString, _ := json.Marshal(usersDB)
		file.Write(jsonString)

		break

	}

}

// // //сохраняем в файл для отдачи данных
// // fileU, _ := os.Create("orders.json")
// // data, _ := json.Marshal(user)
// // fileU.Write(data)
