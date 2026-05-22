# typed: false
# frozen_string_literal: true

require_relative "../../lib/sentry_config"

Sentry.init do |config|
  config.dsn = Wikino.config.sentry_dsn
  config.breadcrumbs_logger = %i[active_support_logger http_logger]

  # Restrict Sentry to production/staging so that a stray DSN in dev or test
  # never leaks events outside of the deployed environments.
  # [Ja] DSN が誤って dev / test に設定されても Sentry に送信されないよう、
  # production / staging のみ送信を有効化する。
  config.enabled_environments = %w[production staging]

  # Fall back to Rails.env when the operator does not override the tag.
  # [Ja] WIKINO_SENTRY_ENVIRONMENT 未指定時は Rails.env を environment タグとして使う。
  config.environment = Wikino.config.sentry_environment.presence || Rails.env

  # Tag events with the deployed asset version so error grouping respects
  # release boundaries. Skip when the version is missing to keep Sentry's
  # auto-detection from being overridden with an empty string.
  # [Ja] デプロイ単位でエラーを分離できるよう、リリースタグにアセットバージョンを設定する。
  # 値が空の場合は Sentry の自動検出を空文字で上書きしないよう設定自体を行わない。
  asset_version = Wikino.config.asset_version.presence
  config.release = asset_version if asset_version

  config.traces_sample_rate = SentryConfig.resolve_traces_sample_rate(Wikino.config.sentry_traces_sample_rate)
  config.profiles_sample_rate = 0.5

  # Pair with the SentryEventFilter below: never auto-attach PII so the scrub
  # only has to defend against accidentally-captured request payloads.
  # [Ja] 後段の SentryEventFilter と合わせて、PII を自動添付しない方針を明示する。
  # 想定外のリクエストペイロード捕捉に対する多層防御として動作する。
  config.send_default_pii = false

  # Drop client-disconnect and malformed-query noise on top of the SDK defaults
  # (ActionController::RoutingError などはデフォルトで既に除外されている)。
  # [Ja] SDK のデフォルト除外に加え、クライアント切断起因のノイズと不正クエリ起因の
  # エラーを除外する。
  config.excluded_exceptions += %w[
    Errno::EPIPE
    Errno::ECONNRESET
    Rack::QueryParser::ParameterTypeError
  ]

  config.before_send = ->(event, hint) { SentryEventFilter.call(event, hint) }
end
