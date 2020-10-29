# Nonoto

## Running the app

```
$ sudo sh -c "echo '127.0.0.1  nonoto.test' >> /etc/hosts"
$ git clone git@github.com:HakunaLtd/nonoto.git
$ cd nonoto
$ touch .env.development.local
$ docker-compose up
$ docker-compose exec rails bundle exec rails db:setup
```

You should then be able to open [http://nonoto.test:3000](http://nonoto.test:3000) in your browser.
