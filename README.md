# Telegram bot for [portalcinema.com.ua](http://portalcinema.com.ua)

Based on go-tgbot library https://github.com/go-telegram-bot-api/telegram-bot-api/tree/v4.2.1

Requirements
===
MongoDB 2.4.9 and higher

The launching of the bot inside the docker environment
===
The procedure description step by step:  

1) Run the MongoDB container and configure it, create user for the bot
```shell
$ docker run -h <mongodb-server-name> --name=<mongodb-server-name> --restart=always --log-driver=json-file --log-opt max-size=100m --log-opt max-file=5 -v /mongo_data:/data/db -d mongo
1b68ad6faa06e97686347d88d8b7aa5f0ee8905089bf6a675fa093ccec27742a

$ docker exec -it <mongodb-server-name> mongo <database_name>
MongoDB shell version: 3.2.6
connecting to: portal_cinema
Server has startup warnings:
2016-05-23T14:21:29.936+0000 I CONTROL  [initandlisten]
2016-05-23T14:21:29.936+0000 I CONTROL  [initandlisten] ** WARNING: /sys/kernel/mm/transparent_hugepage/enabled is 'always'.
2016-05-23T14:21:29.936+0000 I CONTROL  [initandlisten] **        We suggest setting it to 'never'
2016-05-23T14:21:29.936+0000 I CONTROL  [initandlisten]
2016-05-23T14:21:29.936+0000 I CONTROL  [initandlisten] ** WARNING: /sys/kernel/mm/transparent_hugepage/defrag is 'always'.
2016-05-23T14:21:29.936+0000 I CONTROL  [initandlisten] **        We suggest setting it to 'never'
2016-05-23T14:21:29.936+0000 I CONTROL  [initandlisten]
>
> db.createUser({ user: '<user_name>', pwd: '<user_password>', roles: [ { role: "readWrite", db: "<database_name>" } ] });
Successfully added user: {
 "user" : "<user_name>",
 "roles" : [
  {
   "role" : "readWrite",
   "db" : "<database_name>"
  }
 ]
}
> exit
bye
$
```

2) Prepare the configuration file. Copy it from the example and make some changes in it among them the address of **mongodb** server for example how it was set above (<mongodb-server-name>)
```shell
$ cd < project_folder >
$ cp cp conf.json.example conf.json
```

3) Build the the dockerfile that is placed in the project folder
```shell
$ docker build -t <image_name> ./
```

4) Run the bot
```shell
docker run -d -h <container_name> --name=<container_name> --link="<mongodb-server-name>" --restart=always --log-driver=json-file --log-opt max-size=100m --log-opt max-file=5 <image_name>
```
**PS:** You have to substitute text like as **<container_name>** to the some string for example **portal_cinema**.   

To update bot it is required to:

1. Replay step #3
1. Stop the container <container_name> `docker stop <container_name>`
1. Remove it `docker rm <container_name>`
1. Create it again (step #4)
