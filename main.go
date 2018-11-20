/**
   Captcha Bank - Solve reCAPTCHAs from any site and store the g-recaptcha-response tokens.
   Copyright (C) 2018 Gianluca Oliva

   This program is free software; you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation; either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program; if not, see http://www.gnu.org/licenses/.
 */

package main

import (
	"encoding/json"
	"fmt"
	"github.com/lobre/goodhosts" // Fork of https://github.com/lextoumbourou/goodhosts/ but it closes the hosts file after it's done.
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var hostname = "supremenewyork.com"
var siteKey = "6LeWwRkUAAAAAOBsau7KpuC9AV-6J8mhw4AjC3Xz"
var bank []tokenInfo

type tokenInfo struct {
	Token   string `json:"token"`
	Created int    `json:"created"`
	Expires int    `json:"expires"`
	Host    string `json:"host"`
	SiteKey string `json:"siteKey"`
}

func main() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Captured ctrl+c, removing record.")
		removeRecord()
		os.Exit(1)
	}()

	hosts, _ := goodhosts.NewHosts()
	fmt.Println("Checking for " + hostname + " in the hosts file...")
	if hosts.Has("127.0.0.1", "www."+hostname) && hosts.Has("127.0.0.1", hostname) {
		fmt.Println("Entry found!")
	} else {
		fmt.Println("Entry not found, creating it...")
		hosts.Add("127.0.0.1", "www."+hostname, hostname)
		fmt.Println("Entry created!")
	}
	if err := hosts.Flush(); err != nil {
		panic(err)
	}

	http.HandleFunc("/", bankContents)
	http.HandleFunc("/solve", captchaSolver)
	http.HandleFunc("/submit", captchaSubmitted)

	fmt.Println("Started local web server on port 80.")
	fmt.Println("Access " + hostname + "/solve to solve a captcha")
	fmt.Println("Access " + hostname + " to view valid g-recaptcha-response tokens")

	if err := http.ListenAndServe(":80", nil); err != nil {
		panic(err)
	}
}

func removeRecord() {
	hosts, _ := goodhosts.NewHosts()
	hosts.Remove("127.0.0.1", "www."+hostname, hostname)
	if err := hosts.Flush(); err != nil {
		panic(err)
	}
}

func bankContents(w http.ResponseWriter, r *http.Request) {
	str, _ := json.Marshal(bank)
	w.Write([]byte(str))
}

func captchaSolver(w http.ResponseWriter, r *http.Request) {
	template.Must(template.New("solve").ParseFiles("solve.html")).ExecuteTemplate(w, "solve.html", siteKey)
}

func captchaSubmitted(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		fmt.Println("Added new token to captcha bank!")
		r.ParseForm()

		now := int(time.Now().Unix())
		bank = append(bank, tokenInfo{
			Token:   r.Form.Get("g-recaptcha-response"),
			Created: now,
			Expires: now + 120,
			// Pointless for now but you could adapt this code to work for multiple sites thus tokens may have different sitekeys and hostnames.
			Host:    r.Host,
			SiteKey: siteKey,
		})
	}
}

//TODO: Remove the token from the list once it has expired.
//TODO: Make a function to get a token (returns a valid token that's closest to expiring).
//ISSUE: Because of the host record users wont be able to load supremenewyork.com themselves, at least while the code is running
//I'll try and find away of spoofing supremenewyork.com in an electron window without having to use the hosts file, but for now this basic example will do.
