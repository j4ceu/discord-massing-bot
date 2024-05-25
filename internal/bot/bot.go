package bot

import (
	"MassingDiscordBot/internal/config"
	"MassingDiscordBot/internal/sheets"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var BotToken string
var ApplicationID string
var threadSpreadsheetMap = make(map[string]string)
var threadMessageIDs = make(map[string][]string)

func checkNilErr(e error) {
	if e != nil {
		log.Fatal("Error message")
	}
}

func Run() {

	// create a session
	discord, err := discordgo.New("Bot " + BotToken)
	checkNilErr(err)

	// add a event handler
	discord.AddHandler(newMessage)
	discord.AddHandler(commandHandler)
	slashCommand := &discordgo.ApplicationCommand{
		Name:        "crot-massing",
		Description: "Prepare for war and display party data from a spreadsheet",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "title",
				Description: "The title of the war",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "time",
				Description: "The time of the war",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "objective",
				Description: "The objective of the war",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "spreadsheetid",
				Description: "The ID of the spreadsheet containing party data",
				Required:    true,
			},
		},
	}

	_, err = discord.ApplicationCommandCreate(ApplicationID, "", slashCommand)
	if err != nil {
		log.Fatalf("Error registering slash command: %v", err)
		return
	}

	// Register the ping command
	_, err = discord.ApplicationCommandCreate(ApplicationID, "", &discordgo.ApplicationCommand{
		Name:        "ping",
		Description: "Ping the bot to check if it's alive",
	})
	if err != nil {
		log.Fatalf("Error registering slash command: %v", err)
		return
	}

	// Register the /ikutan-crot command with static options for now
	_, err = discord.ApplicationCommandCreate(ApplicationID, "", &discordgo.ApplicationCommand{
		Name:        "ikutan-crot",
		Description: "Ikutan crot!",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "no",
				Description: "Nomor",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "party",
				Description: "Nama party",
				Required:    true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Error registering slash command: %v", err)
		return
	}

	// open session
	discord.Open()
	defer discord.Close() // close session, after function termination

	// keep bot running untill there is NO os interruption (ctrl + C)
	fmt.Println("Bot running....")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

}

func commandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Name {
	case "ping":
		// Respond to the ping command
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Pong!",
			},
		})
	case "crot-massing":
		Massing(s, i)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Waiting!",
			},
		})
	case "ikutan-crot":
		IkutanCrot(s, i)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Waiting!",
			},
		})
	}

}

func IkutanCrot(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Fetch dynamic data for options (e.g., party names)
	threadID := i.ChannelID
	spreadsheetID, ok := threadSpreadsheetMap[threadID]

	// Check if the spreadsheet ID is found
	if !ok {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Spreadsheet ID not found for this thread.",
			},
		})
		return
	}

	// Set the spreadsheet ID
	sheets.SetSpreadSheetID(spreadsheetID)
	srv := config.ConfigSheets()

	// Read party data from the spreadsheet
	_, partyNames := sheets.ReadSheet(srv, "Mooncrat")

	no := i.ApplicationCommandData().Options[0].StringValue()
	party := i.ApplicationCommandData().Options[1].StringValue()

	partyExists := false
	var partyIndex int
	for i, name := range partyNames {
		if name == party {
			partyExists = true
			partyIndex = i
			break
		}
	}

	if !partyExists {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Party not found",
			},
		})
		return
	}
	// Get the user's nickname or username
	var playerName string
	member, err := s.GuildMember(i.GuildID, i.Member.User.ID)
	if err != nil {
		log.Println("Error getting guild member:", err)
		playerName = i.Member.User.Username // Fall back to username
	} else {
		if member.Nick != "" {
			playerName = member.Nick // Use nickname if available
		} else {
			playerName = i.Member.User.Username // Fall back to username
		}
	}

	err = sheets.AssignPlayerToSheet(srv, "Mooncrat", party, no, playerName)
	if err != nil {
		// Handle error
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to assign player: " + err.Error(),
			},
		})
	} else {
		// Send a response to the user
		response := fmt.Sprintf("%s joined party '%s' with number %s", playerName, party, no)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: response,
			},
		})
		s.ChannelMessageSend(threadID, response)
		EditMessage(threadID, partyIndex, s, i)

	}

}

