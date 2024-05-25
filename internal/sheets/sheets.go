package sheets

import (
	"fmt"
	"log"
	"strconv"

	"google.golang.org/api/sheets/v4"
)

// Data struct to hold the party information
type PartyData struct {
	No     string
	Role   string
	Weapon string
	Notes  string
	Player string
}

var spreadsheetID = ""

func SetSpreadSheetID(id string) {
	spreadsheetID = id
}

func ReadSheet(srv *sheets.Service, sheetName string) ([][]PartyData, []string) {
	// Retrieve the party names dynamically
	partyNames, err := getPartyNames(srv, sheetName)
	if err != nil {
		log.Fatalf("Error retrieving party names: %v", err)
	}
	var partyResult [][]PartyData
	// Read data for each party
	for i, partyName := range partyNames {
		startColumn, endColumn := getPartyColumns(i)
		readRange := fmt.Sprintf("%s!%s3:%s", sheetName, startColumn, endColumn)

		resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve data from sheet: %v", err)
		}

		if len(resp.Values) == 0 {
			fmt.Printf("No data found for party %s.\n", partyName)
			continue
		}

		// Parse the data into PartyData structs
		var partyData []PartyData
		var prevRole string
		for _, row := range resp.Values {
			role := getStringFromRow(row, 1)
			if role == "" {
				role = prevRole // Copy previous role if the current role is empty
			} else {
				prevRole = role // Update previous role
			}

			party := PartyData{
				No:     getStringFromRow(row, 0),
				Role:   role,
				Weapon: getStringFromRow(row, 2),
				Notes:  getStringFromRow(row, 3),
				Player: getStringFromRow(row, 4),
			}
			partyData = append(partyData, party)
		}
		partyResult = append(partyResult, partyData)

		log.Print("[ReadSheet] : Succes created party result and party name")
	}

	return partyResult, partyNames
}

func DeletePlayer(srv *sheets.Service, sheetName string, partyName string, no string) error {

	partyIndex, err := findPartyIndex(srv, sheetName, partyName)
	if err != nil {
		return fmt.Errorf("failed to find party index: %v", err)
	}

	_, endColumn := getPartyColumns(partyIndex)

	rowToDelete, err := strconv.ParseInt(no, 10, 64)
	if err != nil {
		return fmt.Errorf("failed catcth row")
	}

	readRange := fmt.Sprintf("%s!%s%d", sheetName, endColumn, rowToDelete+2)

	cvr := &sheets.ClearValuesRequest{}
	_, err = srv.Spreadsheets.Values.Clear(spreadsheetID, readRange, cvr).Do()

	if err != nil {
		log.Fatalf("Unable to clear data from sheet: %v", err)
	}

	return nil

}

func AssignPlayerToSheet(srv *sheets.Service, sheetName string, partyName string, no string, player string) error {
	partyIndex, err := findPartyIndex(srv, sheetName, partyName)
	if err != nil {
		return fmt.Errorf("failed to find party index: %v", err)
	}

	_, endColumn := getPartyColumns(partyIndex)

	rowToUpdate, err := strconv.ParseInt(no, 10, 64)
	if err != nil {
		return fmt.Errorf("failed catcth row")
	}

	readRange := fmt.Sprintf("%s!%s%d", sheetName, endColumn, rowToUpdate+2)
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return fmt.Errorf("failed to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) != 0 {
		return fmt.Errorf("already filled")
	}

	valueInputOption := "RAW"
	values := &sheets.ValueRange{
		Values: [][]interface{}{{
			player,
		}},
	}

	_, err = srv.Spreadsheets.Values.Update(spreadsheetID, readRange, values).ValueInputOption(valueInputOption).Do()
	if err != nil {
		return fmt.Errorf("failed to update player value: %v", err)
	}

	return nil
}

func findPartyIndex(srv *sheets.Service, sheetName string, partyName string) (int, error) {
	readRange := fmt.Sprintf("%s!A1:Z1", sheetName)

	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		return -1, fmt.Errorf("failed to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) == 0 {
		return -1, fmt.Errorf("no header row found in sheet")
	}

	headerRow := resp.Values[0]
	for i, cell := range headerRow {
		if cell == partyName {
			return i / 5, nil // Each party occupies 5 columns
		}
	}

	return -1, fmt.Errorf("party not found in sheet")
}

func getStringFromRow(row []interface{}, index int) string {
	if index < len(row) {
		return getString(row[index])
	}
	return ""
}

func getString(value interface{}) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

// Function to retrieve party names dynamically
func getPartyNames(srv *sheets.Service, sheetName string) ([]string, error) {
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, fmt.Sprintf("%s!A1:Z1", sheetName)).Do()
	if err != nil {
		return nil, err
	}

	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("No header row found")
	}

	headerRow := resp.Values[0]
	var partyNames []string
	for i := 0; i < len(headerRow); i++ {
		if i%5 == 0 && headerRow[i] != nil {
			partyNames = append(partyNames, headerRow[i].(string))
		}
	}
	return partyNames, nil
}

func getPartyColumns(partyIndex int) (string, string) {
	startColumn := 'A' + (partyIndex * 5)
	endColumn := startColumn + 4
	return string(startColumn), string(endColumn)
}
