# typed: false
# frozen_string_literal: true

# Lightweight stand-ins for `Sentry::ErrorEvent` and `Sentry::RequestInterface`.
# Using Struct (instead of OpenStruct) avoids the Ruby 3.5 ostruct default-gem
# warning and keeps the spec free of SDK initialization coupling.
#
# [Ja] `Sentry::ErrorEvent` / `Sentry::RequestInterface` の軽量ダブル。
# Struct を採用することで Ruby 3.5 で警告対象になる ostruct への依存を避け、
# SDK の初期化に依存せずフィルタの挙動だけを検証できるようにしている。
SentrySpecEventDouble = Struct.new(:request, keyword_init: true)
SentrySpecRequestDouble = Struct.new(:data, keyword_init: true)

RSpec.describe "config/initializers/sentry.rb" do # rubocop:disable RSpec/DescribeClass
  let(:config) { Sentry.configuration }

  describe "enabled_environments" do
    it "本番 / ステージング以外では Sentry を有効化しないこと" do
      expect(config.enabled_environments).to eq(%w[production staging])
    end
  end

  describe "send_default_pii" do
    it "リクエストボディや Cookie の自動添付を無効化していること" do
      expect(config.send_default_pii).to be(false)
    end
  end

  describe "environment" do
    it "WIKINO_SENTRY_ENVIRONMENT 未指定時は Rails.env が使われること" do
      expect(config.environment).to eq(Rails.env)
    end
  end

  describe "release" do
    it "WIKINO_ASSET_VERSION 未指定時は release を設定せず SDK の自動検出に委ねること" do
      # `Wikino.config.asset_version.presence` が nil の場合、initializer は
      # `config.release` を一切代入しないため、`Sentry.configuration.release` は
      # SDK の自動検出に依存する。テスト環境では自動検出ソースが無いので nil/空となる。
      # [Ja] asset_version が空のときに release を空文字で上書きしないという
      # 実装判断 (採用しなかった方針も参照) を回帰防止する。
      if Wikino.config.asset_version.present?
        skip "WIKINO_ASSET_VERSION がセットされている環境ではこのケースを検証できない"
      end

      expect(config.release).to be_blank
    end
  end

  describe "traces_sample_rate" do
    it "0.0〜1.0 の範囲に収まること" do
      expect(config.traces_sample_rate).to be_between(0.0, 1.0)
    end

    it "WIKINO_SENTRY_TRACES_SAMPLE_RATE 未指定時は 0.5 (既定値) になること" do
      expect(config.traces_sample_rate).to eq(0.5)
    end
  end

  describe "profiles_sample_rate" do
    it "0.0〜1.0 の範囲に収まること" do
      expect(config.profiles_sample_rate).to be_between(0.0, 1.0)
    end
  end

  describe "excluded_exceptions" do
    it "クライアント切断ノイズ (Errno::EPIPE) を除外していること" do
      expect(config.excluded_exceptions).to include("Errno::EPIPE")
    end

    it "クライアント切断ノイズ (Errno::ECONNRESET) を除外していること" do
      expect(config.excluded_exceptions).to include("Errno::ECONNRESET")
    end

    it "不正クエリ起因のノイズ (Rack::QueryParser::ParameterTypeError) を除外していること" do
      expect(config.excluded_exceptions).to include("Rack::QueryParser::ParameterTypeError")
    end

    it "SDK の既定除外 (ActionController::RoutingError) を保持していること" do
      expect(config.excluded_exceptions).to include("ActionController::RoutingError")
    end
  end

  describe "before_send" do
    let(:before_send) { config.before_send }

    it "lambda として登録されていること" do
      expect(before_send).to respond_to(:call)
    end

    it "password を [FILTERED] に置き換えること" do
      event = build_event(data: {"password" => "super-secret", "email" => "user@example.com"})

      result = before_send.call(event, {})

      expect(result.request.data["password"]).to eq("[FILTERED]")
      expect(result.request.data["email"]).to eq("user@example.com")
    end

    it "password_confirmation を [FILTERED] に置き換えること" do
      event = build_event(data: {"password_confirmation" => "super-secret"})

      result = before_send.call(event, {})

      expect(result.request.data["password_confirmation"]).to eq("[FILTERED]")
    end

    it "csrf_token / authenticity_token を [FILTERED] に置き換えること" do
      event = build_event(data: {
        "csrf_token" => "csrf-abc",
        "authenticity_token" => "auth-xyz"
      })

      result = before_send.call(event, {})

      expect(result.request.data["csrf_token"]).to eq("[FILTERED]")
      expect(result.request.data["authenticity_token"]).to eq("[FILTERED]")
    end

    it "turnstile_response / cf-turnstile-response を [FILTERED] に置き換えること" do
      event = build_event(data: {
        "turnstile_response" => "turnstile-abc",
        "cf-turnstile-response" => "cf-turnstile-xyz"
      })

      result = before_send.call(event, {})

      expect(result.request.data["turnstile_response"]).to eq("[FILTERED]")
      expect(result.request.data["cf-turnstile-response"]).to eq("[FILTERED]")
    end

    it "ネストしたハッシュ内のセンシティブキーも置き換えること" do
      event = build_event(data: {"user" => {"password" => "secret"}})

      result = before_send.call(event, {})

      expect(result.request.data["user"]["password"]).to eq("[FILTERED]")
    end

    it "配列の中のハッシュに含まれるセンシティブキーも置き換えること" do
      event = build_event(data: {"items" => [{"password" => "secret-a"}, {"password" => "secret-b"}]})

      result = before_send.call(event, {})

      expect(result.request.data["items"][0]["password"]).to eq("[FILTERED]")
      expect(result.request.data["items"][1]["password"]).to eq("[FILTERED]")
    end

    it "配列の中のハッシュにさらにネストしたセンシティブキーも置き換えること" do
      event = build_event(data: {"items" => [{"user" => {"password" => "secret"}}]})

      result = before_send.call(event, {})

      expect(result.request.data["items"][0]["user"]["password"]).to eq("[FILTERED]")
    end

    it "request.data が nil でも例外にならないこと" do
      event = build_event(data: nil)

      expect { before_send.call(event, {}) }.not_to raise_error
    end

    it "request 自体がない event でも例外にならないこと" do
      event = SentrySpecEventDouble.new(request: nil)

      expect { before_send.call(event, {}) }.not_to raise_error
    end
  end

  def build_event(data:)
    SentrySpecEventDouble.new(request: SentrySpecRequestDouble.new(data: data))
  end
end
