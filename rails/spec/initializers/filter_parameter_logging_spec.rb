# typed: false
# frozen_string_literal: true

# Specs covering Rails.application.config.filter_parameters. They make the
# implicit substring-matching coverage explicit so that a future deletion of a
# token (e.g. removing `:passw` or `:token`) trips the suite instead of
# silently leaking secrets into logs and Sentry events — sentry-rails honors
# filter_parameters when attaching request data to events, so this list is the
# source of truth for both log scrubbing and Sentry scrubbing.
#
# The masked-key list intentionally includes `email`, which is not in the
# task 5-2 scope but has been part of `filter_parameters` for a long time;
# pinning its behavior here protects that pre-existing filter from being
# regressed alongside the new Turnstile additions.
#
# [Ja] Rails.application.config.filter_parameters の挙動を保証するスペック。
# 部分一致による暗黙のカバレッジを明示化することで、将来トークン
# (例: `:passw` / `:token`) が削除された際にテストで検出できる。sentry-rails は
# リクエストデータをイベントに添付する際にこの設定を尊重するため、本スペックは
# Rails ログ・Sentry イベント双方の機微情報スクラブの正本を担保する。
#
# マスク対象の一覧には作業計画書 タスク 5-2 のスコープ外である `email` を
# 意図的に含めている。`:email` は以前から `filter_parameters` に入っていた
# 既存のフィルタで、Turnstile 系の追加に伴って既存挙動が後退しないよう、
# 本スペックで併せて回帰防止する。
RSpec.describe "config/initializers/filter_parameter_logging.rb" do # rubocop:disable RSpec/DescribeClass
  subject(:filter) { ActiveSupport::ParameterFilter.new(Rails.application.config.filter_parameters) }

  describe "センシティブパラメータのマスク" do
    let(:plain_value) { "leaked-secret" }
    let(:filtered_value) { "[FILTERED]" }

    [
      "password",
      "password_confirmation",
      "current_password",
      "csrf_token",
      "_csrf_token",
      "authenticity_token",
      "turnstile_response",
      "cf-turnstile-response",
      "api_key",
      "secret",
      "token",
      "email"
    ].each do |key|
      it "#{key} を [FILTERED] に置き換えること" do
        expect(filter.filter(key => plain_value)).to eq(key => filtered_value)
      end
    end

    it "良性のキーは変更しないこと" do
      benign = {"id" => 42, "title" => "hello"}
      expect(filter.filter(benign)).to eq(benign)
    end

    it "ネストしたハッシュの中の機微キーもマスクすること" do
      payload = {"user" => {"password" => plain_value, "name" => "alice"}}
      expect(filter.filter(payload)).to eq("user" => {"password" => filtered_value, "name" => "alice"})
    end

    # ActiveSupport::ParameterFilter compiles each filter into a case-insensitive
    # regexp, so request keys coming from headers (typically mixed-case or
    # SCREAMING-CASE) are masked by the same short tokens. Pinning the behavior
    # here guards against a future Rails change that would drop IGNORECASE or a
    # swap to a custom filter implementation.
    #
    # [Ja] ActiveSupport::ParameterFilter は各フィルタを大文字小文字を区別しない
    # 正規表現にコンパイルするため、リクエストヘッダー由来の (大文字混じり・
    # 全大文字の) キーも同じ短いトークンでマスクされる。Rails 側で IGNORECASE
    # が外れたり独自フィルタ実装に差し替えられたりした場合に検知できるよう、
    # この挙動をスペックで固定する。
    it "大文字小文字を区別せずマスクすること" do
      aggregate_failures do
        expect(filter.filter("Password" => plain_value)).to eq("Password" => filtered_value)
        expect(filter.filter("CSRF-TOKEN" => plain_value)).to eq("CSRF-TOKEN" => filtered_value)
        expect(filter.filter("API_KEY" => plain_value)).to eq("API_KEY" => filtered_value)
      end
    end
  end
end
