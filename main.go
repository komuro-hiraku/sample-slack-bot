package main

// https://qiita.com/frozenbonito/items/cf75dadce12ef9a048e9
import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

const (
	selectGohanAction  = "select-gohan"
	confirmGohanAction = "confirm-gohan"
)

func main() {

	api := slack.New(os.Getenv("SLACK_BOT_TOKEN")) // Create slack bot API

	http.HandleFunc("/slack/events", slackVerificationMiddleware(func(w http.ResponseWriter, r *http.Request) {

		// SlackからのRequestであることを検証する
		// verifier, err := slack.NewSecretsVerifier(r.Header, os.Getenv("SLACK_SIGNING_SECRET"))
		// if err != nil {
		// 	log.Println(err)
		// 	w.WriteHeader(http.StatusInternalServerError)
		// 	return
		// }

		// bodyReader := io.TeeReader(r.Body, &verifier)
		// // Request Bodyを全部読み込む
		// body, err := ioutil.ReadAll(bodyReader)

		// if err != nil {
		// 	log.Println(err)                              // errorをログ表示
		// 	w.WriteHeader(http.StatusInternalServerError) // 500エラーをヘッダに書き込んでレスポンス
		// 	return
		// }

		// if err := verifier.Ensure(); err != nil {
		// 	log.Println(err)
		// 	w.WriteHeader(http.StatusBadRequest) // verifyに失敗したらBadRequest
		// 	return
		// }

		// 諸々上記コードは slackVerificationMiddleware で検証済み
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		switch eventsAPIEvent.Type {
		case slackevents.URLVerification:
			// SlackからのURL検証のレスポンスで利用する
			var res *slackevents.ChallengeResponse

			// bodyを slackevents.ChallengeReponse の Json に変換
			if err := json.Unmarshal(body, &res); err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			if _, err := w.Write([]byte(res.Challenge)); err != nil { // Challenge Response を返す
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		case slackevents.CallbackEvent:
			innerEvent := eventsAPIEvent.InnerEvent

			// innerEvent.Data.(type) ってなんだ・・・
			switch event := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				message := strings.Split(event.Text, " ")
				if len(message) < 2 {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				command := message[1] // 入力された2つ目の項目が command

				// コマンドを判定
				switch command {
				case "ping":
					if _, _, err := api.PostMessage(event.Channel, slack.MsgOptionText("pong", false)); err != nil {
						log.Println(err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				case "gohan":
					// https://qiita.com/frozenbonito/items/1df9bb685e6173160991
					// Text Object作成
					text := slack.NewTextBlockObject(slack.MarkdownType, "Please select Gohan", false, false)
					// Section Block を作る。複数のテキストをまとめたりボタンとか組み合わせが可能
					textSection := slack.NewSectionBlock(text, nil, nil)

					// 選択肢を構築
					gohans := []string{"とんかつ", "お刺身", "ハンバーグ", "鯖味噌", "寿司"}
					options := make([]*slack.OptionBlockObject, 0, len(gohans))

					// 選択肢から Text Object に変換して OptionBlock Object を作成していく
					// OptionBlock Object はSelectメニューのオプションとして機能する
					for _, v := range gohans {
						optionText := slack.NewTextBlockObject(slack.PlainTextType, v, false, false)
						options = append(options, slack.NewOptionBlockObject(v, optionText))
					}

					placeholder := slack.NewTextBlockObject(slack.PlainTextType, "Select Gohan", false, false)
					selectmenu := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, placeholder, "", options...)

					// Action Blockはボタンみたいなインタラクティブな要素を持てるBlock
					actionBlock := slack.NewActionBlock(selectGohanAction, selectmenu)

					fallbackText := slack.MsgOptionText("This client is not supported.", false)
					blocks := slack.MsgOptionBlocks(textSection, actionBlock)

					// ユーザーだけに見える一時メッセージの送信
					if _, err := api.PostEphemeral(event.Channel, event.User, fallbackText, blocks); err != nil {
						log.Println(err)
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
			}
		}
	}))

	http.HandleFunc("/slack/actions", slackVerificationMiddleware(func(w http.ResponseWriter, r *http.Request) {
		var payload *slack.InteractionCallback

		if err := json.Unmarshal([]byte(r.FormValue("payload")), &payload); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		switch payload.Type {
		case slack.InteractionTypeBlockActions:
			if len(payload.ActionCallback.BlockActions) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			// 上で存在チェックをしているので必ず値はある
			action := payload.ActionCallback.BlockActions[0]

			// ごはん Action
			switch action.BlockID {
			case selectGohanAction:

				gohan := action.SelectedOption.Value

				text := slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("お前は%sが食べたいのか？", gohan), false, false)
				textSection := slack.NewSectionBlock(text, nil, nil)

				// Confirm Button のセットアップ
				confirmButtonText := slack.NewTextBlockObject(slack.PlainTextType, "食べたい", false, false)
				confirmButton := slack.NewButtonBlockElement("", gohan, confirmButtonText)
				confirmButton.WithStyle(slack.StylePrimary)

				// Deny Buttonのセットアップ
				denyButtonText := slack.NewTextBlockObject(slack.PlainTextType, "いいえ", false, false)
				denyButton := slack.NewButtonBlockElement("", "いいえ", denyButtonText)
				denyButton.WithStyle(slack.StylePrimary)

				actionBlock := slack.NewActionBlock(confirmGohanAction, confirmButton, denyButton)

				fallbackText := slack.MsgOptionText("This client is not supported.", false)
				blocks := slack.MsgOptionBlocks(textSection, actionBlock)

				replaceOriginal := slack.MsgOptionReplaceOriginal(payload.ResponseURL)
				if _, _, _, err := api.SendMessage("", replaceOriginal, fallbackText, blocks); err != nil {
					log.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

			// ごはん Confirm Action
			case confirmGohanAction:
				gohan := action.Value
				log.Printf("select Gohan: %s\n", gohan)

				go func() {
					// 開始メッセージをPost
					startMsg := slack.MsgOptionText(
						fmt.Sprintf("<@%s> 了解しました. 10秒待つのです.", payload.User.ID), false)

					if _, _, err := api.PostMessage(payload.Channel.ID, startMsg); err != nil {
						log.Println(err)
					}

					cookinggohan(gohan)

					// 終了メッセージをPost
					endMsg := slack.MsgOptionText(
						fmt.Sprintf("<@%s> の今日のご飯は `%s` に決定しました", payload.User.ID, gohan), false)
					if _, _, err := api.PostMessage(payload.Channel.ID, endMsg); err != nil {
						log.Println(err)
					}
				}()
			}
		}
	}))

	log.Println("[INFO] Server listening")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// Verifyする処理を別Functionに分解
func slackVerificationMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		verifier, err := slack.NewSecretsVerifier(r.Header, os.Getenv("SLACK_SIGNING_SECRET"))
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		bodyReader := io.TeeReader(r.Body, &verifier)
		body, err := ioutil.ReadAll(bodyReader)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		next.ServeHTTP(w, r)
	}
}

func cookinggohan(gohan string) {
	log.Printf("cooking %s now. wait a minutes", gohan)
	time.Sleep(10 * time.Second)
}
