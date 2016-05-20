package main

import (
    "os"
    "net/http"
    "net/url"
    "fmt"
    "io/ioutil"
    "encoding/json"
    "regexp"
    "time"
    "sync"
    "bytes"
    "strings"
    "sort"
    "strconv"
//    "gopkg.in/telegram-bot-api.v4"
    "./tgbotapi"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
//    "gopkg.in/robfig/cron.v2"
)

type (

    Config struct {
        MongoHost             string
        MongoDB               string
        MongoUser             string
        MongoPassword         string
        BotId                 string
        FilmsUpdateFreequency string
    }

    subscriptionStruct struct {
        ID         string   "_id,omitempty"
        FilmId     string `bson:"film_id"`
        TelegramId int64 `bson:"telegramid"`
    }

    userStruct struct {
        ID         string `_id,omitempty`
        Type       string `json:"type",bson:"type"`
        TelegramId int64  `json:"telegramid",bson:"telegramid"`
        Name       string `json:"name",bson:"name"`
    }

    scheduleStruct struct {
        ID     string `json:"id",bson:"id"`
        FilmId string `json:"film_id",bson:"film_id"`
        Day    string `json:"day",bson:"day""`
        Time   string `json:"time",bson:"time"`
        Price  string `json:"price",bson:"price"`
    }

    filmInfoStrust struct {
        Title       string `json:"title",bson:"title"`
        Description string `json:"text",bson:"description"`
    }

    resultStruct struct {
        ID       string `json:"id",bson:"id"`
        Position int `json:"position",bson:"position"`
        IsActive int `json:"enabled",bson:"enabled"`
        Fact     string `json:"fact",bson:"fact"`
        Info     filmInfoStrust `json:"info",bson:"info"`
        Start    string `json:"pokazyvat_s",bson:"start"`
        End      string `json:"pokazyvat_do",bson:"end"`
        Title    string `json:"originalnoe_nazvanie_filma",bson:"title"`
        Year     string `json:"god",bson:"year"`
        Created  string `json:"proizvodstvo",bson:"created"`
        Country  string `json:"strana",bson:"country"`
        Length   string `json:"dlitelnost_min",bson:"length"`
        Dub      string `json:"dubljazh",bson:"dub"`
        Format   string `json:"format",bson:"format"`
        Genre    []string `json:"zhanr",bson:"genre"`
        Director string `json:"rezhisser",bson:"director"`
        Actors   string `json:"aktery",bson:"actors"`
    }

    filmStruct struct {
        ID        string `json:"-",bson:"_id,omitempty"`
        Time      string `json:"time",bson:"time"`
        Status    string `bson:status`
        Seance    []string `json:"seance",bson:"seance"`
        Result    resultStruct `json:"result",bson:"result"`
        MainPhoto string `json:"main_photo",bson:"main_photo"`
        Schedule  map[string][]scheduleStruct `json:"schedule",bson:"schedule"`
        AvgRate   float64 `json:"avg",bson:"avg_rate"`
    }
)

var (
    bot = &tgbotapi.BotAPI{}

    mongoDBDialInfo = &mgo.DialInfo{}

    filmRegExp = [2]string{"load_film_info\\(([0-9]+)\\);", "load_film_info\\(([0-9]+), 'anonce'\\);"}

    mongoSession *mgo.Session
    err, mongoError error

    commandDelimiter string = "@"

    conf Config
    siteUrl = "http://portalcinema.com.ua/"
//siteImagePath = "uploads/products/main/"

    dayNames = map[string]string{
        "1": "Понедельник",
        "2": "Вторник",
        "3": "Среда",
        "4": "Четверг",
        "5": "Пятница",
        "6": "Суббота",
        "7": "Воскресенье"}

    user userStruct
)

func log(msg string) {
    fmt.Println(time.Now().Format("2006-01-02 15:04:05"), msg)
}

func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}

func getFilmIds(regex string) (ids []string) {
    resp, _ := http.Get(siteUrl)
    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)
    r, _ := regexp.Compile(regex)
    filmIds := r.FindAllStringSubmatch(string(body), -1)
    for key := range (filmIds) {
        if !stringInSlice(filmIds[key][1], ids) {
            ids = append(ids, filmIds[key][1])
        }
    }
    return ids
}

