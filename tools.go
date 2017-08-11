package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
	"gopkg.in/mgo.v2"
)

// Error constants.
var (
	ErrMsgEnding = errors.New("reached ending message")
)

// Color constants for embeded messages.
const (
	ColorMaroon = 0x800000
	ColorGreen  = 0x3B8040
	ColorBlue   = 0x5B6991
	ColorBlack  = 0x000000
	ColorGray   = 0x343434
	ColorYellow = 0xFEEB65
)

func strToCommands(io string) (bool, []string) {
	var cmd bool
	var slice []string

	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case c == '"':
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}

	var str = io
	if strings.HasPrefix(io, envCMDPrefix) {
		str = strings.TrimPrefix(io, envCMDPrefix)
		cmd = true
	}

	s := strings.FieldsFunc(str, f)
	for _, w := range s {
		if strings.HasPrefix(w, "\"") {
			w = strings.TrimPrefix(w, "\"")
		}
		if strings.HasSuffix(w, "\"") {
			w = strings.TrimSuffix(w, "\"")
		}
		slice = append(slice, w)
	}

	return cmd, slice
}

func msgToIOdat(msg *discordgo.MessageCreate) *IOdat {
	var io IOdat
	u := msg.Author

	io.command, io.io = strToCommands(msg.Content)
	io.input = msg.Content
	io.user = &User{ID: u.ID, Username: u.Username, Discriminator: u.Discriminator, Bot: u.Bot}
	io.msg = msg

	return &io
}

func sliceToIOdat(b *godbot.Core, s []string) *IOdat {
	u := b.User
	var io IOdat
	io.user = &User{ID: u.ID, Username: u.Username, Discriminator: u.Discriminator, Bot: u.Bot}
	io.command, io.io = strToCommands(strings.Join(s, " "))

	return &io
}

func tsConvert(ts string) string {
	a := strings.FieldsFunc(fmt.Sprintf("%s", ts), tsSplit)
	return fmt.Sprintf("%s %s", a[0], a[1])
}

func tsSplit(r rune) bool {
	return r == 'T' || r == '.' || r == '+'
}

func idSplit(r rune) bool {
	return r == '<' || r == '@' || r == '>' || r == ':' || r == ' '
}

func usernameAdd(username, discriminator string) string {
	return fmt.Sprintf("%s#%s", username, discriminator)
}

func usernameSplit(username string) []string {
	return strings.Split(username, "#")
}

func (cfg *Config) ioHandler(io *IOdat) (err error) {
	if len(io.io) < 1 {
		// Not enough arguments to do anything.
		// Prevents accessing nil pointer.
		return nil
	}

	// Make sure the channel is allowed to have bot commmands.
	if io.io[0] != "channel" {
		ch := ChannelNew(io.msg.ChannelID, io.guild.Name)
		if !io.user.HasPermission(io.guild.ID, permModerator) && !ch.Check() {
			io.msgEmbed = embedCreator("Bot commands have been disabled here.", ColorGray)
			return nil
		}
	}

	// Check if an alias here
	alias := AliasNew(io.io[0], "", io.user)
	link, err := alias.Check()
	if err != nil {
		if err != mgo.ErrNotFound {
			return err
		}
		err = nil
	} else {
		io.io = aliasConv(io.io[0], link, io.input)
	}

	command := io.io[0]
	switch strings.ToLower(command) {
	case "help":
		io.output = globalHelp()
	case "roll":
		io.miscRoll()
	case "top10":
		io.miscTop10()
	case "gen":
		io.roomGen()
	case "sz":
		io.msgEmbed = embedCreator(msgSize(io.msg.Message), ColorYellow)
	case "invite":
		io.msgEmbed = embedCreator(botInvite(), ColorGreen)
	case "ally":
		err = cfg.CoreAlliance(io)
	case "user":
		err = io.CoreUser()
	case "alias":
		err = io.CoreAlias()
	case "histo":
		err = io.histograph(cfg.Core.Session)
	case "channel":
		err = io.ChannelCore()
	case "event", "events":
		err = io.CoreEvent()
	case "cmd", "command":
		err = io.CoreDatabase()
	case "script", "scripts":
		err = io.CoreLibrary()
	case "echo":
		io.output = strings.Join(io.io[1:], " ")
		return
	}
	return
}

func embedCreator(description string, color int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       color,
		Description: description,
		Fields:      []*discordgo.MessageEmbedField{},
	}
}

func (cfg *Config) takeoverCheck(mID, cID, gID, content string, toggle bool, u *User) bool {
	s := cfg.Core.Session
	if toggle {
		if ok := u.HasPermission(gID, permAscended); ok {
			cfg.textTakeoverToggle(u.ID)
			s.ChannelMessageSend(cID, fmt.Sprintf("Takover enabled: %s", strconv.FormatBool(cfg.Takeover)))
			return true
		}
	} else if cfg.Takeover && cfg.TakeoverID == u.ID {
		s.ChannelMessageDelete(cID, mID)
		s.ChannelMessageSend(cID, u.StringPretty()+": "+content)
		return true
	}
	return false
}

func (cfg *Config) textTakeoverToggle(uID string) {
	if cfg.Takeover {
		cfg.Takeover = false
		cfg.TakeoverID = ""
	} else {
		cfg.Takeover = true
		cfg.TakeoverID = uID
	}
}

func botInvite() string {
	var msg string
	msg += fmt.Sprintf(
		"Invite me to your server!\n"+
			"Click to Add-> %s\n\n"+
			"Bot Support Server -> %s\n",
		"https://discordapp.com/oauth2/authorize?&client_id=290843164892463104&scope=bot&permissions=2080898303",
		"https://discord.gg/pk7eUwP",
	)
	return msg
}

func globalHelp() string {
	var msg = "*Most commands have a '--help' ability."
	for t, cmd := range cmds {
		msg += fmt.Sprintf("\n\n[ %s ]", t)
		for c, txt := range cmd {
			msg += fmt.Sprintf("\n\t%s\n\t\t%s", c, txt)
		}
	}
	return "```" + msg + "```"
}

func msgSize(m *discordgo.Message) string {
	var sz int
	// Author sizes
	usr := func(u *discordgo.User) {
		sz += len(u.ID)
		sz += len(u.Username)
		sz += len(u.Avatar)
		sz += len(u.Discriminator)
		sz++ // Verified Bool
		sz++ // Bot account
	}

	msgE := func(e *discordgo.MessageEmbed) {
		sz += len(e.URL)
		sz += len(e.Type)
		sz += len(e.Title)
		sz += len(e.Description)
		sz += len(e.Timestamp)
	}

	sz += len(m.ID)
	sz += len(m.ChannelID)
	sz += len(m.Content[4:]) // Reduce for the ',sz '
	sz += len(m.Timestamp)
	sz += len(m.EditedTimestamp)
	for _, mr := range m.MentionRoles {
		sz += len(mr)
	}

	sz++ //Tts
	sz++ // Mention everyone
	usr(m.Author)
	for _, u := range m.Mentions {
		usr(u)
	}
	for _, e := range m.Embeds {
		msgE(e)
	}

	return fmt.Sprintf("\nContent:\n%s\n\nSize of message: %d bytes\n", m.Content[4:], sz)
}
