# typed: false
# frozen_string_literal: true

# SentryEventFilter scrubs sensitive parameters from Sentry events before they
# leave the application. Acts as defense in depth on top of Sentry's
# `send_default_pii = false` so that even if request data is captured the
# obvious secrets (passwords, CSRF tokens, Turnstile responses) are masked.
#
# [Ja] Sentry へ送信される直前のイベントから機密パラメータを除去するフィルタ。
# `send_default_pii = false` の挙動と併用して、平文パスワード・CSRF トークン・
# Turnstile レスポンスといった機微情報が Sentry の管理画面に残らないように
# 多層防御として動作する。
class SentryEventFilter
  SENSITIVE_KEYS = %w[
    password
    password_confirmation
    csrf_token
    _csrf_token
    authenticity_token
    turnstile_response
    cf-turnstile-response
  ].freeze

  FILTERED_VALUE = "[FILTERED]"

  def self.call(event, _hint = nil)
    return event unless event.respond_to?(:request)

    request = event.request
    return event unless request

    filter_request!(request)
    event
  end

  def self.filter_request!(request)
    data = request.respond_to?(:data) ? request.data : nil
    filter_hash!(data) if data.is_a?(Hash)
  end

  def self.filter_hash!(hash)
    hash.each do |key, value|
      if SENSITIVE_KEYS.include?(key.to_s.downcase)
        hash[key] = FILTERED_VALUE
      else
        filter_value!(value)
      end
    end
  end

  # Recurse into Hash and Array values so that sensitive keys nested inside
  # collections of objects (e.g. `{"items" => [{"password" => "x"}]}`) are
  # also masked. Non-collection values are left untouched.
  #
  # [Ja] ハッシュや配列の中にネストしたセンシティブキー
  # (例: `{"items" => [{"password" => "x"}]}`) もマスクできるよう、両方の
  # コレクションに再帰する。コレクション以外の値は変更しない。
  def self.filter_value!(value)
    case value
    when Hash
      filter_hash!(value)
    when Array
      value.each { |element| filter_value!(element) }
    end
  end
end
