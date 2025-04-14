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

    sig(:final) { returns(T.nilable(Locale)) }
    private def instant_locale
      @instant_locale ||= T.let(Locale.try_deserialize(params[:locale]), T.nilable(Locale))
    end

    sig(:final) { returns(Locale) }
    private def preferred_locale
      preferred_languages = http_accept_language.user_preferred_languages
      # Chrome returns "ja", but Safari would return "ja-JP", not "ja".
      if preferred_languages.present? && preferred_languages.all? { |lang| !lang.match?(/ja/) }
        Locale::En
      else
        default_locale
      end
    end

    sig(:final) { returns(Locale) }
    private def default_locale
      Locale::Ja
    end

    sig(:final) { returns(Locale) }
    private def current_locale
      instant_locale.presence || Current.viewer&.locale.presence || preferred_locale
    end
  end
end