func loadFilmInfo(filmId string, channel chan filmStruct, wg *sync.WaitGroup) {
    var film filmStruct
    resp, err := http.PostForm(
        siteUrl + "products/index/getinfo",
        url.Values{"film": {filmId}})
    defer resp.Body.Close()
    if ( err != nil ) {
        log(fmt.Sprintf("Cant load film #%v", filmId))
    }
    body, _ := ioutil.ReadAll(resp.Body)
    json.Unmarshal(body, &film)
    channel <- film
    wg.Done()
}

func loadFilms(regExp string) (films []filmStruct) {
    var wg sync.WaitGroup
    channel := make(chan filmStruct)
    currentIds := getFilmIds(regExp)
    for _, id := range currentIds {
        wg.Add(1)
        go loadFilmInfo(id, channel, &wg)
    }
    go func() {
        for filmData := range channel {
            films = append(films, filmData)
        }
    }()
    wg.Wait()
    return
}

func updateFilms() {
    session := getSession()
    defer session.Close()
    c := session.DB(conf.MongoDB).C("Films")
    c.RemoveAll(nil)
    for _, film := range loadFilms(filmRegExp[0]) {
        film.Status = "current"
        mongoError = c.Insert(&film)
        if (mongoError != nil) {
            log("Cant save 'current' film")
        }
    }
    for _, film := range loadFilms(filmRegExp[1]) {
        film.Status = "announce"
        mongoError = c.Insert(&film)
        if (mongoError != nil) {
            log(fmt.Sprintf("Cant save 'announce' film", mongoError))
        }
    }
}

func loadConfig() {
    file, err := os.Open("conf.json")
    if (err != nil) {
        panic(err)
    }
    decoder := json.NewDecoder(file)
    err = decoder.Decode(&conf)
    if err != nil {
        panic(err)
    }
    mongoDBDialInfo = &mgo.DialInfo{
        Addrs:    []string{conf.MongoHost},
        Timeout:  60 * time.Second,
        Database: conf.MongoDB,
        Username: conf.MongoUser,
        Password: conf.MongoPassword,
    }
}

func init() {
    loadConfig()
    films := []filmStruct{}
    session := getSession()
    defer session.Close()
    c := session.DB(conf.MongoDB).C("Films")
    c.Find(bson.M{}).All(&films)
    if (len(films) == 0) {
        updateFilms()
    }
}

func getSession() *mgo.Session {
    if mongoSession == nil {
        mongoSession, mongoError = mgo.DialWithInfo(mongoDBDialInfo)
        if mongoError != nil {
            log(fmt.Sprintf("Mongo CreateSession: %s", mongoError))
        }
    }
    return mongoSession.Clone()
}

func searchFilms(query bson.M) (films []filmStruct) {
    session := getSession()
    defer session.Close()
    c := session.DB(conf.MongoDB).C("Films")
    c.Find(query).All(&films)
    return
}

func concat(str... string) string {
    var buff bytes.Buffer
    for _, part := range (str) {
        buff.WriteString(part)
    }
    return buff.String()
}

func getCommandArguments(m *tgbotapi.Message) string {
    if !m.IsCommand() {
        return ""
    }
    split := strings.SplitN(m.Text, commandDelimiter, 2)
    if len(split) != 2 {
        return ""
    }
    return strings.SplitN(m.Text, commandDelimiter, 2)[1]
}

func stripTags(str string) string {
    re, _ := regexp.Compile("\\<[\\S\\s]+?\\>")
    str = re.ReplaceAllString(str, "")
    return str
}

func getAllMessage() (message string) {
    films := searchFilms(bson.M{"status": "current"})
    for _, film := range films {
        message = concat(message,
            "\n", "*",
            film.Result.Info.Title,
            "* (", fmt.Sprintf("%.1f", film.AvgRate),
            "\u2606",
            ")\nПодробнее: /info",
            commandDelimiter,
            film.Result.ID,
            " Сеансы: /seances",
            commandDelimiter,
            film.Result.ID,
            "\n")
    }
    return
}

func getAnnouncementMessage() (message string) {
    films := searchFilms(bson.M{"status": "announce"})
    for _, film := range films {
        message = concat(message, "\n", "*", film.Result.Info.Title, "* (", fmt.Sprintf("%.1f", film.AvgRate), "\u2606", ")\n")
        message = concat(message, "*Премьера:* ", film.startDate(), "\n")
        message = concat(message, "Подробнее: /info", commandDelimiter, film.Result.ID, " Напомнить: /remind", commandDelimiter, film.Result.ID, "\n")
    }
    return
}

