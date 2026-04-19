package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ikawaha/kagome-dict/uni"
	"github.com/0x307e/go-haiku"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func main() {
	haiku.UseDict(uni.Dict()) // 形態素解析用の辞書をセット

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")

	if botToken == "" || appToken == "" {
		log.Fatal("[エラー] 環境変数 SLACK_BOT_TOKEN または SLACK_APP_TOKEN が設定されていません。")
	}

	api := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
		// slack.OptionDebug(true), // Slack API自体の通信ログを見たい場合はコメントアウトを外す
	)
	client := socketmode.New(api)

	go handleEvents(client, api)

	log.Println("Slack川柳Botを起動しました... メッセージの待機を開始します。")
	if err := client.Run(); err != nil {
		log.Fatalf("[致命的なエラー] Socket Modeの実行エラー: %v", err)
	}
}

func handleEvents(client *socketmode.Client, api *slack.Client) {
	for evt := range client.Events {
		switch evt.Type {
		case socketmode.EventTypeEventsAPI:
			eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
			if !ok {
				continue
			}

			client.Ack(*evt.Request)

			switch eventsAPIEvent.Type {
			case slackevents.CallbackEvent:
				innerEvent := eventsAPIEvent.InnerEvent
				switch ev := innerEvent.Data.(type) {
				case *slackevents.MessageEvent:

					// 1. メッセージを受信した時点のログ
					log.Printf("[受信] チャンネル:%s | ユーザー:%s | テキスト: %s", ev.Channel, ev.User, ev.Text)

					// 2. スキップ条件の判定ログ
					if ev.BotID != "" {
						log.Printf(" ├─ [スキップ] Bot自身のメッセージです (BotID: %s)", ev.BotID)
						continue
					}
					if ev.SubType != "" {
						log.Printf(" ├─ [スキップ] 通常のメッセージではありません (SubType: %s)", ev.SubType)
						continue
					}

					detectAndReplySenryu(api, ev.Channel, ev.Text, ev.TimeStamp)
				}
			}
		}
	}
}

// 3. 川柳の判定処理
func detectAndReplySenryu(api *slack.Client, channelID, text, ts string) {

	matches := haiku.Find(text, []int{5, 7, 5})
	
	if len(matches) == 0 {
		log.Printf(" └─ [結果] 川柳は検出されませんでした")
		return
	}

	found := matches[0]
	log.Printf(" ├─ [検出成功] 川柳が見つかりました: %s", found)
	senryuText := fmt.Sprintf("📝 *川柳を検出しました！*\n\n%s", found)

	// 4. Slackへの送信とエラーログ
	_, _, err := api.PostMessage(
		channelID,
		slack.MsgOptionText(senryuText, false),
		slack.MsgOptionTS(ts),
	)
	
	if err != nil {
		log.Printf(" └─ [エラー] Slackへのメッセージ送信に失敗しました: %v", err)
	} else {
		log.Printf(" └─ [送信成功] スレッドへの返信を完了しました")
	}
}