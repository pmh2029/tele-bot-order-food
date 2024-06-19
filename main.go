package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/robfig/cron/v3"

	_ "github.com/joho/godotenv/autoload"
)

var (
	cuuemCount      = make(map[int64]int)
	mu              sync.Mutex
	users           = make(map[int64]map[int64]string) // Map to store user names
	pendingHogia    = make(map[int64]map[int64]bool)
	numberHogia     = make(map[int64]map[int64]int) //
	isTraChieuTime  = make(map[int64]bool)
	traChieuPeople  = make(map[int64]map[int64]string)
	traChieuCount   = make(map[int64]int)
	pendingTraChieu = make(map[int64]map[int64]bool)
	numberTraChieu  = make(map[int64]map[int64]int) //
	userVote        = make(map[int64]map[string][]UserOrder)
	orderPoll       = make(map[int64]string)
	optionToUsers   = make(map[int64]map[int][]string)
)

var thucdon = map[int]string{
	1:  "Bún Riêu Út Phương",
	2:  "Bánh cuốn Cao Bằng Tống Thêm",
	3:  "Bún bò Huế An Cựu",
	4:  "Nem nướng Minh Đức",
	5:  "Miến/ Bánh đa trộn Cây Xoài",
	6:  "Cơm thố Anh Nguyễn",
	7:  "Cơm thố Bách Khoa",
	8:  "Bánh mỳ Vũ",
	9:  "Bún đậu mắm tôm",
	10: "Cơm gà nhị vị Nam Kinh",
	11: "Kim bap",
}

type UserOrder struct {
	UserID   int64
	Fullname string
	OptionID int
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler2),
		bot.WithDebug(),
		bot.WithDefaultHandler(noxauhandler),
	}

	b, err := bot.New(os.Getenv("BOT_TOKEN"), opts...)
	if err != nil {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/hello", bot.MatchTypePrefix, defaultHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/cuuem", bot.MatchTypePrefix, cuuemHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/boem", bot.MatchTypePrefix, boemHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/countcuuem", bot.MatchTypePrefix, countCuuemHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/khachmoi", bot.MatchTypePrefix, hogiaHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/trachieu", bot.MatchTypePrefix, traChieuTime)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/tinhtrachieu", bot.MatchTypePrefix, countTraChieu)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/order", bot.MatchTypePrefix, orderHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/chotdon", bot.MatchTypePrefix, chotDon)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/noxau", bot.MatchTypePrefix, noxauhandler)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return pendingHogia[update.Message.Chat.ID][update.Message.From.ID] || pendingTraChieu[update.Message.Chat.ID][update.Message.From.ID]
	}, messageHandler)

	b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{
				Command:     "/cuuem",
				Description: "Cú pháp cíu em + stk để lên đò",
			},
			{
				Command:     "/boem",
				Description: "Cú pháp bỏ em - stk để xuống đò",
			},
			{
				Command:     "/khachmoi",
				Description: "Cú pháp khách mời để mua vé cho khách mời đặc biệt",
			},
			{
				Command:     "/countcuuem",
				Description: "Số người lên đò",
			},
			{
				Command:     "/trachieu",
				Description: "Cú pháp gọi đò trà chiều",
			},
			{
				Command:     "/tinhtrachieu",
				Description: "Cú pháp xem danh sách đò viên trà chiều",
			},
			{
				Command:     "/order",
				Description: "Cú pháp gọi món",
			},
			{
				Command:     "/chotdon",
				Description: "Cú pháp xem chốt đơn",
			},
		},
	})
	go b.Start(ctx)

	c := cron.New()

	c.AddFunc("0 10 * * *", func() {
		orderJobHandler(ctx, b, os.Getenv("CHAT_ID"))
	})

	// Start the cron scheduler
	c.Start()

	// Wait for termination signal
	<-ctx.Done()

	// Stop the cron scheduler
	c.Stop()
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Hello, *" + bot.EscapeMarkdown(update.Message.From.LastName) + "*",
		ParseMode: models.ParseModeMarkdown,
	})
}

func orderJobHandler(ctx context.Context, b *bot.Bot, chatID any) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Mẹ bảo, có hai thứ mà đời người không được bỏ lỡ. Một là chuyến đò cuối cùng về nhà, hai là hôm nay ăn món gì thế?",
	})
}

func cuuemHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !isAllowedTime() && !isTraChieuTime[update.Message.Chat.ID] {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " ơi, giờ lên đò là từ 10h đến 10h30 em nhé, đi sớm thì bị bỏ rơi mà đi muộn thì lỡ chuyến đò, căn đúng giờ em nhé!",
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	userID := update.Message.From.ID
	userName := update.Message.From.FirstName + " " + update.Message.From.LastName

	if users[update.Message.Chat.ID] == nil {
		users[update.Message.Chat.ID] = make(map[int64]string)
	}

	if traChieuPeople[update.Message.Chat.ID] == nil {
		traChieuPeople[update.Message.Chat.ID] = make(map[int64]string)
	}
	_, ok := users[update.Message.Chat.ID][userID]
	_, traok := traChieuPeople[update.Message.Chat.ID][userID]
	if isAllowedTime() {
		if !ok {
			users[update.Message.Chat.ID][userID] = userName
			cuuemCount[update.Message.Chat.ID]++

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Đã thêm em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " lên đò",
			})
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " ơi, lên đò chứ có phải đinh lển đâu mà em lên gớm thế?",
			})
		}

	}

	if isTraChieuTime[update.Message.Chat.ID] {
		if !traok {
			traChieuPeople[update.Message.Chat.ID][userID] = userName
			traChieuCount[update.Message.Chat.ID]++

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Đã thêm em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " lên đò",
			})
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " ơi, lên đò chứ có phải đinh lển đâu mà em lên gớm thế?",
			})
		}
	}
}

func boemHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !isAllowedTime() && !isTraChieuTime[update.Message.Chat.ID] {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " ơi, giờ lên đò là từ 10h đến 10h30 em nhé, đi sớm thì bị bỏ rơi mà đi muộn thì lỡ chuyến đò, căn đúng giờ em nhé!",
		})
		return
	}
	mu.Lock()
	defer mu.Unlock()

	userID := update.Message.From.ID

	_, ok := users[update.Message.Chat.ID]
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Đò chưa ra khơi em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé!",
		})
		return
	} else {
		_, ok := users[update.Message.Chat.ID][userID]
		_, traChieuOk := traChieuPeople[update.Message.Chat.ID][userID]
		if ok || traChieuOk {
			if isAllowedTime() {
				delete(users[update.Message.Chat.ID], userID)
				cuuemCount[update.Message.Chat.ID]--
			}

			if isTraChieuTime[update.Message.Chat.ID] {
				delete(traChieuPeople[update.Message.Chat.ID], userID)
				traChieuCount[update.Message.Chat.ID]--
			}

			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Đã cho em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " cook khỏi đò",
			})
			return
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " chưa lên thì sao mà xuống =))",
			})
		}
	}
}

func countCuuemHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	mu.Lock()
	count := cuuemCount[update.Message.Chat.ID]
	countHoGia := 0
	for _, hogia := range numberHogia {
		for _, v := range hogia {
			countHoGia += v
		}
	}

	userNames := make([]string, 0, len(users))

	if users[update.Message.Chat.ID] != nil {
		for _, name := range users[update.Message.Chat.ID] {
			userNames = append(userNames, name)
		}
	} else {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Đò chưa ra khơi em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé!",
		})
		return
	}
	mu.Unlock()

	message := fmt.Sprintf("Số người lên đò là: %d\n", count+countHoGia)
	message += "Đò viên:\n"
	for _, name := range userNames {
		message += "- " + name + "\n"
	}

	message += "Khách mời đặc biệt: " + strconv.Itoa(countHoGia)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   message,
	})

}

func countTraChieu(ctx context.Context, b *bot.Bot, update *models.Update) {
	mu.Lock()
	count := traChieuCount[update.Message.Chat.ID]
	countHoGia := 0
	for _, hogia := range numberTraChieu {
		for _, v := range hogia {
			countHoGia += v
		}
	}

	userNames := make([]string, 0, len(users))

	if traChieuPeople[update.Message.Chat.ID] != nil {
		for _, name := range traChieuPeople[update.Message.Chat.ID] {
			userNames = append(userNames, name)
		}
	} else {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Đò chưa ra khơi em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé!",
		})
		return
	}
	mu.Unlock()

	message := fmt.Sprintf("Số người lên đò là: %d\n", count+countHoGia)
	message += "Đò viên:\n"
	for _, name := range userNames {
		message += "- " + name + "\n"
	}

	message += "Khách mời đặc biệt: " + strconv.Itoa(countHoGia)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   message,
	})
}

func hogiaHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !isAllowedTime() && !isTraChieuTime[update.Message.Chat.ID] {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " ơi, giờ lên đò là từ 10h đến 10h30 em nhé, đi sớm thì bị bỏ rơi mà đi muộn thì lỡ chuyến đò, căn đúng giờ em nhé!",
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()
	_, ok := users[update.Message.Chat.ID]
	_, traChieu := traChieuPeople[update.Message.Chat.ID]
	if !ok && !traChieu {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Đò chưa ra khơi em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé!",
		})
	} else {
		userID := update.Message.From.ID

		// Mark user as pending to input number
		if isAllowedTime() {
			if pendingHogia[update.Message.Chat.ID] == nil {
				pendingHogia[update.Message.Chat.ID] = make(map[int64]bool)
			}
			pendingHogia[update.Message.Chat.ID][userID] = true
		}

		if isTraChieuTime[update.Message.Chat.ID] {
			if pendingTraChieu[update.Message.Chat.ID] == nil {
				pendingTraChieu[update.Message.Chat.ID] = make(map[int64]bool)
			}
			pendingTraChieu[update.Message.Chat.ID][userID] = true
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Rep lại tin nhắn này để thêm khách mời nhé em " + update.Message.From.FirstName + " " + update.Message.From.LastName + "!",
		})
	}
}

func messageHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	mu.Lock()
	defer mu.Unlock()

	userID := update.Message.From.ID
	_, ok := users[update.Message.Chat.ID]
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Đò chưa ra khơi em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé!",
		})
	} else {
		pendingUser := pendingHogia[update.Message.Chat.ID]
		pendingTraChieuPeople := pendingTraChieu[update.Message.Chat.ID]
		_, ok := pendingUser[userID]
		_, traChieuOk := pendingTraChieuPeople[userID]
		if !ok && !traChieuOk {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Nhắn tin theo cú pháp /khachmoi + stk để mua vé đã em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé!",
			})
			return
		}
		if ok || traChieuOk {
			// Check if the message is a number
			number, err := strconv.Atoi(update.Message.Text)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Người chứ có phải vật đâu mà em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé",
				})
				return
			}
			if isAllowedTime() {
				if numberHogia[update.Message.Chat.ID] == nil {
					numberHogia[update.Message.Chat.ID] = make(map[int64]int)
				}
				numberHogia[update.Message.Chat.ID][userID] = number
				delete(pendingHogia[update.Message.Chat.ID], userID)

			}

			if isTraChieuTime[update.Message.Chat.ID] {
				if numberTraChieu[update.Message.Chat.ID] == nil {
					numberTraChieu[update.Message.Chat.ID] = make(map[int64]int)
				}
				numberTraChieu[update.Message.Chat.ID][userID] = number
				delete(pendingTraChieu[update.Message.Chat.ID], userID)
			}

			// Handle the number (e.g., store it, use it in some logic)
			// For now, just send a confirmation message
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   fmt.Sprintf("Em "+update.Message.From.FirstName+" "+update.Message.From.LastName+" có %d khách mời đặc biệt", number),
			})

			// Clear the pending state for the user
		}
	}
}

func orderHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if _, ok := orderPoll[update.Message.Chat.ID]; ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Trâu chậm thì uống nước đục thôi em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé!",
		})
		return
	}

	msg, _ := b.SendPoll(ctx, &bot.SendPollParams{
		ChatID:   update.Message.Chat.ID,
		Question: "Chuyến đò trưa hôm nay hân hạnh được phục vụ các em những món sau:",
		Options: []models.InputPollOption{
			{
				Text: "Bún Riêu Út Phương",
				// TextEntities: []models.MessageEntity{},
			},
			{
				Text: "Bánh cuốn Cao Bằng Tống Thêm",
			},
			{
				Text: "Bún bò Huế An Cựu",
			},
			{
				Text: "Nem nướng Minh Đức",
			},
			{
				Text: "Miến/ Bánh đa trộn Cây Xoài",
			},
			{
				Text: "Cơm thố Anh Nguyễn",
			},
			{
				Text: "Cơm thố Bách Khoa",
			},
			{
				Text: "Bánh mỳ Vũ",
			},
			{
				Text: "Bún đậu mắm tôm",
			},
			{
				Text: "Cơm gà nhị vị Nam Kinh",
			},
			{
				Text: "Kim bap",
			},
		},
		IsAnonymous: bot.False(),
	})

	orderPoll[update.Message.Chat.ID] = msg.Poll.ID
}

