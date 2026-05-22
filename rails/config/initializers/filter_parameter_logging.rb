# typed: strict
# frozen_string_literal: true

# Be sure to restart your server when you modify this file.

# Configure parameters to be partially matched (e.g. passw matches password) and filtered from the log file.
# Use this to limit dissemination of sensitive information.
# See the ActiveSupport::ParameterFilter documentation for supported notations and behaviors.
#
# ActiveSupport::ParameterFilter performs case-insensitive substring matching
# (internally it builds `Regexp.new(filter, Regexp::IGNORECASE)`), so a short
# token covers many related parameter names regardless of case: `:passw` masks
# `password` / `password_confirmation` / `current_password` / `Password`,
# `:token` masks `csrf_token` / `_csrf_token` / `authenticity_token` /
# `CSRF-TOKEN`, and `:_key` masks `api_key`. The hyphenated Turnstile parameter
# (`cf-turnstile-response`) does not match `:turnstile_response` because
# hyphens and underscores are distinct (case folding does not bridge them), so
# it is listed separately. sentry-rails honors these filters when attaching
# request data to events, so this list is the source of truth for both log
# scrubbing and Sentry scrubbing.
#
# [Ja] ActiveSupport::ParameterFilter は文字列の部分一致 (大文字小文字以外は厳密)
# で動作するため、短いトークン 1 つで大文字小文字を問わず関連パラメータ名を
# まとめてカバーできる。例: `:passw` は `password` / `password_confirmation` /
# `current_password` / `Password`、`:token` は `csrf_token` / `_csrf_token` /
# `authenticity_token` / `CSRF-TOKEN`、`:_key` は `api_key` をマスクする。
# Turnstile のハイフン区切りパラメータ (`cf-turnstile-response`) はハイフンと
# アンダースコアが別文字として扱われる関係で (大文字小文字の畳み込みでも
# 橋渡しされない) `:turnstile_response` ではマッチしないため、別途列挙している。
# sentry-rails はイベントにリクエストデータを添付する際にこの設定を尊重するため、
# Rails ログのスクラブ・Sentry のスクラブの双方の正本としてこのリストを管理する。
Rails.application.config.filter_parameters += [
  :passw, :email, :secret, :token, :_key, :crypt, :salt, :certificate, :otp, :ssn, :cvv, :cvc,
  :turnstile_response, :"cf-turnstile-response"
]