func getFilmInfo(id string) (film *filmStruct) {
    session := getSession()
    defer session.Close()
    c := session.DB(conf.MongoDB).C("Films")
    c.Find(bson.M{"result.id" : id}).One(&film)
    return
}

func getFooterLinks() string {
    return "\n*Все фильмы:* /all  *Анонс:* /announcement  *Помощь:* /help"
}

func getFilmMessage(id string) (message string) {
    film := getFilmInfo(id)
    if (film != nil) {
        message = concat(
            "*", film.Result.Info.Title, "*  (", fmt.Sprintf("%.1f", film.AvgRate), "\u2606", ")\n",
            "_", film.Result.Title, "_ \n\n",
            "*Даты показа:* ", film.startDate(), " - ", film.endDate(), "\n",
            "*Жанр:* ", strings.Join(film.Result.Genre, ", "), "\n",
            "*Режисер:* ", film.Result.Director, "\n",
            "*Актеры:* ", film.Result.Actors, "\n",
            "*Производство:* ", film.Result.Country, " (", film.Result.Year, ")\n",
            "*Продолжительность:* ", film.Result.Length, " минут\n")
        if film.Status == "current" {
            message = concat(message, "*Расписание сеансов: * /seances", commandDelimiter, film.Result.ID, "\n")
        } else {
            message = concat(message, "*Напомнить:* /remind", commandDelimiter, film.Result.ID, "\n")
        }
        message = concat(message, stripTags(film.Result.Info.Description), getFooterLinks())
    }else {
        message = "Фильм не найден"
    }
    return
}

func remindFilm(id string) (message string) {
    session := getSession()
    defer session.Close()
    c := session.DB(conf.MongoDB).C("Subscriptions")
    count, _ := c.Find(bson.M{"telegramid" : user.TelegramId, "film_id": id}).Limit(1).Count()
    if (count == 0) {
        mongoError = c.Insert(&subscriptionStruct{
            FilmId: id,
            TelegramId: user.TelegramId})
        if (mongoError == nil) {
            message = "Подписка оформлена"
        } else {
            message = "Ошибка сохранения. Попробуйте еще раз."
        }
    } else {
        message = "Вы уже подписаны на этот фильм"
    }
    return
}

func getSeancesMessage(id string) (message string) {
    film := getFilmInfo(id)
    if (film != nil) {
        if (film.Status == "current") {
            var keys []string
            for k := range film.Schedule {
                keys = append(keys, k)
            }
            sort.Strings(keys)
            for _, k := range keys {
                message = concat(message, "*", dayNames[k], "*\n")
                for _, seaances := range film.Schedule[k] {
                    message = concat(message, seaances.Time, " ", seaances.Price, "\n")
                }
                message = concat(message, "\n")
            }
            message = concat(message, getFooterLinks())
        }else {
            message = "Фильм еще не вышел в прокат."
        }
    } else {
        message = "Фильм не найден"
    }
    return
}

func (film filmStruct) startDate() string {
    t, _ := time.Parse("2006-01-02", film.Result.Start)
    return t.Format("02.01.2006")
}

func (film filmStruct) endDate() string {
    t, _ := time.Parse("2006-01-02", film.Result.End)
    return t.Format("02.01.2006")
}

func (film filmStruct) daysLeft() string {
    start, _ := time.Parse("2006-01-02", film.Result.Start)
    diff := int((start.Unix() - time.Now().Unix()) / 86400)
    days := strconv.Itoa(diff)
    switch diff {
    case 0: days = "Сегодня"
    case 1: days = concat(days, " день")
    case 2, 3, 4: days = concat(days, " дня")
    default: days = concat(days, " дней")
    }
    return days
}

func (user userStruct) subscribe() bool {
    session := getSession()
    defer session.Close()
    c := session.DB(conf.MongoDB).C("Users")
    mongoError = c.Insert(&user)
    if (mongoError == nil) {
        return true
    }
    return false
}

func (user userStruct) isSubscribed() bool {
    session := getSession()
    defer session.Close()
    c := session.DB(conf.MongoDB).C("Users")
    count, _ := c.Find(bson.M{"telegramid" : user.TelegramId}).Limit(1).Count()
    if (count > 0) {
        return true
    }
    return false
}

func getHelpMessage() string {
    return "*Список фильмов:* /all\n*Анонс:* /announcement\n*Напоминания:* /myreminds\n*Помощь:* /help\n*Отменить подписку:* /stop"
}