func isAllowedTime() bool {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), 19, 30, 0, 0, now.Location())
	return now.After(start) && now.Before(end)
}

func traChieuTime(ctx context.Context, b *bot.Bot, update *models.Update) {
	if isAllowedTime() {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Đến giờ đò trưa rồi em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé! Yêu thì muốn yêu người chung thủy mà chân em lại đạp hai đò là sao?",
		})
		return
	}

	if isTraChieuTime[update.Message.Chat.ID] {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " hơi bị tham lam nhé đò lúc nào cũng phải đi 2 chuyến là seo?",
		})
		return
	}

	isTraChieuTime[update.Message.Chat.ID] = true
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Trà chiều hôm nay có những ai nèo?",
	})
}

func defaultHandler2(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID, err := strconv.ParseInt(os.Getenv("CHAT_ID"), 10, 64)
	if err != nil {
		return
	}
	if update.PollAnswer != nil {
		handlePollAnswer(update, chatID)
	}
}

func handlePollAnswer(update *models.Update, chatID int64) {
	mu.Lock()
	defer mu.Unlock()

	answer := update.PollAnswer
	if len(answer.OptionIDs) == 0 {
		return
	}
	userID := answer.User.ID
	fullName := answer.User.FirstName + " " + answer.User.LastName

	if userVote[chatID] == nil {
		userVote[chatID] = make(map[string][]UserOrder)
	}

	if userVote[chatID][answer.PollID] == nil {
		userVote[chatID][answer.PollID] = make([]UserOrder, 0)
	}
	if optionToUsers[chatID] == nil {
		optionToUsers[chatID] = make(map[int][]string)
	}

	// Check if the user has already voted and remove their previous vote
	if previousResponse, exists := userVote[chatID][answer.PollID]; exists {
		for _, prevOptionID := range previousResponse {
			// Remove the user from the previous option
			optionToUsers[chatID][prevOptionID.OptionID] = removeUser(optionToUsers[chatID][prevOptionID.OptionID], fullName)
		}
	}

	userOrder := UserOrder{
		UserID:   userID,
		Fullname: fullName,
		OptionID: answer.OptionIDs[0],
	}

	if len(answer.OptionIDs) > 0 {

		optionToUsers[chatID][answer.OptionIDs[0]] = append(optionToUsers[chatID][answer.OptionIDs[0]], fullName)
		found := false
		for i, v := range userVote[chatID][answer.PollID] {
			if v.UserID == userID {
				userVote[chatID][answer.PollID][i].OptionID = answer.OptionIDs[0]
				found = true
				break
			}
		}

		// Nếu không tìm thấy userID, thêm mới vào userVote
		if !found {
			userVote[chatID][answer.PollID] = append(userVote[chatID][answer.PollID], userOrder)
		}
	} else {
		// If no options selected, remove the user response
		for _, v := range userVote[chatID][answer.PollID] {
			if v.UserID == userID {
				userVote[chatID][answer.PollID] = removeOrderByUserID(userVote[chatID][answer.PollID], userID)
				break
			}
		}
	}
}

func removeOrderByUserID(orders []UserOrder, userID int64) []UserOrder {
	for i, order := range orders {
		if order.UserID == userID {
			// Loại bỏ phần tử tại vị trí i
			return append(orders[:i], orders[i+1:]...)
		}
	}
	// Nếu không tìm thấy UserID, trả về mảng gốc
	return orders
}

func removeUser(userList []string, userName string) []string {
	for i, user := range userList {
		if user == userName {
			return append(userList[:i], userList[i+1:]...)
		}
	}
	return userList
}