func Massing(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Extract information from the interaction
	title := i.ApplicationCommandData().Options[0].StringValue()
	time := i.ApplicationCommandData().Options[1].StringValue()
	objective := i.ApplicationCommandData().Options[2].StringValue()
	spreadsheetID := i.ApplicationCommandData().Options[3].StringValue()
	url := "https://docs.google.com/spreadsheets/d/" + spreadsheetID

	// Set the spreadsheet ID
	sheets.SetSpreadSheetID(spreadsheetID)
	srv := config.ConfigSheets()

	// Read party data from the spreadsheet
	partyResult, partyNames := sheets.ReadSheet(srv, "Mooncrat")

	// Create a war preparation embed
	warEmbed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("⚔️ **%s** ⚔️", title),
		Description: fmt.Sprintf("Prepare yourselves for the upcoming war! 🛡️\n\n**Time:** %s\n**Objective:** %s\n\nLet's show them our strength and determination! 💪🔥", time, objective),
		Color:       0xff4500, // Orange-Red color for excitement
		Type:        discordgo.EmbedTypeRich,
		URL:         url,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL:      "https://cdn.discordapp.com/attachments/1238887854848081962/1238887968887017582/mooncrat_logo_4_transparent.png?ex=6652b815&is=66516695&hm=c9e57276cdc1af4a2a7d7c819dcc95bd20500c41a9a807ea27476631b14e7d70&", // URL of the thumbnail image
			ProxyURL: "",                                                                                                                                                                                                            // Proxy URL, leave empty if not needed
			Width:    0,                                                                                                                                                                                                             // Width of the thumbnail (optional)
			Height:   0,                                                                                                                                                                                                             // Height of the thumbnail (optional)
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Instructions",
				Value:  "1. FILL THE PARTY.\n2. Gear up with your best equipment.\n3. Be ready at the designated time.\n4. Follow your party leader's commands.",
				Inline: false,
			},
			{
				Name:   "Why Join?",
				Value:  "Glory, honor, and fantastic rewards await! Let's MOONCRAT BARENG BARENG! <@&1238104075220942892>",
				Inline: false,
			},
		},
	}

	// Send the war preparation embed
	warMessage, err := s.ChannelMessageSendEmbed(i.ChannelID, warEmbed)
	if err != nil {
		log.Printf("Error sending war preparation message: %v", err)
		return
	}

	// Create a thread from the war preparation message
	thread, err := s.MessageThreadStart(i.ChannelID, warMessage.ID, "War Party Assignments", 1440) // 1440 minutes = 24 hours
	if err != nil {
		log.Printf("Error creating thread: %v", err)
		return
	}
	threadSpreadsheetMap[thread.ID] = spreadsheetID

	// Send party data as separate messages
	for i, partyData := range partyResult {
		// Format party data for the current party
		embed := &discordgo.MessageEmbed{
			Title:       fmt.Sprintf("Party Name : %s", partyNames[i]),
			Color:       0x1750ac, // Green color
			Description: "```" + formatPartyData(partyData) + "```",
		}

		message, err := s.ChannelMessageSendEmbed(thread.ID, embed)
		if err != nil {
			log.Printf("Error sending message: %v", err)
		}
		threadMessageIDs[thread.ID] = append(threadMessageIDs[thread.ID], message.ID)
	}
}

func EditMessage(threadID string, partyIndex int, s *discordgo.Session, i *discordgo.InteractionCreate) {
	srv := config.ConfigSheets()

	spreadsheetID := threadSpreadsheetMap[threadID]
	messageID := threadMessageIDs[threadID][partyIndex]
	sheets.SetSpreadSheetID(spreadsheetID)

	// Read party data from the spreadsheet
	partyResult, partyNames := sheets.ReadSheet(srv, "Mooncrat")
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Party Name : %s", partyNames[partyIndex]),
		Color:       0x1750ac, // Green color
		Description: "```" + formatPartyData(partyResult[partyIndex]) + "```",
	}
	_, err := s.ChannelMessageEditEmbed(threadID, messageID, embed)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func newMessage(discord *discordgo.Session, message *discordgo.MessageCreate) {

	/* prevent bot responding to its own message
	this is achived by looking into the message author id
	if message.author.id is same as bot.author.id then just return
	*/
	if message.Author.ID == discord.State.User.ID {
		return
	}

	// respond to user message if it contains `!help` or `!bye`
	switch {
	case strings.Contains(message.Content, "hello"):
		discord.ChannelMessageSend(message.ChannelID, "Hello World😃")

	case strings.Contains(message.Content, "!bye"):
		discord.ChannelMessageSend(message.ChannelID, "Good Bye👋")
		// add more cases if required
	case strings.Contains(message.Content, "fatir"):
		discord.ChannelMessageSend(message.ChannelID, "https://cdn.discordapp.com/attachments/1238106615291445320/1242513662133010535/Screenshot_2024-05-21_at_23.25.32.png?ex=6652b9c5&is=66516845&hm=36bbdd2994265b266d9564930974a43cf78a537bca2ece8ac7e2ed9b911e98e8&")
	case strings.Contains(message.Content, "pija"):
		discord.ChannelMessageSend(message.ChannelID, "https://cdn.discordapp.com/attachments/1238106615291445320/1242751609784762438/image.png?ex=6652eea0&is=66519d20&hm=bfd470064e43dc12e0e2fceae51d12357970ae971d31e95daa554253b3229d92&")
	}

}

func formatPartyData(partyData []sheets.PartyData) string {
	var builder strings.Builder

	// Add headers
	builder.WriteString(fmt.Sprintf("%-3s %-6s %-15s %-20s %-10s\n", "No", "Role", "Weapon", "Notes", "Player"))
	builder.WriteString(strings.Repeat("-", 55) + "\n")

	// Add rows
	for _, pd := range partyData {
		builder.WriteString(fmt.Sprintf("%-3s %-6s %-15s %-20s %-10s\n", truncateString(pd.No, 3), truncateString(pd.Role, 6), truncateString(pd.Weapon, 15), truncateString(pd.Notes, 20), truncateString(pd.Player, 10)))
	}

	return builder.String()
}

func truncateString(str string, num int) string {
	if len(str) > num {
		return str[0:num]
	}
	return str
}