func (user userStruct) unsubscribe() {
    session := getSession()
    defer session.Close()
    c := session.DB(conf.MongoDB).C("Users")
    c.RemoveAll(bson.M{"telegramid": user.TelegramId})
    c = session.DB(conf.MongoDB).C("Subscriptions")
    c.RemoveAll(bson.M{"telegramid": user.TelegramId})
}

func getReminds() (message string) {
    subscriptions := []subscriptionStruct{}
    films := []filmStruct{}
    filmIds := []string{}
    session := getSession()
    defer session.Close()
    c := session.DB(conf.MongoDB).C("Subscriptions")
    c.Find(bson.M{"telegramid": user.TelegramId}).All(&subscriptions)

    if (len(subscriptions) > 0) {
        for _, subscription := range (subscriptions) {
            filmIds = append(filmIds, subscription.FilmId)
        }
        if (len(filmIds) > 0) {
            c = session.DB(conf.MongoDB).C("Films")
            c.Find(bson.M{"result.id": bson.M{"$in": filmIds}}).All(&films)
            for _, film := range (films) {
                message = concat(message, "*", film.Result.Info.Title, "*\n",
                    "*Премьера:* ", film.startDate(),
                    " *Осталось:* ", film.daysLeft(),
                    "\n*Подробнее:* /info", commandDelimiter, film.Result.ID,
                    " *Отменить напоминание:* ", "/cancel", commandDelimiter, film.Result.ID, "\n\n")
            }
        } else {
            message = "Список напоминаний пуст."
        }
    } else {
        message = "Список напоминаний пуст."
    }
    return
}

func removeNotification(id string) string {
    session := getSession()
    defer session.Close()
    c := session.DB(conf.MongoDB).C("Subscriptions")
    fmt.Println(c.RemoveAll(bson.M{"film_id": id, "telegramid": user.TelegramId}))
    return "Подписка отменена"
}

func initMessage() string {
    return "Для начала общения с ботом используйте команду /start\nДля отменя подписки на рьновления используйте команду /stop"
}


func remindUser(TelegramId int64, film filmStruct){
    session := getSession()
    defer session.Close()
    c := session.DB(conf.MongoDB).C("Subscriptions")
    c.RemoveAll(bson.M{"telegramid" : TelegramId, "film_id": film.Result.ID})
    message := concat("*",film.Result.Title, "* cегодня выходит в прокат\n/info", commandDelimiter, film.Result.ID)
    msg := tgbotapi.NewMessage(TelegramId, message)
    msg.ParseMode = "Markdown"
    bot.Send(msg)
}

func main() {
    bot, err = tgbotapi.NewBotAPI(conf.BotId)
    if err != nil {
        panic(err)
    }
    //if conf.FilmsUpdateFreequency != "" {
    //    c := cron.New()
    //    c.AddFunc(conf.FilmsUpdateFreequency, func() {
    //        updateFilms()
    //    })
    //    c.Start()
    //}


    fmt.Printf("Authorized on account %s\n", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates, _ := bot.GetUpdatesChan(u)

    for update := range updates {
        command := update.Message.Command()
        args := getCommandArguments(update.Message)
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

        user = userStruct{
            Type: update.Message.Chat.Type,
            TelegramId: update.Message.Chat.ID,
            Name: update.Message.Chat.UserName}
        if (user.isSubscribed()) {
            switch command {
            case "all" : msg.Text = getAllMessage()
            case "announcement" : msg.Text = getAnnouncementMessage()
            case "seances" :
                if args != "" {
                    msg.Text = getSeancesMessage(args)
                }
            case "info" :
                if args != "" {
                    msg.Text = getFilmMessage(args)
                }
            case "remind" :
                if args != "" {
                    msg.Text = remindFilm(args)
                }
            case "myreminds" : msg.Text = getReminds()
            case "cancel":
                if args != "" {
                    msg.Text = removeNotification(args)
                }
            case "help" :
                msg.Text = getHelpMessage()
            case "stop":
                user.unsubscribe()
                msg.Text = "Bye message"
            }

            if msg.Text == "" {
                msg.Text = getFooterLinks()
            }
        } else {
            if (command == "start") {
                if (user.subscribe()) {
                    msg.Text = getHelpMessage()
                }else {
                    msg.Text = initMessage()
                }
            }else {
                msg.Text = initMessage()
            }
        }
        msg.ParseMode = "Markdown"
        bot.Send(msg)
    }
}