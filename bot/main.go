package main

//подключение требуемых пакетов
import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"context"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"google.golang.org/api/drive/v3"
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
		Date     int `json:"date"`
		Document struct {
			FileName  string `json:"file_name"`
			MimeType  string `json:"mime_type"`
			Thumbnail struct {
				FileID       string `json:"file_id"`
				FileUniqueID string `json:"file_unique_id"`
				FileSize     int    `json:"file_size"`
				Width        int    `json:"width"`
				Height       int    `json:"height"`
			} `json:"thumbnail"`
			Thumb struct {
				FileID       string `json:"file_id"`
				FileUniqueID string `json:"file_unique_id"`
				FileSize     int    `json:"file_size"`
				Width        int    `json:"width"`
				Height       int    `json:"height"`
			} `json:"thumb"`
			FileID       string `json:"file_id"`
			FileUniqueID string `json:"file_unique_id"`
			FileSize     int    `json:"file_size"`
		} `json:"document"`
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
	Description       string   `json:"description"`
	Count             string   `json:"count"`
	SelectedPackaging []string `json:"selected_packaging"`
	Marking           string   `json:"marking"`
	Barcode           string   `json:"barcode"`
}

// переменные для подключения к боту
var host string = "https://api.telegram.org/bot"

var token string = os.Getenv("BOT_TOKEN")

// переменная для тг канала
var channelName string = os.Getenv("CHANEL_NAME")

// Идентификатор таблицы, в которую будут записываться данные
var spreadsheetID string = os.Getenv("SPREAD_SHEET_ID")

// Замените "Sheet1" на название листа, в который вы хотите добавить данные.
var sheetName string = os.Getenv("SHEET_NAME")

var srv *drive.Service

// данные всеx пользователей
var usersDB map[int]UserT
var newProduct = Product{}

