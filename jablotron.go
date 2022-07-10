package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var t1 time.Time
var oktimer time.Time
var JabloPIN = "485120"
var JStates map[string]string
var JPG map[string]string
var JCommands map[time.Time]string

func TestPub(jd map[int]bool, client mqtt.Client, token mqtt.Token) {

	// publish to topic
	for key, value := range jd {
		TOP := "jablotron/device/" + fmt.Sprintf("%d", key)
		var STA string
		if value == true {
			STA = "ON"
		} else {
			STA = "OFF"
		}
		token = client.Publish(TOP, byte(0), false, STA)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
		}

	}

}

func PubPG(pgid string, pgstate string, client mqtt.Client, token mqtt.Token) {

	// publish to topic

	TOP := "jablotron/pg/" + pgid
	pgstate = strings.TrimSpace(pgstate)
	fmt.Println("Wysylam do ", TOP, "wartosc ", pgstate)
	token = client.Publish(TOP, byte(0), false, pgstate)
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to publish, %v", token.Error())
	}

}

type HASupervisorConfig struct {
	Result string `json:"result"`
	Data   struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Ssl      bool   `json:"ssl"`
		Protocol string `json:"protocol"`
		Username string `json:"username"`
		Password string `json:"password"`
		Addon    string `json:"addon"`
	} `json:"data"`
}

func GetMqttConfigFromHA(token string) (HASupervisorConfig, error) {
	var haconfig HASupervisorConfig
	url := "http://supervisor/services/mqtt"

	// Create a Bearer string by appending string access token
	var bearer = "Bearer " + token

	// Create a new request using http
	req, err := http.NewRequest("GET", url, nil)

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)

	// Send req using http Client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
		return haconfig, err

	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println("---------------------------")
	log.Println(string([]byte(body)))
	fmt.Println("---------------------------")

	if err != nil {
		log.Println("Error while reading the response bytes:", err)
		return haconfig, err

	}
	err = json.Unmarshal(body, &haconfig)
	if err != nil {
		log.Println("Error while reading the response bytes:", err)
		return haconfig, err

	}
	log.Println(string([]byte(body)))
	return haconfig, nil
}

func PublishAlarm(AlarmType string, AlarmZone string, AlarmState string, client mqtt.Client, token mqtt.Token) {
	AlarmState = strings.Trim(strings.TrimSpace(AlarmState), " ")

	// publish to topic
	AlarmType = strings.TrimSpace(AlarmType)
	TOP := "jablotron/alert/" + AlarmType + "/" + AlarmZone
	fmt.Println("Wysylam do ", TOP, "wartosc ", AlarmState)
	token = client.Publish(TOP, byte(0), false, AlarmState)
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to publish, %v", token.Error())
	}

}

var err error

func main() {
	var stok string
	var hacfg HASupervisorConfig
	cf, err := ShowOptionsFile()
	stok = os.Getenv("SUPERVISOR_TOKEN")

	if len(stok) < 2 {
		oa := os.Args
		if len(oa) > 1 {
			stok = os.Args[1]
			hacfg, err = GetMqttConfigFromHA(stok)
		} else {
			hacfg.Data.Host = cf.MQTTHost
			hacfg.Data.Password = cf.MQTTPassword
			hacfg.Data.Username = cf.MQTTUser
			hacfg.Data.Port = cf.MQTTPort
		}
	} else {
		hacfg, err = GetMqttConfigFromHA(stok)
	}
	log.Println(stok)
	fmt.Println(os.Environ())

	if err != nil {
		fmt.Println("Cant Read Config From Supervisor")
		hacfg.Data.Host = cf.MQTTHost
		hacfg.Data.Password = cf.MQTTPassword
		hacfg.Data.Username = cf.MQTTUser
		hacfg.Data.Port = cf.MQTTPort

	}
	JStates = make(map[string]string)
	JPG = make(map[string]string)
	JCommands = make(map[time.Time]string)
	for {
		mainloop(hacfg, cf)
	}

}
func mainloop(hacfg HASupervisorConfig, cf ConfigFile) {
	client, token := MakeMQTTConn(hacfg)
	for {
		if token.Wait() && token.Error() != nil {
			fmt.Println("Wjechal Error Polaczenie MQTT resetuje", token.Error())
			return
		}
		GetFromJablo(client, token, cf)
		time.Sleep(2 * time.Second)
	}
}

