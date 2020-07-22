# Nonoto

## Running the app

```
$ sudo sh -c "echo '127.0.0.1  nonoto.test' >> /etc/hosts"
$ git clone git@github.com:HakunaLtd/nonoto.git
$ cd nonoto
$ bundle install
$ touch .env.development.local
$ bundle exec rails db:setup
$ docker-compose up --build
$ bundle exec rails s -p 3001
```

You should then be able to open [http://nonoto.test:3001](http://nonoto.test:3001) in your browser.
