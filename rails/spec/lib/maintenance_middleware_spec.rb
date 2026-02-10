# typed: false
# frozen_string_literal: true

RSpec.describe MaintenanceMiddleware do
  describe "#call" do
    it "メンテナンスモード無効時、通常のレスポンスを返すこと" do
      app = ->(_env) { [200, {}, ["OK"]] }
      middleware = MaintenanceMiddleware.new(app)
      env = Rack::MockRequest.env_for("/")

      stub_env(
        "WIKINO_MAINTENANCE_MODE" => nil
      ) do
        status, _headers, body = middleware.call(env)

        expect(status).to eq(200)
        expect(body).to eq(["OK"])
      end
    end

    it "メンテナンスモード有効かつ管理者IPのとき、通常のレスポンスを返すこと" do
      app = ->(_env) { [200, {}, ["OK"]] }
      middleware = MaintenanceMiddleware.new(app)
      env = Rack::MockRequest.env_for(
        "/",
        "REMOTE_ADDR" => "192.168.1.1"
      )

      stub_env(
        "WIKINO_MAINTENANCE_MODE" => "on",
        "WIKINO_ADMIN_IP" => "192.168.1.1"
      ) do
        status, _headers, body = middleware.call(env)

        expect(status).to eq(200)
        expect(body).to eq(["OK"])
      end
    end

    it "メンテナンスモード有効かつ一般IPのとき、503を返すこと" do
      app = ->(_env) { [200, {}, ["OK"]] }
      middleware = MaintenanceMiddleware.new(app)
      env = Rack::MockRequest.env_for(
        "/",
        "REMOTE_ADDR" => "192.168.1.100"
      )

      stub_env(
        "WIKINO_MAINTENANCE_MODE" => "on",
        "WIKINO_ADMIN_IP" => "10.0.0.1"
      ) do
        status, headers, body = middleware.call(env)

        expect(status).to eq(503)
        expect(headers["content-type"]).to eq(
          "text/html; charset=utf-8"
        )
        expect(headers["retry-after"]).to be_present
        expect(body.first).to include("メンテナンス中")
      end
    end

    it "メンテナンスモード有効かつヘルスチェックのとき、通常のレスポンスを返すこと" do
      app = ->(_env) { [200, {}, ["OK"]] }
      middleware = MaintenanceMiddleware.new(app)
      env = Rack::MockRequest.env_for("/health")

      stub_env(
        "WIKINO_MAINTENANCE_MODE" => "on",
        "WIKINO_ADMIN_IP" => ""
      ) do
        status, _headers, body = middleware.call(env)

        expect(status).to eq(200)
        expect(body).to eq(["OK"])
      end
    end

    it "複数の管理者IPが設定されているとき、各IPからアクセスできること" do
      app = ->(_env) { [200, {}, ["OK"]] }
      middleware = MaintenanceMiddleware.new(app)

      stub_env(
        "WIKINO_MAINTENANCE_MODE" => "on",
        "WIKINO_ADMIN_IP" => "192.168.1.1, 10.0.0.1"
      ) do
        env1 = Rack::MockRequest.env_for(
          "/",
          "REMOTE_ADDR" => "192.168.1.1"
        )
        status1, = middleware.call(env1)
        expect(status1).to eq(200)

        env2 = Rack::MockRequest.env_for(
          "/",
          "REMOTE_ADDR" => "10.0.0.1"
        )
        status2, = middleware.call(env2)
        expect(status2).to eq(200)
      end
    end

    it "CF-Connecting-IP経由のIP判定が動作すること" do
      app = ->(_env) { [200, {}, ["OK"]] }
      middleware = MaintenanceMiddleware.new(app)
      env = Rack::MockRequest.env_for(
        "/",
        "HTTP_CF_CONNECTING_IP" => "192.168.1.1",
        "REMOTE_ADDR" => "10.0.0.99"
      )

      stub_env(
        "WIKINO_MAINTENANCE_MODE" => "on",
        "WIKINO_ADMIN_IP" => "192.168.1.1"
      ) do
        status, _headers, _body = middleware.call(env)

        expect(status).to eq(200)
      end
    end

    it "X-Forwarded-For経由のIP判定が動作すること" do
      app = ->(_env) { [200, {}, ["OK"]] }
      middleware = MaintenanceMiddleware.new(app)
      env = Rack::MockRequest.env_for(
        "/",
        "HTTP_X_FORWARDED_FOR" => "192.168.1.1, 10.0.0.2",
        "REMOTE_ADDR" => "10.0.0.99"
      )

      stub_env(
        "WIKINO_MAINTENANCE_MODE" => "on",
        "WIKINO_ADMIN_IP" => "192.168.1.1"
      ) do
        status, _headers, _body = middleware.call(env)

        expect(status).to eq(200)
      end
    end

    it "X-Real-IP経由のIP判定が動作すること" do
      app = ->(_env) { [200, {}, ["OK"]] }
      middleware = MaintenanceMiddleware.new(app)
      env = Rack::MockRequest.env_for(
        "/",
        "HTTP_X_REAL_IP" => "192.168.1.1",
        "REMOTE_ADDR" => "10.0.0.99"
      )

      stub_env(
        "WIKINO_MAINTENANCE_MODE" => "on",
        "WIKINO_ADMIN_IP" => "192.168.1.1"
      ) do
        status, _headers, _body = middleware.call(env)

        expect(status).to eq(200)
      end
    end

    it "管理者IP未設定時、すべてのアクセスで503が返ること" do
      app = ->(_env) { [200, {}, ["OK"]] }
      middleware = MaintenanceMiddleware.new(app)
      env = Rack::MockRequest.env_for(
        "/",
        "REMOTE_ADDR" => "192.168.1.1"
      )

      stub_env(
        "WIKINO_MAINTENANCE_MODE" => "on",
        "WIKINO_ADMIN_IP" => nil
      ) do
        status, _headers, _body = middleware.call(env)

        expect(status).to eq(503)
      end
    end

    it "Retry-Afterヘッダーが設定されること" do
      app = ->(_env) { [200, {}, ["OK"]] }
      middleware = MaintenanceMiddleware.new(app)
      env = Rack::MockRequest.env_for("/")

      stub_env(
        "WIKINO_MAINTENANCE_MODE" => "on",
        "WIKINO_ADMIN_IP" => ""
      ) do
        _status, headers, _body = middleware.call(env)

        expect(headers["retry-after"]).to be_present
        retry_time = Time.httpdate(headers["retry-after"])
        expect(retry_time).to be > Time.now.utc
      end
    end

    it "メンテナンスページの内容が正しいこと" do
      app = ->(_env) { [200, {}, ["OK"]] }
      middleware = MaintenanceMiddleware.new(app)
      env = Rack::MockRequest.env_for("/")

      stub_env(
        "WIKINO_MAINTENANCE_MODE" => "on",
        "WIKINO_ADMIN_IP" => ""
      ) do
        _status, _headers, body = middleware.call(env)

        html = body.first
        expect(html).to include("Wikino")
        expect(html).to include("メンテナンス中")
        expect(html).to include("しばらくしてから再度アクセス")
      end
    end
  end

  # ENV操作のヘルパー
  private def stub_env(vars, &block)
    originals = {}
    vars.each do |key, value|
      originals[key] = ENV[key]
      if value.nil?
        ENV.delete(key)
      else
        ENV[key] = value
      end
    end
    block.call
  ensure
    originals.each do |key, original|
      if original.nil?
        ENV.delete(key)
      else
        ENV[key] = original
      end
    end
  end
end
