package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	logMachine "GazPromAPI/logMachine"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Api_site  string
	Api_token string
	Login     string
	Password  string
	Contract  string
}

type sessionInfo struct {
	Session_id  string
	Contract_id string
}

type errTypes struct {
	Type    string
	Message string
}

type contract struct {
	Id     string
	Number string
}

type authAnswer struct {
	Status struct {
		Code   int
		Errors []errTypes
	}
	Data struct {
		Client_id  string
		Session_id string
		Contracts  []contract
	}
}

type infoAnswer struct {
	Status struct {
		Code   int
		Errors []errTypes
	}
	Data struct {
		BalanceData struct {
			Balance string
		}
	}
}

func readConfig(filename string) Config {
	yfile, err := ioutil.ReadFile(filename)

	if err != nil {
		logMachine.Error("Ошибка открытия файла конфигурации")
		logMachine.Fatal(err)
	}

	data := Config{}

	err = yaml.Unmarshal(yfile, &data)

	if err != nil {
		logMachine.Error("Ошибка разбора файла конфигурации")
		logMachine.Fatal(err)
	}

	logMachine.Info("Файл конфигурации ", filename, " успешно разобран")

	return data
}

func authRequest(config Config) authAnswer {
	data := url.Values{
		"login":    {config.Login},
		"password": {config.Password},
	}

	u, _ := url.ParseRequestURI(config.Api_site)
	u.Path = "/vip/v1/authUser"
	urlStr := fmt.Sprintf("%v", u)

	req, err := http.NewRequest("POST", urlStr, strings.NewReader(data.Encode()))
	if err != nil {
		logMachine.Error("Ошибка формирования запроса авторизации")
		logMachine.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("api_key", config.Api_token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logMachine.Error("Ошибка выполенения запроса авторизации")
		logMachine.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logMachine.Fatal(err)
	}

	answer := authAnswer{}

	err = json.Unmarshal(body, &answer)
	if err != nil {
		logMachine.Error("Ошибка разбора ответа авторизации")
		logMachine.Fatal(err)
	}

	return answer
}

func getSessionFromServer(config Config) sessionInfo {
	logMachine.Info("Получение id сессии с сервера")

	answer := authRequest(config)

	// Обработка ошибок авторизации
	if answer.Status.Code != 200 {
		logMachine.Error("Неуспешная авторизация")
		logMachine.Error("Код: " + strconv.Itoa(answer.Status.Code))
		logMachine.Fatal(answer.Status.Errors[len(answer.Status.Errors)-1].Message)
	}

	logMachine.Info("Успешная авторизация ", config.Login)

	// ------------------------------------------------------------------------------

	s := sessionInfo{answer.Data.Session_id, ""}

	// если в конфиге нет указания договора, берём первый
	if config.Contract == "" {
		s.Contract_id = answer.Data.Contracts[0].Id
		logMachine.Info("Используется договор: ", answer.Data.Contracts[0].Number)
		return s
	}

	// если в конфиге есть номер договора, ищем его
	for _, c := range answer.Data.Contracts {
		if strings.Contains(c.Number, config.Contract) {
			s.Contract_id = c.Id
			logMachine.Info("Используется договор: ", c.Number)
			return s
		}
	}

	logMachine.Fatal("Договор " + config.Contract + " не найден")

	return s
}

func getSessionFromFile(file string) (sessionInfo, error) {
	logMachine.Info("Чтение id сессии из файла")

	data := sessionInfo{}
	f, err := os.Open(file)
	if err != nil {
		return data, err
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	json.Unmarshal(buf, &data)
	return data, err
}

func saveSessionToFile(filename string, session_info sessionInfo) {
	logMachine.Info("Запись id сессии в файл")

	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		logMachine.Fatal("Невозможно создать файл с id сессии: %v", err)
	}
	defer f.Close()

	data, _ := json.Marshal(session_info)

	_, err = f.Write(data)
	if err != nil {
		logMachine.Fatal("Невозможно записать файл с id сессии: %v", err)
	}
}

func getSession(config Config, file string, force bool) sessionInfo {
	session_info, err := getSessionFromFile(file)
	if (err != nil) || force {
		// не получилось взять id сессии из файла
		// получаем новый id от сервера
		session_info = getSessionFromServer(config)
		// записываем id в файл
		saveSessionToFile(file, session_info)
	}
	return session_info
}

func getInfo(config Config, s sessionInfo) infoAnswer {
	logMachine.Info("Запрос информации по договору id = ", s.Contract_id)

	data := url.Values{
		"contract_id": {s.Contract_id},
	}
	u, _ := url.ParseRequestURI(config.Api_site)
	u.Path = "/vip/v1/getPartContractData"
	u.RawQuery = data.Encode()
	urlStr := fmt.Sprintf("%v", u)

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		logMachine.Fatal(err)
	}
	req.Header.Add("api_key", config.Api_token)
	req.Header.Add("session_id", s.Session_id)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logMachine.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logMachine.Fatal(err)
	}

	answer := infoAnswer{}

	err = json.Unmarshal(body, &answer)
	if err != nil {
		logMachine.Error("Ошибка разбора ответа авторизации")
		logMachine.Fatal(err)
	}

	logMachine.Info("Код ответа: ", answer.Status.Code)
	for _, e_msg := range answer.Status.Errors {
		logMachine.Info("Сообщение сервера: ", e_msg.Message)
	}

	return answer
}

func main() {

	// 0 - только ошибки
	// >= 1 - инфо + ошибки
	v := flag.Int("v", 0, "Уровень подробности лога")
	logFileName := flag.String("log", "", "Имя log-файла")
	configFileName := flag.String("config", "config.yaml", "Имя конфиг-файла")
	flag.Parse()

	/**v = 1
	*logFileName = "info.log"
	*configFileName = "config.demo.yaml"*/

	if *logFileName != "" {
		logMachine.SetLogFile(*logFileName)
		//defer logMachine.Default().File.Close()
	}
	logMachine.SetVerbLevel(*v)

	var config = readConfig(*configFileName)

	s := getSession(config, "session.id", false)

	info := getInfo(config, s)

	// кроме 401 кода "Необходима авторизация"
	// спустя сутки появляется ещё
	// 403 "Нет доступа (Не найден контракт, карта или группа)"
	// поэтому переавторизуемся при любом коде кроме 200
	if info.Status.Code != 200 {
		s = getSession(config, "session.id", true)
		info = getInfo(config, s)
	}

	fmt.Println(info.Data.BalanceData.Balance)

}
