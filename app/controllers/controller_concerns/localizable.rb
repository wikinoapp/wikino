# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Localizable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { params(action: Proc).returns(T.untyped) }
    def set_locale(&action)
      I18n.with_locale(current_locale.serialize, &action)
    end

    sig(:final) { returns(T.nilable(UserLocale)) }
    private def instant_locale
      @instant_locale ||= T.let(UserLocale.try_deserialize(params[:locale]), T.nilable(UserLocale))
    end

    sig(:final) { returns(UserLocale) }
    private def preferred_locale
      preferred_languages = http_accept_language.user_preferred_languages
      # Chrome returns "ja", but Safari would return "ja-JP", not "ja".
      (preferred_languages.present? && preferred_languages.all? { |lang| !lang.match?(/ja/) }) ? UserLocale::En : default_locale
    end

    sig(:final) { returns(UserLocale) }
    private def default_locale
      UserLocale::Ja
    end

    sig(:final) { returns(UserLocale) }
    private def current_locale
      instant_locale.presence || Current.user&.deserialized_locale.presence || preferred_locale
    end
  end
end
