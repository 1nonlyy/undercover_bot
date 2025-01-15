package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	//"golang.org/x/exp/rand"
)

type Game struct {
	players            []string
	roles              map[string]string
	words              [2]string
	currentPlayerIndex int
	active             bool
}

var game *Game
var playerListMessageID int
var currentPlayerIndex int

func main() {
	er := godotenv.Load()
	if er != nil {
		log.Fatalf("Error loading .env file")
	}
	apikey := os.Getenv("API_KEY")
	bot, err := tgbotapi.NewBotAPI(apikey)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			handleMessage(bot, update.Message)
		} else if update.CallbackQuery != nil {
			handleRevealCallback(bot, update.CallbackQuery)
		}
	}
}
func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	switch message.Command() {
	case "startgame":
		startGame(bot, message)
	case "addplayer":
		addPlayer(bot, message)
	case "done":
		assignRoles(bot, message)
	case "reveal":
		revealRole(bot, message)
	default:
		msg := tgbotapi.NewMessage(message.Chat.ID, "Unknown command. Use /startgame to start.")
		bot.Send(msg)
	}
}
func startGame(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	game = &Game{
		players:            []string{},
		roles:              make(map[string]string),
		words:              [2]string{"Sun", "Moon"},
		currentPlayerIndex: 0,
		active:             true,
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, "Game started! Use /addplayer [name] to add players.")
	bot.Send(msg)
}

func addPlayer(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if game == nil || !game.active {
		msg := tgbotapi.NewMessage(message.Chat.ID, "No active game. Use /startgame to start.")
		bot.Send(msg)
		return
	}

	name := message.CommandArguments()
	if name == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please provide a player name. Example: /addplayer Alice")
		bot.Send(msg)
		return
	}

	game.players = append(game.players, name)
	playerList := "Current players:\n"
	for i, player := range game.players {
		playerList += string(i+1) + ". " + player + "\n"
	}
	if playerListMessageID == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, playerList)
		sentMsg, err := bot.Send(msg)
		if err != nil {
			log.Printf("Error sending player list message: %v", err)
			return
		}
		playerListMessageID = sentMsg.MessageID
	} else {
		editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, playerListMessageID, playerList)
		_, err := bot.Send(editMsg)
		if err != nil {
			log.Printf("Error editing player list message: %v", err)
		}
	}
}
func assignRoles(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if game == nil || !game.active {
		msg := tgbotapi.NewMessage(message.Chat.ID, "No active game. Use /startgame to start.")
		bot.Send(msg)
		return
	}

	numPlayers := len(game.players)
	if numPlayers < 3 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Not enough players! Add at least 3 players.")
		bot.Send(msg)
		return
	}

	// Assign roles randomly
	rand.Seed(time.Now().UnixNano())
	numUndercover := 1
	numMisterWhite := 1
	numCivilians := numPlayers - numUndercover - numMisterWhite

	roles := []string{}
	for i := 0; i < numCivilians; i++ {
		roles = append(roles, "Civilian")
	}
	for i := 0; i < numUndercover; i++ {
		roles = append(roles, "Undercover")
	}
	for i := 0; i < numMisterWhite; i++ {
		roles = append(roles, "Mister White")
	}
	rand.Shuffle(len(roles), func(i, j int) { roles[i], roles[j] = roles[j], roles[i] })

	for i, player := range game.players {
		game.roles[player] = roles[i]
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "Roles assigned! Use /reveal to reveal roles.")
	bot.Send(msg)
}

func revealRole(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if game == nil || !game.active || len(game.players) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "No active game or players to reveal words for.")
		bot.Send(msg)
		return
	}

	currentPlayerIndex = 0 // Start with the first player
	sendRevealPrompt(bot, message.Chat.ID)
}
func sendRevealPrompt(bot *tgbotapi.BotAPI, chatID int64) {
	if currentPlayerIndex >= len(game.players) {
		// If all players have been revealed, end the process
		msg := tgbotapi.NewMessage(chatID, "All players have seen their words. Game ready to start!")
		bot.Send(msg)
		currentPlayerIndex = 0
		return
	}

	playerName := game.players[currentPlayerIndex]

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("It's %s's turn. Pass the phone to them.", playerName))
	revealButton := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Reveal Word", fmt.Sprintf("reveal_%d", currentPlayerIndex)),
		),
	)
	msg.ReplyMarkup = revealButton

	bot.Send(msg)
}

func handleRevealCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery) {
	var playerIndex int
	fmt.Sscanf(callback.Data, "reveal_%d", &playerIndex)

	if playerIndex != currentPlayerIndex {
		return
	}

	playerWord := ""
	role := game.roles[game.players[playerIndex]]
	if role == "Civilian" {
		playerWord = game.words[0]
	} else if role == "Undercover" {
		playerWord = game.words[1]
	} else {
		playerWord = "You are Mister White! Your goal is to guess the correct word!"
	}

	callbackResponse := tgbotapi.NewCallbackWithAlert(callback.ID, fmt.Sprintf("Your word is: %s", playerWord))
	if _, err := bot.Request(callbackResponse); err != nil {
		log.Printf("Error sending callback response: %v", err)
		return
	}

	//revealMsg := tgbotapi.NewMessage(callback.Message.Chat.ID, fmt.Sprintf("Your word is: %s", playerWord))
	//bot.Send(revealMsg)

	currentPlayerIndex++
	sendRevealPrompt(bot, callback.Message.Chat.ID)
}