func connLostHandler(c mqtt.Client, err error) {
	fmt.Printf("Connection lost, reason: %v\n", err)

	//Perform additional action...
}

func startsub(c mqtt.Client) {
	c.Subscribe("jablotron/+/+/set", 2, HandleMSGfromMQTT)

	//Perform additional action...
}

func HandleMSGfromMQTT(client mqtt.Client, msg mqtt.Message) {
	s := strings.Split(msg.Topic(), "/")
	if len(s) > 2 {
		switch s[1] {
		case "pg":
			if fmt.Sprintf("%s", msg.Payload()) == "ON" {
				JCommands[time.Now()] = JabloPIN + " PGON " + s[2]
			} else {
				JCommands[time.Now()] = JabloPIN + " PGOFF " + s[2]
			}
		case "state":
			if fmt.Sprintf("%s", msg.Payload()) == "ON" {
				JCommands[time.Now()] = JabloPIN + " SET " + s[2]
			} else {
				JCommands[time.Now()] = JabloPIN + " UNSET " + s[2]
			}
		}
	}

	fmt.Printf("* [%s] %s\n", msg.Topic(), string(msg.Payload()))
	fmt.Printf(".")
}

type ConfigFile struct {
	JablotronPIN  int    `json:"JablotronPIN"`
	JablotronIP   string `json:"JablotronIP"`
	JablotronPort int    `json:"JablotronPort"`
	MQTTHost      string `json:"MQTTHost"`
	MQTTPort      int    `json:"MQTTPort"`
	MQTTUser      string `json:"MQTTUser"`
	MQTTPassword  string `json:"MQTTPassword"`
	MQTTProtocol  string `json:"MQTTProtocol"`
}

func ShowOptionsFile() (ConfigFile, error) {
	var cf ConfigFile
	file, err := os.Open("/data/options.json")
	if err != nil {
		log.Println(err)
		return cf, err

	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Println(err)

		}
	}()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return cf, err
	}
	err = json.Unmarshal(b, &cf)
	if err != nil {
		return cf, err
	}
	fmt.Println("---------------------------")
	fmt.Println(string(b))
	fmt.Println("---------------------------")
	return cf, nil
}

func MakeMQTTConn(hacfg HASupervisorConfig) (mqtt.Client, mqtt.Token) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", "tcp", hacfg.Data.Host, hacfg.Data.Port))
	opts.SetPassword(hacfg.Data.Username)
	opts.SetUsername(hacfg.Data.Password)
	fmt.Println("Connstring", fmt.Sprintf("%s://%s:%d", "tcp", hacfg.Data.Host, hacfg.Data.Port), "usernamee", hacfg.Data.Username, "password", hacfg.Data.Password)
	//opts.SetClientID(hacfg.Data.Username)
	opts.SetKeepAlive(time.Second * time.Duration(60))
	opts.SetOnConnectHandler(startsub)
	opts.SetConnectionLostHandler(connLostHandler)
	if hacfg.Data.Protocol == "3.1.1" {
		fmt.Println("Proto v4")
		opts.SetProtocolVersion(4)
	}
	// connect to broker
	client := mqtt.NewClient(opts)
	//defer client.Disconnect(uint(2))

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to connect broker, %v", token.Error())
	}
	return client, token

}

