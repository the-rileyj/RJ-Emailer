package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	mailgun "gopkg.in/mailgun/mailgun-go.v1"
)

//Struct to hold the private and public keys for the MailGun API
type info struct {
	Private string `json:"Private"`
	Public  string `json:"Public"`
}

//Function to send an email to the email address(es) with a subject and body depicted by the "address", "subject", and "body" arguments, respectively,
//from the address depicted to by "from", send by the dereferenced mg pointer (we can use a pointer here because since there is no use in the main goroutine
//it means we don't have to worry about data racing and therefor don't have to copy the whole struct); will print out an error if such a thing occurs
func sendEmails(from, subject, body string, addresses []string, mg *mailgun.Mailgun) {
	for _, address := range addresses {
		_, _, err := mailgun.Mailgun(*mg).Send(mailgun.NewMessage(from, subject, body, address))
		if err != nil {
			log.Println(err)
		} else {
			fmt.Printf("Email Sent to %s Successfully\n", address)
		}
	}
}

func main() {
	/* 	Variables	- Arg Inclusion	- Description
	subject 	- not manditory	- the subject of the email, recieved from the command line arg "sub"
	mailserver	- manditory		- the mailserver that will be used to send the email, should correspond to one you've setup on the mailgun site
	keys		- manditory		- string representing the path to the json file with the public and private keys for the mailgun API as 'Public' and 'Private' as the names of the fields in the json object
	body		- not manditory	- the body of the email, recieved from the command line arg "body"
	address		- not manditory - the address(es) which will recieve an email, more than one seperated by spaces, recieved from the command line arg "addr"
	sDate		- manditory		- the date determining when the emails will begin to be sent, recieved from the command line arg "sdate"
	eDate		- not manditory - the date determining when the emails will stop being sent, recieved from the command line arg "edate"
	repeat		- not manditory*- the time interval determining how often an email will be sent, recieved from the command line arg "repeat"; manditory if the command line arg "edate" is depicted
	from		- manditory		- the address which the email will show it was sent from
	*/
	var address, body, eDate, from, keys, mailServer, sDate, rpt, subject string

	/*	Variables	- Description
		eTime		- The ending time parsed from the string variable "eDate"
		nTime		- The variable which will hold the next time an email will be sent later in the email sending loop
		sTime		- The starting time parsed from the string variable "sDate"
	*/
	var eTime, nTime, sTime time.Time

	// 	rTime		- The duration between when emails will be sent parsed from the string variable rpt
	var rTime time.Duration

	//	err			- The variable which catches the various errors throughout the program
	var err error

	//Getting command line arguments and parsing them into their corresponding variables
	flag.StringVar(&subject, "sub", "", "Subject of the email")
	flag.StringVar(&body, "body", "", "Body of the email")
	flag.StringVar(&mailServer, "mailserver", "", "The mailserver setup on the mailgun site which will be sending the email")
	flag.StringVar(&keys, "keys", "", "Path to the json file with the public and private keys for the mailgun API")
	flag.StringVar(&from, "from", "", "Address the email will be sent from")
	flag.StringVar(&address, "addr", "", "Address(es) which will be recieving the email, more than one address should be seperated by a space")
	flag.StringVar(&sDate, "sdate", "", "Date which you want the email sent on, in the format: Jan _2 15:04:05 2006")
	flag.StringVar(&eDate, "edate", "", "Date which you want the emails (in the case of repitition) to stop being sent, in the format: Jan _2 15:04:05 2006")
	flag.StringVar(&rpt, "repeat", "", "How often emails are to be sent following the start sDate, in the format: [number][unit], where unit can be: 'ns', 'us', 'ms', 's', 'm', 'h'")
	flag.Parse()

	//Assuring the address argument isn't empty
	if address == "" {
		log.Fatal("Address argument cannot be blank\n")
	}

	//Assuring the from argument isn't empty
	if from == "" {
		log.Fatal("From address argument cannot be blank\n")
	}

	//Assuring the mailServer argument isn't empty
	if mailServer == "" {
		log.Fatal("Mail server argument cannot be blank\n")
	}

	//Assuring the keys argument isn't empty
	if keys == "" {
		log.Fatal("Keys argument cannot be blank\n")
	}

	//Assuring the starting date argument isn't empty and if it is defaulting to right now
	if sDate != "" {
		sTime, err = time.ParseInLocation("Jan _2 15:04:05 2006", sDate, time.Local) //Parse the start time into the local time
		if err != nil {
			log.Fatal("Error parsing starting time")
		}
	} else {
		sTime = time.Now()
	}

	//Parsing the repetition period of the emails and ending date of the repition if each respective argument is specified
	if rpt != "" || eDate != "" {
		if rpt == "" && eDate != "" {
			log.Fatal("Need both repeat and end date arguments to be specified if including an ending date")
		}
		rTime, err = time.ParseDuration(rpt)
		if err != nil {
			log.Fatal("Error parsing repeating duration")
		}
		if eDate != "" {
			eTime, err = time.ParseInLocation("Jan _2 15:04:05 2006", eDate, time.Local)
			if err != nil {
				log.Fatal("Error parsing end time")
			}
			if sTime.After(eTime) {
				log.Fatal("End time is before start time")
			}
		}
	}

	//Address(es) being parsed from the "address variable"
	addresses := strings.Split(address, " ")

	//Declaration of the type "info" variable "temp", used for getting the public and private key from the json file to initialize a new MailGun struct
	var temp info
	//Attempting to read the file from the path specified by the "keys" command line argument
	data, err := ioutil.ReadFile(keys)
	if err != nil {
		log.Fatal("Error reading data")
	}

	//Unmarshling the slice of bytes from the json file representing the object into the "temp" struct
	json.Unmarshal(data, &temp)
	mg := mailgun.NewMailgun(mailServer, temp.Private, temp.Public)

	fmt.Printf("Subject: %s\nBody: %s\naddress(es): %v\nStart Date: %s (%v from now)\nEnd Date: %s (%v from now)\nRepeat: %s\n", subject, body, addresses, sDate, sTime.Sub(time.Now()), eDate, eTime.Sub(time.Now()), rpt)
	//Initializing the next time with the starting time and then determining where the next time should be from a comparison between the time
	//now and the starting time, just sleeping in the case that the current time is before the starting time
	nTime = sTime
	if time.Now().After(sTime) {
		if rpt == "" {
			//Checking to see if there's a repetition period, and in the case the repetition period is blank it would instrinsically mean that
			//the period to send the single email has passed and therefore quits
			log.Fatal("The start time has already passed, specify a new time")
		} else if eDate != "" && time.Now().After(eTime) {
			//Checing to see if there's an end date and if there is, then whether or not right now is after the ending time, which would mean
			//that the range has passed and the program should quit
			log.Fatal("The time range specified has already passed, specify a new range")
		} else {
			//Since right now is within the starting and ending time range (or just after the starting time in the case an infinite sending
			//is specified through not providing an ending date after specifying a repetition period) the next time an email is to be sent is
			//initialized with the correct time and then set to sleep until that time
			for time.Now().After(nTime) {
				nTime = nTime.Add(rTime)
			}
			time.Sleep(nTime.Sub(time.Now()))
		}
	} else {
		time.Sleep(sTime.Sub(time.Now())) //Sleep until the start time
	}

	//If the repitition period is not blank then it begins the email loop to send that the interval specified,
	//otherwise a single email is sent out as specified by the lack of repitition period
	if rpt != "" {
		if eDate != "" {
			for eTime.After(time.Now().Add(time.Second)) { //Run while the end time is after the time right now
				go sendEmails(from, subject, body, addresses, &mg)
				nTime = nTime.Add(rTime)
				time.Sleep(nTime.Sub(time.Now()))
			}
		} else {
			for { // Run until program is killed
				go sendEmails(from, subject, body, addresses, &mg)
				nTime = nTime.Add(rTime)
				time.Sleep(nTime.Sub(time.Now()))
			}
		}
		sendEmails(from, subject, body, addresses, &mg)
	} else {
		sendEmails(from, subject, body, addresses, &mg)
	}
}
