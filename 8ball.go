package main

/**
 * Initial cut at a magic-8 ball bot for Slack.
 *
 * Created by Chris Baumbauer <cab@cabnetworks.net>
 */

import (
    "encoding/json"
    "io/ioutil"
    "log"
    "math/rand"
    "net/http"
    "os"
    "regexp"
    "sync/atomic"
    "strings"

    "github.com/gorilla/websocket"
)

/* Define the slack rtm.start packet */
type responseRtmStart struct {
    Ok    bool         `json:"ok"`
    Error string       `json:"error"`
    Url   string       `json:"url"`
    Self  responseSelf `json:"self"`
}

type responseSelf struct {
    Id string `json:"id"`
}

/* Received messages so we know what to respond to */
type incomingMessage struct {
    Type        string      `json:"type"`
    SubType     string      `json:"subtype"`
    Channel     string      `json:"channel"`
    User        string      `json:"user"`
    Text        string      `json:"text"`
    TimeStamp   string      `json:"ts"`
}

type outgoingMessage struct {
    Id          uint64       `json:"id",omitempty`
    Type        string      `json:"type"`
    Channel     string      `json:"channel"`
    Text        string      `json:"text"`
}

/* An atomic identifier for messages we post during the session */
var idCounter uint64
var userId string

/* Our array of canned responses */
var magicResponse = []string {
    "Signs point to no.",
    "Yes.",
    "Reply hazy, try again.",
    "Without a doubt.",
    "My sources say no.",
    "As I see it, yes.",
    "You may rely on it.",
    "Concentrate and ask again.",
    "Outlook not so good.",
    "It is decidedly so.",
    "Better not tell you now.",
    "Very doubtful.",
    "Yes - definitely.",
    "It is certain.",
    "Cannot predict now.",
    "Most likely.",
    "Ask again later.",
    "My reply is no.",
    "Outlook good.",
    "Don't count on it.",
}

func main() {
    if len(os.Args) != 2 {
        log.Fatal("ERROR: Expecting a session token")
    }

    sessionToken := os.Args[1]
    urlString := "https://slack.com/api/rtm.start?token=" + sessionToken

    /*
     * We need to make a standard http call first to retrieve the initial
     * data packet.  Then we can "upgrade" the socket, and listen from there
     */
    handshake, err := http.Get(urlString)
    if err != nil {
        log.Fatalf("Error contacting slack: %v\n", err)
    }

    handshakeBody, _ := ioutil.ReadAll(handshake.Body)
    var slackHeader responseRtmStart
    err = json.Unmarshal(handshakeBody, &slackHeader)
    if err != nil {
        log.Fatalf("Unable to parse handshake: %v\n", err)
    } else if !slackHeader.Ok {
        log.Fatalf("Bad header: %v\n", slackHeader.Error)
    } else {
        userId = slackHeader.Self.Id
    }

    con, resp, err := websocket.DefaultDialer.Dial(slackHeader.Url, nil)
    if err != nil {
        if err == websocket.ErrBadHandshake {
            log.Printf("Error with the handshake:\n")
            log.Printf("\tStatus: %d\n", resp.StatusCode)
            for k, v := range resp.Header {
                log.Printf("\t %s = %s\n", k, v)
            }
            
        }

        log.Fatalf("Error establishing connection: %v\n", err)
    }

    defer con.Close()

    /* Listen and reply */
    for {
        _, p, err := con.ReadMessage()
        if err != nil {
            log.Printf("Error: %v\n", err)
            break
        }

        var msg incomingMessage
        err = json.Unmarshal(p, &msg)

        if msg.Type == "message" {
            /* Check if it's directed at us, and if so craft a reply */
            //log.Printf("Found message from %s: %s\n", msg.User, msg.Text)
            mention := "<@" + userId + ">"
            if (strings.HasPrefix(msg.Text, mention)) {
                m := strings.TrimPrefix(msg.Text, mention)
                m = strings.TrimSpace(m)
                go postReply(con, m, msg.Channel)
            }
        }
    }
}

func postReply(con *websocket.Conn, query string, channel string) {
    if !strings.HasSuffix(query, "?") {
        generateResponse(con, channel, "Where's the question?")
        return
    }

    result, err := regexp.MatchString("^(who|what|when|where|why|how|if).*", strings.ToLower(query))
    if err != nil {
        log.Printf("Error matching string: %v\n", err)
        return
    } else if result {
        generateResponse(con, channel, "I'm not a tarot deck. Yes or no questions please.")
        return
    }

    /* randomize message based on time */
    answer := magicResponse[rand.Int() % len(magicResponse)]
    generateResponse(con, channel, answer)

}

func generateResponse(con *websocket.Conn, channel string, text string) {
    id := atomic.AddUint64(&idCounter, 1)
    msg := outgoingMessage{
        Id: id,
        Channel: channel,
        Type: "message",
        Text: text,
    }

    response, err := json.Marshal(msg)
    if err != nil {
        log.Printf("Error: cannot generate response: %v\n", err)
    }
    err = con.WriteMessage(websocket.TextMessage, response)
    if err != nil {
        log.Printf("Error: cannot send response: %v\n", err)
    }
}