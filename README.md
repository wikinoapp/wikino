# Nonoto API

## Running the app

```
$ sudo sh -c "echo '127.0.0.1  api.nonoto.test' >> /etc/hosts"
$ git clone git@github.com:kiraka/nonoto-api.git
$ cd nonoto
$ touch .env.development.local
$ docker compose up
$ bin/setup
$ bin/rails s
```
