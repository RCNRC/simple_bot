package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
)

// Update - представляет собой тип обновления в чате, которе происходит в момент отправки сообщения
type Update struct {
	UpdateID int            `json:"update_id"`
	Message  *IncomeMessage `json:"message"`
}

// IncomeMessage - представляет собой тип входящего сообщения
type IncomeMessage struct {
	ID   int    `json:"message_id"`
	Chat Chat   `json:"chat"`
	Text string `json:"text"`
}

// Chat - представляет собой тип чата из входящего сообщения
type Chat struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Type     string `json:"type"`
}

// OutcomeMessage - выходящее сообщение, которое получит пользователь
type OutcomeMessage struct {
	ChatID      int         `json:"chat_id"`
	Text        string      `json:"text"`
	ReplyMarkup ReplyMarkup `json:"reply_markup"`
}

// ReplyMarkup - набор кнопок
type ReplyMarkup struct {
	KeyBoard [][]string `json:"keyboard"`
}

// WorkFlowManager - имитация БД пользователей
type WorkFlowManager struct {
	Manager map[string]*WorkFlow
}

// WorkFlow - описывает рабочий поток для работы с одним пользователем
type WorkFlow struct {
	NeedNextCommand bool
	CurrentCommand  string
	CurrentParams   [2]float64
	Step            int
}

var (
	// BotAPI - ссылка на стартовую ссылку телеграмм апи для ботов.
	BotAPI = "https://api.telegram.org/bot"
	// BotToken - токен от бота для тестирования напрямую.
	BotToken = "6118030405:AAEdosL5xIaJOp0lqJVej0ZF010SQ5nEFF8" // чтобы тесты проходили нужно закомментить в тестах `WebhookURL` и `BotToken` и запустить ngrock словно для своего пользования
	// WebhookURL - созданный в ngrok адресс для тестирования напрямую.
	WebhookURL    = "https://280c-46-138-171-228.ngrok-free.app" // ngrok http 8081
	defaultPort   = ":8081"
	updateChannel chan Update

	firstButtonLine  = []string{"x+y", "x*y", "x^y", "sqrt[y](x)"}
	secondButtonLine = []string{"x-y", "сбросить"}
)

// commandFlow - обрабатывает ситуацию, когда нужно последовательно ввести два числа, в конце возвращает, закнчился ли ввод
func commandFlow(update Update, wf *WorkFlow, baseKeyBoard ReplyMarkup, botURL, command string) bool {
	var err error
	if wf.Step == 0 {
		wf.NeedNextCommand = false
		wf.CurrentCommand = command
		wf.Step = 1
		sendMessage(botURL, update.Message.Chat.ID, "введите x:", baseKeyBoard)
	} else if wf.Step == 1 {
		tmp := wf.CurrentParams
		tmp[0], err = strconv.ParseFloat(update.Message.Text, 64)
		if err != nil {
			sendMessage(botURL, update.Message.Chat.ID, "не распознал числа, введите ещё раз:", baseKeyBoard)
			return false
		}
		wf.CurrentParams = tmp
		wf.Step = 2
		sendMessage(botURL, update.Message.Chat.ID, "введите y:", baseKeyBoard)
	} else {
		tmp := wf.CurrentParams
		tmp[1], err = strconv.ParseFloat(update.Message.Text, 64)
		if err != nil {
			sendMessage(botURL, update.Message.Chat.ID, "не распознал числа, введите ещё раз:", baseKeyBoard)
			return false
		}
		wf.CurrentParams = tmp
		wf.Step = 3
		return true
	}
	return false
}

// commandPlus - функция обрабатывает комманду сложения
func commandPlus(update Update, wf *WorkFlow, baseKeyBoard ReplyMarkup, botURL, command string) {
	if commandFlow(update, wf, baseKeyBoard, botURL, command) {
		sendMessage(botURL, update.Message.Chat.ID, command+" = "+fmt.Sprintf("%f", wf.CurrentParams[0]+wf.CurrentParams[1]), baseKeyBoard)
		wf.emptyWorkFlow()
	}
}

// commandPlus - функция обрабатывает комманду умножения
func commandMultiply(update Update, wf *WorkFlow, baseKeyBoard ReplyMarkup, botURL, command string) {
	if commandFlow(update, wf, baseKeyBoard, botURL, command) {
		sendMessage(botURL, update.Message.Chat.ID, command+" = "+fmt.Sprintf("%f", wf.CurrentParams[0]*wf.CurrentParams[1]), baseKeyBoard)
		wf.emptyWorkFlow()
	}
}

// commandPlus - функция обрабатывает комманду возведения в степень
func commandPow(update Update, wf *WorkFlow, baseKeyBoard ReplyMarkup, botURL, command string) {
	if commandFlow(update, wf, baseKeyBoard, botURL, command) {
		sendMessage(botURL, update.Message.Chat.ID, command+" = "+fmt.Sprintf("%f", math.Pow(wf.CurrentParams[0], wf.CurrentParams[1])), baseKeyBoard)
		wf.emptyWorkFlow()
	}
}

// commandPlus - функция обрабатывает комманду вычисления корня
func commandSqrt(update Update, wf *WorkFlow, baseKeyBoard ReplyMarkup, botURL, command string) {
	if commandFlow(update, wf, baseKeyBoard, botURL, command) {
		sendMessage(botURL, update.Message.Chat.ID, command+" = "+fmt.Sprintf("%f", math.Pow(wf.CurrentParams[0], 1/wf.CurrentParams[1])), baseKeyBoard)
		wf.emptyWorkFlow()
	}
}

