// +build interactive

package main

import (
	"bufio"
	"fmt"
	"github.com/thejerf/suture"
	"os"
	"github.com/upwrd/sift"
	"github.com/upwrd/sift/db"
	"strconv"
	"strings"
)

func main() {
	//sift.SetLogLevel("debug")
	//ipv4.Log.SetHandler(log.DiscardHandler()) // ignore ipv4 scanner stuff

	// Instantiate a SIFT server
	server, err := sift.NewServer(sift.DefaultDBFilepath)
	if err != nil {
		panic(err)
	}
	if err = server.AddDefaults(); err != nil {
		panic(err)
	}

	// Start the server as a suture process
	supervisor := suture.NewSimple("interactive terminal (SIFT app)")
	servToken := supervisor.Add(server)
	defer supervisor.Remove(servToken)
	go supervisor.ServeBackground()

	// Run the SIFT script
	runInteractive(server)
}

// runInteractive provides an interactive terminal interface for manipulating
// a SIFT server
func runInteractive(server *sift.Server) {
	topLevel(server)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func topLevel(server *sift.Server) {
	fmt.Printf("-- Main Menu --\n")
	fmt.Printf("[enter L for locations, D for Devices, Q to quit]\n")
	bio := bufio.NewReader(os.Stdin)
	lineByte, _, err := bio.ReadLine()
	checkErr(err)
	line := string(lineByte)

	switch {
	case strings.HasPrefix(line, "l"), strings.HasPrefix(line, "L"):
		locations(server)
	case strings.HasPrefix(line, "d"), strings.HasPrefix(line, "D"):
		devices(server)
	case strings.HasPrefix(line, "q"), strings.HasPrefix(line, "Q"):
		fmt.Printf("quitting...\n")
	default:
		// Any other input, just repeat the prompt
		topLevel(server)
	}
}

func locations(server *sift.Server) {
	// Get the locations from database
	siftDB, err := server.DB()
	checkErr(err)
	locs := []db.Location{}
	siftDB.Select(&locs, "SELECT * FROM location")

	// Print the prompt
	fmt.Printf("-- Locations --\n")
	for _, loc := range locs {
		fmt.Printf("   %v %v\n", loc.ID, loc.Name)
	}

	fmt.Printf("[enter R to refresh, A to add a location, an ID to edit, or blank to go back]\n")
	bio := bufio.NewReader(os.Stdin)
	lineByte, _, err := bio.ReadLine()
	checkErr(err)
	line := string(lineByte)

	switch {
	case line == "":
		topLevel(server) // return to top level
	case strings.HasPrefix(line, "r"), strings.HasPrefix(line, "R"):
		locations(server) // refresh by recalling the function
	case strings.HasPrefix(line, "a"), strings.HasPrefix(line, "A"):
		addLocation(server)
	default:
		// Try to parse the input as a number
		id, err := strconv.Atoi(string(line))
		if err != nil {
			locations(server)
		}
		editLocation(server, int64(id))
	}
}

func addLocation(server *sift.Server) {
	// Print the prompt
	fmt.Printf("-- Add Location --\n")
	fmt.Printf("[Enter name (blank to go back)]:\n")
	bio := bufio.NewReader(os.Stdin)
	lineByte, _, err := bio.ReadLine()
	checkErr(err)
	line := string(lineByte)

	switch {
	case line == "":
		locations(server) // go back up to locations
	default:
		siftDB, err := server.DB()
		checkErr(err)
		res, err := siftDB.Exec("INSERT INTO location (name) VALUES (?)", line)
		checkErr(err)
		id, err := res.LastInsertId()
		checkErr(err)
		fmt.Printf(">> inserted %v as location %v\n", line, id)
		locations(server)
	}
}

func editLocation(server *sift.Server, id int64) {
	// Print the prompt
	fmt.Printf("-- Edit Location %v --\n", id)
	fmt.Printf("[enter N to change name, D to delete, or blank to go back]\n")
	bio := bufio.NewReader(os.Stdin)
	lineByte, _, err := bio.ReadLine()
	checkErr(err)
	line := string(lineByte)

	switch {
	case line == "":
		locations(server) // go back up to locations
	case strings.HasPrefix(line, "d"), strings.HasPrefix(line, "D"):
		// perform delete
		siftDB, err := server.DB()
		checkErr(err)
		_, err = siftDB.Exec("DELETE FROM location WHERE id=?", id)
		checkErr(err)
		fmt.Printf(">> deleted location %v\n", id)
		locations(server)
	case strings.HasPrefix(line, "n"), strings.HasPrefix(line, "N"):
		// change the name
		editLocationName(server, id)
	default:
		// if unknown, repeat prompt
		editLocation(server, id)
	}
}

func editLocationName(server *sift.Server, id int64) {
	// Print the prompt
	fmt.Printf("-- Edit name for Location %v--\n", id)
	fmt.Printf("Name? (blank to go back):\n")
	bio := bufio.NewReader(os.Stdin)
	lineByte, _, err := bio.ReadLine()
	checkErr(err)
	line := string(lineByte)

	switch {
	case line == "":
		locations(server) // go back up to locations
	default:
		siftDB, err := server.DB()
		checkErr(err)
		res, err := siftDB.Exec("UPDATE location SET name=? WHERE id=?", line, id)
		checkErr(err)
		n, err := res.RowsAffected()
		checkErr(err)
		if n == 0 {
			fmt.Printf(">> no rows affected, try again?\n")
		} else {
			fmt.Printf(">> location %v is now named %v\n", id, line)
		}
		locations(server)
	}
}

func devices(server *sift.Server) {
	// Get the devices from database
	devs, err := server.SiftDB.GetDevices(db.ExpandNone)
	checkErr(err)

	// Print the prompt
	fmt.Printf("-- Devices --\n")
	for id, dev := range devs {
		fmt.Printf("   %v: %+v\n", id, dev)
	}

	fmt.Printf("[enter R to refresh, an ID to edit, or blank to go back]\n")
	bio := bufio.NewReader(os.Stdin)
	lineByte, _, err := bio.ReadLine()
	checkErr(err)
	line := string(lineByte)

	switch {
	case line == "":
		topLevel(server) // return to top level
	case strings.HasPrefix(line, "r"), strings.HasPrefix(line, "R"):
		devices(server) // refresh by recalling the function
	default:
		// Try to parse the input as a number
		id, err := strconv.Atoi(string(line))
		if err != nil {
			devices(server)
		}
		editDevice(server, int64(id))
	}
}

func editDevice(server *sift.Server, id int64) {
	// Print the prompt
	fmt.Printf("-- Edit Device %v --\n", id)
	fmt.Printf("[enter N to change name, L to set location, or blank to go back]\n")
	bio := bufio.NewReader(os.Stdin)
	lineByte, _, err := bio.ReadLine()
	checkErr(err)
	line := string(lineByte)

	switch {
	case line == "":
		locations(server) // go back up to locations
	case strings.HasPrefix(line, "n"), strings.HasPrefix(line, "N"):
		// change the name
		fmt.Printf("TODO: go to edit Device name...\n")
		editDevice(server, id)
	case strings.HasPrefix(line, "l"), strings.HasPrefix(line, "L"):
		// edit location
		editDeviceLocation(server, id)
	default:
		// if unknown, repeat prompt
		editLocation(server, id)
	}
}

func editDeviceLocation(server *sift.Server, id int64) {
	// Get available locations from database
	siftDB, err := server.DB()
	checkErr(err)
	locs := []db.Location{}
	siftDB.Select(&locs, "SELECT * FROM location")

	// Print the prompt
	fmt.Printf("-- Set Location for Device %v --\n", id)
	fmt.Printf("Available Locations:\n")
	for _, loc := range locs {
		fmt.Printf("   %v %v\n", loc.ID, loc.Name)
	}
	fmt.Printf("[enter an ID to set Location, R to refresh Locations, or blank to go back]\n")

	bio := bufio.NewReader(os.Stdin)
	lineByte, _, err := bio.ReadLine()
	checkErr(err)
	line := string(lineByte)

	switch {
	case line == "":
		editDevice(server, id) // return to top level
	case strings.HasPrefix(line, "r"), strings.HasPrefix(line, "R"):
		editDeviceLocation(server, id) // refresh by recalling the function
	default:
		// Try to parse the input as a number
		locID, err := strconv.Atoi(string(line))
		if err != nil {
			editDeviceLocation(server, id)
		}
		res, err := siftDB.Exec("UPDATE device SET location_id=? WHERE id=?", locID, id)
		checkErr(err)
		n, err := res.RowsAffected()
		checkErr(err)
		if n == 0 {
			fmt.Printf(">> no rows affected, try again?\n")
		} else {
			fmt.Printf(">> device %v is now in location %v\n", id, locID)
		}
		devices(server)
	}
}
