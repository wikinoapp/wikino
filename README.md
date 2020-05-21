# Nonoto

## Running the app

```
$ git clone git@github.com:nonoto/nonoto.git
$ cd nonoto
$ touch .env.development.local
$ docker-compose up
$ docker-compose exec api bundle exec rails db:setup
```

You should then be able to open [http://localhost:4000](http://localhost:4000) in your browser.