// commandPlus - функция обрабатывает комманду вычитания
func commandMinus(update Update, wf *WorkFlow, baseKeyBoard ReplyMarkup, botURL, command string) {
	if commandFlow(update, wf, baseKeyBoard, botURL, command) {
		sendMessage(botURL, update.Message.Chat.ID, command+" = "+fmt.Sprintf("%f", wf.CurrentParams[0]-wf.CurrentParams[1]), baseKeyBoard)
		wf.emptyWorkFlow()
	}
}

// emptyWorkFlow - ставит WorkFlow в дефолтное состояние
func (wf *WorkFlow) emptyWorkFlow() {
	wf.NeedNextCommand = true
	wf.CurrentCommand = ""
	wf.CurrentParams = [2]float64{0, 0}
	wf.Step = 0
}

// getOrCreate - возвращает WorkFlow от привязанного пользователя, если же такого нет, создает и возвращает WorkFlow
func (wfm *WorkFlowManager) getOrCreate(username string) *WorkFlow {
	var wf *WorkFlow
	wf, ok := wfm.Manager[username]
	if ok {
		return wf
	}
	wf = &WorkFlow{}
	wf.emptyWorkFlow()
	wfm.Manager[username] = wf
	return wf
}

// makeCommand = обрабатывает пользовательскй непустой ввод.
func makeCommand(update Update, wf *WorkFlow, baseKeyBoard ReplyMarkup, botURL string) {
	var command string

	if update.Message.Text == secondButtonLine[1] {
		wf.emptyWorkFlow()
		sendMessage(botURL, update.Message.Chat.ID, "память очищена, введите комманду:", baseKeyBoard)
		return
	}

	if wf.NeedNextCommand {
		command = update.Message.Text
	} else {
		command = wf.CurrentCommand
	}

	switch {
	case command == "/start":
		sendMessage(botURL, update.Message.Chat.ID, "Приветствую, я умею вычислять простые операции, они представлены на панели ниже.", baseKeyBoard)
	case command == firstButtonLine[0]:
		commandPlus(update, wf, baseKeyBoard, botURL, command)
	case command == firstButtonLine[1]:
		commandMultiply(update, wf, baseKeyBoard, botURL, command)
	case command == firstButtonLine[2]:
		commandPow(update, wf, baseKeyBoard, botURL, command)
	case command == firstButtonLine[3]:
		commandSqrt(update, wf, baseKeyBoard, botURL, command)
	case command == secondButtonLine[0]:
		commandMinus(update, wf, baseKeyBoard, botURL, command)
	default:
		sendMessage(botURL, update.Message.Chat.ID, "не распознал команды", baseKeyBoard)
	}
}

// sendMessage - отправляет сообщение в чат
func sendMessage(botURL string, chatID int, text string, replyMarkup ReplyMarkup) (bool, error) {
	var message OutcomeMessage
	message.ChatID = chatID
	message.Text = text
	message.ReplyMarkup = replyMarkup

	buff, err := json.Marshal(message)
	if err != nil {
		return false, fmt.Errorf("Error when marshalling outcome message: %w", err)
	}
	resp, err := http.Post(botURL+"/sendMessage", "application/json", bytes.NewBuffer(buff))
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Response code of sending message is not 200:", resp.StatusCode)
	}
	if err != nil {
		return false, fmt.Errorf("Error during sending message: %w", err)
	}
	return true, nil
}

// setWebhook - удаляет старый и устанавливает новый вебхук
func setWebhook(botURL string) error {
	emptyBytes := []byte(`{}`)
	resp, err := http.Get(botURL + "/setWebhook?remove")
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Last web hook remove status code: ", resp.StatusCode)
	}
	defer resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Last web hook remove failed: %w", err)
	}
	resp, err = http.Post(botURL+"/setWebhook?url="+WebhookURL, "application/json", bytes.NewBuffer(emptyBytes))
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Last web hook remove status code: ", resp.StatusCode)
	}
	if err != nil {
		return fmt.Errorf("Webhook setup faile: %w", err)
	}
	return nil
}

// get_update - функция хендлер, получает обновление чата в телеграме, записывает в отдельный канал необходимые распарсенные данные типа Update
func getUpdate(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	var update Update
	err = json.Unmarshal(body, &update)
	if err != nil {
		fmt.Println(err)
		return
	}
	updateChannel <- update
}

// listenUpdates - запускает сервер, который принимает всё, что приходит на локальный сервер localhost:port/
func listenUpdates(botURL, port string) {
	defer close(updateChannel)
	http.HandleFunc("/", getUpdate)
	err := http.ListenAndServe(port, nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

// startTaskBot - запускает бота целиком, работает до нажатия Ctrl+C
func startTaskBot(ctx context.Context) error {
	botURL := BotAPI + BotToken

	wfm := &WorkFlowManager{
		Manager: make(map[string]*WorkFlow),
	}
	var wf *WorkFlow

	baseKeyBoard := ReplyMarkup{
		KeyBoard: [][]string{
			firstButtonLine,
			secondButtonLine,
		},
	}

	err := setWebhook(botURL)
	if err != nil {
		fmt.Println("setWebhook function failure: ", err)
	}
	updateChannel = make(chan Update)
	go listenUpdates(botURL, defaultPort)

	for update := range updateChannel {
		wf = wfm.getOrCreate(update.Message.Chat.Username)
		makeCommand(update, wf, baseKeyBoard, botURL)
	}
	return nil
}

func main() {
	err := startTaskBot(context.Background())
	if err != nil {
		panic(err)
	}
}
