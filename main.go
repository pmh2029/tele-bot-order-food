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
	b.RegisterHandler(bot.HandlerTypeMessageText, "/trachieu", bot.MatchTypePrefix, traChieuTime)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/tinhtrachieu", bot.MatchTypePrefix, countTraChieu)
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

func isAllowedTime() bool {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), 10, 30, 0, 0, now.Location())
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