// главная функция работы бота
func main() {

	//достаем юзеров из кэша
	getUsers()

	//обнуление последнего id сообщения
	lastMessage := 0

	ctx := context.Background()
	b, err := os.ReadFile("client.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, _ = drive.NewService(ctx, option.WithHTTPClient(client))

	r, err := srv.Files.List().PageSize(10).
		Fields("nextPageToken, files(id, name)").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}
	fmt.Println("Files:")
	if len(r.Files) == 0 {
		fmt.Println("No files found.")
	} else {
		for _, i := range r.Files {
			fmt.Printf("%s (%s)\n", i.Name, i.Id)
		}
	}

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

// функция для отправки сообщения в канал
func sendMessageToChanel(apiURL string) {
	fmt.Println("sendMessage")
	requestURL, err := url.Parse(apiURL)
	if err != nil {
		log.Fatal(err)
	}

	// Создание HTTP GET-запроса с параметрами
	request, err := http.NewRequest("GET", requestURL.String(), nil)
	if err != nil {
		log.Fatal(err)
	}

	// Отправка запроса
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	// Чтение ответа
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Вывод конечной ссылки запроса
	finalURL := request.URL.String()
	fmt.Println("Final URL:", finalURL)

	// Вывод ответа от сервера
	fmt.Println("Response:", string(responseData))
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
	file_id := message.Message.Document.FileID

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
		if user.Order.Products[productIndex].Marketplace == "yandex" || user.Order.Products[productIndex].Marketplace == "wildberris" {
			user.Order.Products[productIndex].Barcode = text
		} else {

			// Создайте URL для получения метаданных файла.
			metadataURL := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", token, file_id)

			// Отправьте GET-запрос для получения метаданных файла.
			resp, err := http.Get(metadataURL)
			if err != nil {
				log.Fatalf("Ошибка при запросе метаданных файла: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("Ошибка: %v", resp.Status)
			}

			// Распарсите JSON-ответ, чтобы получить путь к файлу.
			var fileInfo struct {
				Ok     bool `json:"ok"`
				Result struct {
					FilePath string `json:"file_path"`
				} `json:"result"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&fileInfo); err != nil {
				log.Fatalf("Ошибка при разборе JSON: %v", err)
			}

			// Укажите путь к папке, в которую вы хотите сохранить файл.
			targetFolder := "barcodes" // Укажите свой путь к папке

			// Проверьте, существует ли указанная папка, и создайте ее, если нет.
			if err := os.MkdirAll(targetFolder, os.ModePerm); err != nil {
				log.Fatalf("Ошибка при создании папки: %v", err)
			}

			// Определите имя файла на основе его пути.
			_, fileName := filepath.Split(fileInfo.Result.FilePath)

			// Создайте локальный файл в указанной папке для сохранения.
			localFilePath := filepath.Join(targetFolder, fileName)
			localFile, err := os.Create(localFilePath)
			if err != nil {
				log.Fatalf("Ошибка при создании локального файла: %v", err)
			}
			defer localFile.Close()

			// Создайте URL для скачивания файла.
			fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", token, fileInfo.Result.FilePath)

			// Отправьте GET-запрос для скачивания файла.
			resp, err = http.Get(fileURL)
			if err != nil {
				log.Fatalf("Ошибка при скачивании файла: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				log.Fatalf("Ошибка: %v", resp.Status)
			}

			// Скопируйте содержимое файла из ответа в локальный файл.
			_, err = io.Copy(localFile, resp.Body)
			if err != nil {
				log.Fatalf("Ошибка при копировании содержимого: %v", err)
			}

			fmt.Printf("Файл успешно сохранен в %s\n", localFilePath)

			file, err := os.Open(localFilePath)
			if err != nil {
				log.Fatalln(err)
			}

			stat, err := file.Stat()
			if err != nil {
				log.Fatalln(err)
			}
			defer file.Close()

			res, err := srv.Files.Create(
				&drive.File{
					Parents: []string{"1OaWHZirxpar-eGkrSmwGDGc8CZ77N83o"},
					Name:    "barcode.pdf",
				},
			).Media(file, googleapi.ChunkSize(int(stat.Size()))).Do()
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Printf("%s\n", res.Id)

			srv.Permissions.Create(res.Id, &drive.Permission{
				Role: "reader",
				Type: "anyone",
			}).Do()

			user.Order.Products[productIndex].Barcode = "https://drive.google.com/file/d/" + res.Id

			fmt.Println("https://drive.google.com/file/d/" + res.Id)

		}

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

	case usersDB[chatId].Step == 13:
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

		//переменая с уникальным номером заявки
		ticketNumber := generateUniqueTicketNumber()

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

		// Загрузите файл учетных данных вашего проекта Google Cloud в переменную `credentials`.
		credentials, err := ioutil.ReadFile("credentials.json")
		if err != nil {
			log.Fatalf("Unable to read client secret file: %v", err)
		}

		// Извлеките конфигурацию OAuth2 из файла учетных данных и получите токен доступа.
		config, err := google.JWTConfigFromJSON(credentials, sheets.SpreadsheetsScope)
		if err != nil {
			log.Fatalf("Unable to parse client secret file to config: %v", err)
		}
		client := config.Client(context.Background())

		// Создайте клиент Google Sheets API.
		sheetsService, err := sheets.New(client)
		if err != nil {
			log.Fatalf("Unable to retrieve Sheets client: %v", err)
		}

		// Загрузите файл с данными в формате JSON.
		content, err := ioutil.ReadFile("db.json")
		if err != nil {
			log.Fatalf("Unable to read data file: %v", err)
		}

		// Распарсите JSON и извлеките данные.
		var jsonData map[string]interface{}
		if err := json.Unmarshal(content, &jsonData); err != nil {
			log.Fatalf("Unable to parse JSON data: %v", err)
		}

		fmt.Println(jsonData)

		// Создайте список значений для новой строки.
		var values [][]interface{}

		// Проверяем существование и не nil поля "order" в jsonData
		if orderData, ok := jsonData[strconv.Itoa(chatId)].(map[string]interface{}); ok && orderData != nil {

			//текст сообщения в канал
			chanText := "Заявка №" + ticketNumber + " от " + orderData["order"].(map[string]interface{})["reg_date"].(string) + "\n\nПринял: " + orderData["first_name"].(string) + " " + orderData["last_name"].(string) + "\n\nНаименование клиента: " +
				orderData["order"].(map[string]interface{})["organization_name"].(string) + "\n\n"

			// Проход по каждому продукту в массиве product
			for _, product := range orderData["order"].(map[string]interface{})["product"].([]interface{}) {
				p := product.(map[string]interface{})

				// Преобразование selected_packaging в []string
				var packaging []string
				if selectedPackaging, ok := p["selected_packaging"].([]interface{}); ok {
					for _, item := range selectedPackaging {
						packaging = append(packaging, item.(string))
					}
				}

				row := []interface{}{
					ticketNumber,
					orderData["first_name"],
					orderData["last_name"],
					orderData["order"].(map[string]interface{})["reg_date"],
					orderData["order"].(map[string]interface{})["organization_name"],
					p["marketplace"],
					p["type"],
					p["size"],
					p["weight"],
					p["count"],
					p["description"],
					p["marking"],
					p["barcode"],
					strings.Join(packaging, ", "),
					orderData["order"].(map[string]interface{})["shipment"],
					orderData["order"].(map[string]interface{})["supply"],
					orderData["order"].(map[string]interface{})["comment"],
				}

				values = append(values, row)

				chanText += "Отгрузка на " + p["marketplace"].(string) + "\n\nТовары:\n\n" + p["type"].(string) + ". Размер: " + p["size"].(string) + ". Вес: " + p["weight"].(string) + ". Количество: " + p["count"].(string) + ". Описание: " + p["description"].(string) + ". Маркировка: " +
					p["marking"].(string) + ". Баркод: " + p["barcode"].(string) + ". Упаковка: " + strings.Join(packaging, ", ") + "\n\n"
			}

			chanText += "Отгрузка: " + orderData["order"].(map[string]interface{})["shipment"].(string) + "\n\nДата отгрузки: " + orderData["order"].(map[string]interface{})["supply"].(string) + "\n\nОсобые отметки: " + orderData["order"].(map[string]interface{})["comment"].(string)

			apiURL := "https://api.telegram.org/bot" + token + "/sendMessage?chat_id=" + url.QueryEscape(channelName) + "&text=" + url.QueryEscape(chanText)

			sendMessageToChanel(apiURL)

			// Создайте объект ValueRange для добавления новых строк.
			rangeValue := fmt.Sprintf("%s!A2:R", sheetName)
			vr := sheets.ValueRange{Values: values, MajorDimension: "ROWS"}
			_, err = sheetsService.Spreadsheets.Values.Append(spreadsheetID, rangeValue, &vr).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Do()
			if err != nil {
				log.Fatalf("Unable to append values: %v", err)
			}
			fmt.Println("Values appended successfully.")
		} else {
			fmt.Println("Order data not found or is nil.")
		}

		user.Order = Order{}
		usersDB[chatId] = user

		file, _ = os.Create("db.json")
		jsonString, _ = json.Marshal(usersDB)
		file.Write(jsonString)

		break

	}

}

// функция для генерации случайных номеров заявок
func generateUniqueTicketNumber() string {
	// Получите текущее время в миллисекундах
	currentTimeMillis := time.Now().UnixNano() / 1e6

	// Сгенерируйте случайное число в заданном диапазоне (например, от 1000 до 9999)
	rand.Seed(time.Now().UnixNano())
	randomPart := rand.Intn(9000) + 1000

	// Объедините текущее время и случайное число для создания уникального номера
	uniqueNumber := fmt.Sprintf("%d%d", currentTimeMillis, randomPart)

	return uniqueNumber
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