func GetFromJablo(client mqtt.Client, token mqtt.Token, cf ConfigFile) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", cf.JablotronIP, cf.JablotronPort))
	if err != nil {
		return
	}
	t1 = time.Now()
	go TouchJablo(conn)
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}
		if strings.HasPrefix(message, "OK\r") {
			//	fmt.Print("PONG FROM JABLOTRON: " + message)
			oktimer = time.Now()
		} else {
			s := strings.Split(message, " ")
			if len(s) > 1 {
				switchstr := strings.Trim(strings.TrimSpace(s[0]), " ")
				reg, err := regexp.Compile("[^a-zA-Z0-9]+")
				if err != nil {
					fmt.Print(err)
				}
				switchstr = reg.ReplaceAllString(switchstr, "")

				switch switchstr {
				case "PRFSTATE":
					fmt.Printf("Przyszedl status jablotron\n")
					t1 = time.Now()
					str := strings.TrimSpace(s[1])
					JDEV, err := ParseJablotronDevices(str)
					if err != nil {
						fmt.Println(err)
						return
					} else {
						TestPub(JDEV, client, token)
					}

				case "PG":
					fmt.Printf("Przyszedl Status Wyjscia PG - %s\n", message)
					JPG[s[1]] = s[2]
					PubPG(s[1], s[2], client, token)

				case "STATE":
					fmt.Printf("Przyszedl Status Strefy - %s\n", message)
					JStates[s[1]] = s[2]
					PublishStates(client, token)
				case "ENTRY":
					PublishAlarm(switchstr, s[1], s[2], client, token)
				case "EXIT":
					PublishAlarm(switchstr, s[1], s[2], client, token)
				case "INTERNALWARNING":
					PublishAlarm(switchstr, s[1], s[2], client, token)
				case "EXTERNALWARNING":
					PublishAlarm(switchstr, s[1], s[2], client, token)
				case "INTRUDERALARM":
					PublishAlarm(switchstr, s[1], s[2], client, token)
				case "PANICALARM":
					PublishAlarm(switchstr, s[1], s[2], client, token)
				case "FIREALARM":
					PublishAlarm(switchstr, s[1], s[2], client, token)

				default:
					fmt.Println("nieobsluzona wiadomosc: ", message, err)
				}
			}
		}

		//

	}
	return
}

func PublishStates(client mqtt.Client, token mqtt.Token) {
	for key, value := range JStates {
		fmt.Println("Key:", key, "Value:", value)
		TOP := "jablotron/state/" + fmt.Sprintf("%s", key)
		fmt.Println("Publikuje do ", TOP, "warosc", value)
		value = strings.TrimSpace(value)

		token = client.Publish(TOP, byte(0), false, value)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
		}
	}

}

func TouchJablo(conn net.Conn) {
	for {
		for key, value := range JCommands {
			fmt.Println("Wysylam komende ", value)
			//conn.Write([]byte("\n\r"))
			_, err := conn.Write([]byte(value + "\n\r"))
			time.Sleep(1 * time.Second)
			if err != nil {
				fmt.Println("NIe udalo sie wyslac polaczenie zepsute ", err)
				return
			} else {
				//	conn.Write([]byte("\n\r"))
				delete(JCommands, key)
			}
		}

		//	diff := time.Now().Sub(t1)
		if len(JStates) < 1 {
			fmt.Println("NIe mam tabeli statusow wszystkich stref pytam")
			JCommands[time.Now()] = "STATE"

		}
		if len(JPG) < 1 {
			fmt.Println("NIe mam tabeli statusow wszystkich PG pytam")
			JCommands[time.Now()] = "PGSTATE"

		}
		//	if diff.Seconds() > 4 {
		//		fmt.Printf("Nie bylo STANU OD %f pytam \n", diff.Seconds())

		//	JCommands[time.Now()] = "PRFSTATE"

		//	}
		time.Sleep(1 * time.Second)

	}
}

func utb(a uint64) bool {
	if a == uint64(1) {
		return true
	}
	return false
}
func ParseJablotronDevices(instr string) (map[int]bool, error) {
	marker := 0
	m := make(map[int]bool)
	inmarker := 0
	var err error
	for {
		end := marker + 2
		if end <= len(instr) {
			first2 := instr[marker:end]
			//fmt.Println(first2)

			i, err := strconv.ParseUint(first2, 16, 32)
			if err != nil {
				return m, err
			}
			iarr := asBits(i)
			m[inmarker] = iarr[0]
			m[inmarker+1] = iarr[1]
			m[inmarker+2] = iarr[2]
			m[inmarker+3] = iarr[3]
			m[inmarker+4] = iarr[4]
			m[inmarker+5] = iarr[5]
			m[inmarker+6] = iarr[6]
			m[inmarker+7] = iarr[7]

			//	fmt.Println(iarr)
			inmarker = inmarker + 8
			marker = end
		} else {
			break
		}
	}
	return m, err

}

func asBits(val uint64) []bool {
	bits := []bool{}
	//bits2 := []uint64{}

	for i := 0; i < 8; i++ {
		//bits = append([]uint64{val & 0x1}, bits...)
		// or
		bits = append(bits, utb(val&0x1))
		//	bits2 = append(bits2, val&0x1)

		// depending on the order you want
		val = val >> 1
	}
	///	fmt.Println(bits2)
	return bits
}
