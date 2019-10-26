# Nonoto

## Running the app

```
$ git clone git@github.com:nonoto/nonoto.git
$ cd nonoto
$ bundle install
$ touch .env.development.local
$ bundle exec rails db:setup
$ docker-compose up --build
$ bundle exec rails s
```

You should then be able to open [http://localhost:3000](http://localhost:3000) in your browser.
