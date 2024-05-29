package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/robfig/cron/v3"

	_ "github.com/joho/godotenv/autoload"
)

var (
	cuuemCount   = make(map[int64]int)
	mu           sync.Mutex
	users        = make(map[int64]map[int64]string) // Map to store user names
	pendingHogia = make(map[int64]map[int64]bool)
	numberHogia  = make(map[int64]map[int64]int)
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		// bot.WithDefaultHandler(defaultHandler),
		bot.WithDebug(),
	}

	b, err := bot.New("7021374994:AAHqS2-YOSeExuDTSn7Ivq-ZSMQLAYcEnfw", opts...)
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
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return pendingHogia[update.Message.Chat.ID][update.Message.From.ID]
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
		},
	})
	go b.Start(ctx)

	c := cron.New()

	c.AddFunc("15 10 * * *", func() {
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
	if !isAllowedTime() {
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
	_, ok := users[update.Message.Chat.ID][userID]
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

func boemHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !isAllowedTime() {
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
		if _, ok := users[update.Message.Chat.ID][userID]; ok {
			delete(users[update.Message.Chat.ID], userID)
			cuuemCount[update.Message.Chat.ID]--

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

func hogiaHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if !isAllowedTime() {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " ơi, giờ lên đò là từ 10h đến 10h30 em nhé, đi sớm thì bị bỏ rơi mà đi muộn thì lỡ chuyến đò, căn đúng giờ em nhé!",
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()
	_, ok := users[update.Message.Chat.ID]
	if !ok {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Đò chưa ra khơi em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé!",
		})
	} else {
		userID := update.Message.From.ID

		// Mark user as pending to input number
		if pendingHogia[update.Message.Chat.ID] == nil {
			pendingHogia[update.Message.Chat.ID] = make(map[int64]bool)
		}
		pendingHogia[update.Message.Chat.ID][userID] = true
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
		_, ok := pendingUser[userID]
		if !ok {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Nhắn tin theo cú pháp `/khachmoi` + stk để mua vé đã em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé!",
			})
			return
		}
		if ok {
			// Check if the message is a number
			number, err := strconv.Atoi(update.Message.Text)
			if err != nil {
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Người chứ có phải vật đâu mà em " + update.Message.From.FirstName + " " + update.Message.From.LastName + " nhé",
				})
				return
			}
			if numberHogia[update.Message.Chat.ID] == nil {
				numberHogia[update.Message.Chat.ID] = make(map[int64]int)
			}

			numberHogia[update.Message.Chat.ID][userID] = number
			// Handle the number (e.g., store it, use it in some logic)
			// For now, just send a confirmation message
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   fmt.Sprintf("Em "+update.Message.From.FirstName+" "+update.Message.From.LastName+" có %d khách mời đặc biệt", number),
			})

			// Clear the pending state for the user
			delete(pendingHogia[update.Message.Chat.ID], userID)
		}
	}
}

func isAllowedTime() bool {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), 10, 30, 0, 0, now.Location())
	return now.After(start) && now.Before(end)
}