func chotDon(ctx context.Context, b *bot.Bot, update *models.Update) {
	mu.Lock()
	defer mu.Unlock()

	// for _, userOrder := range userVote[update.Message.Chat.ID][orderPoll[update.Message.Chat.ID]] {
	// 	optionToUsers[update.Message.Chat.ID][userOrder.OptionID] = append(optionToUsers[update.Message.Chat.ID][userOrder.OptionID], userOrder.Fullname)
	// }

	totalVote := len(userVote[update.Message.Chat.ID][orderPoll[update.Message.Chat.ID]])
	maxVotes := 0

	for _, user := range optionToUsers[update.Message.Chat.ID] {
		if len(user) > maxVotes {
			maxVotes = len(user)
		}
	}

	var mostVotedOptions []int
	var resultMessage string
	resultMessage = "Các em hơi nhiều yêu cầu đấy nhé:\n"
	for optionID, users := range optionToUsers[update.Message.Chat.ID] {
		if len(users) > 0 {
			percentage := (float64(len(users)) / float64(totalVote)) * 100
			resultMessage += fmt.Sprintf("- %s (%.2f%%):\n", thucdon[optionID+1], percentage)
			for _, user := range users {
				resultMessage += fmt.Sprintf("  * %s\n", user)
			}
			if len(users) == maxVotes {
				mostVotedOptions = append(mostVotedOptions, optionID)
			}
		}
	}

	resultMessage += fmt.Sprintf("\nĐò trưa nay chia %d thuyền các em nhé:\n", len(mostVotedOptions))
	for optionID, users := range optionToUsers[update.Message.Chat.ID] {
		if len(users) == maxVotes {
			resultMessage += fmt.Sprintf("- Thuyền %s:\n", thucdon[optionID+1])
			for _, user := range users {
				resultMessage += fmt.Sprintf("  * %s\n", user)
			}
			resultMessage += "\n"
		}
	}
	var selectedOptionIDs []int
	for _, userOrder := range userVote[update.Message.Chat.ID][orderPoll[update.Message.Chat.ID]] {
		selectedOptionIDs = append(selectedOptionIDs, userOrder.OptionID)
	}
	if !slices.Equal(mostVotedOptions, selectedOptionIDs) {
		resultMessage += "Các em "
		for i, userOrder := range userVote[update.Message.Chat.ID][orderPoll[update.Message.Chat.ID]] {
			if !slices.Contains(mostVotedOptions, userOrder.OptionID) {
				resultMessage += userOrder.Fullname
				if len(userVote[update.Message.Chat.ID][orderPoll[update.Message.Chat.ID]]) > 1 && i < len(userVote[update.Message.Chat.ID][orderPoll[update.Message.Chat.ID]])-1 {
					resultMessage += ", "
				}
			}
		}
		resultMessage = strings.TrimSuffix(resultMessage, ", ")
		resultMessage += " chọn lại thuyền cho mình các em nhé!"
	}

	msg := &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   resultMessage,
	}
	_, err := b.SendMessage(ctx, msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func noXau(ctx context.Context, b *bot.Bot, update *models.Update) {
	// b.SendMessage(ctx, &bot.SendMessageParams{
	// 	ChatID: -4175362958,
	// 	Text:   "Không phải thách ^-^",
	// })

	// fileData, errReadFile := os.ReadFile("./photo_2024-05-07_14-29-21.png")
	// if errReadFile != nil {
	// 	fmt.Printf("error read file, %v\n", errReadFile)
	// 	return
	// }

	// params := &bot.SendPhotoParams{
	// 	ChatID:  -4175362958,
	// 	Photo:   &models.InputFileUpload{Data: bytes.NewReader(fileData)},
	// 	Caption: "Thanks for your opinion, I appreciate it!",
	// }

	// b.SendPhoto(ctx, params)

	// b.BanChatMember(ctx, &bot.BanChatMemberParams{
	// 	UserID: 939425786,
	// 	ChatID: -4175362958,
	// 	RevokeMessages: true,
	// })
}

func noxauhandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	admin := os.Getenv("ADMIN")
	adminIDs := strings.Split(admin, ",")
	ids := []int64{}
	for _, id := range adminIDs {
		people, _ := strconv.ParseInt(id, 10, 64)
		ids = append(ids, people)
	}
	if update.Message != nil && slices.Contains(ids, update.Message.From.ID) && update.Message.Chat.ID != -4175362958 {
		if update.Message.Text != "" {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: os.Getenv("CHAT_ID"),
				Text:   update.Message.Text,
			})
		}
		if update.Message.Sticker != nil {
			b.SendSticker(ctx, &bot.SendStickerParams{
				ChatID: os.Getenv("CHAT_ID"),
				Sticker: &models.InputFileString{
					Data: update.Message.Sticker.FileID,
				},
			})
		}
		if update.Message.Photo != nil {
			b.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID: os.Getenv("CHAT_ID"),
				Photo: &models.InputFileString{
					Data: update.Message.Photo[0].FileID,
				},
			})

		}
	}
}
